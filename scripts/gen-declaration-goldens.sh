#!/usr/bin/env bash
# Purpose: build testdata/declarations/fixture with real JS roblox-ts@3.0.0 (TS 5.5.3) and keep its output tree in
#          testdata/declarations/golden, for compile's TestDeclarationGolden. Never edit the golden by hand.
# Run:     bash scripts/gen-declaration-goldens.sh, after scripts/setup-diff-corpus.sh.
# Studio:  not needed.
# Date:    2026-11-05, declaration emit.
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
fixture="$repo_root/testdata/declarations/fixture"
work="$repo_root/.scratch/decl-js"
rm -rf "$work"
mkdir -p "$work/node_modules"
cp -r "$fixture/." "$work/"
cp -r "$repo_root/.scratch/diff/tests/node_modules/@rbxts" "$work/node_modules/@rbxts"
(cd "$work" && NO_COLOR=1 node "$repo_root/.scratch/golden-deps/node_modules/roblox-ts/out/CLI/cli.js")
golden="$repo_root/testdata/declarations/golden"
rm -rf "$golden"
cp -r "$work/out" "$golden"
find "$golden" -type f | wc -l
