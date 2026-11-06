package difftest

// Tests of the fixes that differ from JS 3.0.0, as the CLI runs them. See testdata/fixemit/README.md. These need what
// scripts/setup-diff-corpus.sh sets up, plus rojo and lune, and skip without them, as the other CLI tests here do.

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	fixEmitBinaryOnce sync.Once
	fixEmitBinary     string
	fixEmitBinaryErr  string
)

// fixEmitCompiler builds the binary once per test run.
func fixEmitCompiler(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	fixEmitBinaryOnce.Do(func() {
		fixEmitBinary = filepath.Join(root, ".scratch", "diff", "fixemit-rbxtsgo.exe")
		goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
		command := exec.Command(goBin, "build", "-o", fixEmitBinary, "./cmd/rbxtsgo")
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			fixEmitBinaryErr = string(output)
		}
	})
	if fixEmitBinaryErr != "" {
		t.Fatal(fixEmitBinaryErr)
	}
	return fixEmitBinary
}

// fixEmitDir makes a folder under .scratch for one project, with the type packages in it. Only src, out and include
// change between runs, so the copy of the packages stays.
func fixEmitDir(t *testing.T, name string) string {
	t.Helper()
	root := repoRoot(t)
	packages := filepath.Join(root, ".scratch", "diff", "setget-lines", "node_modules", "@rbxts")
	if _, err := os.Stat(filepath.Join(packages, "types")); err != nil {
		t.Skip("run scripts/setup-diff-corpus.sh")
	}
	dir := filepath.Join(root, ".scratch", "diff", "fixemit", name)
	copied := filepath.Join(dir, "node_modules", "@rbxts")
	if _, err := os.Stat(filepath.Join(copied, "types")); err != nil {
		if err := os.MkdirAll(filepath.Dir(copied), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.CopyFS(copied, os.DirFS(packages)); err != nil {
			t.Fatal(err)
		}
	}
	for _, sub := range []string{"src", "out", "include"} {
		if err := os.RemoveAll(filepath.Join(dir, sub)); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// layOutProject copies a fixture project over the folder.
func layOutProject(t *testing.T, fixture string, dir string) {
	t.Helper()
	source := filepath.Join(repoRoot(t), "testdata", "fixemit", fixture)
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o777)
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, text, 0o666)
	}); err != nil {
		t.Fatal(err)
	}
}

func jsCompiler(t *testing.T) string {
	t.Helper()
	cli := filepath.Join(repoRoot(t), ".scratch", "golden-deps", "node_modules", "roblox-ts", "out", "CLI", "cli.js")
	if _, err := os.Stat(cli); err != nil {
		t.Skip("run scripts/setup-diff-corpus.sh")
	}
	return cli
}

func luneBinary(t *testing.T) string {
	t.Helper()
	home, _ := os.UserHomeDir()
	matches, _ := filepath.Glob(filepath.Join(home, ".aftman", "tool-storage", "lune-org", "lune", "*", "lune*"))
	if len(matches) == 0 {
		t.Skip("lune isn't installed")
	}
	return matches[len(matches)-1]
}

// tree reads every file under the folder, by path relative to it.
func tree(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	result := map[string][]byte{}
	if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(rel)] = text
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return result
}

func compareTrees(t *testing.T, label string, want, got map[string][]byte) {
	t.Helper()
	for name, text := range want {
		other, ok := got[name]
		if !ok {
			t.Errorf("%s: %s is missing", label, name)
		} else if !bytes.Equal(text, other) {
			t.Errorf("%s: %s differs: %s", label, name, FirstDifference(text, other))
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("%s: %s wasn't expected", label, name)
		}
	}
}

// buildFixture cleans the output and compiles. Output goes to out in dir, and is returned.
func buildFixture(t *testing.T, bin, dir string, args ...string) map[string][]byte {
	t.Helper()
	for _, sub := range []string{"out", "include"} {
		if err := os.RemoveAll(filepath.Join(dir, sub)); err != nil {
			t.Fatal(err)
		}
	}
	if output, code := runCommand(t, dir, bin, args...); code != 0 {
		t.Fatalf("%v: exit %d\n%s", args, code, output)
	}
	return tree(t, filepath.Join(dir, "out"))
}

// runInLune builds the place from what's in dir, and runs the cases.
func runInLune(t *testing.T, dir string) string {
	t.Helper()
	if _, err := exec.LookPath("rojo"); err != nil {
		t.Skip("rojo isn't on PATH")
	}
	place := filepath.Join(dir, "test.rbxl")
	if output, code := runCommand(t, dir, "rojo", "build", ".", "-o", place); code != 0 {
		t.Fatal(output)
	}
	runner := filepath.Join(repoRoot(t), "testdata", "fixemit", "run.luau")
	output, code := runCommand(t, dir, luneBinary(t), "run", runner, place)
	if code != 0 {
		t.Fatalf("lune: exit %d\n%s", code, output)
	}
	return strings.ReplaceAll(output, "\r\n", "\n")
}

// jsSemantics is what the fixture's cases print as plain JS, compiled by tsc and run by Node.
func jsSemantics(t *testing.T, dir string) string {
	t.Helper()
	root := repoRoot(t)
	typescript := filepath.Join(root, ".scratch", "golden-deps", "node_modules", "typescript")
	if _, err := os.Stat(typescript); err != nil {
		t.Skip("run scripts/setup-diff-corpus.sh")
	}
	oracle := filepath.Join(root, "testdata", "fixemit", "oracle.cjs")
	output, code := runCommand(t, dir, "node", oracle, typescript, filepath.Join(dir, "src"))
	if code != 0 {
		t.Fatalf("oracle: exit %d\n%s", code, output)
	}
	return strings.ReplaceAll(output, "\r\n", "\n")
}

func readExpected(t *testing.T, parts ...string) string {
	t.Helper()
	text, err := os.ReadFile(filepath.Join(append([]string{repoRoot(t), "testdata", "fixemit", "expected"}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(text), "\r\n", "\n")
}

// writeModelConfig writes what a model project of plain files needs.
func writeModelConfig(t *testing.T, dir string) {
	t.Helper()
	for name, text := range map[string]string{
		"tsconfig.json":        `{ "compilerOptions": { "allowSyntheticDefaultImports": true, "downlevelIteration": true, "module": "commonjs", "moduleResolution": "Node", "noLib": true, "forceConsistentCasingInFileNames": true, "moduleDetection": "force", "strict": true, "target": "ESNext", "typeRoots": ["node_modules/@rbxts"], "rootDir": "src", "outDir": "out", "baseUrl": "src" }, "include": ["src/**/*"] }`,
		"package.json":         `{ "name": "fixemit", "version": "1.0.0" }`,
		"default.project.json": `{ "name": "fixemit", "tree": { "$className": "DataModel", "ReplicatedStorage": { "$className": "ReplicatedStorage", "rbxts_include": { "$path": "include" } }, "ServerScriptService": { "$className": "ServerScriptService", "ts": { "$path": "out" } } } }`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o666); err != nil {
			t.Fatal(err)
		}
	}
}

// layOutOneFile makes a project of one source file in dir.
func layOutOneFile(t *testing.T, dir, source string) {
	t.Helper()
	writeModelConfig(t, dir)
	if err := os.RemoveAll(filepath.Join(dir, "src")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o777); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", filepath.Base(source)), text, 0o666); err != nil {
		t.Fatal(err)
	}
}

// TestFixEmitOption: the option's surface: --help, a bad value, and a plugin only JS can run.
func TestFixEmitOption(t *testing.T) {
	bin := fixEmitCompiler(t)
	dir := t.TempDir()
	help, _ := runCommand(t, dir, bin, "--help")
	if !strings.Contains(help, "fix-emit") || !strings.Contains(help, "fixEmit") {
		t.Errorf("--help doesn't mention the option:\n%s", help)
	}
	config := filepath.Join(dir, "tsconfig.json")
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(config, []byte(text), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	write(`{ "rbxtsgo": { "fixEmit": "yes" } }`)
	if output, code := runCommand(t, dir, bin); code == 0 || !strings.Contains(output, "must be true or false") {
		t.Errorf("a bad value: exit %d\n%s", code, output)
	}
	// Handing this to the JS compiler would silently lose the fixes.
	write(`{ "compilerOptions": { "plugins": [{ "transform": "other" }] } }`)
	if output, code := runCommand(t, dir, bin, "--fix-emit"); code == 0 || !strings.Contains(output, "doesn't have --fix-emit") {
		t.Errorf("with a plugin only JS can run: exit %d\n%s", code, output)
	}
}
