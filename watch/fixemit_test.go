package watch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// fixEmitSource compiles differently with the flag on: its continue has to get out of the switch.
const fixEmitSource = `for (const value of [1, 2]) {
	switch (value) {
		case 1:
			continue;
	}
	print(value);
}

export {};
`

// fixEmitMarker is in the output only with the flag on.
const fixEmitMarker = "_continue"

var firstBrace = regexp.MustCompile(`^\s*\{`)

func withFixEmitBlock(tsconfig string, on string) string {
	return firstBrace.ReplaceAllString(tsconfig, `{ "rbxtsgo": { "fixEmit": `+on+` },`)
}

func writeTestFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o666); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

// TestFixEmitForcesIncrementalRebuild: nothing in the sources changes, so only the flag can make the next build a real
// one, and it has to leave what a clean build leaves.
func TestFixEmitForcesIncrementalRebuild(t *testing.T) {
	state := setgetState()
	_, live, clean := newDirs(t, "incfixemit")
	files := renderIncremental(state)
	writeState(t, live, files, nil)
	source := filepath.Join(live, "src", "shared", "fixemit.ts")
	writeTestFile(t, source, fixEmitSource)
	output := filepath.Join(live, "out", "shared", "fixemit.luau")
	build := func(label string, wantMarker bool, extra ...string) {
		t.Helper()
		if text, code := runBuild(t, live, state, extra...); code != 0 {
			t.Fatalf("%s: %s", label, text)
		}
		if got := strings.Contains(readTestFile(t, output), fixEmitMarker); got != wantMarker {
			t.Errorf("%s: fix in the output is %v, and should be %v:\n%s", label, got, wantMarker, readTestFile(t, output))
		}
	}
	build("off", false)
	build("off, no change", false)
	build("flag on", true, "--fix-emit")
	build("flag on, no change", true, "--fix-emit")
	build("flag off again", false)

	// By the tsconfig block, which changes what's hashed twice over, so this also shows the option on its own works: the
	// flag cases above keep the config the same.
	tsconfig := filepath.Join(live, "tsconfig.json")
	plain := readTestFile(t, tsconfig)
	writeTestFile(t, tsconfig, withFixEmitBlock(plain, "true"))
	build("block on", true)
	build("flag over the block", false, "--no-fix-emit")
	build("block on again", true)

	// What's left is what a clean build leaves, in another folder.
	writeState(t, clean, files, nil)
	writeTestFile(t, filepath.Join(clean, "src", "shared", "fixemit.ts"), fixEmitSource)
	writeTestFile(t, filepath.Join(clean, "tsconfig.json"), withFixEmitBlock(plain, "true"))
	if text, code := runBuild(t, clean, state); code != 0 {
		t.Fatal(text)
	}
	if got, want := readTestFile(t, output), readTestFile(t, filepath.Join(clean, "out", "shared", "fixemit.luau")); got != want {
		t.Errorf("not what a clean build leaves:\n%s\nclean:\n%s", got, want)
	}
}

// TestFixEmitInWatchMode: changing the tsconfig block in a running watcher recompiles with the new setting.
func TestFixEmitInWatchMode(t *testing.T) {
	state := setgetState()
	_, live, _ := newDirs(t, "watchfixemit")
	writeState(t, live, render(state), nil)
	writeTestFile(t, filepath.Join(live, "src", "shared", "fixemit.ts"), fixEmitSource)
	tsconfig := filepath.Join(live, "tsconfig.json")
	plain := readTestFile(t, tsconfig)
	output := filepath.Join(live, "out", "shared", "fixemit.luau")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	args := []string{"-p", live, "--watch", "--usePolling"}
	if state[setgetOn] == 1 {
		args = append(args, "--setget")
	}
	command := exec.CommandContext(ctx, testBinary(t), args...)
	command.Dir = live
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = command.Wait()
	}()

	wait := func(label string, wantMarker bool) {
		t.Helper()
		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			if text, err := os.ReadFile(output); err == nil && strings.Contains(string(text), fixEmitMarker) == wantMarker {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		text, _ := os.ReadFile(output)
		t.Fatalf("%s: fix in the output never became %v:\n%s", label, wantMarker, text)
	}
	wait("first build", false)
	writeTestFile(t, tsconfig, withFixEmitBlock(plain, "true"))
	wait("turned on", true)
	writeTestFile(t, tsconfig, withFixEmitBlock(plain, "false"))
	wait("turned off", false)
}
