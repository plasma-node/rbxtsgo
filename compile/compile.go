// Package compile is the port of src/Project/functions/compileFiles.ts after the checks that project.PrepareCompile
// does: pre-emit diagnostics, transforming in parallel, and writing the results.
//
// Upstream handles files one at a time. It adds a file's pre-emit diagnostics, stops transforming as soon as any error
// exists, and otherwise transforms the file. So file i is transformed only if files 0..i have no pre-emit errors and
// files 0..i-1 have no transform errors. This port transforms in parallel with one worker per checker, keeps each
// file's results apart, and merges them in file order under that same rule. Which diagnostics get reported then doesn't
// depend on scheduling. See the notes on Run for the two pieces of per-compile state this affects.
//
// Declaration emit is in declarations.go, and emitBuildInfo is in incremental.go.
package compile

import (
	"context"
	"fmt"
	"os"
	"time"
	"unicode/utf8"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/pathtranslator"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/transformer"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

// Mode says how much state one Run shares between files.
type Mode int

const (
	// ModeWhole is one compileFiles call with every file. An error stops later files from being transformed.
	ModeWhole Mode = iota
	// ModeIsolated is one compileFiles call per file on one program, as upstream's src/CLI/test.ts does. Each file gets
	// a new MultiTransformState, and an error in one file doesn't affect the others.
	ModeIsolated
)

// FileResult is what one file produced.
type FileResult struct {
	File *ast.SourceFile
	// Source is the rendered Luau, with lone surrogates written the way JS writes them. Check HasSource.
	Source    string
	HasSource bool
	// Diagnostics holds this file's pre-emit diagnostics followed by the ones from transforming it, as far as upstream's
	// stopping rule lets them through.
	Diagnostics []*shared.Diagnostic
	// Panic holds an internal compiler error from transforming this file: an upstream assert or a port bug.
	Panic string
}

// Context is what a compile needs that doesn't change between Run calls.
type Context struct {
	Project    *tsbridge.Project
	Data       *shared.ProjectData
	Translator *pathtranslator.Translator
	Prepared   *project.CompileContext
	// Replay holds the accumulated text of context diagnostics. It must outlive one Run, as in upstream.
	Replay *shared.ContextReplay
	// Times receives where the time goes, when it's set.
	Times *Times
}

type transformed struct {
	source      string
	hasSource   bool
	diagnostics []*shared.Diagnostic
	panicText   string
	ran         bool
}

// Run transforms files and returns a result per file, in input order.
//
// Two pieces of state are per worker here and global upstream. The once-per-symbol diagnostic caches can't lose or
// repeat anything that gets reported: reporting one means an error, the merge drops everything after the first file
// with an error, and no earlier file in any worker can have set the cache. And context diagnostics accumulate text in
// the order they're created, so the merge rebuilds that text in file order with Context.Replay.
func Run(ctx context.Context, c *Context, files []*ast.SourceFile, mode Mode) ([]*FileResult, error) {
	p := c.Project
	options := p.Options()

	times := c.Times
	if times == nil {
		times = &Times{}
	}
	phase := time.Now()
	preEmit := make([][]*shared.Diagnostic, len(files))
	for i, list := range p.PreEmitDiagnosticsForFiles(ctx, files) {
		preEmit[i] = append(shared.FromTSList(list), project.FileUsesCommentDirectives(c.Data, files[i])...)
	}

	times.Check = time.Since(phase)
	times.FilesChecked = len(files)
	phase = time.Now()

	// In whole mode, no file at or after the first pre-emit error gets transformed.
	limit := len(files)
	if mode == ModeWhole {
		for i, list := range preEmit {
			if shared.HasErrors(list) {
				limit = i
				break
			}
		}
	}
	index := make(map[*ast.SourceFile]int, len(files))
	for i, file := range files {
		index[file] = i
	}

	results := make([]transformed, len(files))
	stateOptions := transformer.StateOptions{
		Program:        p.Program(),
		Data:           c.Data,
		PathTranslator: c.Translator,
		Compile:        c.Prepared,
		Options:        options,
	}
	var macroErr error
	workers := p.NewWorkers(ctx, files, tsbridge.DefaultMacroSpec())
	macroErrs := make([]error, len(workers))
	workerIndex := make(map[*tsbridge.Worker]int, len(workers))
	for i, w := range workers {
		workerIndex[w] = i
	}
	p.RunWorkers(workers, func(w *tsbridge.Worker) {
		macros, err := w.Macros()
		if err != nil {
			macroErrs[workerIndex[w]] = err
			return
		}
		services := &transformer.TransformServices{MacroManager: macros}
		multi := transformer.NewMultiTransformState()
		for _, file := range w.Files() {
			i := index[file]
			if i >= limit {
				continue
			}
			if mode == ModeIsolated {
				multi = transformer.NewMultiTransformState()
				if shared.HasErrors(preEmit[i]) {
					continue
				}
			}
			result := &results[i]
			transformOne(ctx, w, stateOptions, services, multi, file, result)
			// Nothing after a failed file can be reported, so don't spend time on it.
			if mode == ModeWhole && (result.panicText != "" || shared.HasErrors(result.diagnostics)) {
				break
			}
		}
	})
	for _, err := range macroErrs {
		if err != nil {
			macroErr = err
			break
		}
	}
	if macroErr != nil {
		return nil, shared.NewProjectError(macroErr.Error())
	}

	times.Lower = time.Since(phase)
	for i := range results {
		if results[i].ran {
			times.FilesLowered++
		}
	}

	// Merge in file order.
	output := make([]*FileResult, len(files))
	stopped := false
	for i, file := range files {
		result := &FileResult{File: file, Diagnostics: preEmit[i]}
		output[i] = result
		if mode == ModeWhole {
			if stopped {
				continue
			}
			if shared.HasErrors(preEmit[i]) {
				stopped = true
				continue
			}
		} else if shared.HasErrors(preEmit[i]) {
			c.Replay.Apply(result.Diagnostics)
			continue
		}
		t := &results[i]
		result.Panic = t.panicText
		result.Diagnostics = append(result.Diagnostics, t.diagnostics...)
		c.Replay.Apply(result.Diagnostics)
		if t.panicText != "" || shared.HasErrors(t.diagnostics) {
			stopped = true
			continue
		}
		result.Source = t.source
		result.HasSource = t.hasSource
	}
	// compileFiles writes nothing when any error was reported, including files that compiled before it.
	if mode == ModeWhole && stopped {
		for _, result := range output {
			result.Source = ""
			result.HasSource = false
		}
	}
	return output, nil
}

func transformOne(ctx context.Context, w *tsbridge.Worker, options transformer.StateOptions, services *transformer.TransformServices, multi *transformer.MultiTransformState, file *ast.SourceFile, result *transformed) {
	result.ran = true
	service := &shared.DiagnosticService{}
	defer func() {
		result.diagnostics = service.Flush()
		if r := recover(); r != nil {
			result.panicText = panicText(r)
		}
	}()
	state := transformer.NewTransformState(ctx, w, options, services, multi, service, file)
	statements := transformer.TransformSourceFile(state, file)
	if service.HasErrors() {
		return
	}
	result.source = WriteableText(luau.RenderAST(statements))
	result.hasSource = true
}

func panicText(r any) string {
	switch v := r.(type) {
	case string:
		return v
	case error:
		return v.Error()
	}
	return fmt.Sprint(r)
}

// WriteableText turns WTF-8 into what JS writes. A Go string can hold a lone surrogate from an escape such as "\uD83D",
// as the bytes ED A0 BD. JS holds UTF-16 units: two adjacent surrogates that form a pair are one character, and when
// fs.outputFileSync encodes a lone one, it writes U+FFFD. Everything else is left as it is.
func WriteableText(text string) string {
	isSurrogate := func(i int) bool {
		return i+2 < len(text) && text[i] == 0xED && text[i+1] >= 0xA0 && text[i+1] <= 0xBF
	}
	found := false
	for i := range len(text) {
		if isSurrogate(i) {
			found = true
			break
		}
	}
	if !found {
		return text
	}
	unit := func(i int) rune {
		return 0xD000 | rune(text[i+1]&0x3F)<<6 | rune(text[i+2]&0x3F)
	}
	out := make([]byte, 0, len(text))
	for i := 0; i < len(text); {
		if !isSurrogate(i) {
			out = append(out, text[i])
			i++
			continue
		}
		high := unit(i)
		if high < 0xDC00 && isSurrogate(i+3) {
			if low := unit(i + 3); low >= 0xDC00 {
				out = utf8.AppendRune(out, 0x10000+(high-0xD800)<<10+(low-0xDC00))
				i += 6
				continue
			}
		}
		out = append(out, 0xEF, 0xBF, 0xBD)
		i += 3
	}
	return string(out)
}

// EmitResult is ts.EmitResult, as compileFiles returns it.
type EmitResult struct {
	EmittedFiles []string
	EmitSkipped  bool
	Diagnostics  []*shared.Diagnostic
}

// CompileFiles is the rest of compileFiles, for one whole compile. It writes the output files in file order.
func CompileFiles(ctx context.Context, c *Context, files []*ast.SourceFile) (*EmitResult, error) {
	shared.LogService.WriteLineIfVerbose("compiling as " + string(c.Prepared.ProjectType) + "..")
	results, err := Run(ctx, c, files, ModeWhole)
	if err != nil {
		return nil, err
	}

	var all []*shared.Diagnostic
	for _, result := range results {
		if result.Panic != "" {
			return nil, fmt.Errorf("internal error while compiling %s: %s", result.File.FileName().AsString(), result.Panic)
		}
		all = append(all, result.Diagnostics...)
	}
	if shared.HasErrors(all) {
		return &EmitResult{EmitSkipped: true, Diagnostics: all}, nil
	}

	phase := time.Now()
	emitted, err := WriteResults(c, results)
	if c.Times != nil {
		c.Times.Write = time.Since(phase)
	}
	if err == nil && c.Project.Options().Declaration.IsTrue() {
		phase = time.Now()
		var declarations []string
		declarations, err = EmitDeclarations(ctx, c, files)
		if c.Times != nil {
			c.Times.Declarations = time.Since(phase)
		}
		// Upstream's emittedFiles has only the Luau, because ts.sys.writeFile writes the declarations. Watch's report
		// of what it wrote wants both, and nothing else reads them.
		emitted = append(emitted, declarations...)
	}
	if err != nil {
		return nil, err
	}
	return &EmitResult{EmittedFiles: emitted, Diagnostics: all}, nil
}

// WriteResults writes each result that has source, in order, and returns the paths it wrote.
func WriteResults(c *Context, results []*FileResult) ([]string, error) {
	var emitted []string
	for _, result := range results {
		if !result.HasSource {
			continue
		}
		outPath := c.Translator.GetOutputPath(nodepath.Native.Normalize(result.File.FileName().AsString()))
		if c.Data.ProjectOptions.WriteOnlyChanged {
			if current, err := os.ReadFile(outPath); err == nil && string(current) == result.Source {
				continue
			}
		}
		// fs.outputFileSync makes the folder and writes the bytes it was given.
		if err := os.MkdirAll(nodepath.Native.Dirname(outPath), 0o777); err != nil {
			return emitted, err
		}
		if err := os.WriteFile(outPath, []byte(result.Source), 0o666); err != nil {
			return emitted, err
		}
		emitted = append(emitted, outPath)
	}
	return emitted, nil
}

func failure(message string) *EmitResult {
	return &EmitResult{EmitSkipped: true, Diagnostics: []*shared.Diagnostic{shared.TextDiagnostic(message, diagnostics.CategoryError)}}
}
