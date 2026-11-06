package tsbridge

import (
	"context"
	"slices"
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
)

// PreEmitDiagnostics mirrors ts.getPreEmitDiagnostics(program, sourceFile), which is what compileFiles.ts
// calls per source file: config, options, this file's syntax, globals, this file's semantics, and declaration
// diagnostics when declaration emit is on. Program.GetSemanticDiagnostics applies @ts-ignore and
// @ts-expect-error, and includes bind diagnostics. Package .d.ts files are not checked unless passed in.
func (p *Project) PreEmitDiagnostics(ctx context.Context, file *ast.SourceFile) []*ast.Diagnostic {
	p.assertNotInWorkerPhase()
	// Check first, because the global diagnostics of a checker can grow during checking.
	p.program.GetSemanticDiagnostics(ctx, file)
	return p.preEmitDiagnostics(ctx, file, p.programWideDiagnostics(ctx))
}

// PreEmitDiagnosticsForFiles returns PreEmitDiagnostics for each file, checking in parallel with one goroutine
// per checker. Results are per input file, so program-wide diagnostics repeat for each file, as they do when
// compileFiles.ts collects them file by file. Concatenating in input order gives the same list as rbxtsc.
func (p *Project) PreEmitDiagnosticsForFiles(ctx context.Context, files []*ast.SourceFile) [][]*ast.Diagnostic {
	p.assertNotInWorkerPhase()
	var wg sync.WaitGroup
	for _, group := range p.groupByChecker(ctx, files) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, file := range group.files {
				// This only forces the check. The result is read again below, from the checker's cache.
				p.program.GetSemanticDiagnostics(ctx, file)
			}
		}()
	}
	wg.Wait()
	global := p.programWideDiagnostics(ctx)
	results := make([][]*ast.Diagnostic, len(files))
	for i, file := range files {
		results[i] = p.preEmitDiagnostics(ctx, file, global)
	}
	return results
}

// programWideDiagnostics returns the parts that don't depend on the file.
func (p *Project) programWideDiagnostics(ctx context.Context) []*ast.Diagnostic {
	return slices.Concat(
		p.program.GetConfigFileParsingDiagnostics(),
		p.program.GetProgramDiagnostics(),
		p.program.GetGlobalDiagnostics(ctx),
	)
}

func (p *Project) preEmitDiagnostics(ctx context.Context, file *ast.SourceFile, global []*ast.Diagnostic) []*ast.Diagnostic {
	all := slices.Concat(global, p.program.GetSyntacticDiagnostics(ctx, file), p.program.GetSemanticDiagnostics(ctx, file))
	if p.Options().GetEmitDeclarations() {
		all = append(all, p.program.GetDeclarationDiagnostics(ctx, file)...)
	}
	return compiler.SortAndDeduplicateDiagnostics(FilterDiagnostics(all, p.Options()))
}

// FilterDiagnostics removes diagnostics that TS 7 reports and rbxtsc doesn't.
//
// TS2441 on `Object` is new in TS 7 and means nothing for Luau output. TS2441 on `exports` and `require`
// stays: TS 5.5.3 reports those under CommonJS as well, so rbxtsc rejects them.
//
// With rootDirs and no rootDir, TS 6 and later add TS5011 and TS6059, because they default rootDir to the
// tsconfig's folder. roblox-ts accepts rootDirs on its own, and TS 5.5.3 reports neither. See 0004.
func FilterDiagnostics(all []*ast.Diagnostic, options *core.CompilerOptions) []*ast.Diagnostic {
	return slices.DeleteFunc(slices.Clone(all), func(d *ast.Diagnostic) bool {
		switch d.Code() {
		case diagnostics.Duplicate_identifier_0_Compiler_reserves_name_1_in_top_level_scope_of_a_module.Code():
			args := d.MessageArgs()
			return len(args) > 0 && args[0] == "Object"
		case diagnostics.The_common_source_directory_of_0_is_1_The_rootDir_setting_must_be_explicitly_set_to_this_or_another_path_to_adjust_your_output_s_file_layout.Code(),
			diagnostics.File_0_is_not_under_rootDir_1_rootDir_is_expected_to_contain_all_source_files.Code():
			return options.RootDir == ""
		}
		return false
	})
}

func (p *Project) assertNotInWorkerPhase() {
	if p.workersRunning.Load() {
		panic("tsbridge: diagnostics must not run while workers are running, because workers hold no checker lock")
	}
}
