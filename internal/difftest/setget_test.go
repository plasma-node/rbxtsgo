package difftest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/compile"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/plugins"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

// markerPattern finds what a fixture expects on its own line: `//! TS2322 TS2339`, or `{/*! TS2747 */}` in JSX.
var markerPattern = regexp.MustCompile(`(?://|\{/\*)! ((?:TS\d+|roblox-ts)(?: (?:TS\d+|roblox-ts))*)`)

func wantedMarkers(t *testing.T, sourceDir string) map[string][]string {
	t.Helper()
	wanted := map[string][]string{}
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		text, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for line, content := range strings.Split(string(text), "\n") {
			if match := markerPattern.FindStringSubmatch(content); match != nil {
				for _, code := range strings.Fields(match[1]) {
					key := fmt.Sprintf("src/%s:%d", entry.Name(), line)
					wanted[key] = append(wanted[key], code)
				}
			}
		}
	}
	return wanted
}

func tokenAt(text string, position int) string {
	end := position
	for end < len(text) && (text[end] == '_' || text[end] == '#' || text[end] >= 0x80 || text[end] >= '0' && text[end] <= '9' || text[end] >= 'a' && text[end] <= 'z' || text[end] >= 'A' && text[end] <= 'Z') {
		end++
	}
	if end == position && position < len(text) {
		end++
	}
	return text[position:end]
}

// TestSetgetDiagnosticLines checks the line mapping of the reprint: in the setget-errors fixtures, every diagnostic
// lands on the line of the original source that its marker is on, and at the same token. The same run also checks that
// this is a real change: JS reports many of these on other lines, in text the user never wrote.
func TestSetgetDiagnosticLines(t *testing.T) {
	root := repoRoot(t)
	tsConfig := nodepath.Native.Normalize(filepath.Join(root, ".scratch", "diff", "setget-errors", "tsconfig.json"))
	if _, err := os.Stat(tsConfig); err != nil {
		t.Skip("run scripts/setup-diff-corpus.sh")
	}
	wanted := wantedMarkers(t, filepath.Join(filepath.Dir(tsConfig), "src"))
	data, err := project.CreateProjectData(tsConfig, shared.DefaultProjectOptions())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := project.GetParsedCommandLine(data)
	if err != nil {
		t.Fatal(err)
	}
	p := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{})
	translator := project.CreatePathTranslator(p, data)
	prepared, failure, err := project.PrepareCompile(p, data, translator, p.Program().CaseSensitivity())
	if err != nil || failure != nil {
		t.Fatal(err, failure)
	}
	original := map[string]*ast.SourceFile{}
	files := p.SourceFilesToCompile()
	for _, file := range files {
		original[file.FileName().AsString()] = file
	}
	applied, err := plugins.Apply(context.Background(), p, plugins.Decide(tsConfig, plugins.Options{}), files)
	if err != nil {
		t.Fatal(err)
	}
	c := &compile.Context{Project: applied.Project, Data: data, Translator: translator, Prepared: prepared, Replay: shared.NewContextReplay()}
	results, err := compile.Run(context.Background(), c, applied.Files, compile.ModeIsolated)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(root, "testdata", "differential", "golden", "setget-errors", "diagnostics.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fromJS map[string][]DiagnosticEntry
	if err := json.Unmarshal(raw, &fromJS); err != nil {
		t.Fatal(err)
	}

	projectDir := filepath.ToSlash(filepath.Dir(tsConfig))
	// contexts.ts is there for the text of the reprint. It has no markers, but its diagnostics still have to land on the
	// token they came from.
	const unmarked = "src/contexts.ts"
	got := map[string][]string{}
	total, movedFromJS := 0, 0
	for _, result := range results {
		name := strings.TrimPrefix(strings.TrimPrefix(result.File.FileName().AsString(), projectDir), "/")
		// Mapping can drop a repeat, so each one is mapped alone here to keep it next to its printed form.
		for i, printed := range result.Diagnostics {
			d := applied.MapDiagnostics([]*shared.Diagnostic{printed})[0]
			total++
			if d.File() != original[d.File().FileName().AsString()] {
				t.Errorf("%s: diagnostic %d points at the printed file, not the original", name, i)
			}
			line, _ := lineAndColumn(d.File(), d.Start())
			code := "roblox-ts"
			if !d.IsRobloxTS() {
				code = fmt.Sprintf("TS%d", d.Code())
			}
			key := fmt.Sprintf("%s:%d", name, line)
			if name != unmarked && !slices.Contains(got[key], code) {
				got[key] = append(got[key], code)
			}
			if list := fromJS[name]; i < len(list) && list[i].Line != line {
				movedFromJS++
			}

			printedToken := tokenAt(printed.File().Text(), printed.Start())
			originalToken := tokenAt(d.File().Text(), d.Start())
			wantToken := printedToken
			for _, prefix := range []string{"__get", "__set"} {
				if rest, ok := strings.CutPrefix(printedToken, prefix); ok {
					wantToken = rest
				}
			}
			// Three things in the printed text have no token of their own in the source. __get alone is what a computed or
			// string name parses as. A ( can be a parenthesis the printer added. And a source map skips the whitespace at
			// the start of JSX text.
			exempt := printedToken == "__get" || printedToken == "__set" || printedToken == "(" || strings.TrimSpace(printedToken) == ""
			if !exempt && wantToken != originalToken {
				t.Errorf("%s: %s points at %q in the source, and at %q in the printed text", key, code, originalToken, printedToken)
			}
		}
	}
	for key, codes := range got {
		slices.Sort(codes)
		want := slices.Clone(wanted[key])
		slices.Sort(want)
		if !slices.Equal(codes, want) {
			t.Errorf("%s: reported %v, and the fixture's marker says %v", key, codes, want)
		}
	}
	for key, codes := range wanted {
		if _, ok := got[key]; !ok {
			t.Errorf("%s: the marker says %v, and nothing was reported", key, codes)
		}
	}
	t.Logf("%d diagnostics on the marked source lines, %d of them on a different line in JS", total, movedFromJS)
	if movedFromJS == 0 {
		t.Error("no diagnostic moved, so this test doesn't show anything")
	}
}
