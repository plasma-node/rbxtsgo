package difftest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFixEmitFlagOffIsJS: with the flag off, the fixture project is what JS writes, byte for byte, at either worker
// count, even though the fixture is all code the fixes change.
func TestFixEmitFlagOffIsJS(t *testing.T) {
	cli := jsCompiler(t)
	bin := fixEmitCompiler(t)
	dir := fixEmitDir(t, "off")
	layOutProject(t, "project", dir)
	if output, code := runCommand(t, dir, "node", cli, "build"); code != 0 {
		t.Fatalf("JS: exit %d\n%s", code, output)
	}
	want := tree(t, filepath.Join(dir, "out"))
	for _, checkers := range []string{"1", "8"} {
		compareTrees(t, "flag off, "+checkers+" workers", want, buildFixture(t, bin, dir, "--checkers="+checkers))
	}
	compareTrees(t, "--no-fix-emit", want, buildFixture(t, bin, dir, "--no-fix-emit"))
}

// TestFixEmitOutput: with the flag on, the output is the expected Luau, whichever way the flag came, at either worker
// count, with setget on or off.
func TestFixEmitOutput(t *testing.T) {
	bin := fixEmitCompiler(t)
	dir := fixEmitDir(t, "on")
	layOutProject(t, "project", dir)
	got := buildFixture(t, bin, dir, "--fix-emit", "--checkers=1")
	expectedDir := filepath.Join(repoRoot(t), "testdata", "fixemit", "expected", "out")
	if os.Getenv("UPDATE_FIXEMIT") == "1" {
		if err := os.RemoveAll(expectedDir); err != nil {
			t.Fatal(err)
		}
		if err := os.CopyFS(expectedDir, os.DirFS(filepath.Join(dir, "out"))); err != nil {
			t.Fatal(err)
		}
	}
	compareTrees(t, "against the expected Luau", tree(t, expectedDir), got)
	compareTrees(t, "8 workers", got, buildFixture(t, bin, dir, "--fix-emit", "--checkers=8"))
	compareTrees(t, "setget", got, buildFixture(t, bin, dir, "--fix-emit", "--setget"))

	// The tsconfig block, and the flag over the block.
	config := filepath.Join(dir, "tsconfig.json")
	text, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, append([]byte(`{ "rbxtsgo": { "fixEmit": true }, `), bytes.TrimPrefix(bytes.TrimSpace(text), []byte("{"))...), 0o666); err != nil {
		t.Fatal(err)
	}
	compareTrees(t, "tsconfig block", got, buildFixture(t, bin, dir))
	off := buildFixture(t, bin, dir, "--no-fix-emit")
	if bytes.Equal(off["cases/continue_switch.luau"], got["cases/continue_switch.luau"]) {
		t.Errorf("--no-fix-emit should win over the tsconfig block")
	}
}

// TestFixEmitRuntime: the same cases, run. Off shows the bug, and on does what Node does with the same TypeScript. The
// cases are the sections of what each prints.
func TestFixEmitRuntime(t *testing.T) {
	bin := fixEmitCompiler(t)
	dir := fixEmitDir(t, "runtime")
	layOutProject(t, "project", dir)

	js := sections(jsSemantics(t, dir))
	buildFixture(t, bin, dir, "--no-fix-emit")
	off := sections(runInLune(t, dir))
	buildFixture(t, bin, dir, "--fix-emit")
	on := sections(runInLune(t, dir))
	buildFixture(t, bin, dir, "--fix-emit", "--setget")
	onSetget := sections(runInLune(t, dir))

	if len(js) == 0 {
		t.Fatal("no cases ran")
	}
	expected := filepath.Join(repoRoot(t), "testdata", "fixemit", "expected", "runtime")
	for name, want := range js {
		if on[name] != want {
			t.Errorf("%s with the flag on doesn't do what JS does: %s", name, FirstDifference([]byte(want), []byte(on[name])))
		}
		if onSetget[name] != want {
			t.Errorf("%s with the flag on and setget doesn't do what JS does: %s", name, FirstDifference([]byte(want), []byte(onSetget[name])))
		}
		// A case that runs the same both ways proves nothing about the fix it's for.
		if off[name] == on[name] {
			t.Errorf("%s runs the same with the flag on and off:\n%s", name, on[name])
		}
		if os.Getenv("UPDATE_FIXEMIT") == "1" {
			if err := os.MkdirAll(expected, 0o777); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(expected, name+".off.txt"), []byte(off[name]), 0o666); err != nil {
				t.Fatal(err)
			}
		}
		if want := readExpected(t, "runtime", name+".off.txt"); off[name] != want {
			t.Errorf("%s with the flag off: %s", name, FirstDifference([]byte(want), []byte(off[name])))
		}
	}
}

// sections splits what the cases print by case.
func sections(text string) map[string]string {
	result := map[string]string{}
	for _, section := range strings.Split(text, "== ")[1:] {
		name, body, _ := strings.Cut(section, "\n")
		result[name] = body
	}
	return result
}
