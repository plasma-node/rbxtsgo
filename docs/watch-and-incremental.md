# Watch mode, incremental builds and declaration emit

## The rule that decides what can be skipped

A compile has up to two stages that read a whole program: setget on the source program, and checking and lowering on the printed one. With no plugin those are the same program. A stage's answer for one file depends only on:

1. that file,
2. what the checker can see from it, which is what it imports, transitively, and the global scope,
3. settings: options, Rojo project, packages.

So `tsbridge.AffectedFiles` redoes a changed file plus everything that reaches it through imports. It compares programs by parsed file object in watch mode, and by text hash across runs. That includes `import type`, `import("...")` types, module augmentation names and `/// <reference path>`.

It redoes **everything** when:

- a changed file, in its old or new version, is a declaration file, is not a module, or has `declare global`, `declare module` or an ambient module,
- files were added or removed,
- an unchanged file's imports resolve somewhere else,
- with setget, the set of accessor names changed,
- the options, tsconfig, extended configs, Rojo tree, `package.json`, any package's `package.json`, or the compiler changed,
- any diagnostic with no file appears (`rbxtsc` repeats those per file, and they can grow with what got checked).

This is deliberately wider than TypeScript's own `.d.ts` signature logic: lowering depends on details an importer's own declaration output does not show, such as whether something is a method or a function property.

The two stages are separate on purpose. A file whose printed text comes out the same keeps its parsed object, so a spacing edit is one print and no lowering.

**Only files with no diagnostics at all count as done.** A file with warnings compiles again every time, so its warnings show again.

**Known limit.** Checking order could in principle change a result, since a checker checks a different set of files before a given one when others are skipped. The port already relies on this not mattering, across worker counts and TS 7's file partitioning, and the tests look for it. They cannot rule it out.

## Evidence

`TestEditSequence` and `TestIncrementalBuilds` compare with a clean build after every step of a seeded random sequence, with setget on and off: Luau, copied files, `include`, and for watch also the diagnostics. The fixture covers exported types, inferred types through two files, const enums, re-exports, `.d.ts` globals, `declare global`, `declare module`, `import("...")` types, JSON types, class hierarchies, `Array` against `Set` (a macro-relevant type), cycles, added and deleted files and folders, moved imports, non-compiled files, Rojo and tsconfig changes, type errors, roblox-ts errors, and setget accessors added or removed in another file, through a re-export, and in a `declare global`. The harness asserts that the first build has no errors, because a fixture that always fails would pass every comparison.

## Watch mode

A port of `setupProjectWatchProgram.ts`. Same messages, checked against a recorded JS session (`watch/testdata/js-session.txt`, recorded by `scripts/gen-watch-golden.sh`). Differences from JS, all on purpose:

| Case | JS | rbxtsgo | Why |
|---|---|---|---|
| tsconfig, Rojo project, `package.json`, packages change | ignored until restart | full reload, without restart | A clean build would see them |
| Watching | root folders | also those files and `typeRoots` | Same |
| A file added that `exclude` leaves out | compiled, because it adds to a list | not compiled | Reads the tsconfig patterns again, as a clean build does |
| `.d.ts` or `.json` change | rewrites what imported the changed files only, can go stale | what a clean build says | The edit tests require it |
| Internal error, or a plugin's exit | process ends | reported, watching goes on | A watcher that dies loses more than it saves |
| Writing | rewrites files it compiled | only bytes that changed | Fewer spurious downstream events |
| Debounce | chokidar 50ms + 100ms timer | fswatch 50ms + 20ms quiet | Latency. Bursts still merge |
| `--usePolling` | chokidar polling | a 100ms walk of the watched folders | fswatch has no polling |
| Clock in messages | `toLocaleTimeString()` | `3:04:05 PM` | Same on an en-US machine. Other locales differ |
| Ctrl-C | exit by signal | closes watches, exit 0 | |

A project with a transformer plugin other than setget hands the whole watch to JS at startup, as a build does. If a tsconfig change asks for one while watching, that is reported as an error asking for a restart.

**When saves merge.** A save that lands during a compile, or within the debounce, joins the next compile. Nothing is lost, and the next compile reads what is on disk. So a run of saves closer together than a compile takes gets fewer status lines than saves. JS merges the same way, with a longer window. `TestEverySpacedSaveGetsAReport` and `TestChangeDuringRebuild` check this.

**Optimizations, each added for a measured reason:**

1. The Rojo resolver's walk called `realpath` and `stat` for every child, as the JS does, and Go's `EvalSymlinks` on Windows stats every path component. On a generated project that was most of a rebuild. It now uses the directory entry unless it is a link.
2. The same work is kept per process (`PrepareCache`) until a config or package event.
3. Skipping per the rule above.
4. Up to 8 checkers in watch mode, where builds use 4. `--checkers` overrides it. Worker counts do not change output (`TestWorkerCounts`).

`RBXTSGO_WATCH_FULL=1` turns off 2 and 3, to compare. Keeping checkers alive between rebuilds is not done, because nothing measured needs it.

## Incremental builds

`incremental: true` and `tsBuildInfoFile` use the same path as TypeScript's rules give. The file has its own format, with `"version": "rbxtsgo-1"`, not TypeScript's: TS 7's is not TS 5's either, and it would not hold what this needs (printed text hashes, output stats). `TestBuildInfoSharedWithJS` runs each compiler over the other's file, with edits in between: JS finds no `program` in ours and starts over, and ours rejects anything without its version. Output stayed equal to a clean build every time. The cost of taking turns is a full build each way.

**What it holds.** A key, and per program file: a hash of the source, a hash of what got compiled (the printed text with setget), whether it has global effects, its resolved imports, whether it compiled clean, and the size and mtime of its outputs. The key covers what `createProgramFactory` hashes (compiler version, project type, `isPackage`, plugins, Rojo config) and more: the raw tsconfig and what it extends, the project options that change output, every package's `package.json`, the Rojo tree including nested projects, and the compiler binary's size and mtime.

**Steps.** Upstream's cleanup, include and copying still run first. Then, if the key, every source hash and every output stat match, the build stops: no printing, no checker. Otherwise setget runs as in a clean build, the printed hashes pick the files to compile by the rule above, and any file with a missing or changed output compiles again on its own. A build with errors writes no Luau and no state, so the next run compares with the last good state and reports the same errors. A missing, damaged or foreign state file means a full build.

This goes further than JS in two places: it notices missing outputs, and it reports warnings again.

Watch mode reads and writes the same file when the project has it on, so a watcher starting on an up-to-date project compiles nothing, and a build after a watcher has nothing to do (`TestWatchUsesAndKeepsBuildInfo`). The file can differ by a few bytes between runs on the same sources, because it holds output mtimes as text. Nothing reads it byte for byte.

## Declaration emit

`Program.Emit` with `EmitOnlyDts` per file. TS 7 has no `afterDeclarations`, so `transformPaths` and `transformTypeReferenceDirectives` run on the printed text: parse it, find the same nodes the upstream visitor finds, and replace those string literals. Quotes follow upstream: a rewritten specifier gets double quotes, an untouched one keeps what TypeScript printed. `declarationMap` positions do not account for the rewrite, and that is not tested.

On `testdata/declarations/fixture` the output is byte-identical to JS on all 29 files. Wider projects show real differences from TS 7's printer, listed in [known differences](known-differences.md).

## Project references

Not ported, because `rbxtsc` ignores `references` and has no build mode. See [config normalization](config-normalization.md).
