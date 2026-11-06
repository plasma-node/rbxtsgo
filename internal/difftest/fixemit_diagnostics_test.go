package difftest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUnsupportedIterationDiagnostics: where JS 3.0.0 crashes on a type it can't iterate, this reports what upstream's
// master reports (#3052), with or without the flag, and without taking down the rest of the build. noUnsupportedIteration.*
// are upstream's tests.
func TestUnsupportedIterationDiagnostics(t *testing.T) {
	cli := jsCompiler(t)
	bin := fixEmitCompiler(t)
	dir := fixEmitDir(t, "diagnostics")
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), "testdata", "fixemit", "diagnostics"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 4 {
		t.Fatalf("only %d cases", len(entries))
	}
	for _, entry := range entries {
		layOutOneFile(t, dir, filepath.Join(repoRoot(t), "testdata", "fixemit", "diagnostics", entry.Name()))
		js, code := runCommand(t, dir, "node", cli, "build", "--type=model")
		if code == 0 || !strings.Contains(js, "Assertion Failed!") {
			t.Errorf("%s: JS 3.0.0 should crash on this, and gave exit %d:\n%s", entry.Name(), code, js)
		}
		for _, args := range [][]string{{"--type=model"}, {"--type=model", "--fix-emit"}, {"--type=model", "--setget"}} {
			output, code := runCommand(t, dir, bin, args...)
			if code != 1 || strings.Contains(output, "internal error") || strings.Contains(output, "Assertion Failed") {
				t.Errorf("%s %v: exit %d\n%s", entry.Name(), args, code, output)
			}
			if !strings.Contains(ansi.ReplaceAllString(output, ""), "Cannot iterate over this type!") {
				t.Errorf("%s %v: wrong diagnostic\n%s", entry.Name(), args, output)
			}
		}
	}
}
