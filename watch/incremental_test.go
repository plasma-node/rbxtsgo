package watch

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"maps"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// These tests are here because they share the edit fixture, and they run the real binary because the build command's
// wiring is part of what's tested.
var (
	binaryOnce sync.Once
	binaryPath string
	binaryErr  error
)

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(file))
}

func testBinary(t testing.TB) string {
	t.Helper()
	binaryOnce.Do(func() {
		root := filepath.Join(repoRoot(), ".scratch", "watchtest")
		binaryPath = filepath.Join(root, "rbxtsgo-test")
		if runtime.GOOS == "windows" {
			binaryPath += ".exe"
		}
		goBin, err := exec.LookPath("go")
		if err != nil {
			binaryErr = fmt.Errorf("go was not found on PATH: install Go (the version in go.mod) and make sure it is on PATH: %v", err)
			return
		}
		cmd := exec.Command(goBin, "build", "-o", binaryPath, "./cmd/rbxtsgo")
		cmd.Dir = repoRoot()
		if output, err := cmd.CombinedOutput(); err != nil {
			binaryErr = fmt.Errorf("%v: %s", err, output)
		}
	})
	if binaryErr != nil {
		t.Fatal(binaryErr)
	}
	return binaryPath
}

func runBuild(t testing.TB, dir string, state map[string]int, extra ...string) (string, int) {
	t.Helper()
	args := []string{"-p", dir}
	if state[setgetOn] == 1 {
		args = append(args, "--setget")
	}
	cmd := exec.Command(testBinary(t), append(args, extra...)...)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = exitErr.ExitCode()
	}
	return output.String(), code
}

var withIncremental = regexp.MustCompile(`"compilerOptions": \{`)

// renderIncremental is the fixture with incremental builds on.
func renderIncremental(state map[string]int) map[string]string {
	files := render(state)
	files["tsconfig.json"] = withIncremental.ReplaceAllString(files["tsconfig.json"], "\"compilerOptions\": {\n\t\t\"incremental\": true,\n\t\t\"tsBuildInfoFile\": \"out/build.tsbuildinfo\",")
	return files
}

func withoutBuildInfo(tree map[string]string) map[string]string {
	result := maps.Clone(tree)
	delete(result, "out/build.tsbuildinfo")
	return result
}

// TestIncrementalBuilds: after each edit in a randomized sequence, a warm incremental build must leave the same output as
// a clean build of the same sources.
func TestIncrementalBuilds(t *testing.T) {
	seeds := []uint64{21, 22, 23}
	steps := 30
	if testing.Short() {
		seeds = seeds[:1]
		steps = 10
	}
	for _, seed := range seeds {
		for _, variant := range []string{"", "setget", "declarations"} {
			name := fmt.Sprintf("seed%d%s", seed, variant)
			start := baseState()
			switch variant {
			case "setget":
				start = setgetState()
			case "declarations":
				start[declarationsOn] = 1
			}
			t.Run(name, func(t *testing.T) {
				_, live, clean := newDirs(t, "inc"+name)
				rng := rand.New(rand.NewPCG(seed, 0))
				writeState(t, live, renderIncremental(start), nil)
				state := start
				if output, code := runBuild(t, live, state); code != 0 {
					t.Fatalf("first build failed: %s", output)
				}
				if _, err := os.Stat(filepath.Join(live, "out", "build.tsbuildinfo")); err != nil {
					t.Fatal("no build info after a build: ", err)
				}
				for step := range steps {
					next := randomStep(rng, state)
					label := fmt.Sprintf("seed %d step %d (%s)", seed, step, describeChange(state, next))
					t.Log(label)
					writeState(t, live, renderIncremental(next), renderIncremental(state))
					state = next
					output, code := runBuild(t, live, state)
					_, cleanErrors := cleanBuild(t, live, clean, state)
					if (code != 0) != (cleanErrors != 0) {
						t.Fatalf("%s: exit code %d, and a clean build has %d errors\n%s", label, code, cleanErrors, output)
					}
					if code != 0 {
						continue
					}
					want := readTree(t, filepath.Join(clean, "out"), filepath.Join(clean, "include"))
					got := withoutBuildInfo(readTree(t, filepath.Join(live, "out"), filepath.Join(live, "include")))
					if problem := compareTrees(want, got); problem != "" {
						t.Fatalf("%s: output differs from a clean build:\n%s", label, problem)
					}
				}
			})
		}
	}
}

func modTimes(t testing.TB, root string) map[string]time.Time {
	t.Helper()
	result := map[string]time.Time{}
	for name := range readTree(t, root) {
		if strings.HasSuffix(name, "/") {
			continue
		}
		info, err := os.Stat(filepath.Join(filepath.Dir(root), filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		result[name] = info.ModTime()
	}
	return result
}

func compiledFiles(t testing.TB, root string) int {
	count := 0
	for name := range readTree(t, root) {
		if strings.HasSuffix(name, ".luau") {
			count++
		}
	}
	return count
}

// TestIncrementalInvalidation covers what forces work, and what doesn't.
func TestIncrementalInvalidation(t *testing.T) {
	state := setgetState()
	_, live, _ := newDirs(t, "incinval")
	writeState(t, live, renderIncremental(state), nil)
	out := filepath.Join(live, "out")
	build := func(label string, extra ...string) {
		t.Helper()
		if output, code := runBuild(t, live, state, extra...); code != 0 {
			t.Fatalf("%s: %s", label, output)
		}
	}
	build("first build")
	first := modTimes(t, out)

	// Nothing changed: no output gets rewritten.
	build("no change")
	for name, stamp := range modTimes(t, out) {
		if !stamp.Equal(first[name]) && !strings.HasSuffix(name, ".txt") && !strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".lua") {
			t.Errorf("a build with no changes rewrote %s", name)
		}
	}

	// A file that went missing comes back, on its own.
	luau := filepath.Join(out, "shared", "user.luau")
	want, err := os.ReadFile(luau)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(luau); err != nil {
		t.Fatal(err)
	}
	build("output removed")
	if got, err := os.ReadFile(luau); err != nil || !bytes.Equal(got, want) {
		t.Errorf("a removed output didn't come back: %v", err)
	}
	// So does a whole out folder.
	if err := os.RemoveAll(out); err != nil {
		t.Fatal(err)
	}
	build("out removed")
	if got, err := os.ReadFile(luau); err != nil || !bytes.Equal(got, want) {
		t.Errorf("a removed out folder didn't come back: %v", err)
	}
	if count := compiledFiles(t, out); count < 20 {
		t.Errorf("only %d files came back", count)
	}

	// A flag that changes output paths forces a rebuild, with no stale files left behind.
	build("--luau=false", "--luau=false")
	// A damaged build info is no build info.
	buildInfo := filepath.Join(out, "build.tsbuildinfo")
	if err := os.WriteFile(buildInfo, []byte("{ not json"), 0o666); err != nil {
		t.Fatal(err)
	}
	build("damaged build info")
	if text, err := os.ReadFile(buildInfo); err != nil || !strings.Contains(string(text), "rbxtsgo-1") {
		t.Errorf("the damaged build info wasn't replaced: %v", err)
	}
}

// TestErrorsSurviveIncrementalBuilds: a failing build writes nothing and records nothing, so its errors come back on the
// next run even though the file hasn't changed since, and fixing them gives what a clean build gives.
func TestErrorsSurviveIncrementalBuilds(t *testing.T) {
	state := setgetState()
	_, live, clean := newDirs(t, "incerrors")
	writeState(t, live, renderIncremental(state), nil)
	if output, code := runBuild(t, live, state); code != 0 {
		t.Fatal(output)
	}
	out := filepath.Join(live, "out")
	before := readTree(t, out)

	broken := maps.Clone(state)
	broken["bad"] = 1
	broken["msg"] = 1
	writeState(t, live, renderIncremental(broken), renderIncremental(state))
	first, code := runBuild(t, live, broken)
	if code == 0 {
		t.Fatalf("expected an error, got: %s", first)
	}
	second, code := runBuild(t, live, broken)
	if code == 0 || first != second {
		t.Fatalf("the error should come back the same on an unchanged file.\n--- first\n%s\n--- second (exit %d)\n%s", first, code, second)
	}
	if problem := compareTrees(before, readTree(t, out)); problem != "" {
		t.Fatalf("a failing build changed the output:\n%s", problem)
	}

	// A file that breaks because of a change in what it imports has to keep failing too.
	fixed := maps.Clone(broken)
	fixed["bad"] = 0
	writeState(t, live, renderIncremental(fixed), renderIncremental(broken))
	if output, code := runBuild(t, live, fixed); code != 0 {
		t.Fatal(output)
	}
	cleanBuild(t, live, clean, fixed)
	want := readTree(t, filepath.Join(clean, "out"), filepath.Join(clean, "include"))
	got := withoutBuildInfo(readTree(t, filepath.Join(live, "out"), filepath.Join(live, "include")))
	if problem := compareTrees(want, got); problem != "" {
		t.Fatalf("after the fix, output differs from a clean build:\n%s", problem)
	}
	withExtraUser := maps.Clone(fixed)
	withExtraUser["extra"], withExtraUser["extraUser"] = 1, 1
	writeState(t, live, renderIncremental(withExtraUser), renderIncremental(fixed))
	if output, code := runBuild(t, live, withExtraUser); code != 0 {
		t.Fatal(output)
	}
	removed := maps.Clone(withExtraUser)
	removed["extra"] = 0
	writeState(t, live, renderIncremental(removed), renderIncremental(withExtraUser))
	first, code = runBuild(t, live, removed)
	second, code2 := runBuild(t, live, removed)
	if code == 0 || code2 == 0 || first != second {
		t.Fatalf("deleting an import target should keep failing the file that imports it.\n%s\n---\n%s", first, second)
	}
}

// TestPackageChangesForceARebuild: changing an installed package invalidates everything, including where that only shows
// in its package.json.
func TestPackageChangesForceARebuild(t *testing.T) {
	state := baseState()
	_, live, _ := newDirs(t, "incpackages")
	writeState(t, live, renderIncremental(state), nil)
	out := filepath.Join(live, "out")
	build := func() {
		t.Helper()
		if output, code := runBuild(t, live, state); code != 0 {
			t.Fatal(output)
		}
	}
	build()
	rebuilt := func(label string) {
		t.Helper()
		before := modTimes(t, out)
		build()
		for name, stamp := range modTimes(t, out) {
			if strings.HasSuffix(name, ".luau") && stamp.Equal(before[name]) {
				t.Errorf("%s: %s wasn't rebuilt", label, name)
				return
			}
		}
	}
	types := filepath.Join(live, "node_modules", "@rbxts", "compiler-types", "package.json")
	text, err := os.ReadFile(types)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(types, append(text, '\n'), 0o666); err != nil {
		t.Fatal(err)
	}
	rebuilt("package.json")
	declarations, err := filepath.Glob(filepath.Join(live, "node_modules", "@rbxts", "types", "include", "*.d.ts"))
	if err != nil || len(declarations) == 0 {
		t.Fatalf("no declarations found: %v", err)
	}
	text, err = os.ReadFile(declarations[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(declarations[0], append(text, []byte("\ninterface AddedByTheTest {}\n")...), 0o666); err != nil {
		t.Fatal(err)
	}
	rebuilt("a package's declarations")
	if err := os.MkdirAll(filepath.Join(live, "node_modules", "@rbxts", "newpackage"), 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(live, "node_modules", "@rbxts", "newpackage", "package.json"), []byte(`{ "name": "@rbxts/newpackage", "main": "x.lua" }`), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(live, "node_modules", "@rbxts", "newpackage", "index.d.ts"), []byte("export {};\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	rebuilt("a new package")
}

// TestBuildInfoSharedWithJS: rbxtsc and rbxtsgo can take turns on one tsBuildInfoFile. Each just starts over.
func TestBuildInfoSharedWithJS(t *testing.T) {
	cli := filepath.Join(repoRoot(), ".scratch", "golden-deps", "node_modules", "roblox-ts", "out", "CLI", "cli.js")
	if _, err := os.Stat(cli); err != nil {
		t.Skip("no JS compiler: run scripts/setup-diff-corpus.sh")
	}
	state := baseState()
	_, live, clean := newDirs(t, "incjs")
	writeState(t, live, renderIncremental(state), nil)
	js := func() {
		t.Helper()
		cmd := exec.Command("node", cli, "-p", live)
		cmd.Dir = live
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("JS: %v\n%s", err, output)
		}
	}
	check := func(label string) {
		t.Helper()
		cleanBuild(t, live, clean, state)
		want := readTree(t, filepath.Join(clean, "out"), filepath.Join(clean, "include"))
		got := withoutBuildInfo(readTree(t, filepath.Join(live, "out"), filepath.Join(live, "include")))
		if problem := compareTrees(want, got); problem != "" {
			t.Fatalf("%s: output differs from a clean build:\n%s", label, problem)
		}
	}
	if output, code := runBuild(t, live, state); code != 0 {
		t.Fatal(output)
	}
	check("rbxtsgo first")
	js()
	check("JS over rbxtsgo's build info")
	next := maps.Clone(state)
	next["msg"] = 1
	writeState(t, live, renderIncremental(next), renderIncremental(state))
	state = next
	if output, code := runBuild(t, live, state); code != 0 {
		t.Fatal(output)
	}
	check("rbxtsgo over JS's build info")
	next = maps.Clone(state)
	next["msg"] = 2
	writeState(t, live, renderIncremental(next), renderIncremental(state))
	state = next
	js()
	check("JS after an edit, over rbxtsgo's build info")
}

// TestWatchUsesAndKeepsBuildInfo: watch mode reads and writes the same file, so a watcher starting on an up-to-date
// project compiles nothing, and a one-shot build after a watcher skips everything and matches a clean build. It runs the
// binary, because the key includes the compiler's identity.
func TestWatchUsesAndKeepsBuildInfo(t *testing.T) {
	t.Run("plain", func(t *testing.T) { testWatchThenBuild(t, "incwatch", baseState()) })
	t.Run("setget", func(t *testing.T) { testWatchThenBuild(t, "incwatchsetget", setgetState()) })
}

func testWatchThenBuild(t *testing.T, name string, state map[string]int) {
	_, live, clean := newDirs(t, name)
	writeState(t, live, renderIncremental(state), nil)
	if output, code := runBuild(t, live, state); code != 0 {
		t.Fatal(output)
	}
	args := []string{"-w", "--verbose", "-p", live}
	if state[setgetOn] == 1 {
		args = append(args, "--setget")
	}
	cmd := exec.Command(testBinary(t), args...)
	cmd.Dir = live
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := sync.OnceFunc(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	defer stopped()
	timings := make(chan string, 16)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "[timing]") {
				timings <- scanner.Text()
			}
		}
	}()
	wait := func() string {
		t.Helper()
		select {
		case line := <-timings:
			return line
		case <-time.After(2 * time.Minute):
			t.Fatal("no compile finished")
			return ""
		}
	}
	if line := wait(); !strings.Contains(line, "lowered 0,") {
		t.Errorf("starting a watcher on an up-to-date project compiled something: %s", line)
	}
	time.Sleep(200 * time.Millisecond)
	next := maps.Clone(state)
	next["msg"] = 1
	writeState(t, live, renderIncremental(next), renderIncremental(state))
	if line := wait(); !strings.Contains(line, "lowered 1,") {
		t.Errorf("an edit of a leaf file should lower that file: %s", line)
	}
	stopped()

	before := modTimes(t, filepath.Join(live, "out"))
	output, code := runBuild(t, live, next, "--verbose")
	if code != 0 {
		t.Fatal(output)
	}
	if !strings.Contains(output, "checked 0, lowered 0") {
		t.Errorf("a build after the watcher should skip everything: %s", output)
	}
	for name, stamp := range modTimes(t, filepath.Join(live, "out")) {
		if strings.HasSuffix(name, ".luau") && !stamp.Equal(before[name]) {
			t.Errorf("a build after the watcher rewrote %s", name)
		}
	}
	cleanBuild(t, live, clean, next)
	want := readTree(t, filepath.Join(clean, "out"), filepath.Join(clean, "include"))
	got := withoutBuildInfo(readTree(t, filepath.Join(live, "out"), filepath.Join(live, "include")))
	if problem := compareTrees(want, got); problem != "" {
		t.Fatalf("after the watcher and a build, output differs from a clean build:\n%s", problem)
	}
}

// TestEverySpacedSaveGetsAReport: on real file events, a save that lands while nothing is running gets its own compile
// and its own status line. Saves only merge when one lands during a compile or its debounce.
func TestEverySpacedSaveGetsAReport(t *testing.T) {
	state := baseState()
	_, live, _ := newDirs(t, "spaced")
	writeState(t, live, render(state), nil)
	var output syncBuffer
	cycles := make(chan *Cycle, 64)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Config{Load: loadFor(live), Output: &output, OnCycle: func(c *Cycle) { cycles <- c }})
	}()
	defer func() {
		cancel()
		<-done
	}()
	<-cycles
	files := []string{"src/server/main.server.ts", "src/shared/user.ts", "src/shared/types.ts"}
	const saves = 12
	for i := range saves {
		time.Sleep(500 * time.Millisecond)
		full := filepath.Join(live, filepath.FromSlash(files[i%len(files)]))
		text, err := os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, append(text, []byte(fmt.Sprintf("\nexport const added%d = %d;\n", i, i))...), 0o666); err != nil {
			t.Fatal(err)
		}
		select {
		case <-cycles:
		case <-time.After(time.Minute):
			t.Fatalf("save %d got no compile", i)
		}
	}
	time.Sleep(time.Second)
	output.mu.Lock()
	text := output.String()
	output.mu.Unlock()
	if found := strings.Count(text, "Found 0 errors. Watching for file changes."); found < saves+1 {
		t.Errorf("%d status lines for %d saves and a first build:\n%s", found, saves, text)
	}
}
