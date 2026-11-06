#!/usr/bin/env bash
# Purpose: generate the differential goldens in testdata/differential/golden with real JS roblox-ts@3.0.0 (TS 5.5.3).
#          Never edit a golden by hand. Rerun this only when the corpus changes.
# Run:     bash scripts/gen-diff-goldens.sh [corpus name...]
# Studio:  not needed.
# Date:    2026-11-05, differential harness for the transformer port.
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
bash "$repo_root/scripts/setup-diff-corpus.sh"
node "$repo_root/testdata/differential/gen-golden.cjs" "$repo_root/.scratch/golden-deps/node_modules" "$@"
