#!/usr/bin/env bash
# Purpose: runs upstream's runtime suite (TestEZ specs under Lune, on a Rojo-built place) on rbxtsgo's output, in
#          .scratch/diff/tests-cli. JS rbxtsc's output passes 460 of 460 on this machine with these tools.
# Run:     bash scripts/scoreboard-runtime.sh, after scripts/scoreboard-cli.sh, which leaves rbxtsgo's output there.
# Studio:  not needed.
# Date:    2026-11-05, differential harness for the transformer port.
#
# The aftman shim for lune refuses to run outside a project that lists it, so this calls the stored binary directly.
set -uo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root/.scratch/diff/tests-cli" || exit 1
export PATH="$PATH:$HOME/.aftman/bin"
lune="$(ls "$HOME"/.aftman/tool-storage/lune-org/lune/*/lune* | tail -n 1)"
# default.project.json lists out/diagnostics, which this copy has no sources for.
mkdir -p out/diagnostics
rm -f test.rbxl
rojo build . -o test.rbxl || exit 1
"$lune" run runTestsWithLune.lua test.rbxl 2>&1 | tail -n 25
