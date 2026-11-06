package tsbridge

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

var ctx = context.Background()

// copyFixture copies testdata/project to a temp folder, naming _node_modules back to node_modules.
// The repo's .gitignore keeps a real node_modules out of git.
func copyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join("testdata", "project")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = strings.Replace(rel, "_node_modules", "node_modules", 1)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dir, rel), 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func load(t *testing.T, dir, configName string, checkers int) (*ParsedConfig, *Project) {
	t.Helper()
	parsed, err := ParseConfig(filepath.Join(dir, configName))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("parser reported %d diagnostics, first: %v", len(parsed.Diagnostics), parsed.Diagnostics[0])
	}
	return parsed, NewProject(parsed, ProjectOptions{Checkers: checkers})
}

func relName(dir string, name tspath.RootedFilePath) string {
	dir = tspath.NormalizePath(filepath.ToSlash(dir))
	return strings.TrimPrefix(name.AsString(), dir+"/")
}

func formatDiagnostic(dir string, d *ast.Diagnostic) string {
	text := flatten(d)
	if d.File() == nil {
		return fmt.Sprintf("TS%d: %s", d.Code(), text)
	}
	line, character := scanner.GetECMALineAndUTF16CharacterOfPosition(d.File(), d.Pos())
	return fmt.Sprintf("%s(%d,%d): TS%d: %s", relName(dir, d.File().FileName()), line+1, character+1, d.Code(), text)
}

func flatten(d *ast.Diagnostic) string {
	text := d.String()
	for _, child := range d.MessageChain() {
		text += "\n" + flatten(child)
	}
	return text
}

func readGolden(t *testing.T, name string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "golden", name))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimRight(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func compareLines(t *testing.T, label string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s differs from the TS 5.5.3 golden\n--- got\n%s\n--- want\n%s", label, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestNormalizeOptions(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 1)
	opts := project.Options()
	if opts.BaseUrl != "" {
		t.Errorf("baseUrl = %q", opts.BaseUrl)
	}
	if opts.ModuleResolution != core.ModuleResolutionKindBundler {
		t.Errorf("moduleResolution = %v", opts.ModuleResolution)
	}
	if !opts.ResolvePackageJsonExports.IsFalse() || !opts.ResolvePackageJsonImports.IsFalse() {
		t.Errorf("package.json exports and imports should be off, got %v and %v", opts.ResolvePackageJsonExports, opts.ResolvePackageJsonImports)
	}
	if !opts.DownlevelIteration.IsUnknown() {
		t.Errorf("downlevelIteration = %v", opts.DownlevelIteration)
	}
	if !slices.Equal(opts.Types, []string{"*"}) {
		t.Errorf("types = %v", opts.Types)
	}
	if opts.Module != core.ModuleKindCommonJS {
		t.Errorf("module = %v, must stay commonjs", opts.Module)
	}
	if !opts.NoUncheckedSideEffectImports.IsFalse() {
		t.Errorf("noUncheckedSideEffectImports = %v", opts.NoUncheckedSideEffectImports)
	}
	if patterns, ok := opts.Paths.Get("*"); !ok || !slices.Equal(patterns, []string{"./*"}) {
		t.Errorf("paths[*] = %v", patterns)
	}
	if !strings.HasSuffix(opts.PathsBasePath.AsString(), "/src") {
		t.Errorf("pathsBasePath = %q", opts.PathsBasePath)
	}

	again, notes := NormalizeOptions(opts)
	if len(notes) != 0 {
		t.Errorf("second normalization changed %v", notes)
	}
	if again.ModuleResolution != opts.ModuleResolution {
		t.Errorf("second normalization changed moduleResolution")
	}
}

func TestKeepsUserOptions(t *testing.T) {
	opts := &core.CompilerOptions{
		ModuleResolution:          core.ModuleResolutionKindNode10,
		ResolvePackageJsonExports: core.TSTrue,
		Types:                     []string{"a"},
	}
	out, _ := NormalizeOptions(opts)
	if !out.ResolvePackageJsonExports.IsTrue() {
		t.Errorf("an explicit resolvePackageJsonExports must survive")
	}
	if !slices.Equal(out.Types, []string{"a"}) {
		t.Errorf("explicit types must survive, got %v", out.Types)
	}
	if opts.ModuleResolution != core.ModuleResolutionKindNode10 {
		t.Errorf("the input must not change")
	}
}

// The file list must match what TS 5.5.3 loads with moduleResolution Node, including a deep import into
// a package with an exports field and an import through baseUrl.
func TestResolutionMatchesNode(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 1)
	var got []string
	for _, file := range project.Program().SourceFiles() {
		got = append(got, relName(dir, file.FileName()))
	}
	want := readGolden(t, "files.txt")
	compareLines(t, "file list", got, want)
	for _, name := range []string{"node_modules/@rbxts/exportspkg/lib/deep.d.ts", "src/shared/viaBaseUrl.ts", "node_modules/@rbxts/types/index.d.ts"} {
		if !slices.Contains(got, name) {
			t.Errorf("%s not loaded", name)
		}
	}
}

func TestDiagnosticsMatchTypeScript(t *testing.T) {
	for _, test := range []struct{ config, golden string }{
		{"tsconfig.json", "diagnostics.txt"},
		{"tsconfig.errors.json", "errors.txt"},
		{"rootdirs/tsconfig.json", "rootdirs.txt"},
	} {
		for _, checkers := range []int{1, 4, 8} {
			t.Run(fmt.Sprintf("%s/%d", test.config, checkers), func(t *testing.T) {
				dir := copyFixture(t)
				_, project := load(t, dir, test.config, checkers)
				var got []string
				files := project.SourceFilesToCompile()
				for _, diags := range project.PreEmitDiagnosticsForFiles(ctx, files) {
					for _, d := range diags {
						got = append(got, formatDiagnostic(dir, d))
					}
				}
				compareLines(t, "diagnostics", got, readGolden(t, test.golden))
			})
		}
	}
}

func TestDiagnosticTraps(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.errors.json", 2)
	got := map[string][]string{}
	for _, file := range project.SourceFilesToCompile() {
		name := relName(dir, file.FileName())
		for _, d := range project.PreEmitDiagnostics(ctx, file) {
			got[name] = append(got[name], fmt.Sprintf("%d", d.Code()))
		}
	}
	if len(got["errors/reservedObject.ts"]) != 0 {
		t.Errorf("TS2441 on Object must be filtered, got %v", got["errors/reservedObject.ts"])
	}
	if !slices.Equal(got["errors/reservedExports.ts"], []string{"2441"}) {
		t.Errorf("TS2441 on exports must stay, because TS 5.5.3 reports it: %v", got["errors/reservedExports.ts"])
	}
	if !slices.Equal(got["errors/reservedRequire.ts"], []string{"2441"}) {
		t.Errorf("TS2441 on require must stay, because TS 5.5.3 reports it: %v", got["errors/reservedRequire.ts"])
	}
	// One real error, the ignored one suppressed, and an unused @ts-expect-error reported.
	if !slices.Equal(got["errors/plainError.ts"], []string{"2322", "2578"}) {
		t.Errorf("plainError.ts = %v", got["errors/plainError.ts"])
	}
}

func TestNoDiagnosticsFromPackages(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 4)
	for _, diags := range project.PreEmitDiagnosticsForFiles(ctx, project.SourceFilesToCompile()) {
		for _, d := range diags {
			t.Errorf("unexpected diagnostic: %s", formatDiagnostic(dir, d))
		}
	}
	// The package really has an error, so this isn't passing by accident.
	for _, file := range project.Program().SourceFiles() {
		if strings.HasSuffix(file.FileName().AsString(), "brokenpkg/index.d.ts") {
			if len(project.Program().GetSemanticDiagnostics(ctx, file)) == 0 {
				t.Errorf("brokenpkg should have an error when asked for directly")
			}
			return
		}
	}
	t.Errorf("brokenpkg not loaded")
}

func TestPoolGroupsFilesByOwner(t *testing.T) {
	dir := copyFixture(t)
	for _, checkers := range []int{1, 2, 4, 8} {
		_, project := load(t, dir, "tsconfig.json", checkers)
		files := project.SourceFilesToCompile()
		workers := project.NewWorkers(ctx, files, MacroSpec{})
		seen := map[*ast.SourceFile]bool{}
		seenChecker := map[*checker.Checker]bool{}
		for _, w := range workers {
			if seenChecker[w.Checker()] {
				t.Errorf("two workers share a checker")
			}
			seenChecker[w.Checker()] = true
			if len(w.Files()) == 0 {
				t.Errorf("worker with no files")
			}
			for _, file := range w.Files() {
				if seen[file] {
					t.Errorf("%s given to two workers", file.FileName())
				}
				seen[file] = true
				c, done := project.Program().GetTypeCheckerForFile(ctx, file)
				done()
				if c != w.Checker() {
					t.Errorf("%s given to a worker that doesn't own its checker", file.FileName())
				}
			}
		}
		if len(seen) != len(files) {
			t.Errorf("%d of %d files assigned", len(seen), len(files))
		}
		if len(workers) > checkers {
			t.Errorf("%d workers for %d checkers", len(workers), checkers)
		}
	}
}

func TestRunWorkersRunsOneGoroutinePerChecker(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 4)
	workers := project.NewWorkers(ctx, project.SourceFilesToCompile(), MacroSpec{})
	var mu sync.Mutex
	active := map[*checker.Checker]*atomic.Int32{}
	for _, w := range workers {
		active[w.Checker()] = &atomic.Int32{}
	}
	var calls atomic.Int32
	project.RunWorkers(workers, func(w *Worker) {
		calls.Add(1)
		mu.Lock()
		counter := active[w.Checker()]
		mu.Unlock()
		if counter.Add(1) != 1 {
			t.Errorf("two goroutines on one checker")
		}
		for _, file := range w.Files() {
			w.CheckFileAndGetResolver(ctx, file)
		}
		counter.Add(-1)
	})
	if int(calls.Load()) != len(workers) {
		t.Errorf("%d calls for %d workers", calls.Load(), len(workers))
	}
}

func TestDiagnosticsCannotOverlapWorkers(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 2)
	files := project.SourceFilesToCompile()
	workers := project.NewWorkers(ctx, files, MacroSpec{})
	var panicked atomic.Bool
	project.RunWorkers(workers[:1], func(w *Worker) {
		defer func() {
			if recover() != nil {
				panicked.Store(true)
			}
		}()
		project.PreEmitDiagnostics(ctx, files[0])
	})
	if !panicked.Load() {
		t.Errorf("diagnostics during the worker phase must panic")
	}
}

// importAliasNodes yields the same nodes gen-golden.cjs reports.
func importAliasNodes(file *ast.SourceFile, report func(node *ast.Node, label string)) {
	for _, statement := range file.Statements.Nodes {
		if !ast.IsImportDeclaration(statement) || statement.AsImportDeclaration().ImportClause == nil {
			continue
		}
		clauseNode := statement.AsImportDeclaration().ImportClause
		clause := clauseNode.AsImportClause()
		if clause.Name() != nil {
			report(clauseNode, clause.Name().Text())
		}
		if bindings := clause.NamedBindings; bindings != nil {
			if ast.IsNamedImports(bindings) {
				for _, element := range bindings.AsNamedImports().Elements.Nodes {
					report(element, element.Name().Text())
				}
			} else {
				report(bindings, "* as "+bindings.Name().Text())
			}
		}
	}
}

// IsReferencedAliasDeclaration through the bridge must equal what TS 5.5.3's getEmitResolver gives.
func TestReferencedAliasMatchesTypeScript(t *testing.T) {
	for _, checkers := range []int{1, 2, 4, 8} {
		t.Run(fmt.Sprint(checkers), func(t *testing.T) {
			dir := copyFixture(t)
			_, project := load(t, dir, "tsconfig.json", checkers)
			files := project.SourceFilesToCompile()
			workers := project.NewWorkers(ctx, files, MacroSpec{})
			var mu sync.Mutex
			results := map[*ast.SourceFile][]string{}
			project.RunWorkers(workers, func(w *Worker) {
				for _, file := range w.Files() {
					resolver := w.CheckFileAndGetResolver(ctx, file)
					var lines []string
					importAliasNodes(file, func(node *ast.Node, label string) {
						lines = append(lines, fmt.Sprintf("%s\t%s\t%v", relName(dir, file.FileName()), label, resolver.IsReferencedAliasDeclaration(node)))
					})
					mu.Lock()
					results[file] = lines
					mu.Unlock()
				}
			})
			var got []string
			for _, file := range files {
				got = append(got, results[file]...)
			}
			compareLines(t, "aliases", got, readGolden(t, "aliases.txt"))
		})
	}
}

// The trap itself: a resolver made without checking the file answers differently, with no error.
// If this stops failing, CheckFileAndGetResolver's check may be removable, but not before.
func TestUncheckedResolverIsWrong(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 1)
	var file *ast.SourceFile
	for _, f := range project.SourceFilesToCompile() {
		if strings.HasSuffix(f.FileName().AsString(), "aliases.ts") {
			file = f
		}
	}
	workers := project.NewWorkers(ctx, []*ast.SourceFile{file}, MacroSpec{})
	w := workers[0]
	unchecked := map[string]bool{}
	raw := newUncheckedResolverForTest(w)
	importAliasNodes(file, func(node *ast.Node, label string) {
		unchecked[label] = raw.IsReferencedAliasDeclaration(node)
	})
	resolver := w.CheckFileAndGetResolver(ctx, file)
	differ := 0
	importAliasNodes(file, func(node *ast.Node, label string) {
		if resolver.IsReferencedAliasDeclaration(node) != unchecked[label] {
			differ++
		}
	})
	if differ == 0 {
		t.Errorf("expected an unchecked resolver to give different answers, so the trap in 0002 may be gone")
	}
}

func TestResolverRejectsFileFromAnotherChecker(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 4)
	workers := project.NewWorkers(ctx, project.SourceFilesToCompile(), MacroSpec{})
	if len(workers) < 2 {
		t.Skip("the partitioner put every file on one checker")
	}
	defer func() {
		if recover() == nil {
			t.Errorf("a file owned by another checker must panic")
		}
	}()
	workers[0].CheckFileAndGetResolver(ctx, workers[1].Files()[0])
}

// Under the pool's exclusive lock, a resolver method deadlocks. The bridge must panic instead, promptly.
func TestResolverPanicsUnderExclusiveLock(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 2)
	files := project.SourceFilesToCompile()
	workers := project.NewWorkers(ctx, files, MacroSpec{})
	w := workers[0]
	result := make(chan any, 1)
	go func() {
		_, done := project.Program().GetTypeCheckerForFileExclusive(ctx, w.Files()[0])
		defer done()
		defer func() { result <- recover() }()
		w.CheckFileAndGetResolver(ctx, w.Files()[0])
	}()
	select {
	case r := <-result:
		if r == nil {
			t.Errorf("must panic under the exclusive lock")
		} else if !strings.Contains(fmt.Sprint(r), "mutex is held") {
			t.Errorf("wrong panic: %v", r)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("deadlocked under the exclusive lock")
	}
	// After the lock is released, the same call works.
	if w.CheckFileAndGetResolver(ctx, w.Files()[0]) == nil {
		t.Errorf("no resolver")
	}
}

func fixtureMacroSpec() MacroSpec {
	return MacroSpec{
		Calls:        []string{"assert"},
		Constructors: []string{"MapConstructor", "SetConstructor"},
		PropertyCalls: []PropertyMacroClass{
			{Class: "Array", Methods: []string{"push"}},
			{Class: "ReadonlyArray", Methods: []string{"size"}},
		},
	}
}

// fixtureHasAllSymbols adds what the small fixture lib doesn't declare, so NewMacroSymbols can run on it.
func addMissingGlobals(t *testing.T, dir string) {
	t.Helper()
	text := `
interface ArrayConstructor { new <T>(): Array<T>; }
interface WeakSetConstructor { new <T>(): WeakSet<T>; }
interface WeakMapConstructor { new <K, V>(): WeakMap<K, V>; }
interface ReadonlyMapConstructor { new <K, V>(): ReadonlyMap<K, V>; }
interface ReadonlySetConstructor { new <T>(): ReadonlySet<T>; }
interface Generator<T, R, N> {}
interface IterableFunction<T> extends Iterable<T> {}
type LuaTuple<T> = T & { readonly _nominal_LuaTuple: unique symbol };
interface ReadVoxelsArray {}
interface WeakMap<K, V> {}
interface WeakSet<T> {}
declare function $range(a: number, b: number): Iterable<number>;
declare function $tuple<T>(...values: Array<T>): LuaTuple<Array<T>>;
`
	path := filepath.Join(dir, "node_modules", "@rbxts", "types", "index.d.ts")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, text...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The real trap from 0002: Array is cloned per checker once a file augments it, so a shared table is wrong.
func TestMacroSymbolsAreNotSharedAcrossCheckers(t *testing.T) {
	dir := copyFixture(t)
	addMissingGlobals(t, dir)
	_, project := load(t, dir, "tsconfig.json", 4)
	files := project.Program().SourceFiles()
	checkers := map[*checker.Checker]bool{}
	var tables []*MacroSymbols
	for _, file := range files {
		c, done := project.Program().GetTypeCheckerForFile(ctx, file)
		done()
		if !checkers[c] {
			checkers[c] = true
			table, err := NewMacroSymbols(c, fixtureMacroSpec())
			if err != nil {
				t.Fatal(err)
			}
			if table.Checker() != c {
				t.Errorf("table doesn't remember its checker")
			}
			tables = append(tables, table)
		}
	}
	if len(tables) < 2 {
		t.Skip("needs two checkers")
	}
	if tables[0].Symbol("Array") == tables[1].Symbol("Array") {
		t.Errorf("Array should be cloned per checker here, because augment.ts adds to it")
	}
	for i, table := range tables {
		array := table.Symbol("Array")
		if !table.IsMacroOnlyClass(array) {
			t.Errorf("table %d: Array should be macro-only on its own checker", i)
		}
		for j, other := range tables {
			if i != j && other.IsMacroOnlyClass(array) {
				t.Errorf("table %d's Array matched table %d", i, j)
			}
		}
	}
}

func TestWorkerMacrosArePerWorker(t *testing.T) {
	dir := copyFixture(t)
	addMissingGlobals(t, dir)
	_, project := load(t, dir, "tsconfig.json", 4)
	workers := project.NewWorkers(ctx, project.SourceFilesToCompile(), fixtureMacroSpec())
	var mu sync.Mutex
	seen := map[*MacroSymbols]bool{}
	project.RunWorkers(workers, func(w *Worker) {
		table, err := w.Macros()
		if err != nil {
			t.Error(err)
			return
		}
		again, _ := w.Macros()
		if table != again {
			t.Errorf("a worker's table must be stable")
		}
		if table.Checker() != w.Checker() {
			t.Errorf("table from the wrong checker")
		}
		for _, file := range w.Files() {
			w.CheckFileAndGetResolver(ctx, file)
		}
		mu.Lock()
		defer mu.Unlock()
		if seen[table] {
			t.Errorf("two workers share a table")
		}
		seen[table] = true
	})
}

func TestMacroSymbolErrorsUseUpstreamText(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 1)
	c, done := project.Program().GetTypeChecker(ctx)
	defer done()
	for _, test := range []struct {
		spec MacroSpec
		want string
	}{
		{MacroSpec{Calls: []string{"noSuchMacro"}}, "MacroManager could not find symbol for noSuchMacro\nYou may need to update your @rbxts/compiler-types!"},
		{MacroSpec{PropertyCalls: []PropertyMacroClass{{Class: "Array", Methods: []string{"nope"}}}}, "MacroManager could not find method for Array.nope\nYou may need to update your @rbxts/compiler-types!"},
		{MacroSpec{Constructors: []string{"Number"}}, "MacroManager could not find constructor for Number\nYou may need to update your @rbxts/compiler-types!"},
	} {
		if _, err := NewMacroSymbols(c, test.spec); err == nil || err.Error() != test.want {
			t.Errorf("got %v, want %q", err, test.want)
		}
	}
}

// spikeFixture returns the public fixture with real @rbxts packages, or skips if npm ci hasn't been run there.
func spikeFixture(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "spike", "fixture"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "@rbxts", "compiler-types")); err != nil {
		t.Skip("run npm ci in spike/fixture to enable this test")
	}
	return dir
}

// With the real compiler-types, every macro name of roblox-ts 3.0.0 must resolve, on each checker.
func TestRealCompilerTypesResolveEveryMacro(t *testing.T) {
	dir := spikeFixture(t)
	_, project := load(t, dir, "tsconfig.json", 4)
	workers := project.NewWorkers(ctx, project.SourceFilesToCompile(), DefaultMacroSpec())
	project.RunWorkers(workers, func(w *Worker) {
		table, err := w.Macros()
		if err != nil {
			t.Error(err)
			return
		}
		for _, name := range append(slices.Clone(SymbolNames), NominalLuaTupleName) {
			if table.Symbol(name) == nil {
				t.Errorf("no symbol for %s", name)
			}
		}
		for _, file := range w.Files() {
			w.CheckFileAndGetResolver(ctx, file)
		}
	})
	for _, diags := range project.PreEmitDiagnosticsForFiles(ctx, project.SourceFilesToCompile()) {
		for _, d := range diags {
			t.Errorf("unexpected diagnostic on the fixture: %s", formatDiagnostic(dir, d))
		}
	}
}

// The port must reach the traps through this package only.
func TestNoTrapCallsOutsideBridge(t *testing.T) {
	forbidden := regexp.MustCompile(`\.(GetTypeCheckerForFileExclusive|ForEachCheckerParallel|NewEmitResolver)\(`)
	root := ".."
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".scratch", "bin", "node_modules", "third_party", "spike", "tsbridge", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if match := forbidden.Find(data); match != nil {
			t.Errorf("%s calls %s, which only tsbridge may do", path, match)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// rbxtsc checks a file right before transforming it. The port checks everything first, then transforms.
// Both orders must give the same alias answers.
func TestReferencedAliasAfterDiagnosticsPhase(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "tsconfig.json", 4)
	files := project.SourceFilesToCompile()
	project.PreEmitDiagnosticsForFiles(ctx, files)
	workers := project.NewWorkers(ctx, files, MacroSpec{})
	var mu sync.Mutex
	results := map[*ast.SourceFile][]string{}
	project.RunWorkers(workers, func(w *Worker) {
		for _, file := range w.Files() {
			resolver := w.CheckFileAndGetResolver(ctx, file)
			var lines []string
			importAliasNodes(file, func(node *ast.Node, label string) {
				lines = append(lines, fmt.Sprintf("%s\t%s\t%v", relName(dir, file.FileName()), label, resolver.IsReferencedAliasDeclaration(node)))
			})
			mu.Lock()
			results[file] = lines
			mu.Unlock()
		}
	})
	var got []string
	for _, file := range files {
		got = append(got, results[file]...)
	}
	compareLines(t, "aliases", got, readGolden(t, "aliases.txt"))
}

// TS 7 defaults rootDir to the tsconfig's folder and TS 5.5.3 doesn't, which would move every output path of a
// project that only sets rootDirs.
func TestCommonSourceDirectoryMatchesTypeScript(t *testing.T) {
	dir := copyFixture(t)
	_, project := load(t, dir, "rootdirs/tsconfig.json", 1)
	rel := func(path string) string {
		base := tspath.NormalizePath(filepath.ToSlash(dir))
		return strings.TrimPrefix(strings.TrimPrefix(strings.TrimSuffix(path, "/"), base), "/")
	}
	want := ""
	if lines := readGolden(t, "rootdirs-common.txt"); lines != nil {
		want = lines[0]
	}
	if got := rel(project.CommonSourceDirectory().AsString()); got != want {
		t.Errorf("common source directory = %q, TS 5.5.3 gives %q", got, want)
	}
	if rel(project.Program().CommonSourceDirectory().AsString()) == want {
		t.Errorf("Program.CommonSourceDirectory now agrees with TS 5.5.3, so this workaround may be removable")
	}
}
