# rbxtsgo

**A much faster [roblox-ts](https://roblox-ts.com) compiler.** rbxtsgo is written in Go, on top of TypeScript 7's Go compiler. It produces the same Luau as `rbxtsc` 3.0.0, works with your existing `@rbxts/*` packages and tsconfig, and installs as a drop-in `rbxtsc`.

| On a production Roblox game | roblox-ts 3.0.0 | rbxtsgo | |
|---|---|---|---|
| **Save a file in watch mode** | 3.6 s | **87 ms** | ~40x faster |
| **Save a widely imported file in watch mode** | 10.2 s | **317 ms** | ~32x faster |
| **Full build** | 13.1 s | **0.92 s** | ~14x faster |
| **Peak memory, full build** | ~630 MB | **~335 MB** | ~47% less |

The compiled Luau from that game is byte-for-byte identical to what `roblox-ts` writes.

> **Early development.** It's new and has only been run on Windows x64. It hasn't had long-term use, so keep your project in git, check your own output, and playtest before relying on it. Please [open an issue](https://github.com/plasma-node/rbxtsgo/issues) if something breaks or the output differs.

## Why use it

- **Saves feel instant.** Watch mode rebuilds in milliseconds, so your game is synced by the time you've switched windows.
- **Same output.** Luau is compared byte for byte against the real JS compiler. Differences are the exception, and they're [listed](docs/known-differences.md).
- **No migration.** Your tsconfig, packages and scripts stay as they are, including settings TypeScript 7 removed, like `baseUrl` and `moduleResolution: Node`.
- **Real incremental builds.** A build with no changes takes a fraction of a second, and tests compare incremental output against a clean build after every edit.
- **Setget built in.** [`rbxts-transformer-setget`](https://github.com/wAD4444/rbxts-transformer-setget) runs natively with no package needed, and **errors point at the correct line** in your source, which the JS plugin can't do.
- **Build times shown.** Each build tells you how long it took.
- **Crashes fixed.** Some cases that crash the JS compiler or make it write invalid Luau are fixed. Other known bugs have optional fixes behind [`--fix-emit`](docs/fix-emit.md).

## Benchmarks

Medians of 5 to 15 runs on one Windows 11 desktop (22 logical CPUs), comparing `roblox-ts` 3.0.0 on Node 20 with rbxtsgo. Machine speed varied between runs and these aren't lab conditions, so treat them as approximate and try your own project.

### A production Roblox game

A couple hundred TypeScript files, with React UI, setget and a wide set of `@rbxts` packages.

| | roblox-ts 3.0.0 | rbxtsgo | Faster by |
|---|---|---|---|
| Watch: save, rarely imported file | 3.6 s | **87 ms** | ~40x |
| Watch: save, widely imported file | 10.2 s | **317 ms** | ~32x |
| Full cold build | 13.1 s | **0.92 s** | ~14x |
| Incremental: no changes | 5.4 s | **0.25 s** | ~22x |
| Incremental: edit a rarely imported file | 5.1 s | **0.43 s** | ~12x |
| Incremental: edit a widely imported file | 8.2 s | **0.65 s** | ~13x |
| Peak memory, full build | ~630 MB | **~335 MB** | ~47% less |

The watch figures for roblox-ts come from an earlier run than the rest. This project is closed source, so these numbers can't be reproduced. The next ones can.

### [littensy/slither](https://github.com/littensy/slither) (public, MIT)

An open-source game at commit [`56545f1`](https://github.com/littensy/slither/tree/56545f14df4d25d3df7e71fbaeff7b21cf8798a7): 256 TypeScript files, about 12k lines. All 258 output files were byte-identical to JS.

| | roblox-ts 3.0.0 | rbxtsgo | Faster by |
|---|---|---|---|
| Watch: save, rarely imported file | 3.87 s | **91 ms** | ~42x |
| Watch: save, widely imported file (41 importers) | 4.12 s | **210 ms** | ~20x |
| Full cold build | 7.06 s | **0.81 s** | ~9x |
| Incremental: no changes | 3.81 s | **0.32 s** | ~12x |
| Incremental: edit a rarely imported file | 4.07 s | **0.49 s** | ~8x |
| Incremental: edit a widely imported file | 6.51 s | **0.71 s** | ~9x |
| Peak memory, full build | 259 MB | **222 MB** | ~14% less |

Smaller projects have less to gain: on the 10-line [rbxts-rs/template](https://github.com/rbxts-rs/template) at `e446ad9`, a cold build is 1.47 s against 0.13 s, which is mostly Node's startup time. roblox-ts can't read that template's TypeScript 7-style config, so its time is from the same source under a classic config.

**Not faster:** projects using a transformer plugin other than setget, such as Flamework. Those builds are handed to the JS compiler, so they run at JS speed.

## Install

### With Aftman or Rokit

Add it under the name `rbxtsc`, so your scripts and editor keep working. In `aftman.toml`:

```toml
[tools]
rbxtsc = "plasma-node/rbxtsgo@0.1.1"
```

Then:

```sh
aftman install
npm uninstall roblox-ts
```

Removing the package matters, because a local `roblox-ts` takes priority over the tool. Keep it if your project uses a transformer plugin other than setget, since those builds need the JS compiler. In that case, `rbxtsc` stays the JS one, and you run rbxtsgo by installing it under another name.

- **Don't run `npx rbxtsc`.** Without a local install, `npx` downloads an unrelated placeholder package with that name. Run `rbxtsc` directly, and update any `npx rbxtsc` in your scripts.
- **"Being used by another process" during install:** a running tool such as `rojo` is holding its file. Stop it, and run the install again.
- **To go back to JS:** remove the line from `aftman.toml`, then run `npm install -D roblox-ts@3.0.0`.

Aftman's install has been run on Windows x64. Rokit reads `aftman.toml` too, but hasn't been tried, and neither has macOS or Linux, though binaries are built for them.

**VS Code:** the [roblox-ts extension](https://github.com/roblox-ts/vscode-roblox-ts)'s start button uses a local `roblox-ts` if there is one, and otherwise runs `rbxtsc -w` from your `PATH`, so it should pick this up. That's from reading its source. It hasn't been tested, including whether its stop button stops the process on Windows. If it doesn't work, run `rbxtsc -w` in a terminal.

### From source

You need Go (the version in `go.mod`), `git`, and a bash shell (Git Bash on Windows).

```sh
git clone https://github.com/plasma-node/rbxtsgo.git
cd rbxtsgo
bash scripts/setup-third-party.sh     # clones TypeScript 7.1 at a pinned commit and applies patches/typescript/
go build -o rbxtsgo ./cmd/rbxtsgo     # rbxtsgo.exe on Windows
```

## Use

It takes the flags of `rbxtsc build`, and flags go before `build`:

```sh
rbxtsc -p .          # build the project in the current folder
rbxtsc -p . -w       # watch mode
rbxtsc --help        # all flags
```

Extra flags: `--checkers N` sets the number of type checkers, `--setget` turns on the built-in setget transformer, and `--fix-emit` turns on the optional fixes.

### Setget

Existing `rbxts-transformer-setget` entries in `plugins` keep working, with or without the package installed. Or, without a plugin entry, add this at the top level of `tsconfig.json`, next to `compilerOptions`:

```json
"rbxtsgo": { "setget": true }
```

The default method prefix is `__`, the same as the plugin's 2.0.0 release. See [docs/setget.md](docs/setget.md).

## Compatibility

- **Reference:** `roblox-ts@3.0.0`, which uses TypeScript 5.5.3. Expected output comes from running the real JS compiler, never from editing test results by hand.
- **Test corpus:** roblox-ts's own test sources, plus cases written for this project. 186 of 188 Luau files are byte-identical, and the other 2 are cases where JS's output fails to parse. All 460 of roblox-ts's runtime tests pass on rbxtsgo's output under Lune.
- **Real code:** 9 public projects and about 40 popular packages, including `@rbxts/react`, `vide`, `charm`, `reflex` and `profile-store`, came out byte-identical. The exception is `.d.ts` output, below. Two of those projects went through the JS fallback.
- **Config:** tsconfigs written for `rbxtsc` work unchanged. TypeScript 7-style configs (`module: preserve`, `moduleResolution: bundler`) are accepted too. See [docs/config-normalization.md](docs/config-normalization.md).
- **Parallelism:** worker counts don't change output.

### Known differences

- **Error text:** 230 of 243 diagnostics comparisons match. TypeScript 7 words some errors differently, and sometimes reports them on a different line. The code is rejected on both sides, so Luau isn't affected.
- **Declaration files:** `.d.ts` output is printed slightly differently, which matters only if you publish packages.
- **Your editor:** VS Code still uses its own TypeScript for squiggles, so it won't get faster and may occasionally disagree with the build.

The full list is in [docs/known-differences.md](docs/known-differences.md).

### Not done yet

- Only the build command and its flags exist. A few parts of `rbxtsc` aren't ported, such as `VirtualProject`.
- `roblox-ts` 3.1 isn't released yet. When it is, rbxtsgo will need work to match its output.
- Some bugs remain, for example `new.target` crashes, as it does in roblox-ts.
- Concurrency hasn't been checked with Go's race detector. Deterministic output is shown by repeated runs at different worker counts.
- macOS and Linux test data doesn't exist yet, so some tests skip there.

## Related projects

[rbxts-rs](https://github.com/rbxts-rs/rbxts-rs) is a separate native rewrite of roblox-ts, in Rust. It requires tsconfig changes, and it supports Flamework natively through its own plugin system, which rbxtsgo doesn't. It's worth a look if that's what you need.

## Development

```sh
bash scripts/setup-third-party.sh
go build ./...
go test ./...
bash scripts/setup-diff-corpus.sh     # needs node and npm: installs roblox-ts@3.0.0 and the corpus dependencies into .scratch/
bash scripts/scoreboard.sh            # the differential scoreboard
```

Some tests skip themselves when their dependencies aren't installed. [docs/](docs/README.md) has the design notes, and [testdata/differential/README.md](testdata/differential/README.md) describes the differential harness.

## License

MIT. See [LICENSE](LICENSE).

rbxtsgo is a port of roblox-ts and its libraries, and is built on TypeScript. Their license texts are in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
