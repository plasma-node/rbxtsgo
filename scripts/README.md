# scripts/

Developer tooling run from a terminal. The scripts are bash (Git Bash on Windows). They have only been run on Windows. They need `go`, `node` and `git` on `PATH`.
Each script opens with a header: purpose, exact run command, whether it needs Studio, date/request.

| Script | Does |
|---|---|
| `gen-goldens.sh` | Regenerates every differential golden from the real npm packages, into the testdata folders |
| `setup-diff-corpus.sh` | Sets up the differential corpus in `.scratch/diff`. See [the harness README](../testdata/differential/README.md) |
| `gen-diff-goldens.sh [corpus...]` | Regenerates the differential goldens with JS `roblox-ts@3.0.0` |
| `scoreboard.sh [-v] [--cli] [--runtime]` | The scoreboard. `--cli` also compares whole-project builds and worker counts, and `--runtime` runs the Lune suite |
| `scoreboard-plugins.sh` | Setget through both CLIs at 1 and 8 workers, each way to turn it on, the plugin's exits, and the fallback to the JS compiler. `scoreboard-cli.sh` runs it |
| `bench-watch.sh <project> <file> [N]` | Save-to-output latency and memory of watch mode, on any project. Restores the file, and `--generate` writes a synthetic project |
| `bench-incremental.sh <project> [runs] [file...]` | Full, no-change and per-file edit times of incremental builds |
| `gen-watch-golden.sh` | Records a session of JS `rbxtsc -w` for the watch tests |
| `gen-declaration-goldens.sh` | Builds the declaration fixture with JS `rbxtsc` |
| `setup-third-party.sh [dir]` | Reproduces `third_party/` from the pins and applies `patches/typescript/`. Resets the clones |

## `local/` (gitignored)
