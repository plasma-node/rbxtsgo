package compile_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/compile"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

func readAll(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(item string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(item)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, item)
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func copyTree(t *testing.T, src, dest string) {
	t.Helper()
	err := filepath.WalkDir(src, func(item string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, item)
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o777)
		}
		data, err := os.ReadFile(item)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o666)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestDeclarationGolden builds a package project and compares every output file with what JS rbxtsc wrote. Differences
// that are accepted are listed in acceptedDeclarationDiffs, with the reason in docs/known-differences.md.
func TestDeclarationGolden(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repo := filepath.Dir(filepath.Dir(file))
	packages := filepath.Join(repo, ".scratch", "diff", "tests", "node_modules", "@rbxts")
	if _, err := os.Stat(filepath.Join(packages, "types")); err != nil {
		t.Skip("no packages: run scripts/setup-diff-corpus.sh")
	}
	work := filepath.Join(repo, ".scratch", fmt.Sprintf("decl-go-%d", os.Getpid()))
	if err := os.RemoveAll(work); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(work) })
	copyTree(t, filepath.Join(repo, "testdata", "declarations", "fixture"), work)
	copyTree(t, packages, filepath.Join(work, "node_modules", "@rbxts"))

	data, err := project.CreateProjectData(filepath.Join(work, "tsconfig.json"), shared.DefaultProjectOptions())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := project.GetParsedCommandLine(data)
	if err != nil {
		t.Fatal(err)
	}
	p := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{})
	translator := project.CreatePathTranslator(p, data)
	if err := project.CopyFiles(data, translator, project.GetRootDirs(p.Options())); err != nil {
		t.Fatal(err)
	}
	prepared, failure, err := project.PrepareCompile(p, data, translator, p.Program().CaseSensitivity())
	if err != nil || failure != nil {
		t.Fatal(err, failure)
	}
	result, err := compile.CompileFiles(context.Background(), &compile.Context{
		Project: p, Data: data, Translator: translator, Prepared: prepared, Replay: shared.NewContextReplay(),
	}, p.SourceFilesToCompile())
	if err != nil {
		t.Fatal(err)
	}
	if shared.HasErrors(result.Diagnostics) {
		t.Fatal(shared.FormatDiagnostics(result.Diagnostics, shared.ProcessCwd(), p.Program().CaseSensitivity(), "\n"))
	}

	want := readAll(t, filepath.Join(repo, "testdata", "declarations", "golden"))
	got := readAll(t, filepath.Join(work, "out"))
	var names []string
	for name := range want {
		names = append(names, name)
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	for _, name := range names {
		expected, inWant := want[name]
		actual, inGot := got[name]
		switch {
		case !inWant:
			t.Errorf("%s: only rbxtsgo wrote this", name)
		case !inGot:
			t.Errorf("%s: only JS wrote this", name)
		case expected != actual:
			if accepted, ok := acceptedDeclarationDiffs[name]; ok && accepted == actual {
				continue
			}
			t.Errorf("%s differs\n--- JS\n%s\n--- rbxtsgo\n%s", name, expected, actual)
		}
	}
}

// acceptedDeclarationDiffs holds the whole file TS 7 prints where it differs from TS 5.5.3.
var acceptedDeclarationDiffs = map[string]string{}
