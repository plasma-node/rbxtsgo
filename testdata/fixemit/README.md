# Tests of the fixes that differ from JS 3.0.0

rbxtsgo stays byte-identical to JS `roblox-ts` 3.0.0 for code that compiles and runs correctly there. A few upstream fixes are cherry-picked anyway, in two classes. See [docs/fix-emit.md](../../docs/fix-emit.md), and [docs/known-differences.md](../../docs/known-differences.md) for the list.

- **Always on:** where JS crashes, or writes Luau that fails to parse. No working output can change.
- **`--fix-emit`:** where JS writes Luau that parses and does the wrong thing, so working projects could see a change.

The goldens in `testdata/differential` come from JS, so they can't hold what JS doesn't do. These are the only hand-checked expected outputs in the repo, and they're tested in `internal/difftest/fixemit_*_test.go`. Those tests skip without what `scripts/setup-diff-corpus.sh` sets up, or without `rojo` and `lune`.

```sh
go test ./internal/difftest -run 'FixEmit|UnsupportedIteration|ParenthesizedArray|AcceptedString'
UPDATE_FIXEMIT=1 go test ./internal/difftest -run 'FixEmit'   # rewrite expected/ and strings/expected.txt, then read the diff
```

| Path | What it is |
|---|---|
| `project/` | The `--fix-emit` cases, written from scratch or ported from upstream's tests where a file says so. JS compiles all of it, and each case is wrong in JS |
| `project-crashes/` | Cases JS crashes on and rbxtsgo compiles, flag or no flag |
| `diagnostics/` | Cases JS crashes on and rbxtsgo reports. `noUnsupportedIteration.*` are upstream's |
| `strings/` | The string matrix: the escape cases, each its own file, because one that fails to parse would hide the others. `expected.txt` records each case's status in JS, flag off and flag on |
| `expected/out/` | What `project/` compiles to with the flag on |
| `expected/runtime/` | What each case in `project/` prints with the flag off, which shows the bug |
| `run.luau` | Runs a place built by Rojo under Lune, as `scripts/scoreboard-runtime.sh` does for upstream's suite |
| `oracle.cjs` | Runs the same TypeScript as plain JS, through tsc and Node |

## What the tests check

- **Flag off:** `project/` is byte-identical to what JS writes, at 1 and 8 workers.
- **Flag on:** `project/` is `expected/out`, at 1 and 8 workers, with setget on or off, and from the flag or the tsconfig block. `--no-fix-emit` wins over the block.
- **Runtime:** each case prints with the flag on, with or without setget, exactly what Node prints for the same TypeScript. So the oracle for "right" is JS, and not something written by hand. With the flag off each case prints something else, which shows that the bug is real.
- **Strings:** every case's value is compared with what JS's `eval` gives. With the flag off, a case changes only if JS's Luau for it fails to parse, and then it has the right value. With the flag on, every case has the right value.
- **Rebuilds:** `watch/fixemit_test.go` changes only the setting, in an incremental build and in a running watcher, and expects new output.
