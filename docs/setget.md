# Setget, and the fallback for other plugins

rbxtsgo has a built-in port of `rbxts-transformer-setget@2.0.0` (get and set accessors lowered to `__get` and `__set` methods). Its license is in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).

## Turning it on

Setget is off by default. Any one of these turns it on, and none of them needs a package installed:

| Way | Example |
|---|---|
| Top-level tsconfig block | `"rbxtsgo": { "setget": true }` |
| Flag | `--setget`, and `--no-setget` to override the block |
| Plugin entry, as with JS | `"compilerOptions": { "plugins": [{ "transform": "rbxts-transformer-setget" }] }` |

The prefix defaults to `__`, so `get x()` becomes `__getx()`. To change it, use `"setgetPrefix"` in the block, `--setgetPrefix`, or the plugin's `customPrefix`. The flag beats the block, and the block beats the plugin entry.

The block is a top-level key for the same reason as roblox-ts's `"rbxts"`: TypeScript ignores top-level keys it does not know. `tsc --showConfig` (5.5.3) and JS `rbxtsc` both accept it, and JS ignores it, so **JS needs the plugin entry to build a project with accessors.** A project that builds with both compilers should keep the entry. `--no-setget` turns off the built-in option only, and a plugin entry stays on. No editor was tested.

## What JS does

`compileFiles.ts` runs plugins on every file with the first program, prints each file with `ts.createPrinter()`, and makes a new program from the text. Everything after that sees printed source: checking, comments, literals and diagnostic positions.

## Measurements against JS

**The reprint changes Luau on its own.** On upstream's 44 accessor-free spec files, with and without the plugin, 2 files differ: `1_0_0_0_0` becomes `10000`, because `createPrinter()` has no target and that turns off numeric separators, and a multi-line comment gets new indents.

**Turning accessors into methods changes what type checks.** For example (all in `testdata/differential/fixtures/setget/errors`):

| Source | JS |
|---|---|
| `class C implements HasX { get x() }` | TS2420: the class no longer has `x` |
| `const { x } = new C()` | TS2339 |
| `c["x"]` | TS7053 |
| `++c.x` | TS2357 |
| `class Child extends Base { get v() }` over a property | Accepted, though plain TypeScript rejects it |
| `print(c.v;` | Accepted: the printer writes the missing `)` |

**The TS 7 printer can match.** With an identity transform over 357 files, including roblox-ts's own source and about 7,000 generated operand and context combinations, JS and Go differed only in added parentheses, such as `(a!).b`. The small printer option in `patches/typescript/0004` fixes that, and then all 357 are byte-identical. No target setting matches JS's literals.

## Design

| Option | Luau identical | Original lines in diagnostics | Verdict |
|---|---|---|---|
| Lower accessors natively, no reprint | No: it would have to imitate the printer's comment and literal effects and the type changes above | Yes | Rejected |
| Port the plugin, print, rebuild, report printed positions | Yes | No | Matches JS, but points at lines that are not in the file on disk |
| **Port the plugin, print, rebuild, map positions back** | Yes | Yes | **Chosen** |

The steps, in `plugins/apply.go`:

1. Scan every program file for accessor names, in parallel, with no checker.
2. Run the plugin port on each file, with a checker only for files with a property access that could be an accessor.
3. Print each file with a source map. The source map has no effect on the text.
4. Make a new program from the text. Files that were not printed, such as declaration files, keep their parse and binding.
5. Check and lower that program, as usual.
6. `Applied.MapDiagnostics` moves each diagnostic to the original file, including related locations. The original file is what gets shown.

The printed positions stay available before step 6. The differential test compares those with JS, so it still catches any change in what gets reported.

The first program only answers `getSymbolAtLocation` for property accesses whose name matches an accessor somewhere in the program. A computed accessor name, or a mapped type with an `as` clause when accessors exist, turns that filter off, because then only the checker knows. With no name matches the plugin step is a print and a parse. When common names match in most files it is about as expensive as a full check. These are rough figures from a single process, not speed claims.

## The plugin's behavior, which the port keeps

- **Union types.** The checker's union property merges declarations, so one getter in any member rewrites the access for all of them. With a plain field in the other member, that is TS2339 on `__getv`. Two classes that both have the accessor work.
- **Inheritance.** Decided by the declarations of the symbol found. `super.x` becomes `super.__getx()`. A getter over a base property is not rejected, because it is a method by then.
- **Its own diagnostics.** It has no diagnostics. It writes `[rbxts-transformer-setget]: ...` to stdout and calls `process.exit(1)`, in four cases: compound assignment or postfix with no getter, and either of them with `new` in the target. The first in file order wins, at any worker count.
- **Quirks.** `y = o.x` becomes `o.__setx(o.__getx())` when `x` has a setter, losing `y =`, because the first property access anywhere under the expression decides. `arr[o.x] = 1` becomes `o.__setx(1)`. Compound assignment evaluates the object twice. Postfix keeps the object expression as written, so `o.inner.n++` fails if `inner` is an accessor. `++x` is not handled. These are bugs in the plugin, and JS output has them.

## Fallback to the JS compiler

`plugins.Decide` follows upstream's `getPluginConfigs`: only entries with a string `transform` count, a parent's entries add to the child's, and a child's empty list turns them off. Language service plugins and comments are ignored. Anything rbxtsgo cannot reproduce goes to `node_modules/roblox-ts/out/CLI/cli.js` with the same flags, minus rbxtsgo's own, and its exit code comes back:

- any other transformer plugin
- setget listed twice, or with another `type` or `import`
- an installed package at a version other than 2.0.0, unless the built-in option is on
- a list in `extends`, or one that cannot be resolved, when plugins are on, because upstream throws there

If the JS compiler is not installed, the error names the plugin, the path and the install command. The built-in option together with another plugin is an error, because the JS compiler that plugin needs does not have the option.

In watch mode, `Decide` runs per config load and `Apply` per rebuild, always on the unprinted project.

## Tests

- `setget-*` corpora: goldens from real JS with the real plugin. `reprint` cases compare the printed text as well.
- `TestSetgetDiagnosticLines`: in-process, on the markers in the `setget/errors` fixtures.
- `TestSetgetCLILines`: through the real binary, against the same code written as methods, on the same lines.
- `scripts/scoreboard-plugins.sh`: whole projects through both CLIs at 1 and 8 workers, the three ways to turn it on, the plugin's exits and the fallback.
