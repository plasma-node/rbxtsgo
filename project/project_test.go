package project

import (
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/parser"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/pathtranslator"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/rojoresolver"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

var _ embed.FS

type goldenComment struct {
	Message string `json:"message"`
	Start   int    `json:"start"`
	Length  int    `json:"length"`
	ID      int    `json:"id"`
	Text    string `json:"text"`
}

type goldenData struct {
	IsPackage       bool     `json:"isPackage"`
	NodeModulesPath string   `json:"nodeModulesPath"`
	ProjectPath     string   `json:"projectPath"`
	RojoConfigPath  *string  `json:"rojoConfigPath"`
	IncludePath     string   `json:"includePath"`
	Logs            []string `json:"logs"`
}

type golden struct {
	Validate    map[string]string          `json:"validate"`
	ProjectData map[string]goldenData      `json:"projectData"`
	Mapping     []string                   `json:"mapping"`
	Copy        map[string][]string        `json:"copy"`
	Include     map[string][]string        `json:"include"`
	Cleanup     map[string][]string        `json:"cleanup"`
	FileNames   map[string][]string        `json:"fileNames"`
	RojoCheck   []string                   `json:"rojoCheck"`
	RojoNoConf  []string                   `json:"rojoCheckNoConfig"`
	Comments    map[string][]goldenComment `json:"comments"`
	Ancestor    []struct {
		Dirs   []string `json:"dirs"`
		Result string   `json:"result"`
	} `json:"ancestor"`
	Descendant []struct {
		File   string `json:"file"`
		Dir    string `json:"dir"`
		Result bool   `json:"result"`
	} `json:"descendant"`
	Translators map[string]struct {
		RootDir             string  `json:"rootDir"`
		OutDir              string  `json:"outDir"`
		BuildInfoOutputPath *string `json:"buildInfoOutputPath"`
		Declaration         bool    `json:"declaration"`
		UseLuauExtension    bool    `json:"useLuauExtension"`
	} `json:"translators"`
	Compile map[string]struct {
		Skipped     bool     `json:"skipped"`
		Diagnostics []string `json:"diagnostics"`
	} `json:"compile"`
}

func readGolden(t *testing.T) struct {
	Color golden `json:"color"`
	Plain golden `json:"plain"`
} {
	t.Helper()
	var result struct {
		Color golden `json:"color"`
		Plain golden `json:"plain"`
	}
	data, err := os.ReadFile(filepath.Join("testdata", "golden."+runtime.GOOS+".json"))
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("no golden for this platform: run scripts/gen-goldens.sh here")
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func fixtureRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("testdata", "fixtures"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// setColor sets the process-wide color and definitions for one test, like running under a different environment.
func setColor(t *testing.T, enabled bool) {
	t.Helper()
	oldColor, oldErrors, oldWarnings := shared.Color, shared.Errors, shared.Warnings
	shared.Color = shared.Kleur{Enabled: enabled}
	shared.Errors, shared.Warnings = shared.NewDefinitions(shared.Color)
	t.Cleanup(func() { shared.Color, shared.Errors, shared.Warnings = oldColor, oldErrors, oldWarnings })
}

func forEachMode(t *testing.T, fn func(t *testing.T, want golden)) {
	all := readGolden(t)
	t.Run("color", func(t *testing.T) {
		setColor(t, true)
		fn(t, all.Color)
	})
	t.Run("plain", func(t *testing.T) {
		setColor(t, false)
		fn(t, all.Plain)
	})
}

func TestValidateCompilerOptions(t *testing.T) {
	forEachMode(t, func(t *testing.T, want golden) {
		root := fixtureRoot(t)
		if len(want.Validate) < 15 {
			t.Fatalf("only %d cases", len(want.Validate))
		}
		for name, wantText := range want.Validate {
			dir := filepath.Join(root, "validate")
			_, err := GetParsedCommandLine(&shared.ProjectData{TsConfigPath: filepath.Join(dir, name), ProjectPath: dir})
			got := "ok"
			if err != nil {
				var diagnosticError *shared.DiagnosticError
				if !errors.As(err, &diagnosticError) {
					t.Errorf("%s: unexpected error %v", name, err)
					continue
				}
				var messages []string
				for _, d := range diagnosticError.Diagnostics {
					messages = append(messages, d.Message())
				}
				got = strings.ReplaceAll(strings.Join(messages, "\n"), root, "<ROOT>")
			}
			if got != wantText {
				t.Errorf("%s\n got %q\nwant %q", name, got, wantText)
			}
		}
	})
}

func TestCreateProjectData(t *testing.T) {
	root := fixtureRoot(t)
	want := readGolden(t).Plain
	cases := map[string]struct {
		rel     string
		options func(*shared.ProjectOptions)
	}{
		"app":        {"data/app/tsconfig.json", nil},
		"pkg":        {"data/pkg/tsconfig.json", nil},
		"nested":     {"data/nested/deep/er/tsconfig.json", nil},
		"rojos":      {"data/rojos/tsconfig.json", nil},
		"noname":     {"data/noname/tsconfig.json", nil},
		"badjson":    {"data/badjson/tsconfig.json", nil},
		"rojoOption": {"data/app/tsconfig.json", func(o *shared.ProjectOptions) { o.Rojo = filepath.Join(root, "data", "rojos", "a.project.json") }},
		"includeOption": {"data/app/tsconfig.json", func(o *shared.ProjectOptions) {
			o.IncludePath = root + string(filepath.Separator) + filepath.Join("data", "custom", "..", "include2")
		}},
		"typeOption": {"data/pkg/tsconfig.json", func(o *shared.ProjectOptions) { o.Type = shared.ProjectTypeGame; o.NoInclude = true }},
	}
	if len(cases) != len(want.ProjectData) {
		t.Errorf("%d cases here, %d in the golden", len(cases), len(want.ProjectData))
	}
	clean := func(text string) string { return strings.ReplaceAll(text, root, "<ROOT>") }
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			options := shared.DefaultProjectOptions()
			if c.options != nil {
				c.options(&options)
			}
			data, err := CreateProjectData(filepath.Join(root, filepath.FromSlash(c.rel)), options)
			if err != nil {
				t.Fatal(err)
			}
			w := want.ProjectData[name]
			if data.IsPackage != w.IsPackage {
				t.Errorf("isPackage = %v, want %v", data.IsPackage, w.IsPackage)
			}
			if got := clean(data.NodeModulesPath); got != w.NodeModulesPath {
				t.Errorf("nodeModulesPath = %q, want %q", got, w.NodeModulesPath)
			}
			if got := clean(data.ProjectPath); got != w.ProjectPath {
				t.Errorf("projectPath = %q, want %q", got, w.ProjectPath)
			}
			wantRojo := ""
			if w.RojoConfigPath != nil {
				wantRojo = *w.RojoConfigPath
			}
			if got := clean(data.RojoConfigPath); got != wantRojo {
				t.Errorf("rojoConfigPath = %q, want %q", got, wantRojo)
			}
			if got := clean(data.ProjectOptions.IncludePath); got != w.IncludePath {
				t.Errorf("includePath = %q, want %q", got, w.IncludePath)
			}
		})
	}
}

func TestCreateNodeModulesPathMapping(t *testing.T) {
	root := fixtureRoot(t)
	want := readGolden(t).Plain.Mapping
	sensitivity := tspath.CaseSensitive
	if runtime.GOOS == "windows" {
		sensitivity = tspath.CaseInsensitive
	}
	mapping, err := CreateNodeModulesPathMapping([]string{filepath.Join(root, "mapping", "@rbxts"), filepath.Join(root, "mapping", "missing")}, sensitivity)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	lowerRoot := sensitivity.Canonicalize(root)
	for key, value := range mapping {
		key = strings.ReplaceAll(strings.ReplaceAll(key, lowerRoot, "<ROOT>"), root, "<ROOT>")
		got = append(got, key+" -> "+strings.ReplaceAll(value, root, "<ROOT>"))
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("mapping\n got %q\nwant %q", got, want)
	}
}

// treeLines mirrors tree() in gen-golden.cjs: sorted, with file contents in base64.
func treeLines(t *testing.T, dir string, withContents bool) []string {
	t.Helper()
	var lines []string
	var walk func(current, prefix string)
	walk = func(current, prefix string) {
		entries, err := os.ReadDir(current)
		if err != nil {
			return
		}
		for _, entry := range entries {
			full := filepath.Join(current, entry.Name())
			if entry.IsDir() {
				lines = append(lines, prefix+entry.Name()+"/")
				walk(full, prefix+entry.Name()+"/")
				continue
			}
			if !withContents {
				lines = append(lines, prefix+entry.Name())
				continue
			}
			data, err := os.ReadFile(full)
			if err != nil {
				t.Fatal(err)
			}
			lines = append(lines, prefix+entry.Name()+" = "+base64.StdEncoding.EncodeToString(data))
		}
	}
	walk(dir, "")
	return lines
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCopyItem(t *testing.T) {
	root := fixtureRoot(t)
	want := readGolden(t).Plain.Copy
	if len(want) != 8 {
		t.Fatalf("%d cases", len(want))
	}
	for _, declaration := range []bool{false, true} {
		for _, writeOnlyChanged := range []bool{false, true} {
			for _, existing := range []bool{false, true} {
				name := fmt.Sprintf("decl=%v,changed=%v,existing=%v", declaration, writeOnlyChanged, existing)
				t.Run(name, func(t *testing.T) {
					outDir := filepath.Join(t.TempDir(), "out")
					if existing {
						writeFile(t, filepath.Join(outDir, "f.txt"), "old content\n")
						writeFile(t, filepath.Join(outDir, "d.lua"), "content of copy/src/d.lua\n")
						writeFile(t, filepath.Join(outDir, "sub", "h.luau"), "old\n")
						writeFile(t, filepath.Join(outDir, "stale.txt"), "stays\n")
						writeFile(t, filepath.Join(outDir, "sub", "stale", "x.lua"), "stays\n")
					}
					translator := pathtranslator.New(filepath.Join(root, "copy", "src"), outDir, "", declaration, true)
					options := shared.DefaultProjectOptions()
					options.WriteOnlyChanged = writeOnlyChanged
					if err := CopyItem(&shared.ProjectData{ProjectOptions: options}, translator, filepath.Join(root, "copy", "src")); err != nil {
						t.Fatal(err)
					}
					if got := treeLines(t, outDir, true); !slices.Equal(got, want[name]) {
						t.Errorf("tree\n got %q\nwant %q", got, want[name])
					}
				})
			}
		}
	}
}

// With writeOnlyChanged, an identical file must not be rewritten. That is what makes a Rojo sync quiet.
func TestCopyItemLeavesUnchangedFilesAlone(t *testing.T) {
	root := fixtureRoot(t)
	outDir := filepath.Join(t.TempDir(), "out")
	writeFile(t, filepath.Join(outDir, "d.lua"), "content of copy/src/d.lua\n")
	dest := filepath.Join(outDir, "d.lua")
	// Read-only: a rewrite would remove and recreate it, losing this mode.
	info1, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	old := info1.ModTime().Add(-100 * 3600 * 1e9)
	if err := os.Chtimes(dest, old, old); err != nil {
		t.Fatal(err)
	}
	options := shared.DefaultProjectOptions()
	options.WriteOnlyChanged = true
	translator := pathtranslator.New(filepath.Join(root, "copy", "src"), outDir, "", false, true)
	if err := CopyItem(&shared.ProjectData{ProjectOptions: options}, translator, filepath.Join(root, "copy", "src")); err != nil {
		t.Fatal(err)
	}
	info2, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !info2.ModTime().Equal(old) {
		t.Errorf("an unchanged file was rewritten")
	}
}

func TestCopyInclude(t *testing.T) {
	want := readGolden(t).Plain.Include
	for name, c := range map[string]struct {
		noInclude bool
		typ       shared.ProjectType
		isPackage bool
	}{
		"default":            {},
		"noInclude":          {noInclude: true},
		"typePackage":        {typ: shared.ProjectTypePackage},
		"isPackage":          {isPackage: true},
		"isPackageTypeModel": {typ: shared.ProjectTypeModel, isPackage: true},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			options := shared.DefaultProjectOptions()
			options.IncludePath = filepath.Join(dir, "inc", "lower")
			options.NoInclude = c.noInclude
			options.Type = c.typ
			if err := CopyInclude(&shared.ProjectData{IsPackage: c.isPackage, ProjectOptions: options}); err != nil {
				t.Fatal(err)
			}
			if got := treeLines(t, dir, true); !slices.Equal(got, want[name]) {
				t.Errorf("tree differs: %d lines, want %d", len(got), len(want[name]))
			}
		})
	}
	t.Run("overwrite", func(t *testing.T) {
		dir := t.TempDir()
		options := shared.DefaultProjectOptions()
		options.IncludePath = filepath.Join(dir, "inc")
		writeFile(t, filepath.Join(options.IncludePath, "RuntimeLib.lua"), "overwritten\n")
		writeFile(t, filepath.Join(options.IncludePath, "Extra.lua"), "stays\n")
		if err := CopyInclude(&shared.ProjectData{ProjectOptions: options}); err != nil {
			t.Fatal(err)
		}
		if got := treeLines(t, dir, true); !slices.Equal(got, want["overwrite"]) {
			t.Errorf("tree differs: %d lines, want %d", len(got), len(want["overwrite"]))
		}
	})
}

// The embedded runtime files must stay byte-identical to the pinned clone, which matches the tarball (see 0004).
func TestIncludeFilesMatchPinnedClone(t *testing.T) {
	want := map[string]string{}
	for _, name := range []string{"Promise.lua", "RuntimeLib.lua"} {
		data, err := includeFiles.ReadFile("include/" + name)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		want[name] = hex.EncodeToString(sum[:])
		pinned, err := os.ReadFile(filepath.Join("..", "third_party", "roblox-ts", "include", name))
		if err != nil {
			t.Skip("third_party/roblox-ts is not set up")
		}
		if string(pinned) != string(data) {
			t.Errorf("%s differs from third_party/roblox-ts/include", name)
		}
	}
}

func TestCleanup(t *testing.T) {
	want := readGolden(t).Plain.Cleanup
	for _, declaration := range []bool{false, true} {
		for _, luau := range []bool{false, true} {
			name := fmt.Sprintf("decl=%v,luau=%v", declaration, luau)
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				src := filepath.Join(dir, "src")
				outDir := filepath.Join(dir, "out")
				for _, file := range []string{"a.ts", "b.tsx", "index.ts", "sub/c.ts", "keep.txt", "sub/index.tsx", "typed.d.ts"} {
					writeFile(t, filepath.Join(src, filepath.FromSlash(file)), "x\n")
				}
				for _, file := range []string{
					"a.lua", "a.luau", "b.lua", "b.luau", "init.lua", "init.luau", "index.lua", "orphan.lua", "orphan.luau", "keep.txt", "gone.txt",
					"sub/c.lua", "sub/c.luau", "sub/init.lua", "sub/init.luau", "sub/orphan.lua", "empty/gone.lua", "a.d.ts", "orphan.d.ts", "typed.d.ts", "tsconfig.tsbuildinfo", ".git/keep.lua",
				} {
					writeFile(t, filepath.Join(outDir, filepath.FromSlash(file)), "x\n")
				}
				translator := pathtranslator.New(src, outDir, filepath.Join(outDir, "tsconfig.tsbuildinfo"), declaration, luau)
				if err := Cleanup(translator); err != nil {
					t.Fatal(err)
				}
				if got := treeLines(t, outDir, false); !slices.Equal(got, want[name]) {
					t.Errorf("tree\n got %q\nwant %q", got, want[name])
				}
			})
		}
	}
}

func messagesOf(list []*shared.Diagnostic, root string) []string {
	result := []string{}
	for _, d := range list {
		result = append(result, strings.ReplaceAll(strings.ReplaceAll(d.Message(), root, "<ROOT>"), filepath.ToSlash(root), "<ROOT>"))
	}
	return result
}

func TestFileNameAndRojoChecks(t *testing.T) {
	forEachMode(t, func(t *testing.T, want golden) {
		root := fixtureRoot(t)
		for name, wantMessages := range want.FileNames {
			service := &shared.DiagnosticService{}
			CheckFileName(service, "/some/dir/"+name)
			if got := messagesOf(service.Flush(), root); !slices.Equal(got, wantMessages) {
				t.Errorf("%s\n got %q\nwant %q", name, got, wantMessages)
			}
		}
		configPath := filepath.Join(root, "rojocheck", "default.project.json")
		resolver, err := rojoresolver.FromPath(configPath)
		if err != nil {
			t.Fatal(err)
		}
		srcDir := filepath.Join(root, "rojocheck", "src")
		translator := pathtranslator.New(srcDir, filepath.Join(root, "rojocheck", "out"), "", false, true)
		service := &shared.DiagnosticService{}
		CheckRojoConfig(service, &shared.ProjectData{RojoConfigPath: configPath}, resolver, []string{srcDir}, translator)
		if got := messagesOf(service.Flush(), root); !slices.Equal(got, want.RojoCheck) {
			t.Errorf("rojo check\n got %q\nwant %q", got, want.RojoCheck)
		}
		CheckRojoConfig(service, &shared.ProjectData{}, resolver, []string{srcDir}, translator)
		if got := messagesOf(service.Flush(), root); !slices.Equal(got, want.RojoNoConf) {
			t.Errorf("rojo check with no config\n got %q\nwant %q", got, want.RojoNoConf)
		}
	})
}

func parseFile(t *testing.T, path string) *ast.SourceFile {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	name := tspath.RootedFilePathFromAbsolute(tspath.NormalizePath(filepath.ToSlash(path)))
	options := ast.SourceFileParseOptions{FileName: name, PathKey: tspath.CaseSensitive.PathKey(name.AsPath())}
	return parser.ParseSourceFile(options, string(text), core.ScriptKindTS)
}

func TestFileUsesCommentDirectives(t *testing.T) {
	forEachMode(t, func(t *testing.T, want golden) {
		root := fixtureRoot(t)
		if len(want.Comments) < 5 {
			t.Fatalf("only %d files", len(want.Comments))
		}
		options := shared.DefaultProjectOptions()
		data := &shared.ProjectData{ProjectOptions: options}
		for name, wantList := range want.Comments {
			file := parseFile(t, filepath.Join(root, "comments", name))
			got := FileUsesCommentDirectives(data, file)
			if len(got) != len(wantList) {
				t.Errorf("%s: %d diagnostics, want %d", name, len(got), len(wantList))
				continue
			}
			for i, d := range got {
				w := wantList[i]
				if d.Message() != w.Message || d.ID() != w.ID {
					t.Errorf("%s[%d] message or id differs", name, i)
				}
				// Offsets are bytes here and UTF-16 in JS, so compare the text they cover.
				text := file.Text()
				end := min(d.Start()+d.Length(), len(text))
				if name == "unicode.ts" {
					// Upstream stores range.end as the length, and range.end differs here: bytes against UTF-16 units.
					if !strings.HasPrefix(text[d.Start():end], "// @ts-ignore") || !strings.HasPrefix(w.Text, "// @ts-ignore") {
						t.Errorf("%s[%d] covers %q, want it to start at the directive", name, i, text[d.Start():end])
					}
					continue
				}
				if text[d.Start():end] != w.Text {
					t.Errorf("%s[%d] covers %q, want %q", name, i, text[d.Start():end], w.Text)
				}
			}
			allowed := options
			allowed.AllowCommentDirectives = true
			if len(FileUsesCommentDirectives(&shared.ProjectData{ProjectOptions: allowed}, file)) != 0 {
				t.Errorf("%s: allowCommentDirectives must silence this", name)
			}
		}
	})
}

func TestSharedPathHelpers(t *testing.T) {
	want := readGolden(t).Plain
	for _, c := range want.Ancestor {
		if got := shared.FindAncestorDir(c.Dirs); got != c.Result {
			t.Errorf("findAncestorDir(%q) = %q, want %q", c.Dirs, got, c.Result)
		}
	}
	for _, c := range want.Descendant {
		if got := shared.IsPathDescendantOf(c.File, c.Dir); got != c.Result {
			t.Errorf("isPathDescendantOf(%q, %q) = %v, want %v", c.File, c.Dir, got, c.Result)
		}
	}
}

func TestCreatePathTranslator(t *testing.T) {
	root := fixtureRoot(t)
	want := readGolden(t).Plain.Translators
	clean := func(text string) string { return strings.ReplaceAll(text, root, "<ROOT>") }
	for name, w := range want {
		t.Run(name, func(t *testing.T) {
			options := shared.DefaultProjectOptions()
			options.Rojo = filepath.Join(root, "program", "rojo-ok.json")
			options.IncludePath = filepath.Join(root, "program", "include")
			data, err := CreateProjectData(filepath.Join(root, "program", name), options)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := GetParsedCommandLine(data)
			if err != nil {
				t.Fatal(err)
			}
			project := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{Checkers: 1})
			translator := CreatePathTranslator(project, data)
			if got := clean(translator.RootDir); got != w.RootDir {
				t.Errorf("rootDir = %q, want %q", got, w.RootDir)
			}
			// Paths from the TS parser use forward slashes on every platform. Upstream's outDir is like that too.
			if got := clean(strings.ReplaceAll(translator.OutDir, filepath.ToSlash(root), root)); got != strings.ReplaceAll(w.OutDir, filepath.ToSlash(root), root) {
				t.Errorf("outDir = %q, want %q", got, w.OutDir)
			}
			wantBuildInfo := ""
			if w.BuildInfoOutputPath != nil {
				wantBuildInfo = *w.BuildInfoOutputPath
			}
			if got := clean(translator.BuildInfoOutputPath); got != wantBuildInfo {
				t.Errorf("buildInfoOutputPath = %q, want %q", got, wantBuildInfo)
			}
			if translator.Declaration != w.Declaration || translator.UseLuauExtension != w.UseLuauExtension {
				t.Errorf("declaration or luau differ")
			}
		})
	}
}

func TestPrepareCompile(t *testing.T) {
	root := fixtureRoot(t)
	forEachMode(t, func(t *testing.T, want golden) {
		cases := []struct {
			name, project, rojo, include string
			typ                          shared.ProjectType
		}{
			{"ok", "program", "rojo-ok.json", "include", ""},
			{"model", "program", "rojo-model.json", "include", ""},
			{"noRojoFile", "program", "", "include", ""},
			{"noIncludeData", "program", "rojo-ok.json", "elsewhere", ""},
			{"serverInclude", "program", "rojo-server.json", "include", ""},
			{"clientInclude", "program", "rojo-client.json", "include", ""},
			{"isolatedInclude", "program", "rojo-isolated.json", "include", ""},
			{"typePackage", "program", "", "include", shared.ProjectTypePackage},
			{"inSrc", "program", "rojo-insrc.json", "include", ""},
			{"badName", "badname", "rojo-ok.json", "include", ""},
		}
		if len(cases) != len(want.Compile) {
			t.Errorf("%d cases here, %d in the golden", len(cases), len(want.Compile))
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				options := shared.DefaultProjectOptions()
				if c.rojo != "" {
					options.Rojo = filepath.Join(root, c.project, c.rojo)
				}
				options.IncludePath = filepath.Join(root, c.project, c.include)
				options.Type = c.typ
				data, err := CreateProjectData(filepath.Join(root, c.project, "tsconfig.json"), options)
				if err != nil {
					t.Fatal(err)
				}
				if c.rojo == "" {
					data.RojoConfigPath = ""
				}
				parsed, err := GetParsedCommandLine(data)
				if err != nil {
					t.Fatal(err)
				}
				project := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{Checkers: 1})
				translator := CreatePathTranslator(project, data)
				sensitivity := tspath.CaseSensitive
				if runtime.GOOS == "windows" {
					sensitivity = tspath.CaseInsensitive
				}
				context, failure, err := PrepareCompile(project, data, translator, sensitivity)
				if err != nil {
					t.Fatal(err)
				}
				w := want.Compile[c.name]
				if got := messagesOf(failure, root); !slices.Equal(got, w.Diagnostics) && len(got)+len(w.Diagnostics) != 0 {
					t.Errorf("diagnostics\n got %q\nwant %q", got, w.Diagnostics)
				}
				if (failure != nil) != w.Skipped {
					t.Errorf("failed = %v, JS emitSkipped = %v", failure != nil, w.Skipped)
				}
				if failure == nil && context == nil {
					t.Errorf("no context")
				}
			})
		}
	})
}
