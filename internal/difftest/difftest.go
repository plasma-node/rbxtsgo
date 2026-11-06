// Package difftest compares rbxtsgo with the goldens that real JS roblox-ts@3.0.0 produced, in
// testdata/differential/golden. See testdata/differential/README.md. Every output file must match byte for byte,
// and each file's diagnostics must match in order. It's the scoreboard and the gate for the port.
package difftest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/compile"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/plugins"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/pathtranslator"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

// Corpus is one entry of testdata/differential/corpora.json.
type Corpus struct {
	Name     string         `json:"name"`
	TsConfig string         `json:"tsconfig"`
	Mode     string         `json:"mode"`
	Options  map[string]any `json:"options"`
	// Golden names the corpus whose goldens this one compares with, when JS can't read this one's config.
	Golden string `json:"golden"`
}

// DiagnosticEntry is the structured form of a diagnostic in diagnostics.json. Column is in UTF-16 units on both sides.
type DiagnosticEntry struct {
	Source   string `json:"source"`
	Code     int    `json:"code"`
	ID       int    `json:"id"`
	Category string `json:"category"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

func (d DiagnosticEntry) String() string {
	return fmt.Sprintf("%s:%d:%d %s %s code=%d id=%d %q", d.File, d.Line, d.Column, d.Source, d.Category, d.Code, d.ID, d.Message)
}

// Result is the outcome for one case. Kind is "output" for one file under outDir and "diagnostics" for one source file's
// diagnostics, or the whole list in a whole-program corpus.
type Result struct {
	Corpus string
	Kind   string
	Name   string
	Pass   bool
	// Detail says why it failed.
	Detail string
	// Panic is the internal error behind a failure, if any. It's what the scoreboard groups by.
	Panic string
}

// Key identifies a case in the baseline file.
func (r Result) Key() string { return r.Corpus + " " + r.Kind + " " + r.Name }

// Report is the result of one run over all corpora. Sources holds each corpus's output by path, for comparing runs.
type Report struct {
	Results []Result
	Outputs map[string]string
}

// Counts returns how many cases passed out of how many, for a kind.
func (r *Report) Counts(kind string) (pass, total int) {
	for _, result := range r.Results {
		if result.Kind == kind {
			total++
			if result.Pass {
				pass++
			}
		}
	}
	return pass, total
}

// RepoRoot walks up from dir to the folder holding testdata/differential.
func RepoRoot(dir string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(dir, "testdata", "differential", "corpora.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no testdata/differential above the current folder")
		}
		dir = parent
	}
}

// LoadCorpora reads corpora.json.
func LoadCorpora(repoRoot string) ([]Corpus, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "testdata", "differential", "corpora.json"))
	if err != nil {
		return nil, err
	}
	var corpora []Corpus
	if err := json.Unmarshal(data, &corpora); err != nil {
		return nil, err
	}
	return corpora, nil
}

// Options says how to run.
type Options struct {
	Checkers int
	// Only limits the run to these corpora, if not empty.
	Only []string
}

// Run compiles each corpus and compares it with its goldens.
func Run(repoRoot string, options Options) (*Report, error) {
	corpora, err := LoadCorpora(repoRoot)
	if err != nil {
		return nil, err
	}
	report := &Report{Outputs: map[string]string{}}
	for _, corpus := range corpora {
		if len(options.Only) > 0 && !slices.Contains(options.Only, corpus.Name) {
			continue
		}
		if err := runCorpus(repoRoot, corpus, options, report); err != nil {
			return nil, fmt.Errorf("corpus %s: %w", corpus.Name, err)
		}
	}
	return report, nil
}

func projectOptions(corpus Corpus) shared.ProjectOptions {
	options := shared.DefaultProjectOptions()
	if value, ok := corpus.Options["allowCommentDirectives"].(bool); ok {
		options.AllowCommentDirectives = value
	}
	if value, ok := corpus.Options["optimizedLoops"].(bool); ok {
		options.OptimizedLoops = value
	}
	return options
}

func runCorpus(repoRoot string, corpus Corpus, options Options, report *Report) error {
	tsConfig := nodepath.Native.Normalize(filepath.Join(repoRoot, filepath.FromSlash(corpus.TsConfig)))
	if _, err := os.Stat(tsConfig); err != nil {
		return fmt.Errorf("%s is missing. Run: bash scripts/setup-diff-corpus.sh", tsConfig)
	}
	goldenName := corpus.Name
	if corpus.Golden != "" {
		goldenName = corpus.Golden
	}
	goldenDir := filepath.Join(repoRoot, "testdata", "differential", "golden", goldenName)

	data, err := project.CreateProjectData(tsConfig, projectOptions(corpus))
	if err != nil {
		return err
	}
	parsed, err := project.GetParsedCommandLine(data)
	if err != nil {
		return err
	}
	p := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{Checkers: options.Checkers})
	translator := project.CreatePathTranslator(p, data)
	prepared, failure, err := project.PrepareCompile(p, data, translator, p.Program().CaseSensitivity())
	if err != nil {
		return err
	}
	if failure != nil {
		return fmt.Errorf("prepare failed: %s", failure[0].Message())
	}

	files := p.SourceFilesToCompile()
	decision := plugins.Decide(tsConfig, plugins.Options{})
	if corpus.Mode == "plugin-exit" {
		return comparePluginExits(corpus, goldenDir, tsConfig, p, decision, files, report)
	}
	applied, err := plugins.Apply(context.Background(), p, decision, files)
	if err != nil {
		return err
	}
	if applied.Exit != nil {
		return fmt.Errorf("the plugin exited: %s", strings.Join(applied.Exit, "\n"))
	}
	p, files = applied.Project, applied.Files
	if err := compareReprints(corpus, goldenDir, applied, translator, p.Options().OutDir.AsString(), report); err != nil {
		return err
	}
	mode := compile.ModeWhole
	if corpus.Mode == "perfile" {
		mode = compile.ModeIsolated
	}
	c := &compile.Context{Project: p, Data: data, Translator: translator, Prepared: prepared, Replay: shared.NewContextReplay()}
	results, err := compile.Run(context.Background(), c, files, mode)
	if err != nil {
		return err
	}

	projectDir := filepath.ToSlash(filepath.Dir(tsConfig))
	outDir := p.Options().OutDir.AsString()
	relProject := func(name string) string {
		return strings.TrimPrefix(strings.TrimPrefix(filepath.ToSlash(name), projectDir), "/")
	}

	// Outputs.
	outputs := map[string]string{}
	panics := map[string]string{} // output path to the internal error that kept it from existing
	for _, result := range results {
		outPath := filepath.ToSlash(translator.GetOutputPath(nodepath.Native.Normalize(result.File.FileName().AsString())))
		rel := strings.TrimPrefix(strings.TrimPrefix(outPath, filepath.ToSlash(outDir)), "/")
		if result.Panic != "" {
			panics[rel] = result.Panic
		}
		if result.HasSource {
			outputs[rel] = result.Source
			report.Outputs[corpus.Name+"/"+rel] = result.Source
		}
	}
	golden := map[string][]byte{}
	goldenFiles := filepath.Join(goldenDir, "files")
	if _, statErr := os.Stat(goldenFiles); statErr == nil {
		err = filepath.WalkDir(goldenFiles, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(goldenFiles, path)
			if err != nil {
				return err
			}
			golden[filepath.ToSlash(rel)] = content
			return nil
		})
		if err != nil {
			return err
		}
	}
	names := make([]string, 0, len(golden)+len(outputs))
	for name := range golden {
		names = append(names, name)
	}
	for name := range outputs {
		if _, ok := golden[name]; !ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		result := Result{Corpus: corpus.Name, Kind: "output", Name: name, Panic: panics[name]}
		want, inGolden := golden[name]
		got, inGo := outputs[name]
		switch {
		case inGolden && inGo:
			if bytes.Equal(want, []byte(got)) {
				result.Pass = true
			} else {
				result.Detail = FirstDifference(want, []byte(got))
			}
		case inGolden:
			result.Detail = "rbxtsgo wrote no file"
			if result.Panic != "" {
				result.Detail += ": " + result.Panic
			}
		default:
			result.Detail = "JS wrote no file, and rbxtsgo did"
		}
		report.Results = append(report.Results, result)
	}

	// Diagnostics.
	var wantDiagnostics map[string][]DiagnosticEntry
	raw, err := os.ReadFile(filepath.Join(goldenDir, "diagnostics.json"))
	if err != nil {
		return fmt.Errorf("no goldens: %w. Run: bash scripts/gen-diff-goldens.sh", err)
	}
	if err := json.Unmarshal(raw, &wantDiagnostics); err != nil {
		return err
	}
	compareDiagnostics := func(name string, got []*shared.Diagnostic, panicText string) {
		entries := make([]DiagnosticEntry, len(got))
		for i, d := range got {
			entries[i] = describe(d, projectDir, relProject)
		}
		result := Result{Corpus: corpus.Name, Kind: "diagnostics", Name: name, Panic: panicText}
		want := wantDiagnostics[name]
		switch {
		case panicText != "":
			result.Detail = "internal error: " + panicText
		case slices.Equal(want, entries):
			result.Pass = true
		default:
			result.Detail = firstDiagnosticDifference(want, entries)
		}
		report.Results = append(report.Results, result)
	}
	if mode == compile.ModeIsolated {
		for _, result := range results {
			compareDiagnostics(relProject(result.File.FileName().AsString()), result.Diagnostics, result.Panic)
		}
	} else {
		var all []*shared.Diagnostic
		panicText := ""
		for _, result := range results {
			all = append(all, result.Diagnostics...)
			if result.Panic != "" && panicText == "" {
				panicText = result.Panic
			}
		}
		compareDiagnostics("<whole>", all, panicText)
	}
	return nil
}

// comparePluginExits runs the plugin on each file alone, and compares what it wrote before exiting.
func comparePluginExits(corpus Corpus, goldenDir, tsConfig string, p *tsbridge.Project, decision plugins.Decision, files []*ast.SourceFile, report *Report) error {
	var want map[string][]DiagnosticEntry
	raw, err := os.ReadFile(filepath.Join(goldenDir, "diagnostics.json"))
	if err != nil {
		return fmt.Errorf("no goldens: %w. Run: bash scripts/gen-diff-goldens.sh", err)
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		return err
	}
	projectDir := filepath.ToSlash(filepath.Dir(tsConfig))
	for _, file := range files {
		name := strings.TrimPrefix(strings.TrimPrefix(file.FileName().AsString(), projectDir), "/")
		applied, err := plugins.Apply(context.Background(), p, decision, []*ast.SourceFile{file})
		if err != nil {
			return err
		}
		var got []DiagnosticEntry
		if applied.Exit != nil {
			message := ansi.ReplaceAllString(strings.Join(applied.Exit, "\n"), "")
			got = []DiagnosticEntry{{Source: "plugin", Code: 1, ID: -1, Category: "error", Message: message, Line: -1, Column: -1}}
		}
		result := Result{Corpus: corpus.Name, Kind: "diagnostics", Name: name, Pass: slices.Equal(want[name], got)}
		if !result.Pass {
			result.Detail = firstDiagnosticDifference(want[name], got)
		}
		report.Results = append(report.Results, result)
	}
	return nil
}

// compareReprints compares each file's reprinted text with what JS wrote for the writeTransformedFiles option. The Luau
// comparison is the requirement. This one says where a difference starts, and diagnostic positions depend on it.
func compareReprints(corpus Corpus, goldenDir string, applied *plugins.Applied, translator *pathtranslator.Translator, outDir string, report *Report) error {
	if applied.Printed == nil {
		return nil
	}
	for _, file := range applied.Files {
		name := file.FileName().AsString()
		outPath := filepath.ToSlash(translator.GetOutputTransformedPath(nodepath.Native.Normalize(name)))
		rel := strings.TrimPrefix(strings.TrimPrefix(outPath, filepath.ToSlash(outDir)), "/")
		result := Result{Corpus: corpus.Name, Kind: "reprint", Name: rel}
		want, err := os.ReadFile(filepath.Join(goldenDir, "transformed", filepath.FromSlash(rel)))
		switch {
		case err != nil:
			result.Detail = "JS wrote no reprint"
		case string(want) == applied.Printed[name]:
			result.Pass = true
		default:
			result.Detail = FirstDifference(want, []byte(applied.Printed[name]))
		}
		report.Results = append(report.Results, result)
		report.Outputs[corpus.Name+"/"+rel] = applied.Printed[name]
	}
	return nil
}

var ansi = regexp.MustCompile("\x1b\\[[0-9;]*m")

func describe(d *shared.Diagnostic, projectDir string, relProject func(string) string) DiagnosticEntry {
	entry := DiagnosticEntry{Code: -1, ID: d.ID(), Line: -1, Column: -1}
	if d.IsRobloxTS() {
		entry.Source = "rbxts"
	} else {
		entry.Source = "ts"
		entry.Code = int(d.Code())
	}
	switch d.Category() {
	case diagnostics.CategoryError:
		entry.Category = "error"
	case diagnostics.CategoryWarning:
		entry.Category = "warning"
	case diagnostics.CategorySuggestion:
		entry.Category = "suggestion"
	default:
		entry.Category = "message"
	}
	message := ansi.ReplaceAllString(d.Message(), "")
	message = strings.ReplaceAll(message, filepath.FromSlash(projectDir), "<ROOT>")
	entry.Message = strings.ReplaceAll(message, projectDir, "<ROOT>")
	if file := d.File(); file != nil {
		entry.File = relProject(file.FileName().AsString())
		entry.Line, entry.Column = lineAndColumn(file, d.Start())
	}
	return entry
}

// lineAndColumn is getLineAndCharacterOfPosition, with the column in UTF-16 units as JS has it.
func lineAndColumn(file *ast.SourceFile, position int) (line, column int) {
	text := file.Text()
	lineStart := 0
	for i := 0; i < position && i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		next := i + size
		switch r {
		case '\r':
			if next < len(text) && text[next] == '\n' {
				next++
			}
			fallthrough
		case '\n', ' ', ' ':
			if next <= position {
				line++
				lineStart = next
			}
		}
		i = next
	}
	for _, r := range text[lineStart:min(position, len(text))] {
		column += utf16.RuneLen(r)
	}
	return line, column
}

func firstDiagnosticDifference(want, got []DiagnosticEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "want %d diagnostics, got %d", len(want), len(got))
	different := 0
	for i := range min(len(want), len(got)) {
		if want[i] != got[i] {
			different++
		}
	}
	fmt.Fprintf(&b, ", %d differ", different)
	for i := range max(len(want), len(got)) {
		var w, g string
		if i < len(want) {
			w = want[i].String()
		}
		if i < len(got) {
			g = got[i].String()
		}
		if w != g {
			fmt.Fprintf(&b, "; first difference at #%d\n    JS: %s\n    Go: %s", i, w, g)
			break
		}
	}
	return b.String()
}

// FirstDifference describes the first place two outputs differ, with the lines around it.
func FirstDifference(want, got []byte) string {
	i := 0
	for i < len(want) && i < len(got) && want[i] == got[i] {
		i++
	}
	line := bytes.Count(want[:i], []byte("\n")) + 1
	context := func(text []byte) string {
		start := i
		for n := 0; start > 0 && n < 3; start-- {
			if text[start-1] == '\n' {
				n++
				if n == 3 {
					break
				}
			}
		}
		end := min(i, len(text))
		for n := 0; end < len(text) && n < 3; end++ {
			if text[end] == '\n' {
				n++
			}
		}
		if start > len(text) {
			start = len(text)
		}
		return string(text[start:end])
	}
	return fmt.Sprintf("first difference at byte %d, line %d\n--- JS\n%s\n--- Go\n%s", i, line, context(want), context(got))
}

// PanicSummary groups failures by their internal error, largest first. An entry for a port that's missing a piece
// reads "not ported: X", which makes this the work list.
func (r *Report) PanicSummary() []string {
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, result := range r.Results {
		key := result.Corpus + " " + result.Name
		if result.Panic != "" && !seen[key] {
			seen[key] = true
			counts[firstLine(result.Panic)]++
		}
	}
	lines := make([]string, 0, len(counts))
	for panicText, count := range counts {
		lines = append(lines, fmt.Sprintf("%4d  %s", count, panicText))
	}
	sort.Slice(lines, func(i, j int) bool {
		if a, b := lines[i][:4], lines[j][:4]; a != b {
			return a > b
		}
		return lines[i] < lines[j]
	})
	return lines
}

func firstLine(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return text[:i]
	}
	return text
}
