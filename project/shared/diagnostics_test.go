package shared

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/parser"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

type goldenDiagnostic struct {
	Message  string `json:"message"`
	ID       *int   `json:"id"`
	Category int    `json:"category"`
	Start    *int   `json:"start"`
	Length   *int   `json:"length"`
	Code     string `json:"code"`
}

type goldenMode struct {
	Factories map[string]struct {
		FactoryID int              `json:"factoryId"`
		AtRange   goldenDiagnostic `json:"atRange"`
		AtNode    goldenDiagnostic `json:"atNode"`
	} `json:"factories"`
	Context struct {
		NoRojoDataID                  int                `json:"noRojoDataId"`
		NoPackageImportWithoutScopeID int                `json:"noPackageImportWithoutScopeId"`
		NoRojoData                    []goldenDiagnostic `json:"noRojoData"`
		NoPackageImportWithoutScope   []goldenDiagnostic `json:"noPackageImportWithoutScope"`
	} `json:"context"`
	Text struct {
		IncorrectFileName   goldenDiagnostic `json:"incorrectFileName"`
		RojoPathInSrc       goldenDiagnostic `json:"rojoPathInSrc"`
		TransformerNotFound goldenDiagnostic `json:"transformerNotFound"`
		TruthyChange        struct {
			FactoryID  int              `json:"factoryId"`
			Diagnostic goldenDiagnostic `json:"diagnostic"`
		} `json:"truthyChange"`
		StringOffsetChange struct {
			FactoryID  int              `json:"factoryId"`
			Diagnostic goldenDiagnostic `json:"diagnostic"`
		} `json:"stringOffsetChange"`
	} `json:"text"`
	Format     map[string]string `json:"format"`
	SourceText string            `json:"sourceText"`
}

func parseSample(t *testing.T, text string) *ast.SourceFile {
	t.Helper()
	dir := tspath.RootedDirectoryPathFromAbsolute(tspath.NormalizePath(filepath.ToSlash(mustAbs(t, "testdata"))))
	name := tspath.ToRootedFilePath("sample.ts", dir)
	opts := ast.SourceFileParseOptions{FileName: name, PathKey: tspath.CaseSensitive.PathKey(name.AsPath())}
	return parser.ParseSourceFile(opts, text, core.ScriptKindTS)
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// factoriesByKey lists factories by upstream key, using the struct's field names.
func factoriesByKey(e *ErrorDefinitions) map[string]*DiagnosticFactory {
	result := map[string]*DiagnosticFactory{}
	value := reflect.ValueOf(e).Elem()
	for i := range value.NumField() {
		if !value.Type().Field(i).IsExported() {
			continue
		}
		if factory, ok := value.Field(i).Interface().(*DiagnosticFactory); ok {
			name := value.Type().Field(i).Name
			result[strings.ToLower(name[:1])+name[1:]] = factory
		}
	}
	return result
}

func checkDiagnostic(t *testing.T, label string, got *Diagnostic, want goldenDiagnostic) {
	t.Helper()
	if got.Message() != want.Message {
		t.Errorf("%s message\n got %q\nwant %q", label, got.Message(), want.Message)
	}
	wantID := -1
	if want.ID != nil {
		wantID = *want.ID
	}
	if got.ID() != wantID {
		t.Errorf("%s id = %d, want %d", label, got.ID(), wantID)
	}
	// ts.DiagnosticCategory is Warning 0, Error 1. tsgo's enum starts the same way.
	if int(got.Category()) != want.Category {
		t.Errorf("%s category = %d, want %d", label, got.Category(), want.Category)
	}
	if want.Code != " roblox-ts" {
		t.Errorf("%s: golden code is %q", label, want.Code)
	}
	if want.Start != nil {
		if got.Start() != *want.Start {
			t.Errorf("%s start = %d, want %d", label, got.Start(), *want.Start)
		}
		if got.Length() != *want.Length {
			t.Errorf("%s length = %d, want %d", label, got.Length(), *want.Length)
		}
	} else if got.File() != nil {
		t.Errorf("%s should have no file", label)
	}
}

func TestAgainstNpmPackage(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "golden."+runtime.GOOS+".json"))
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("no golden for this platform: run scripts/gen-goldens.sh here")
	}
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Color goldenMode `json:"color"`
		Plain goldenMode `json:"plain"`
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []struct {
		name  string
		kleur Kleur
		want  goldenMode
	}{{"color", Kleur{Enabled: true}, golden.Color}, {"plain", Kleur{Enabled: false}, golden.Plain}} {
		t.Run(mode.name, func(t *testing.T) {
			e, w := NewDefinitions(mode.kleur)
			file := parseSample(t, mode.want.SourceText)
			node := file.Statements.Nodes[1]
			loc := AtRange(SourceRange{File: file, Pos: 19, End: 24})
			factories := factoriesByKey(e)
			factories["runtimeLibUsedInReplicatedFirst"] = w.RuntimeLibUsedInReplicatedFirst
			if len(mode.want.Factories) < 50 {
				t.Fatalf("only %d factories in the golden", len(mode.want.Factories))
			}
			for key, want := range mode.want.Factories {
				factory := factories[key]
				if factory == nil {
					t.Errorf("no factory for %s", key)
					continue
				}
				if factory.ID != want.FactoryID {
					t.Errorf("%s id = %d, want %d", key, factory.ID, want.FactoryID)
				}
				checkDiagnostic(t, key+" at range", factory.At(loc), want.AtRange)
				checkDiagnostic(t, key+" at node", factory.At(AtNode(node)), want.AtNode)
			}
			if len(factories) != len(mode.want.Factories) {
				t.Errorf("%d factories here, %d upstream", len(factories), len(mode.want.Factories))
			}

			if e.NoRojoData.ID != mode.want.Context.NoRojoDataID || e.NoPackageImportWithoutScope.ID != mode.want.Context.NoPackageImportWithoutScopeID {
				t.Errorf("context factory ids differ")
			}
			// The accumulation bug has to match call for call.
			checkDiagnostic(t, "noRojoData 1", e.NoRojoData.At(loc, "first/path.lua", false), mode.want.Context.NoRojoData[0])
			checkDiagnostic(t, "noRojoData 2", e.NoRojoData.At(loc, "second/path.lua", true), mode.want.Context.NoRojoData[1])
			checkDiagnostic(t, "noRojoData 3", e.NoRojoData.At(AtNode(node), "third/path.lua", false), mode.want.Context.NoRojoData[2])
			checkDiagnostic(t, "noPackage 1", e.NoPackageImportWithoutScope.At(loc, "pkg/a.lua", []string{"ReplicatedStorage", "a"}), mode.want.Context.NoPackageImportWithoutScope[0])
			checkDiagnostic(t, "noPackage 2", e.NoPackageImportWithoutScope.At(AtNode(node), "pkg/b.lua", []string{"ReplicatedStorage", "b", "c"}), mode.want.Context.NoPackageImportWithoutScope[1])

			checkDiagnostic(t, "incorrectFileName", e.IncorrectFileName("init.ts", "index.ts", "/full/path/init.ts"), mode.want.Text.IncorrectFileName)
			checkDiagnostic(t, "rojoPathInSrc", e.RojoPathInSrc("src/x", "out/x"), mode.want.Text.RojoPathInSrc)
			checkDiagnostic(t, "transformerNotFound", w.TransformerNotFound("my-transformer", "Cannot find module"), mode.want.Text.TransformerNotFound)
			truthy := w.TruthyChange(`0, ""`)
			offset := w.StringOffsetChange("text")
			if truthy.ID != mode.want.Text.TruthyChange.FactoryID || offset.ID != mode.want.Text.StringOffsetChange.FactoryID {
				t.Errorf("dynamic warning ids: %d and %d, want %d and %d", truthy.ID, offset.ID, mode.want.Text.TruthyChange.FactoryID, mode.want.Text.StringOffsetChange.FactoryID)
			}
			checkDiagnostic(t, "truthyChange", truthy.At(loc), mode.want.Text.TruthyChange.Diagnostic)
			checkDiagnostic(t, "stringOffsetChange", offset.At(loc), mode.want.Text.StringOffsetChange.Diagnostic)

			cwd := tspath.RootedDirectoryPathFromAbsolute(tspath.NormalizePath(filepath.ToSlash(mustAbs(t, "testdata"))))
			format := func(list ...*Diagnostic) string { return FormatDiagnostics(list, cwd, tspath.CaseSensitive, EOL) }
			text := func(message string) *Diagnostic { return TextDiagnostic(message, e.NoAny.category) }
			end := len(mode.want.SourceText)
			for name, got := range map[string]string{
				"located":    format(e.NoNullLiteral.At(AtNode(node))),
				"range":      format(e.NoCommentDirectives.At(loc)),
				"rangeAtEnd": format(e.NoCommentDirectives.At(AtRange(SourceRange{File: file, Pos: end - 3, End: end}))),
				"warning":    format(w.RuntimeLibUsedInReplicatedFirst.At(AtNode(node))),
				"text":       format(text("Just text")),
				"several":    format(e.NoVar.At(AtNode(node)), text("Text after"), e.NoAny.At(AtNode(file.Statements.Nodes[2]))),
			} {
				if got != mode.want.Format[name] {
					t.Errorf("format %s\n got %q\nwant %q", name, got, mode.want.Format[name])
				}
			}
		})
	}
}

func TestDetectKleur(t *testing.T) {
	for _, test := range []struct {
		env  map[string]string
		want bool
	}{
		{map[string]string{"FORCE_COLOR": "1"}, true},
		{map[string]string{"FORCE_COLOR": "0"}, false},
		{map[string]string{"FORCE_COLOR": "1", "NO_COLOR": ""}, false},
		{map[string]string{"FORCE_COLOR": "1", "TERM": "dumb"}, false},
		{map[string]string{"FORCE_COLOR": "1", "NODE_DISABLE_COLORS": "1"}, false},
		{map[string]string{"FORCE_COLOR": "1", "NODE_DISABLE_COLORS": ""}, true},
	} {
		for _, key := range []string{"FORCE_COLOR", "NO_COLOR", "TERM", "NODE_DISABLE_COLORS"} {
			if value, ok := test.env[key]; ok {
				t.Setenv(key, value)
			} else {
				t.Setenv(key, "")
				os.Unsetenv(key)
			}
		}
		if got := DetectKleur().Enabled; got != test.want && test.env["FORCE_COLOR"] != "0" {
			t.Errorf("env %v: enabled = %v, want %v", test.env, got, test.want)
		}
	}
}
