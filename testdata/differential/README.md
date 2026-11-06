# Differential test

This compares `rbxtsgo` with real JS `roblox-ts@3.0.0` on TS 5.5.3, byte for byte. It's the gate for all port work.

## Run it

```sh
bash scripts/scoreboard.sh          # the scoreboard: files byte-identical, diagnostics matched, work list
bash scripts/scoreboard.sh -v       # also print the first difference of every failure
bash scripts/scoreboard.sh --cli --runtime   # also whole-project JS vs rbxtsgo, 1 vs 8 workers, and the Lune suite
go test ./internal/difftest         # the gate: fails if anything in baseline.txt regresses, and compares 1 and 8 workers
```

`RBXTSGO_TRACE_UNION=<file>` logs lookups on unions with distinct symbols. See [union order](../../docs/union-order.md).

First-time setup, once per clone: `bash scripts/setup-diff-corpus.sh`. It needs npm and network access, and it puts everything in `.scratch/diff`.

## What's in here

| Path | What it is |
|---|---|
| `corpora.json` | The corpora and how each is compiled |
| `gen-golden.cjs` | The JS side. Run through `scripts/gen-diff-goldens.sh` |
| `golden/<corpus>/files/` | What JS wrote, per file under `outDir` |
| `golden/<corpus>/diagnostics.json` | What JS reported, structured, per file or as `<whole>` |
| `fixtures/` | Our own public fixtures, written from scratch |
| `baseline.txt` | The ratchet: every case that passes |

Never edit a golden by hand. Rerun `scripts/gen-diff-goldens.sh` only when the corpus changes.

## Corpora

| Corpus | Source | Mode |
|---|---|---|
| `upstream-tests` | upstream `tests/src`, including `tests/src/diagnostics`, at the versions in upstream's lock file | `perfile` |
| `fixtures` | `fixtures/core` | `perfile` |
| `jsx-react` | `fixtures/jsx-react`: `React.createElement` and `React.Fragment` against a local stand-in for `@rbxts/react`. Real packages are in `testdata/ecosystem`, see `scripts/validate-ecosystem.sh` | `perfile` |
| `union-isolated` | `fixtures/union-isolated`, with no imports, so program order differs from `fixtures` | `whole` |
| `whole-errors` | `fixtures/whole-errors` | `whole` |
| `spike-fixture` | `spike/fixture` | `whole` |
| `setget-core` | `fixtures/setget/core`, with the plugin | `whole` |
| `setget-errors` | `fixtures/setget/errors`, with the plugin, each file on its own | `perfile` |
| `setget-fatal` | `fixtures/setget/fatal`, where the plugin itself exits | `plugin-exit` |
| `setget-prefix`, `setget-off` | `customPrefix`, and plugin entries that are commented out | `whole` |
| `setget-upstream`, `setget-fixtures`, `setget-spike` | Existing code with the plugin on, which measures the reprint alone | `whole`, `perfile` |
| `ts7-core`, `ts7-errors`, `template`, each `-classic` and `-ts7` | One set of sources under a classic config for JS, and a TS 7 style config for rbxtsgo. See [config normalization](../../docs/config-normalization.md) | `whole`, `perfile` |

`perfile` does what upstream's `src/CLI/test.ts` does: one program, then `compileFiles` on one file at a time, so an error in one file can't stop the others. `whole` calls `compileFiles` once with every file, as `rbxtsc build` does. `plugin-exit` gives each file a process, and records what a plugin wrote before calling `process.exit`. A corpus with a `golden` field has no goldens of its own and compares with that corpus's.

A `perfile` corpus with a plugin must keep each file self-contained: JS's watcher accumulates printed text, so a file compiled before its dependency sees that dependency unprinted.

## What counts as a match

- **Output:** the bytes of each file under `outDir`, compared with no normalization. A file only one side wrote is a failure.
- **Reprints:** with `writeTransformedFiles` in a corpus's options, each file's printed text, compared with no normalization. Diagnostic positions depend on it.
- **Diagnostics:** the source, category, code, roblox-ts id, line, column (in UTF-16 units on both sides), message with ANSI colors removed, and order. For TypeScript's own diagnostics, only the top line of the message chain is compared. Lengths aren't compared, because upstream stores `range.end` as the length, and Go's offsets are bytes.

## The ratchet

`TestDifferential` fails when a case in `baseline.txt` stops matching. A failing case that isn't in the baseline is logged, and doesn't fail the test, so the test can gate an unfinished port. After a batch of work, run `UPDATE_BASELINE=1 go test ./internal/difftest -run TestDifferential` and commit the longer baseline. Never remove a case from it without writing the reason in [known-differences.md](../../docs/known-differences.md).
