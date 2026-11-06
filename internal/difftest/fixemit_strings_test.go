package difftest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestFixEmitStrings: with the flag off, a string changes only if JS's Luau for it fails to parse, and then it parses.
// With the flag on, every string has the value JS gives it.
func TestFixEmitStrings(t *testing.T) {
	cli := jsCompiler(t)
	bin := fixEmitCompiler(t)
	lune := luneBinary(t)
	root := repoRoot(t)
	dir := fixEmitDir(t, "strings")
	writeModelConfig(t, dir)
	if output, code := runCommand(t, dir, "node", filepath.Join(root, "testdata", "fixemit", "strings", "gen.cjs"), dir); code != 0 {
		t.Fatal(output)
	}
	check := func(label, out string) map[string]string {
		t.Helper()
		output, code := runCommand(t, dir, lune, "run", filepath.Join(root, "testdata", "fixemit", "strings", "check.luau"), dir, out)
		if code != 0 {
			t.Fatalf("%s: exit %d\n%s", label, code, output)
		}
		result := map[string]string{}
		for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
			if fields := strings.Fields(line); len(fields) == 2 {
				result[fields[0]] = fields[1]
			}
		}
		return result
	}
	if output, code := runCommand(t, dir, "node", cli, "build", "--type=model"); code != 0 {
		t.Fatalf("JS: exit %d\n%s", code, output)
	}
	jsOut := tree(t, filepath.Join(dir, "out"))
	jsStatus := check("JS", "out")
	if len(jsStatus) < 60 {
		t.Fatalf("only %d cases ran", len(jsStatus))
	}

	offOut := buildFixture(t, bin, dir, "--type=model")
	offStatus := check("off", "out")
	invalid := 0
	for name, status := range jsStatus {
		file := name + ".luau"
		if status == "parse-error" {
			invalid++
			if offStatus[name] != "ok" {
				t.Errorf("%s is %s with the flag off, and it failed to parse in JS, so it can be right:\n%s", name, offStatus[name], offOut[file])
			}
		} else if !bytes.Equal(jsOut[file], offOut[file]) {
			t.Errorf("%s parses in JS's output, so the flag can't be off and have it change. JS:\n%s\nrbxtsgo:\n%s", name, jsOut[file], offOut[file])
		}
	}
	if invalid == 0 {
		t.Error("JS's output for every case parses, which would make the always-on fix unnecessary")
	}

	buildFixture(t, bin, dir, "--type=model", "--fix-emit")
	onStatus := check("on", "out")
	for name, status := range onStatus {
		if status != "ok" {
			t.Errorf("%s with the flag on: %s", name, status)
		}
	}

	// A record of what each case does in JS, to read and to notice if it moves.
	var names []string
	for name := range jsStatus {
		names = append(names, name)
	}
	slices.Sort(names)
	var record strings.Builder
	for _, name := range names {
		fmt.Fprintf(&record, "%s js=%s off=%s on=%s\n", name, jsStatus[name], offStatus[name], onStatus[name])
	}
	path := filepath.Join(root, "testdata", "fixemit", "strings", "expected.txt")
	if os.Getenv("UPDATE_FIXEMIT") == "1" {
		if err := os.WriteFile(path, []byte(record.String()), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := record.String(); strings.ReplaceAll(string(want), "\r\n", "\n") != got {
		t.Errorf("strings/expected.txt: %s", FirstDifference(want, []byte(got)))
	}
}

// TestAcceptedStringGoldens covers the two goldens that left the baseline with this fix, because they have \uD83D in them.
// JS's Luau for that file fails to parse, and the only change to it is the braces. See docs/known-differences.md.
func TestAcceptedStringGoldens(t *testing.T) {
	bin := fixEmitCompiler(t)
	lune := luneBinary(t)
	root := repoRoot(t)
	dir := fixEmitDir(t, "unicode")
	writeModelConfig(t, dir)
	layOutOneFile(t, dir, filepath.Join(root, "testdata", "differential", "fixtures", "core", "src", "shared", "strings_unicode.ts"))
	escape := regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)
	parses := func(path string) bool {
		t.Helper()
		output, code := runCommand(t, dir, lune, "run", filepath.Join(root, "testdata", "fixemit", "strings", "parses.luau"), path)
		if code != 0 {
			t.Fatalf("exit %d\n%s", code, output)
		}
		return strings.TrimSpace(output) == "parses"
	}
	for _, test := range []struct {
		corpus string
		args   []string
	}{
		{corpus: "fixtures", args: []string{"--type=model"}},
		{corpus: "setget-fixtures", args: []string{"--type=model", "--setget"}},
	} {
		golden := filepath.Join(root, "testdata", "differential", "golden", test.corpus, "files", "shared", "strings_unicode.luau")
		text, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if parses(golden) {
			t.Errorf("%s: JS's output parses, so this difference can't be accepted", test.corpus)
		}
		got := buildFixture(t, bin, dir, test.args...)["strings_unicode.luau"]
		if want := escape.ReplaceAll(text, []byte(`\u{$1}`)); !bytes.Equal(want, got) {
			t.Errorf("%s: more changed than the escapes: %s", test.corpus, FirstDifference(want, got))
		}
		if !parses(filepath.Join(dir, "out", "strings_unicode.luau")) {
			t.Errorf("%s: this still fails to parse", test.corpus)
		}
	}
}
