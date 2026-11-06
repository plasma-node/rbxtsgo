# Known differences from `rbxtsc`

Every difference from JS `rbxtsc` 3.0.0 that is accepted gets a row here, with the reason. Expected output is never edited to make a test pass, and no failing case is skipped silently. The differential harness and its baseline are described in [testdata/differential/README.md](../testdata/differential/README.md).

## Accepted

| Case | Difference | Reason |
|---|---|---|
| Setget: where diagnostics point | Line and column are in the user's source. JS reports them in its reprinted text, and shows that text. In the `setget-errors` fixtures, 39 of 57 diagnostics are on a different line in JS | JS's lines are wrong for the file on disk. See [setget](setget.md). The differential test compares the printed positions before mapping, so this hides nothing else |
| Setget: repeated diagnostics | An error in the object of a compound assignment is reported once. JS reports it twice, at two places in its printed text | Both map to one place in the source |
| Setget: plugin entry with no package | Builds as if installed. JS warns, skips the plugin and rejects every accessor | Setget is built in |
| Setget: the `rbxtsgo` block and `--setget` | JS does not have them | rbxtsgo's own option |
| `--fix-emit` and `"rbxtsgo": { "fixEmit": true }` | JS does not have them | rbxtsgo's own option, off by default. It fixes `continue` inside `switch`, default re-exports of `export =` modules, functions declared after `return`, and string escapes that have the wrong value |
| `\uXXXX`, `\{` and a backslash and a line break in a template; `for...of`, spread or destructuring on a type with no iteration; `[(a)] = b` | JS crashes, or writes Luau that fails to parse. rbxtsgo reports a diagnostic, or writes Luau that works | Always on, because no working output can change |
| `module: preserve` with `moduleResolution: bundler` | Accepted, and compiled as the classic pair. JS rejects the config | See [config normalization](config-normalization.md) |
| Declarations: `export =` of an expression | TS 7 prints `export = _default;` then `declare function _default(): void;`. TS 5.5 prints `declare const _default: () => void;` first. Measured on 38 of the 45 declaration files of upstream's specs | TS 7's printer. Same meaning to a reader of the file |
| Declarations: exported destructuring | TS 7 keeps `export declare const { a, b }: {...}`. TS 5.5 prints `export declare const a: number, b: number;` | TS 7's printer. tsc 5.5.3 reads TS 7's form and gives the same types, tested |
| Declarations: union and spread property order | `undefined[] \| [number]` for `[number] \| undefined[]`, and spread properties in a different order | TS 7's type ordering, see [union order](union-order.md). Same types |
| Watch: where it deliberately goes beyond JS | Reloading on config changes, and others | Output equals a clean build, which JS watch does not always. See [watch](watch-and-incremental.md) |

## Differ, and not in the baseline

These show as failures on the scoreboard. None of them changes Luau, because each program is rejected on both sides:

| Case | Difference | Cause |
|---|---|---|
| `setget-errors` `unionField.ts` | Same code and position. The message says `WithField \| WithGetter`, and JS says `WithGetter \| WithField` | TS 7's union order |
| `setget-errors` `jsxText.tsx` | Same code and position. The message's type text is shorter | How TS 7 prints the type |
| `setget-errors` `mixedCoalesce.ts` | Same code and line. TS 7 reports `a ?? b \|\| c` at the start with the operators the other way round | Inferred from the positions: TS 7 seems to parse it as `(a ?? b) \|\| c`, and TS 5.5.3 as `a ?? (b \|\| c)` |
| `ts7-errors-*` `starCallable.ts`, `starClass.ts` | JS has TS2497 at the import. TS 7 has TS2349 or TS2351 at the use, on another line | Open |
| `ts7-errors-*` `overloadErrors.ts` | Same code, TS2769. JS lists every overload at the call. TS 7 reports only the last overload, at the failing argument | TS 7 only elaborates the last candidate |
| `ts7-errors-*` `neverTarget.ts` | Same code and position. TS 7's message names the literal, and JS the widened type | Inferred: TS 5.5.3 widens the literal in this message and TS 7 does not |
| `ts7-errors-*` `topLevelAwait.ts` | Same code and position. TS 7's message lists more `module` values | TS 7's message |

## Setget diagnostics that cannot match the same code written as methods

They are reported, and tested in `TestSetgetCLILines` and `TestSetgetDiagnosticLines`:

- **Syntax errors.** The check sees the printed text, as in JS. The printer can repair a mistake, so `print(c.v;` and `if (true {` report nothing and compile. Or it can change one: `const = 1;` is TS1123 at column 1, where the plain file has TS1134 at column 7. The line is right.
- **Message text.** Messages name what the check saw, such as `__getv`, as in JS.
- **A parenthesis the printer added.** A diagnostic on `(c.__getv())` points at `c`, the first token that exists in the source.
- **JSX text starting with whitespace.** The column is that of the first non-whitespace character. The line is right.
- **Accessors with computed, string or private names.** These print as something that does not parse as written, so the errors are about `__get`. They point at the accessor's name.

## Output that is not Luau, and that no corpus compares

- **Related locations in the CLI.** TS 7's writer prints `file:line:col - message` on one line, and TS 5.5.3 puts the message under the source line. This is true of every project, not only setget.

## Places that could differ and that no corpus case reaches

These are not accepted, because nothing has measured them:

- `getAllSuperTypeNodes` for `@augments` in a JS file: TS 5.5.3 returns the tag's class node, and the port returns the extends element.
- A package duplicated in `node_modules`: TS 7 gives the redirect's target, and JS gives a redirect file under its own path, so the output path can differ.
- `isMethodFromType` and `validateSpread` depend on union and property order. See [union order](union-order.md).
