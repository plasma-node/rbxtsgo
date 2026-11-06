package pathtranslator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
)

type goldenRow struct {
	RootDir              string   `json:"rootDir"`
	OutDir               string   `json:"outDir"`
	Declaration          bool     `json:"declaration"`
	Luau                 bool     `json:"luau"`
	File                 string   `json:"file"`
	Output               string   `json:"output"`
	DeclarationPath      string   `json:"declarationPath"`
	Transformed          string   `json:"transformed"`
	Inputs               []string `json:"inputs"`
	ImportPath           string   `json:"importPath"`
	ImportPathNodeModule string   `json:"importPathNodeModule"`
}

func TestAgainstNpmPackage(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Win32 []goldenRow `json:"win32"`
		Posix []goldenRow `json:"posix"`
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	for _, flavor := range []struct {
		name string
		path *nodepath.Path
		rows []goldenRow
	}{{"win32", nodepath.Win32, golden.Win32}, {"posix", nodepath.Posix, golden.Posix}} {
		t.Run(flavor.name, func(t *testing.T) {
			if len(flavor.rows) < 100 {
				t.Fatalf("only %d rows", len(flavor.rows))
			}
			for _, row := range flavor.rows {
				tr := NewWithPath(flavor.path, row.RootDir, row.OutDir, "", row.Declaration, row.Luau)
				check := func(fn, got, want string) {
					t.Helper()
					if got != want {
						t.Errorf("%s(%q) root=%q out=%q decl=%v luau=%v\n got %q\nwant %q", fn, row.File, row.RootDir, row.OutDir, row.Declaration, row.Luau, got, want)
					}
				}
				check("getOutputPath", tr.GetOutputPath(row.File), row.Output)
				check("getOutputDeclarationPath", tr.GetOutputDeclarationPath(row.File), row.DeclarationPath)
				check("getOutputTransformedPath", tr.GetOutputTransformedPath(row.File), row.Transformed)
				check("getImportPath", tr.GetImportPath(row.File, false), row.ImportPath)
				check("getImportPath node module", tr.GetImportPath(row.File, true), row.ImportPathNodeModule)
				if got := tr.GetInputPaths(row.File); !slices.Equal(got, row.Inputs) {
					t.Errorf("getInputPaths(%q) root=%q decl=%v luau=%v\n got %q\nwant %q", row.File, row.RootDir, row.Declaration, row.Luau, got, row.Inputs)
				}
			}
			t.Logf("%d rows, 6 functions each", len(flavor.rows))
		})
	}
}
