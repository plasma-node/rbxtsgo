// Command tsgo-check is the rbxtsgo feasibility spike. It links TypeScript 7's compiler,
// loads a roblox-ts tsconfig, and exercises the checker queries the transformer needs.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/diagnosticwriter"
	"github.com/microsoft/TypeScript/tsc/internal/execute/tsc"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
)


type options struct {
	noNormalize bool
	checkers    int
	evidence    string
	resolutions string
	diagnostics string
	lockTest    bool
	watchSim    string
	watchRuns   int
}

func main() {
	var o options
	flag.BoolVar(&o.noNormalize, "no-normalize", false, "skip the in-memory tsconfig normalization")
	flag.IntVar(&o.checkers, "checkers", 4, "number of checkers in the pool")
	flag.StringVar(&o.evidence, "evidence", "", "write per-node evidence to this file (default stdout)")
	flag.StringVar(&o.resolutions, "dump-resolutions", "", "write the file list and module resolutions to this file")
	flag.StringVar(&o.diagnostics, "dump-diagnostics", "", "write diagnostics to this file")
	flag.BoolVar(&o.lockTest, "lock-test", false, "test emit resolver calls while holding the checker lock")
	flag.StringVar(&o.watchSim, "watch-sim", "", "time rebuilds after an in-memory edit of this source file")
	flag.IntVar(&o.watchRuns, "watch-runs", 7, "repetitions for -watch-sim")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: tsgo-check [flags] <tsconfig.json>")
		os.Exit(2)
	}
	if err := run(context.Background(), flag.Arg(0), o); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath string, o options) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cwd = tspath.NormalizePath(cwd)
	rootedCwd := tspath.RootedDirectoryPathFromAbsolute(cwd)
	fs := bundled.WrapFS(osvfs.FS())
	absConfig := tspath.ToRootedFilePath(filepath.ToSlash(configPath), rootedCwd)
	projectDir := absConfig.Directory()

	start := time.Now()
	extendedCache := &tsc.ExtendedConfigCache{}
	config, configDiags := tsoptions.GetParsedCommandLineOfConfigFile(absConfig, nil, nil, fs, extendedCache)
	if config == nil {
		printDiagnostics(os.Stdout, configDiags, projectDir)
		return fmt.Errorf("could not parse %s", absConfig)
	}
	fmt.Printf("config errors from the parser before normalization: %d\n", len(configDiags))
	printDiagnostics(os.Stdout, configDiags, projectDir)

	if !o.noNormalize {
		fmt.Println("normalization:")
		for _, note := range normalize(config) {
			fmt.Println("  " + note)
		}
	}
	checkers := o.checkers
	opts := *config.CompilerOptions()
	opts.Checkers = &checkers
	config.SetCompilerOptions(&opts)

	compilerHost := compiler.NewCachedFSCompilerHost(fs, bundled.LibPath(), extendedCache, nil, nil)
	program := compiler.NewProgram(compiler.ProgramOptions{Config: config, Host: compilerHost})
	fmt.Printf("program: %d files, %d root files, in %s\n", len(program.SourceFiles()), len(config.FileNames()), time.Since(start).Round(time.Millisecond))

	if o.resolutions != "" {
		if err := dumpResolutions(program, projectDir, o.resolutions); err != nil {
			return err
		}
	}

	diagStart := time.Now()
	diags := collectDiagnostics(ctx, program)
	fmt.Printf("diagnostics: %d in %s\n", len(diags), time.Since(diagStart).Round(time.Millisecond))
	printUserDiagnostics(diags, projectDir)
	if o.diagnostics != "" {
		f, err := os.Create(o.diagnostics)
		if err != nil {
			return err
		}
		defer f.Close()
		printDiagnostics(f, diags, projectDir)
	}
	if o.noNormalize {
		return nil
	}

	if o.lockTest {
		runLockTest(program)
		return nil
	}
	if o.watchSim != "" {
		return runWatchSim(ctx, config, rootedCwd, fs, extendedCache, o.watchSim, o.watchRuns)
	}

	report := runWalk(ctx, program, projectDir)
	if o.evidence != "" {
		if err := os.WriteFile(o.evidence, []byte(report.evidence), 0o644); err != nil {
			return err
		}
	} else {
		fmt.Print(report.evidence)
	}
	fmt.Print(report.summary())
	runMultiCheckerReport(ctx, program)
	checkLinkname(program)
	return nil
}

func collectDiagnostics(ctx context.Context, program *compiler.Program) []*ast.Diagnostic {
	all := slices.Clone(program.GetConfigFileParsingDiagnostics())
	all = append(all, program.GetProgramDiagnostics()...)
	all = append(all, program.GetGlobalDiagnostics(ctx)...)
	for _, file := range program.SourceFiles() {
		all = append(all, program.GetSyntacticDiagnostics(ctx, file)...)
		all = append(all, program.GetBindDiagnostics(ctx, file)...)
		all = append(all, program.GetSemanticDiagnostics(ctx, file)...)
	}
	return compiler.SortAndDeduplicateDiagnostics(all)
}

func printDiagnostics(w io.Writer, diags []*ast.Diagnostic, projectDir tspath.RootedDirectoryPath) {
	diagnosticwriter.WriteFormatDiagnostics(w, diagnosticwriter.FromASTDiagnostics(diags), &diagnosticwriter.FormattingOptions{
		NewLine:             "\n",
		CaseSensitivity:  tspath.CaseSensitive,
		CurrentDirectory: projectDir,
	})
}

// printUserDiagnostics prints diagnostics outside node_modules and a count of the rest.
func printUserDiagnostics(diags []*ast.Diagnostic, projectDir tspath.RootedDirectoryPath) {
	var user []*ast.Diagnostic
	library := 0
	for _, d := range diags {
		if d.File() != nil && strings.Contains(d.File().FileName().AsString(), "/node_modules/") {
			library++
		} else {
			user = append(user, d)
		}
	}
	printDiagnostics(os.Stdout, user, projectDir)
	fmt.Printf("  outside node_modules: %d, inside node_modules: %d\n", len(user), library)
}

func userFiles(program *compiler.Program) []*ast.SourceFile {
	var files []*ast.SourceFile
	for _, file := range program.SourceFiles() {
		if !file.IsDeclarationFile && !strings.Contains(file.FileName().AsString(), "/node_modules/") {
			files = append(files, file)
		}
	}
	slices.SortFunc(files, func(a, b *ast.SourceFile) int { return a.FileName().Compare(b.FileName()) })
	return files
}

func relative(projectDir tspath.RootedDirectoryPath, path string) string {
	return tspath.ConvertToRelativePath(path, projectDir, tspath.CaseSensitive)
}

func dumpResolutions(program *compiler.Program, projectDir tspath.RootedDirectoryPath, out string) error {
	var lines []string
	for _, file := range program.SourceFiles() {
		name := relative(projectDir, file.FileName().AsString())
		lines = append(lines, "file\t"+name)
		for _, spec := range file.Imports() {
			resolved := "UNRESOLVED"
			if m := program.GetResolvedModuleFromModuleSpecifier(file, spec); m != nil && m.ResolvedFileName != "" {
				resolved = relative(projectDir, m.ResolvedFileName.AsString())
			}
			lines = append(lines, "import\t"+name+"\t"+spec.Text()+"\t"+resolved)
		}
	}
	slices.Sort(lines)
	lines = slices.Compact(lines)
	return os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
