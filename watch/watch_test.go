package watch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
	"github.com/microsoft/TypeScript/tsc/internal/execute/watchmanager"
	"github.com/microsoft/TypeScript/tsc/internal/fswatch"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/compile"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/plugins"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

// All of this works in .scratch/watchtest. roblox-ts wants node_modules next to the tsconfig, so each project folder gets
// a copy of the packages, from one kept there.
func testRoot(t testing.TB) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	repo := filepath.Dir(filepath.Dir(file))
	root := filepath.Join(repo, ".scratch", "watchtest")
	modules := filepath.Join(root, "node_modules", "@rbxts")
	if _, err := os.Stat(filepath.Join(modules, "types")); err != nil {
		source := filepath.Join(repo, ".scratch", "diff", "tests", "node_modules", "@rbxts")
		if _, err := os.Stat(filepath.Join(source, "types")); err != nil {
			t.Skip("no packages: run scripts/setup-diff-corpus.sh")
		}
		if err := copyTree(source, modules); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func copyTree(src, dest string) error {
	return filepath.WalkDir(src, func(item string, entry fs.DirEntry, err error) error {
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
}

// fakeBackend lets a test decide exactly when events happen, so a sequence of edits is deterministic.
type fakeBackend struct {
	mu       sync.Mutex
	requests []watchmanager.WatchDirectoryRequest
}

type fakeCloser struct{}

func (fakeCloser) Close() error { return nil }

func (b *fakeBackend) WatchDirectories(requests []watchmanager.WatchDirectoryRequest) ([]io.Closer, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.requests = append(b.requests, requests...)
	closers := make([]io.Closer, len(requests))
	for i := range requests {
		closers[i] = fakeCloser{}
	}
	return closers, nil
}

// fire reports paths the way ReadDirectoryChangesW does: the item, and an update on the folder it's in.
func (b *fakeBackend) fire(paths ...string) {
	b.mu.Lock()
	requests := slices.Clone(b.requests)
	b.mu.Unlock()
	var events []watchmanager.WatchEvent
	for _, file := range paths {
		slashed := filepath.ToSlash(file)
		kind := fswatch.EventUpdate
		if _, err := os.Stat(file); err != nil {
			kind = fswatch.EventDelete
		}
		events = append(events,
			watchmanager.WatchEvent{Path: tspath.RootedPathFromAbsolute(slashed), Kind: kind},
			watchmanager.WatchEvent{Path: tspath.RootedPathFromAbsolute(filepath.ToSlash(filepath.Dir(file))), Kind: fswatch.EventUpdate})
	}
	if len(requests) > 0 {
		requests[0].Callback(events, nil)
	}
}

// slots are the things a sequence of edits can change. Each variant of each slot changes what some file outputs,
// sometimes a file that doesn't import the one edited.
var slots = map[string]int{
	"api":        2, // method or function property: `a:run()` or `a.run()` in everything using it
	"kind":       3, // const enum values, inlined in other files
	"items":      2, // Array or Set: `.size()` lowers differently
	"msg":        3, // only this file's output
	"base":       2, // method or property on a base class in another file
	"global":     2, // declared in a .d.ts nothing imports
	"augment":    2, // `declare global` in a module nothing imports
	"modaugment": 2, // `declare module` in a module nothing imports
	"json":       2, // a JSON file's type: `+` or `..`
	"extra":      2, // a new file, which is absent at first
	"extraUser":  2, // a file that imports it. Without the extra file, an error
	"bad":        3, // fine, a type error, a roblox-ts error
	"edge":       2, // adds an import to an existing file, so the program has to be rebuilt
	"asset":      3, // a file that only gets copied: absent, one text, another text
	"lua":        2, // a Luau file next to the sources
	"newdir":     2, // a folder with a module and an asset
	"rojo":       2, // renames a Rojo mapping, which changes generated paths
	"strict":     2, // tsconfig: another option on or off
	"cycle":      2, // two files importing each other
	"infer":      2, // a type that passes through inference across two files
}

// setgetSlots only exist with setget on, because roblox-ts rejects accessors without it.
var setgetSlots = map[string]int{
	"accessor":   2, // an accessor or a plain field: in another file, `a.value` or `a.__get_value()`
	"accname":    2, // adds an accessor whose name is used by a property elsewhere, which changes the name filter
	"accglobal":  2, // an accessor or a field in a `declare global` nothing imports
	"accchain":   2, // an accessor that reaches the user through a re-export and inference
	"whitespace": 2, // a change in spacing, so the printed text stays the same and the source positions move
	"accerror":   2, // a type error below an accessor, so its position has to map back to the source
	"accfatal":   2, // an edit that makes the plugin end the process in JS
}

// setgetOn says this state has setget on.
const setgetOn = "@setget"

func setgetState() map[string]int {
	state := baseState()
	state[setgetOn] = 1
	for name := range setgetSlots {
		state[name] = 0
	}
	return state
}

// declarationsOn says this state has declaration emit on.
const declarationsOn = "@declaration"

func declarationOption(state map[string]int) string {
	if state[declarationsOn] == 1 {
		return "\n\t\t\"declaration\": true,"
	}
	return ""
}

func optionsFor(state map[string]int) plugins.Options {
	on := state[setgetOn] == 1
	return plugins.Options{Setget: &on}
}

func slotsOf(state map[string]int) map[string]int {
	result := maps.Clone(slots)
	if state[setgetOn] == 1 {
		maps.Copy(result, setgetSlots)
	}
	return result
}

func baseState() map[string]int {
	state := map[string]int{}
	for name := range slots {
		state[name] = 0
	}
	return state
}

// render gives the whole project for a state: relative path to contents.
func render(state map[string]int) map[string]string {
	files := map[string]string{}
	pick := func(slot string, variants ...string) string { return variants[state[slot]] }

	strict := pick("strict", "true", "false") // noImplicitReturns
	files["tsconfig.json"] = `{
	"compilerOptions": {
		"allowSyntheticDefaultImports": true,
		"downlevelIteration": true,
		"jsx": "react",
		"jsxFactory": "Roact.jsx",
		"module": "commonjs",
		"moduleResolution": "Node",
		"noLib": true,
		"resolveJsonModule": true,
		"forceConsistentCasingInFileNames": true,
		"moduleDetection": "force",
		"strict": true,
		"noImplicitReturns": ` + strict + `,` + declarationOption(state) + `
		"target": "ESNext",
		"typeRoots": ["node_modules/@rbxts"],
		"experimentalDecorators": true,
		"rootDir": "src",
		"outDir": "out",
		"baseUrl": "src"
	},
	"include": ["src/**/*"]
}
`
	files["default.project.json"] = `{
	"name": "watchtest",
	"tree": {
		"$className": "DataModel",
		"ReplicatedStorage": {
			"$className": "ReplicatedStorage",
			"rbxts_include": {
				"$path": "include",
				"node_modules": { "$className": "Folder", "@rbxts": { "$path": "node_modules/@rbxts" } }
			},
			"` + pick("rojo", "TS", "Renamed") + `": { "$path": "out/shared" }
		},
		"ServerScriptService": {
			"$className": "ServerScriptService",
			"TS": { "$path": "out/server" }
		}
	}
}
`
	files["src/shared/types.ts"] = `export interface Api {
	` + pick("api", "run(x: number): number;", "run: (x: number) => number;") + `
}
export const enum Kind {
	A = ` + pick("kind", "1", "10", "100") + `,
	B,
}
export type Items = ` + pick("items", "Array<number>", "Set<number>") + `;
`
	files["src/shared/chain.ts"] = `export * from "./types";
export { Kind as KindAlias } from "./types";
`
	files["src/shared/values.ts"] = `import { Api, Items, Kind, KindAlias } from "./chain";
export const api: Api = {
	run(x) {
		return x + Kind.B;
	},
};
export const alias = KindAlias.A;
export const items = undefined as unknown as Items;
`
	files["src/shared/user.ts"] = `import { api, items } from "./values";
export const user = api.run(items.size());
`
	files["src/server/main.server.ts"] = `import { user } from "shared/user";
print(user, "` + pick("msg", "first", "second", "third") + `");
`
	files["src/shared/base.ts"] = `export class Base {
	` + pick("base", "public go() { return 1; }", "public go = () => 1;") + `
}
export interface Extendable {}
`
	files["src/server/derived.ts"] = `import { Base } from "shared/base";
export class Derived extends Base {
	public twice() {
		return this.go() + this.go();
	}
}
`
	files["src/globals.d.ts"] = `interface GlobalThing {
	` + pick("global", "go(x: number): void;", "go: (x: number) => void;") + `
}
`
	files["src/server/usesglobal.ts"] = `declare const thing: GlobalThing;
export function viaGlobal() {
	thing.go(1);
}
`
	files["src/shared/augment.ts"] = `declare global {
	interface Augmented {
		` + pick("augment", "go(x: number): void;", "go: (x: number) => void;") + `
	}
}
export {};
`
	files["src/server/usesaugment.ts"] = `declare const thing: Augmented;
export function viaAugment() {
	thing.go(1);
}
`
	files["src/shared/modaugment.ts"] = `declare module "./base" {
	interface Extendable {
		` + pick("modaugment", "go(x: number): void;", "go: (x: number) => void;") + `
	}
}
export {};
`
	files["src/server/usesmodaugment.ts"] = `import { Extendable } from "shared/base";
declare const thing: Extendable;
export function viaModuleAugment() {
	thing.go(1);
}
`
	files["src/server/importtype.ts"] = `declare const thing: import("shared/types").Items;
export const size = thing.size();
`
	files["src/shared/data.json"] = pick("json", `{ "v": 1 }`, `{ "v": "text" }`) + "\n"
	files["src/server/usesjson.ts"] = `import data from "shared/data.json";
export const doubled = data.v + data.v;
`
	if state["extra"] == 1 {
		files["src/shared/extra.ts"] = "export const extra = 1;\n"
	}
	if state["extraUser"] == 1 {
		files["src/server/extrauser.ts"] = `import { extra } from "shared/extra";
print(extra);
`
	}
	files["src/shared/bad.ts"] = pick("bad", "export const bad = 1;\n", "export const bad: string = 1;\n",
		"const value: any = 1;\nexport const bad = value.field;\n")
	files["src/server/edge.ts"] = pick("edge", "export const edge = 1;\n", "import { user } from \"shared/user\";\nexport const edge = user;\n")
	switch state["asset"] {
	case 1:
		files["src/shared/asset.txt"] = "one\n"
	case 2:
		files["src/shared/asset.txt"] = "two\n"
	}
	if state["lua"] == 1 {
		files["src/shared/native.lua"] = "return 1\n"
	}
	if state["newdir"] == 1 {
		files["src/shared/fresh/inside.ts"] = "export const inside = 2;\n"
		files["src/shared/fresh/deep/asset.txt"] = "deep\n"
	}
	files["src/shared/cyc1.ts"] = `import { two } from "./cyc2";
export function one() {
	return two;
}
export const count = ` + pick("cycle", "1", `"text"`) + `;
`
	files["src/shared/cyc2.ts"] = `import { count, one } from "./cyc1";
export function two() {
	return one;
}
export const doubledCount = count + count;
`
	files["src/shared/infer1.ts"] = `export const value = { go: ` + pick("infer", "1", `"text"`) + ` };
`
	if state[setgetOn] == 1 {
		files["src/shared/acc.ts"] = `export class Acc {
	private stored = 1;
	` + pick("accessor", "public get value() { return this.stored; }\n	public set value(other: number) { this.stored = other; }", "public value = 1;") + `
	public read() { return this.stored; }
}
`
		files["src/server/useacc.ts"] = `import { Acc } from "shared/acc";
const a = new Acc();
a.value = a.value + 1;
a.value += 2;
print(a.value, a.read());
`
		files["src/shared/accname.ts"] = `export class Named {
	` + pick("accname", "public plain = 1;", "public get foo() { return 1; }") + `
}
`
		files["src/server/plainuse.ts"] = `declare const thing: { foo: number };
print(thing.foo);
`
		files["src/shared/accglobal.ts"] = `declare global {
	interface AccGlobal {
		` + pick("accglobal", "get reach(): number;", "reach: number;") + `
	}
}
export {};
`
		files["src/server/useaccglobal.ts"] = `declare const thing: AccGlobal;
print(thing.reach);
`
		files["src/shared/accchain1.ts"] = `class Chain {
	` + pick("accchain", "public get far() { return 1; }", "public far = 1;") + `
}
export const chained = new Chain();
`
		files["src/shared/accchain2.ts"] = `export * from "./accchain1";
`
		files["src/server/accchain3.ts"] = `import { chained } from "shared/accchain2";
print(chained.far);
`
		files["src/shared/spacing.ts"] = pick("whitespace", "export const spaced = 1;\n", "\n\nexport   const   spaced   =   1;\n")
		files["src/shared/accerror.ts"] = `export class Later {
	public get first() {
		return 1;
	}
}
export const missing: ` + pick("accerror", "number", "string") + ` = 1;
`
		files["src/server/accfatal.ts"] = `class Fatal {
	public set onlySet(other: number) {
		print(other);
	}
	public get both() { return 1; }
	public set both(other: number) { print(other); }
}
const fatal = new Fatal();
` + pick("accfatal", "fatal.both += 1;", "fatal.onlySet += 1;") + "\n"
	}
	files["src/shared/infer2.ts"] = `import { value } from "./infer1";
export const passed = value;
`
	files["src/server/infer3.ts"] = `import { passed } from "shared/infer2";
export const result = passed.go + passed.go;
`
	return files
}

type harness struct {
	t       testing.TB
	dir     string
	live    string
	clean   string
	state   map[string]int
	backend *fakeBackend
	cycles  chan *Cycle
	output  *bytes.Buffer
	stop    func()
	done    chan error
	// incremental says the project on disk has incremental builds on.
	incremental bool
}

func (h *harness) rendered(state map[string]int) map[string]string {
	if h.incremental {
		return renderIncremental(state)
	}
	return render(state)
}

func loadFor(live string) func() (*shared.ProjectData, error) {
	return func() (*shared.ProjectData, error) {
		return project.CreateProjectData(filepath.Join(live, "tsconfig.json"), shared.DefaultProjectOptions())
	}
}

type syncBuffer struct {
	mu sync.Mutex
	bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(p)
}

func newDirs(t testing.TB, name string) (dir, live, clean string) {
	t.Helper()
	root := testRoot(t)
	dir = filepath.Join(root, fmt.Sprintf("%s-%d", name, os.Getpid()))
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	live = filepath.Join(dir, "live")
	clean = filepath.Join(dir, "clean")
	for _, folder := range []string{live, clean} {
		if err := copyTree(filepath.Join(root, "node_modules"), filepath.Join(folder, "node_modules")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "package.json"), []byte(`{ "name": "watchtest", "version": "0.0.0", "private": true }`), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir, live, clean
}

func startHarness(t testing.TB, name string, state map[string]int, configure func(*Config)) *harness {
	t.Helper()
	dir, live, clean := newDirs(t, name)
	h := &harness{t: t, dir: dir, live: live, clean: clean, state: maps.Clone(state), backend: &fakeBackend{}, cycles: make(chan *Cycle, 64)}
	writeState(t, live, render(state), nil)
	return h.start(state, configure)
}

// startExisting starts a watcher on a project already on disk, with incremental builds on.
func startExisting(t testing.TB, live string, state map[string]int) *harness {
	t.Helper()
	h := &harness{t: t, dir: filepath.Dir(live), live: live, clean: filepath.Join(filepath.Dir(live), "clean"), state: maps.Clone(state), backend: &fakeBackend{}, cycles: make(chan *Cycle, 64), incremental: true}
	return h.start(state, nil)
}

func (h *harness) start(state map[string]int, configure func(*Config)) *harness {
	t := h.t
	live := h.live

	ctx, cancel := context.WithCancel(context.Background())
	buffer := &syncBuffer{}
	h.output = &buffer.Buffer
	h.done = make(chan error, 1)
	config := Config{
		Load:          loadFor(live),
		Backend:       h.backend,
		PluginOptions: optionsFor(state),
		Output:        buffer,
		OnCycle: func(c *Cycle) {
			copied := *c
			copied.Result = nil
			copied.Report = formatForCompare(c.Result)
			h.cycles <- &copied
		},
	}
	if configure != nil {
		configure(&config)
	}
	go func() { h.done <- Run(ctx, config) }()
	h.stop = sync.OnceFunc(func() {
		cancel()
		if err := <-h.done; err != nil {
			t.Errorf("Run returned %v", err)
		}
	})
	t.Cleanup(h.stop)
	return h
}

// writeState makes the folder match files, and returns what it touched.
func writeState(t testing.TB, root string, files, previous map[string]string) []string {
	t.Helper()
	var touched []string
	for rel, text := range files {
		if old, ok := previous[rel]; ok && old == text {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o666); err != nil {
			t.Fatal(err)
		}
		touched = append(touched, full)
	}
	for rel := range previous {
		if _, ok := files[rel]; !ok {
			full := filepath.Join(root, filepath.FromSlash(rel))
			if err := os.Remove(full); err != nil {
				t.Fatal(err)
			}
			touched = append(touched, full)
			// Git doesn't track empty folders, and neither does a clean build need them, but out/ would keep a
			// matching empty folder. Remove what's left empty, as deleting a folder in an editor does.
			for dir := filepath.Dir(full); dir != root; dir = filepath.Dir(dir) {
				if entries, err := os.ReadDir(dir); err != nil || len(entries) > 0 || filepath.Base(dir) == "src" {
					break
				}
				if err := os.Remove(dir); err != nil {
					t.Fatal(err)
				}
				touched = append(touched, dir)
			}
		}
	}
	slices.Sort(touched)
	return touched
}

// apply moves the live project to state and waits for the compile.
func (h *harness) apply(state map[string]int) *Cycle {
	h.t.Helper()
	touched := writeState(h.t, h.live, h.rendered(state), h.rendered(h.state))
	h.state = maps.Clone(state)
	if len(touched) == 0 {
		return nil
	}
	h.backend.fire(touched...)
	return h.wait()
}

func (h *harness) wait() *Cycle {
	h.t.Helper()
	select {
	case cycle := <-h.cycles:
		return cycle
	case <-time.After(3 * time.Minute):
		h.t.Fatal("no compile finished")
		return nil
	}
}

var pathInMessage = regexp.MustCompile(`[^\s"'` + "`" + `]*?[/\\](live|clean)[/\\]`)

func formatForCompare(result *compile.EmitResult) string {
	text := shared.FormatDiagnostics(result.Diagnostics, tspath.RootedDirectoryPathFromAbsolute("C:/"), osvfs.FS().CaseSensitivity(), "\n")
	return pathInMessage.ReplaceAllString(text, "<PROJECT>/")
}

// cleanBuild is what `rbxtsc` does with no state: the handler of the build command, on a fresh copy of the sources.
func cleanBuild(t testing.TB, live, clean string, state map[string]int) (string, int) {
	t.Helper()
	for _, folder := range []string{"src", "out", "include"} {
		if err := os.RemoveAll(filepath.Join(clean, folder)); err != nil {
			t.Fatal(err)
		}
	}
	writeState(t, clean, render(state), nil)
	data, err := project.CreateProjectData(filepath.Join(clean, "tsconfig.json"), shared.DefaultProjectOptions())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := project.GetParsedCommandLine(data)
	if err != nil {
		t.Fatal(err)
	}
	p := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{})
	translator := project.CreatePathTranslator(p, data)
	if err := project.Cleanup(translator); err != nil {
		t.Fatal(err)
	}
	if err := project.CopyInclude(data); err != nil {
		t.Fatal(err)
	}
	if err := project.CopyFiles(data, translator, project.GetRootDirs(p.Options())); err != nil {
		t.Fatal(err)
	}
	prepared, failure, err := project.PrepareCompile(p, data, translator, p.Program().CaseSensitivity())
	if err != nil {
		t.Fatal(err)
	}
	if failure != nil {
		t.Fatalf("clean build could not start: %s", formatForCompare(&compile.EmitResult{Diagnostics: failure}))
	}
	applied, err := plugins.Apply(context.Background(), p, plugins.Decide(data.TsConfigPath, optionsFor(state)), p.SourceFilesToCompile())
	if err != nil {
		t.Fatal(err)
	}
	if applied.Exit != nil {
		return formatForCompare(&compile.EmitResult{Diagnostics: []*shared.Diagnostic{shared.TextDiagnostic(strings.Join(applied.Exit, "\n"), diagnostics.CategoryError)}}), 1
	}
	result, err := compile.CompileFiles(context.Background(), &compile.Context{
		Project: applied.Project, Data: data, Translator: translator, Prepared: prepared, Replay: shared.NewContextReplay(),
	}, applied.Files)
	if err != nil {
		t.Fatal(err)
	}
	result.Diagnostics = applied.MapDiagnostics(result.Diagnostics)
	errorCount := 0
	for _, d := range result.Diagnostics {
		if shared.HasErrors([]*shared.Diagnostic{d}) {
			errorCount++
		}
	}
	return formatForCompare(result), errorCount
}

func readTree(t testing.TB, roots ...string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	for _, root := range roots {
		base := filepath.Base(root)
		err := filepath.WalkDir(root, func(item string, entry fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			rel, _ := filepath.Rel(root, item)
			if entry.IsDir() {
				tree[base+"/"+filepath.ToSlash(rel)+"/"] = ""
				return nil
			}
			data, err := os.ReadFile(item)
			if err != nil {
				return err
			}
			tree[base+"/"+filepath.ToSlash(rel)] = string(data)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return tree
}

// sameFiles compares two trees. Empty folders don't count: fs-extra's copySync leaves one for a source folder with no
// file to copy, and a clean build and a watcher can get there differently only by the order of events.
func compareTrees(want, got map[string]string) string {
	var problems []string
	for name, text := range want {
		if strings.HasSuffix(name, "/") {
			continue
		}
		have, ok := got[name]
		switch {
		case !ok:
			problems = append(problems, "missing "+name)
		case have != text:
			problems = append(problems, fmt.Sprintf("differs %s\n--- clean\n%s\n--- watch\n%s", name, text, have))
		}
	}
	for name := range got {
		if strings.HasSuffix(name, "/") {
			continue
		}
		if _, ok := want[name]; !ok {
			problems = append(problems, "extra "+name)
		}
	}
	slices.Sort(problems)
	return strings.Join(problems, "\n")
}

// check compares the watcher's output with a clean build of the same sources.
func (h *harness) check(cycle *Cycle, before map[string]string, step string) {
	h.t.Helper()
	report, cleanErrors := cleanBuild(h.t, h.live, h.clean, h.state)
	if cycle.Report != report {
		h.t.Fatalf("%s: diagnostics differ\n--- clean\n%s\n--- watch\n%s", step, report, cycle.Report)
	}
	if cycle.Errors != cleanErrors {
		h.t.Fatalf("%s: %d errors, a clean build has %d", step, cycle.Errors, cleanErrors)
	}
	got := readTree(h.t, filepath.Join(h.live, "out"), filepath.Join(h.live, "include"))
	if cleanErrors > 0 {
		// Nothing may be half updated. The first path copies files before it compiles, in both.
		if !cycle.Initial {
			if problem := compareTrees(before, got); problem != "" {
				h.t.Fatalf("%s: a failed compile changed the output:\n%s", step, problem)
			}
		}
		return
	}
	want := readTree(h.t, filepath.Join(h.clean, "out"), filepath.Join(h.clean, "include"))
	if problem := compareTrees(want, got); problem != "" {
		h.t.Fatalf("%s: output differs from a clean build:\n%s", step, problem)
	}
}

func describeChange(old, state map[string]int) string {
	var changes []string
	for name, value := range state {
		if old[name] != value {
			changes = append(changes, fmt.Sprintf("%s=%d", name, value))
		}
	}
	slices.Sort(changes)
	return strings.Join(changes, ",")
}

func randomStep(rng *rand.Rand, state map[string]int) map[string]int {
	next := maps.Clone(state)
	available := slotsOf(state)
	names := slices.Sorted(maps.Keys(available))
	// Mostly one change at a time, sometimes several saved together.
	for range 1 + rng.IntN(3)/2*rng.IntN(3) {
		name := names[rng.IntN(len(names))]
		next[name] = (next[name] + 1 + rng.IntN(available[name]-1+1)) % available[name]
	}
	return next
}

// TestEditSequence is the check on everything else: after every edit in a randomized sequence, the output must match a
// clean build of the same sources byte for byte. The seed is printed so a failure can be repeated.
func TestEditSequence(t *testing.T) {
	seeds := []uint64{1, 2, 3, 4, 5, 6}
	steps := 50
	if testing.Short() {
		seeds = seeds[:1]
		steps = 15
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
				rng := rand.New(rand.NewPCG(seed, 0))
				h := startHarness(t, "seq"+name, start, nil)
				first := h.wait()
				h.check(first, nil, "first build")
				if first.Errors != 0 {
					t.Fatalf("the first build has errors:\n%s", first.Report)
				}
				state := start
				for step := range steps {
					next := randomStep(rng, state)
					if describeChange(state, next) == "" {
						continue
					}
					before := readTree(t, filepath.Join(h.live, "out"), filepath.Join(h.live, "include"))
					label := fmt.Sprintf("seed %d step %d (%s)", seed, step, describeChange(state, next))
					t.Log(label)
					if cycle := h.apply(next); cycle != nil {
						h.check(cycle, before, label)
					}
					state = next
				}
			})
		}
	}
}

// TestEachSlot walks every variant of every slot from the base state and back, so that no slot depends on a random
// sequence reaching it.
func TestEachSlot(t *testing.T) {
	t.Run("plain", func(t *testing.T) { testEachSlot(t, "slots", baseState(), nil) })
	t.Run("setget", func(t *testing.T) { testEachSlot(t, "slotsSetget", setgetState(), nil) })
	full := func(config *Config) { config.FullRebuilds = true }
	t.Run("full", func(t *testing.T) { testEachSlot(t, "slotsFull", setgetState(), full) })
}

func testEachSlot(t *testing.T, label string, state map[string]int, configure func(*Config)) {
	h := startHarness(t, label, state, configure)
	first := h.wait()
	h.check(first, nil, "first build")
	if first.Errors != 0 {
		t.Fatalf("the first build has errors: %s", first.Report)
	}
	available := slotsOf(state)
	for _, name := range slices.Sorted(maps.Keys(available)) {
		for variant := range available[name] {
			next := maps.Clone(state)
			next[name] = (variant + 1) % available[name]
			before := readTree(t, filepath.Join(h.live, "out"), filepath.Join(h.live, "include"))
			label := fmt.Sprintf("%s=%d", name, next[name])
			t.Log(label)
			if cycle := h.apply(next); cycle != nil {
				h.check(cycle, before, label)
			}
			state = next
		}
	}
}

// TestLessWorkThanAFullRebuild checks that the shortcuts happen. An edit nothing imports only reaches its own file.
func TestLessWorkThanAFullRebuild(t *testing.T) {
	h := startHarness(t, "narrow", setgetState(), nil)
	first := h.wait()
	if first.Times.FilesLowered < 20 {
		t.Fatalf("the first build lowered %d files, errors %d: %s", first.Times.FilesLowered, first.Errors, first.Report)
	}
	state := maps.Clone(h.state)
	state["msg"] = 1
	cycle := h.apply(state)
	h.check(cycle, nil, "leaf edit")
	if cycle.Times.FilesLowered != 1 || cycle.Times.FilesChecked != 1 || cycle.Times.FilesTransformed != 1 {
		t.Errorf("a leaf edit should redo one file: transformed %d, checked %d, lowered %d", cycle.Times.FilesTransformed, cycle.Times.FilesChecked, cycle.Times.FilesLowered)
	}
	state["whitespace"] = 1
	cycle = h.apply(state)
	h.check(cycle, nil, "spacing edit")
	if cycle.Times.FilesTransformed != 1 || cycle.Times.FilesLowered != 0 {
		t.Errorf("a spacing edit prints the same text, so nothing needs lowering: transformed %d, lowered %d", cycle.Times.FilesTransformed, cycle.Times.FilesLowered)
	}
	state["accessor"] = 1
	cycle = h.apply(state)
	h.check(cycle, nil, "accessor edit")
	if cycle.Times.FilesLowered < 2 || cycle.Times.FilesLowered > 4 {
		t.Errorf("an accessor edit should reach its file and importers, not everything: lowered %d", cycle.Times.FilesLowered)
	}
}

// TestChangeDuringRebuild saves a file while a compile is running. The change must get a compile of its own.
func TestChangeDuringRebuild(t *testing.T) {
	var hook func()
	var mu sync.Mutex
	h := startHarness(t, "during", baseState(), func(config *Config) {
		config.BeforeCompile = func() {
			mu.Lock()
			fn := hook
			hook = nil
			mu.Unlock()
			if fn != nil {
				fn()
			}
		}
	})
	h.wait()

	state := maps.Clone(h.state)
	state["msg"] = 1
	late := maps.Clone(state)
	late["msg"] = 2
	mu.Lock()
	hook = func() {
		touched := writeState(t, h.live, render(late), render(state))
		h.backend.fire(touched...)
	}
	mu.Unlock()
	h.apply(state)
	// The first compile may have read either version. The next one must have the late one.
	cycle := h.wait()
	h.state = late
	h.check(cycle, nil, "after a change during a compile")
	out, err := os.ReadFile(filepath.Join(h.live, "out", "server", "main.server.luau"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "third") {
		t.Fatalf("the change made during a compile was lost:\n%s", out)
	}
}

// TestRapidSaves fires many events for one save burst while the compile is waiting, and they must become one compile.
func TestRapidSaves(t *testing.T) {
	h := startHarness(t, "rapid", baseState(), func(config *Config) { config.Debounce = 100 * time.Millisecond })
	h.wait()
	state := maps.Clone(h.state)
	for i := range 12 {
		state["msg"] = i % 3
		touched := writeState(t, h.live, render(state), render(h.state))
		h.state = maps.Clone(state)
		if len(touched) > 0 {
			h.backend.fire(touched...)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cycle := h.wait()
	select {
	case extra := <-h.cycles:
		t.Fatalf("a burst of saves should compile once. Second compile: %+v", extra)
	case <-time.After(400 * time.Millisecond):
	}
	h.check(cycle, nil, "after a burst of saves")
}

// TestRealFileEvents runs the sequence on the real watcher backend, with real file events. It waits for a marker file to
// be copied, since nothing else tells a test that the watcher has caught up.
func TestRealFileEvents(t *testing.T) {
	testRealEvents(t, "real", nil)
}

// TestPolling is the same test on --usePolling.
func TestPolling(t *testing.T) {
	testRealEvents(t, "poll", func(config *Config) { config.UsePolling = true })
}

func testRealEvents(t *testing.T, name string, configure func(*Config)) {
	dir, live, clean := newDirs(t, name)
	_ = dir
	state := baseState()
	writeState(t, live, render(state), nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	cycles := make(chan *Cycle, 256)
	config := Config{
		Load:   loadFor(live),
		Output: &syncBuffer{},
		OnCycle: func(c *Cycle) {
			copied := *c
			copied.Result = nil
			copied.Report = formatForCompare(c.Result)
			cycles <- &copied
		},
	}
	if configure != nil {
		configure(&config)
	}
	go func() { done <- Run(ctx, config) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Run returned %v", err)
		}
	}()
	settle := func(want string) {
		t.Helper()
		// Quiet for long enough that any event still on its way has arrived.
		for {
			select {
			case <-cycles:
			case <-time.After(1500 * time.Millisecond):
				return
			case <-time.After(3 * time.Minute):
				t.Fatalf("%s: never settled", want)
			}
		}
	}
	<-cycles
	settle("first build")

	rng := rand.New(rand.NewPCG(11, 0))
	for step := range 12 {
		next := randomStep(rng, state)
		// Errors stop output, so this test stays on the states that have some: it's about events.
		next["bad"], next["extraUser"], next["extra"] = 0, 0, 0
		// Empty folders aren't a thing a real delete of one file leaves behind, and they're tested with the fake backend.
		touched := writeState(t, live, render(next), render(state))
		label := fmt.Sprintf("step %d (%s)", step, describeChange(state, next))
		t.Log(label)
		state = next
		if len(touched) == 0 {
			continue
		}
		settle(label)
		_, cleanErrors := cleanBuild(t, live, clean, state)
		if cleanErrors != 0 {
			t.Fatalf("%s: unexpected errors in a clean build", label)
		}
		want := readTree(t, filepath.Join(clean, "out"), filepath.Join(clean, "include"))
		got := readTree(t, filepath.Join(live, "out"), filepath.Join(live, "include"))
		if problem := compareTrees(want, got); problem != "" {
			t.Fatalf("%s: output differs from a clean build:\n%s", label, problem)
		}
	}
}

// TestSessionMatchesUpstream compares what the watcher prints with a session of real JS rbxtsc -w, kept in
// testdata/js-session.txt by scripts/gen-watch-golden.sh. Only the clock is taken out.
func TestSessionMatchesUpstream(t *testing.T) {
	golden, err := os.ReadFile(filepath.Join("testdata", "js-session.txt"))
	if err != nil {
		t.Fatal(err)
	}
	dir, live, _ := newDirs(t, "session")
	_ = dir
	writeState(t, live, map[string]string{
		"tsconfig.json":        render(baseState())["tsconfig.json"],
		"default.project.json": render(baseState())["default.project.json"],
		"src/shared/a.ts":      "export const a = 1;\n",
		"src/server/m.ts":      "import { a } from \"shared/a\";\nprint(a);\n",
		"src/shared/data.txt":  "hello\n",
	}, nil)
	backend := &fakeBackend{}
	cycles := make(chan *Cycle, 16)
	buffer := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	cwd, _ := os.Getwd()
	if err := os.Chdir(live); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	go func() {
		done <- Run(ctx, Config{Load: loadFor(live), Backend: backend, Output: buffer, OnCycle: func(c *Cycle) { cycles <- c }})
	}()
	defer func() {
		cancel()
		<-done
	}()
	<-cycles
	script := []struct{ rel, text string }{
		{"src/shared/a.ts", "export const a = 2;\n"},
		{"src/shared/a.ts", "export const a: string = 2;\n"},
		{"src/shared/a.ts", "export const a = 3;\n"},
		{"src/shared/data.txt", "bye\n"},
		{"src/shared/b.ts", "export const b = 3;\n"},
		{"src/shared/b.ts", ""},
		{"src/shared/data.txt", ""},
	}
	for _, edit := range script {
		full := filepath.Join(live, filepath.FromSlash(edit.rel))
		if edit.text == "" {
			if err := os.Remove(full); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(full, []byte(edit.text), 0o666); err != nil {
			t.Fatal(err)
		}
		backend.fire(full)
		<-cycles
	}
	clock := regexp.MustCompile(`\d+:\d\d:\d\d [AP]M`)
	normalize := func(text string) string {
		return clock.ReplaceAllString(strings.ReplaceAll(text, "\r\n", "\n"), "<TIME>")
	}
	buffer.mu.Lock()
	got := normalize(buffer.String())
	buffer.mu.Unlock()
	if want := normalize(string(golden)); got != want {
		t.Fatalf("session differs from JS rbxtsc -w\n--- JS\n%s\n--- rbxtsgo\n%s", want, got)
	}
}

// TestMemoryOverManyRebuilds is the check on retained memory: after a warm-up, the live heap and the goroutine count
// must stop growing. Old programs and checkers are only collectable if nothing in a long-running process holds them.
func TestMemoryOverManyRebuilds(t *testing.T) {
	rebuilds := 400
	if testing.Short() {
		rebuilds = 80
	}
	h := startHarness(t, "memory", setgetState(), nil)
	h.wait()
	state := maps.Clone(h.state)
	sample := func() (uint64, int) {
		var stats runtime.MemStats
		runtime.GC()
		runtime.GC()
		runtime.ReadMemStats(&stats)
		return stats.HeapAlloc, runtime.NumGoroutine()
	}
	names := []string{"msg", "api", "kind", "edge", "items", "accessor"}
	var heaps []uint64
	var goroutines []int
	for i := range rebuilds {
		name := names[i%len(names)]
		state[name] = (state[name] + 1) % slotsOf(state)[name]
		if h.apply(maps.Clone(state)) == nil {
			t.Fatalf("rebuild %d had no changes", i)
		}
		if i%(rebuilds/20) == 0 {
			heap, routines := sample()
			heaps = append(heaps, heap)
			goroutines = append(goroutines, routines)
		}
	}
	t.Logf("live heap MB over %d rebuilds: %v", rebuilds, megabytes(heaps))
	t.Logf("goroutines: %v", goroutines)
	// Compare the second half with the first half after warm-up, by median so a sample taken mid-rebuild can't decide.
	third := len(heaps) / 3
	early := median(heaps[third : 2*third])
	late := median(heaps[2*third:])
	if float64(late) > float64(early)*1.1+5<<20 {
		t.Errorf("the live heap keeps growing: %d MB, then %d MB", early>>20, late>>20)
	}
	if goroutines[len(goroutines)-1] > goroutines[third]+8 {
		t.Errorf("goroutines keep growing: %v", goroutines)
	}
}

func median(list []uint64) uint64 {
	sorted := slices.Clone(list)
	slices.Sort(sorted)
	return sorted[len(sorted)/2]
}

func megabytes(list []uint64) []uint64 {
	result := make([]uint64, len(list))
	for i, value := range list {
		result[i] = value >> 20
	}
	return result
}

// TestConfigReload: a tsconfig or Rojo change is a full reload, and a broken one is an error that a later fix clears.
func TestConfigReload(t *testing.T) {
	h := startHarness(t, "reload", baseState(), nil)
	h.wait()
	tsconfig := filepath.Join(h.live, "tsconfig.json")
	good, err := os.ReadFile(tsconfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tsconfig, []byte(`{ "compilerOptions": { "outDir": 5 } }`), 0o666); err != nil {
		t.Fatal(err)
	}
	h.backend.fire(tsconfig)
	if cycle := h.wait(); cycle.Errors == 0 {
		t.Fatalf("a broken tsconfig should report an error, got:\n%s", cycle.Report)
	}
	if err := os.WriteFile(tsconfig, good, 0o666); err != nil {
		t.Fatal(err)
	}
	h.backend.fire(tsconfig)
	cycle := h.wait()
	if !cycle.Initial {
		t.Error("fixing the tsconfig should be a full reload")
	}
	h.check(cycle, nil, "after fixing the tsconfig")

	state := maps.Clone(h.state)
	state["rojo"] = 1
	cycle = h.apply(state)
	h.check(cycle, nil, "after a Rojo change")
}

// TestCleanExit: cancelling stops the loop and leaves no goroutines behind beyond fswatch's shared one.
func TestCleanExit(t *testing.T) {
	before := runtime.NumGoroutine()
	h := startHarness(t, "exit", baseState(), nil)
	h.wait()
	h.stop()
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > before+4 {
		if time.Now().After(deadline) {
			t.Fatalf("%d goroutines before, %d after", before, runtime.NumGoroutine())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
