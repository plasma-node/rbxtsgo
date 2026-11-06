# rbxtsgo

A [roblox-ts](https://roblox-ts.com) compiler written in Go, on top of TypeScript 7's Go compiler. It aims to produce the same Luau as `rbxtsc` and to stay compatible with `@rbxts/*` packages.

Status: in development. The core compiler, watch mode, incremental builds, declaration emit and the built-in setget transformer are ported. There are no releases and no prebuilt binaries yet. A few parts of `rbxtsc` are not ported (for example `VirtualProject`), and only the build command and its flags exist.

## Install

### With Aftman or Rokit (not tested yet)

Releases are built by a GitHub Actions workflow (`.github/workflows/release.yml`) when a version tag is pushed. Each release has one zip per target, named like `rbxtsgo-0.1.0-windows-x86_64.zip`, with the binary at the zip root. The names follow the OS and architecture words that Aftman and Rokit look for in their public source code. **No release has been published yet, and neither tool has been tried against one**, so everything in this section is untested.

Once a release exists, add it under the name `rbxtsc`, so existing scripts and editors that call `rbxtsc` use rbxtsgo:

```sh
aftman add plasma-node/rbxtsgo-release rbxtsc
rokit add plasma-node/rbxtsgo-release rbxtsc
```

Then:

- Remove the project's local `roblox-ts` (`npm uninstall roblox-ts`) so `rbxtsc` resolves to the tool and not to `node_modules/.bin`. Keep it if the project uses a transformer plugin other than setget: those builds are handed to the JS compiler in `node_modules` (see below).
- Do not run `npx rbxtsc`. `npx` runs the local npm package when there is one, and downloads roblox-ts when there is not, so it never reaches the tool. Run `rbxtsc` directly, and change `npx rbxtsc` in your `package.json` scripts.
- VS Code: extensions or buttons that start the compiler through `npx` or the local package will not use rbxtsgo. Untested: which of the roblox-ts extension's buttons call `rbxtsc` from `PATH`. If a button still runs the JS compiler, run `rbxtsc -w` in a terminal instead.

### From source

This is the only way that has been tested, and only on Windows.

You need Go (the version in `go.mod`), `git`, and a bash shell (Git Bash on Windows).

```sh
git clone <this repository>
cd rbxtsgo
bash scripts/setup-third-party.sh     # clones TypeScript 7.1 at a pinned commit and applies patches/typescript/
go build -o rbxtsgo ./cmd/rbxtsgo     # rbxtsgo.exe on Windows
```

Release builds use `CGO_ENABLED=0` for Windows, macOS and Linux on x86_64 and arm64. All six cross-compile; only Windows x86_64 has been run.

## Use

It takes the flags of `rbxtsc build`:

```sh
rbxtsgo -p .          # build the project in the current folder
rbxtsgo -p . -w       # watch mode
rbxtsgo --help        # all flags
```

Extra flags: `--checkers N` sets the number of type checkers, and `--setget` turns on the built-in setget transformer (see [docs/setget.md](docs/setget.md)). A project that uses any other transformer plugin is built by the JS compiler, which must be installed in the project's `node_modules`.

## Compatibility

- The reference is `roblox-ts@3.0.0`, which uses TypeScript 5.5.3. Output is compared with it byte for byte on a differential corpus that includes roblox-ts's own test sources, a React JSX corpus and generated cases. Expected output comes from running the real JS compiler, never from editing goldens by hand.
- Tsconfigs written for `rbxtsc` work unchanged. Tsconfigs written in TypeScript 7 style (`module: preserve`, `moduleResolution: bundler`) are accepted and compiled as the classic pair. See [docs/config-normalization.md](docs/config-normalization.md).
- Worker counts do not change output.
- Only Windows has been tested. Linux and macOS have not.

## Known differences

Where the output or the diagnostics differ from `rbxtsc`, and why, is listed in [docs/known-differences.md](docs/known-differences.md). The main ones are in how TypeScript 7 prints types in diagnostic messages and in declaration files, and in diagnostics positions with setget (they point at the file on disk instead of at JS's reprinted text).

## Benchmarks

No speed figures are published in this README. Any benchmark numbers that get published will come from public projects only, with the project and commit, compiler versions, machine and number of runs, and cold, warm and watch runs reported separately. A speedup is not claimed without repeated clean runs.

## Development

```sh
bash scripts/setup-third-party.sh
go build ./...
go test ./...
bash scripts/setup-diff-corpus.sh     # needs node and npm: installs roblox-ts@3.0.0 and the corpus dependencies into .scratch/
bash scripts/scoreboard.sh            # the differential scoreboard
```

Some tests skip themselves when their dependencies are not installed. [docs/](docs/README.md) has the design notes, and [testdata/differential/README.md](testdata/differential/README.md) describes the differential harness.

## License

MIT. See [LICENSE](LICENSE).

## Third-party software

rbxtsgo is a port of roblox-ts and its libraries, and is built on TypeScript. Their license texts are in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
