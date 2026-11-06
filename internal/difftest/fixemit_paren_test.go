package difftest

import (
	"strings"
	"testing"
)

// TestParenthesizedArrayAssignment: JS 3.0.0 crashes on [(a)] = b. Upstream's master compiles it (#3091), and so does
// this, with or without the flag.
func TestParenthesizedArrayAssignment(t *testing.T) {
	cli := jsCompiler(t)
	bin := fixEmitCompiler(t)
	dir := fixEmitDir(t, "paren")
	layOutProject(t, "project-crashes", dir)
	if js, code := runCommand(t, dir, "node", cli, "build"); code == 0 || !strings.Contains(js, "Assertion Failed!") {
		t.Errorf("JS 3.0.0 should crash on this, and gave exit %d:\n%s", code, js)
	}
	want := jsSemantics(t, dir)
	buildFixture(t, bin, dir)
	if got := runInLune(t, dir); got != want {
		t.Errorf("flag off: %s", FirstDifference([]byte(want), []byte(got)))
	}
	buildFixture(t, bin, dir, "--fix-emit")
	if got := runInLune(t, dir); got != want {
		t.Errorf("flag on: %s", FirstDifference([]byte(want), []byte(got)))
	}
}
