package plugins

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/internal/sourcemap"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/pathtranslator"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

// Applied is what's left to compile after the plugin step of compileFiles.
type Applied struct {
	// Project and Files are what to compile: the reprinted program, or the input when no plugin ran.
	Project *tsbridge.Project
	Files   []*ast.SourceFile
	// Exit is set when the plugin ended the process, with the lines it wrote to stdout. Nothing else happens then.
	Exit []string
	// Printed holds each file's reprinted text, by file name.
	Printed map[string]string

	maps map[*ast.SourceFile]*reprintMap
}

// Stats says how much the plugin step asked of the first program.
type Stats struct {
	// Candidates is how many files have a property access that might be an accessor. Only these need a checker.
	Candidates int
	// Filtered is false when a computed accessor name or a mapped type's `as` clause turned the name filter off.
	Filtered bool
	Names    int
}

// Apply is the "running transformers" part of compileFiles. Like JS, it transforms using the first program, prints
// every file, and makes a new program from the printed text, so that everything after this sees printed source.
//
// The first program is never fully checked. See docs/setget.md.
func Apply(ctx context.Context, p *tsbridge.Project, decision Decision, files []*ast.SourceFile) (*Applied, error) {
	result, _, err := ApplyWithStats(ctx, p, decision, files)
	return result, err
}

// ApplyWithStats is Apply, and also says what it asked of the first program.
func ApplyWithStats(ctx context.Context, p *tsbridge.Project, decision Decision, files []*ast.SourceFile) (*Applied, Stats, error) {
	applied := &Applied{Project: p, Files: files}
	var stats Stats
	if decision.Action != ActionSetget {
		return applied, stats, nil
	}

	names := collectAccessorNames(p)
	stats.Filtered = !names.all
	stats.Names = len(names.names)

	outcomes := make([]fileOutcome, len(files))
	index := make(map[*ast.SourceFile]int, len(files))
	var needChecker, noChecker []*ast.SourceFile
	uses := make([]bool, len(files))
	if !names.empty() {
		forEachParallel(files, func(i int, file *ast.SourceFile) { uses[i] = fileUsesName(file, names) })
	}
	for i, file := range files {
		index[file] = i
		if uses[i] {
			needChecker = append(needChecker, file)
		} else {
			noChecker = append(noChecker, file)
		}
	}
	stats.Candidates = len(needChecker)

	// Each worker goes through its files in order and stops at a plugin exit. So the first exit in file order always
	// gets found, whatever the scheduling: every file before it runs to completion.
	if len(needChecker) > 0 {
		workers := p.NewWorkers(ctx, needChecker, tsbridge.DefaultMacroSpec())
		p.RunWorkers(workers, func(w *tsbridge.Worker) {
			for _, file := range w.Files() {
				outcome := &outcomes[index[file]]
				transformFile(file, w.Checker(), decision.Prefix, names, outcome)
				if outcome.exit != nil {
					return
				}
			}
		})
	}
	forEachParallel(noChecker, func(_ int, file *ast.SourceFile) {
		transformFile(file, nil, decision.Prefix, names, &outcomes[index[file]])
	})

	for i := range outcomes {
		if outcomes[i].exit != nil {
			applied.Exit = outcomes[i].exit
			return applied, stats, nil
		}
		if outcomes[i].panicText != "" {
			return nil, stats, fmt.Errorf("internal error while transforming %s: %s", files[i].FileName().AsString(), outcomes[i].panicText)
		}
	}

	printed := make(map[tspath.PathKey]string, len(files))
	applied.Printed = make(map[string]string, len(files))
	for i, file := range files {
		printed[file.PathKey()] = outcomes[i].text
		applied.Printed[file.FileName().AsString()] = outcomes[i].text
	}
	next := p.WithPrintedFiles(printed)
	applied.Project = next
	applied.maps = make(map[*ast.SourceFile]*reprintMap, len(files))
	applied.Files = make([]*ast.SourceFile, len(files))
	byPath := next.Program().FilesByPath()
	for i, file := range files {
		printedFile := byPath[file.PathKey()]
		if printedFile == nil {
			return nil, stats, fmt.Errorf("internal error: %s is missing from the reprinted program", file.FileName().AsString())
		}
		applied.Files[i] = printedFile
		applied.maps[printedFile] = &reprintMap{original: file, generator: outcomes[i].generator}
	}
	return applied, stats, nil
}

type fileOutcome struct {
	text      string
	generator *sourcemap.Generator
	exit      []string
	panicText string
}

func transformFile(file *ast.SourceFile, c *checker.Checker, prefix string, names *accessorNames, outcome *fileOutcome) {
	defer func() {
		switch r := recover().(type) {
		case nil:
		case *pluginExit:
			outcome.exit = r.lines
		default:
			outcome.panicText = fmt.Sprint(r)
		}
	}()
	context := newTransformContext(file, c, prefix, names)
	transformed := context.transformFile()
	// ts.createPrinter() has no options. That means no target, which is why numeric separators don't survive: 1_000
	// prints as 1000.
	options := printer.PrinterOptions{NewLine: core.NewLineKindLF, KeepParseTreeParentheses: true}
	writer := printer.NewTextWriter("\n", 0)
	generator := sourcemap.NewGenerator("", "", file.FileName().Directory(), tspath.CaseSensitive)
	printer.NewPrinter(options, printer.PrintHandlers{}, context.emit).Write(transformed.AsNode(), transformed, writer, generator)
	outcome.text = writer.String()
	outcome.generator = generator
}

// collectAccessorNames walks every file in the program, including declaration files.
func collectAccessorNames(p *tsbridge.Project) *accessorNames {
	files := p.Program().SourceFiles()
	facts := make([]fileAccessorFacts, len(files))
	forEachParallel(files, func(i int, file *ast.SourceFile) { facts[i] = scanAccessors(file) })
	result := &accessorNames{names: map[string]bool{}}
	mapped := false
	computed := false
	for _, fact := range facts {
		for _, name := range fact.names {
			result.names[name] = true
		}
		mapped = mapped || fact.mappedTypeName
		computed = computed || fact.computed
	}
	result.all = computed || (mapped && len(result.names) > 0)
	return result
}

func forEachParallel(files []*ast.SourceFile, fn func(i int, file *ast.SourceFile)) {
	var wg sync.WaitGroup
	gate := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i, file := range files {
		wg.Add(1)
		gate <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-gate }()
			fn(i, file)
		}()
	}
	wg.Wait()
}

// WriteTransformedFiles is the writeTransformedFiles option. JS writes these right after printing, in file order.
func (a *Applied) WriteTransformedFiles(data *shared.ProjectData, translator *pathtranslator.Translator) error {
	if !data.ProjectOptions.WriteTransformedFiles || a.Printed == nil {
		return nil
	}
	for _, file := range a.Files {
		outPath := translator.GetOutputTransformedPath(nodepath.Native.Normalize(file.FileName().AsString()))
		if err := os.MkdirAll(nodepath.Native.Dirname(outPath), 0o777); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte(a.Printed[file.FileName().AsString()]), 0o666); err != nil {
			return err
		}
	}
	return nil
}

// reprintMap maps a position in a printed file back to the source file it came from.
type reprintMap struct {
	original  *ast.SourceFile
	generator *sourcemap.Generator
	once      sync.Once
	mappings  []sourcemap.Mapping
}

func (m *reprintMap) load() {
	m.once.Do(func() {
		decoder := sourcemap.DecodeMappings(m.generator.RawSourceMap().Mappings)
		for mapping := range decoder.Values() {
			if mapping.SourceIndex != sourcemap.MissingSource {
				m.mappings = append(m.mappings, *mapping)
			}
		}
		sort.SliceStable(m.mappings, func(i, j int) bool {
			a, b := &m.mappings[i], &m.mappings[j]
			return a.GeneratedLine < b.GeneratedLine || a.GeneratedLine == b.GeneratedLine && a.GeneratedCharacter < b.GeneratedCharacter
		})
	})
}

// position finds the original position for a printed one: the last mapping at or before it. Every node with a place in
// the source writes a mapping at its first token, so a diagnostic's start normally matches one exactly.
func (m *reprintMap) position(printedFile *ast.SourceFile, pos int) (int, bool) {
	m.load()
	pos = min(pos, len(printedFile.Text()))
	line, character := scanner.GetECMALineAndUTF16CharacterOfPosition(printedFile, pos)
	i := sort.Search(len(m.mappings), func(i int) bool {
		a := &m.mappings[i]
		return a.GeneratedLine > line || a.GeneratedLine == line && a.GeneratedCharacter > character
	})
	if i == 0 {
		return 0, false
	}
	found := &m.mappings[i-1]
	return scanner.GetECMAPositionOfLineAndUTF16Character(m.original, found.SourceLine, found.SourceCharacter), true
}

// locate maps a printed range to the original file. It keeps the printed length when the end doesn't map to a later
// place on the same line.
func (m *reprintMap) locate(printedFile *ast.SourceFile, loc core.TextRange) (core.TextRange, bool) {
	start, ok := m.position(printedFile, loc.Pos())
	if !ok {
		return loc, false
	}
	end := start + loc.Len()
	if mapped, ok := m.position(printedFile, loc.End()); ok && mapped > start {
		end = mapped
	}
	lineEnd := scanner.GetECMAEndLinePosition(m.original, scanner.GetECMALineOfPosition(m.original, start))
	if end > lineEnd && lineEnd > start {
		end = lineEnd
	}
	return core.NewTextRange(start, min(end, len(m.original.Text()))), true
}

// MapDiagnostics points diagnostics at the source the user wrote, not at the printed text. This is where rbxtsgo differs
// from JS on purpose. See docs/known-differences.md. A compound assignment prints its object twice, so an error in it
// is reported twice in the printed text. Those are the same place in the source, and only one stays.
func (a *Applied) MapDiagnostics(list []*shared.Diagnostic) []*shared.Diagnostic {
	if a.maps == nil {
		return list
	}
	result := make([]*shared.Diagnostic, 0, len(list))
	type place struct {
		file    *ast.SourceFile
		start   int
		code    int32
		message string
	}
	seen := make(map[place]bool, len(list))
	relocate := func(file *ast.SourceFile, loc core.TextRange) (*ast.SourceFile, core.TextRange, bool) {
		m := a.maps[file]
		if m == nil {
			return nil, core.TextRange{}, false
		}
		mapped, ok := m.locate(file, loc)
		return m.original, mapped, ok
	}
	for _, d := range list {
		moved := d.Relocated(relocate)
		key := place{moved.File(), moved.Start(), moved.Code(), moved.Message()}
		if moved.File() != nil && seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, moved)
	}
	return result
}
