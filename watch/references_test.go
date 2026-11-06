package watch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectReferencesAreIgnored: rbxtsc gives its program no references, so a tsconfig's "references" change nothing. An
// import into another project is a file outside rootDir. This was checked against JS, which reports the same code.
func TestProjectReferencesAreIgnored(t *testing.T) {
	state := baseState()
	dir, live, _ := newDirs(t, "references")
	writeState(t, live, map[string]string{
		"tsconfig.json":             strings.Replace(render(state)["tsconfig.json"], `"include": ["src/**/*"]`, `"include": ["src/**/*"], "references": [{ "path": "../lib" }]`, 1),
		"default.project.json":      render(state)["default.project.json"],
		"src/server/main.server.ts": "import { fromLib } from \"../../../lib/src/x\";\nprint(fromLib);\n",
	}, nil)
	lib := filepath.Join(dir, "lib")
	if err := os.MkdirAll(filepath.Join(lib, "src"), 0o777); err != nil {
		t.Fatal(err)
	}
	config := strings.Replace(render(state)["tsconfig.json"], `"rootDir": "src",`, `"rootDir": "src", "composite": true, "declaration": true,`, 1)
	if err := os.WriteFile(filepath.Join(lib, "tsconfig.json"), []byte(config), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "src", "x.ts"), []byte("export const fromLib = 1;\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	output, code := runBuild(t, live, state)
	if code == 0 || !strings.Contains(output, "TS6059") || strings.Contains(output, "TS6305") {
		t.Fatalf("expected TS6059, as JS gives. Exit %d:\n%s", code, output)
	}
}
