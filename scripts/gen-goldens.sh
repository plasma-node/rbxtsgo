#!/usr/bin/env bash
# Purpose: regenerate every differential golden from the real npm packages. Public fixtures only.
#          Installs the exact versions into .scratch/golden-deps, which is gitignored. Goldens that need a
#          real filesystem are per-platform, so run this on Windows and in WSL/Linux to refresh both.
# Run:     bash scripts/gen-goldens.sh
# Studio:  not needed.
# Date:    2026-11-05, foundation work for the port.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
deps="$repo_root/.scratch/golden-deps"
mkdir -p "$deps"
if [ ! -f "$deps/node_modules/roblox-ts/package.json" ]; then
	(
		cd "$deps"
		[ -f package.json ] || npm init -y >/dev/null
		npm install --silent --no-audit --no-fund --ignore-scripts roblox-ts@3.0.0
	)
fi
modules="$deps/node_modules"

# path-translator, node path and bridge goldens are the same on any platform, so generate them on Windows only.
case "$(uname -s)" in
MINGW* | MSYS* | CYGWIN*)
	node "$repo_root/project/nodepath/testdata/gen-golden.cjs" "$repo_root/project/nodepath/testdata/golden.json"
	node "$repo_root/project/pathtranslator/testdata/gen-golden.cjs" "$modules" "$repo_root/project/pathtranslator/testdata/golden.json"
	node "$repo_root/tsbridge/testdata/gen-golden.cjs" "$modules/typescript"
	# Needs npm ci in spike/fixture.
	node "$repo_root/transformer/testdata/gen-probe.cjs" "$modules/typescript"
	;;
esac

node "$repo_root/project/rojoresolver/testdata/gen-golden.cjs" "$modules"
node "$repo_root/project/shared/testdata/gen-golden.cjs" "$modules"
node "$repo_root/project/testdata/gen-golden.cjs" "$modules"
