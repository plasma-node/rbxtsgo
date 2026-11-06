#!/usr/bin/env bash
# Purpose: set up the public differential corpus in .scratch/diff: a copy of upstream's tests project plus its
#          dependencies at the commits and versions in upstream's package-lock.json, and JS roblox-ts@3.0.0.
#          Upstream's lock file disagrees with its package.json on @rbxts/roact, so npm ci refuses it. This script
#          installs the locked versions by hand, from tarballs, so it needs no SSH key.
# Run:     bash scripts/setup-diff-corpus.sh
# Studio:  not needed.
# Date:    2026-11-05, differential harness for the transformer port.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
tests="$repo_root/.scratch/diff/tests"
upstream="$repo_root/third_party/roblox-ts/tests"

mkdir -p "$tests"
rm -rf "$tests/src"
cp -r "$upstream/src" "$tests/src"
cp "$upstream/tsconfig.json" "$upstream/default.project.json" "$upstream/runTestsWithLune.lua" "$tests/"

if [ ! -d "$tests/node_modules/@rbxts/types" ]; then
	(
		cd "$tests"
		cat >package.json <<'JSON'
{
	"name": "tests",
	"version": "1.0.0",
	"private": true,
	"dependencies": {
		"@rbxts/compiler-types": "https://codeload.github.com/roblox-ts/compiler-types/tar.gz/9fc6549e778cf7ce0db15fef920f6a6ae2dda288",
		"@rbxts/roact": "https://codeload.github.com/roblox-ts/rbx-roact/tar.gz/1ae4a5b4b01df783081cc23d8ece11541fe0b0d2",
		"@rbxts/services": "1.5.3",
		"@rbxts/testez": "0.4.2-ts.0",
		"@rbxts/types": "1.0.751"
	}
}
JSON
		npm install --ignore-scripts --no-audit --no-fund --silent
	)
fi

deps="$repo_root/.scratch/golden-deps"
if [ ! -f "$deps/node_modules/roblox-ts/package.json" ]; then
	mkdir -p "$deps"
	(
		cd "$deps"
		[ -f package.json ] || npm init -y >/dev/null
		npm install --silent --no-audit --no-fund --ignore-scripts roblox-ts@3.0.0
	)
fi
if [ ! -f "$deps/node_modules/rbxts-transformer-setget/package.json" ]; then
	(
		cd "$deps"
		npm install --silent --no-audit --no-fund --ignore-scripts rbxts-transformer-setget@2.0.0
	)
fi

# Our own fixtures: each project shares the upstream tests' dependencies through a copy of node_modules.
fixtures="$repo_root/testdata/differential/fixtures"
setup_fixture() {
	local name="$1" source="$2"
	local dest="$repo_root/.scratch/diff/$name"
	mkdir -p "$dest"
	rm -rf "$dest/src"
	cp -r "$source" "$dest/src"
	cp "$fixtures/tsconfig.json" "$fixtures/default.project.json" "$fixtures/package.json" "$dest/"
	if [ ! -d "$dest/node_modules/@rbxts/types" ]; then
		cp -r "$tests/node_modules" "$dest/node_modules"
	fi
}
setup_fixture fixtures "$fixtures/core/src"
setup_fixture whole-errors "$fixtures/whole-errors/src"
setup_fixture union-isolated "$fixtures/union-isolated/src"

# JSX with React.createElement and React.Fragment, against a local stand-in for @rbxts/react. Its tsconfig differs.
setup_fixture jsx-react "$fixtures/jsx-react/src"
cp "$fixtures/jsx-react/tsconfig.json" "$repo_root/.scratch/diff/jsx-react/tsconfig.json"

# The spike fixture has its own dependency versions. It has no Rojo project, so it gets the fixture one.
spike="$repo_root/.scratch/diff/spike-fixture"
mkdir -p "$spike"
rm -rf "$spike/src"
cp -r "$repo_root/spike/fixture/src" "$spike/src"
cp "$repo_root/spike/fixture/tsconfig.json" "$repo_root/spike/fixture/package.json" "$fixtures/default.project.json" "$spike/"
if [ ! -d "$spike/node_modules/@rbxts/types" ]; then
	cp -r "$repo_root/spike/fixture/node_modules" "$spike/node_modules"
fi

# The setget corpora. Each project gets a copy of the plugin, which is where JS looks for it. The plugin's own
# require("typescript") finds the roblox-ts one through NODE_PATH, see gen-golden.cjs, because a real install dedupes
# the two: the plugin's peer range takes the =5.5.3 that roblox-ts pins.
setget="$fixtures/setget"
add_setget_plugin() {
	rm -rf "$1/node_modules/rbxts-transformer-setget"
	cp -r "$deps/node_modules/rbxts-transformer-setget" "$1/node_modules/rbxts-transformer-setget"
}
setup_setget() {
	local name="$1" source="$2" config="$3"
	local dest="$repo_root/.scratch/diff/$name"
	mkdir -p "$dest"
	rm -rf "$dest/src"
	cp -r "$source" "$dest/src"
	cp "$setget/$config" "$dest/tsconfig.json"
	cp "$fixtures/default.project.json" "$fixtures/package.json" "$dest/"
	if [ ! -d "$dest/node_modules/@rbxts/types" ]; then
		cp -r "$tests/node_modules" "$dest/node_modules"
	fi
	add_setget_plugin "$dest"
}
setup_setget setget-core "$setget/core/src" tsconfig.json
setup_setget setget-errors "$setget/errors/src" tsconfig.json
setup_setget setget-fatal "$setget/fatal/src" tsconfig.json
setup_setget setget-prefix "$setget/prefix/src" tsconfig.prefix.json
setup_setget setget-off "$setget/off/src" tsconfig.off.json
# Existing code with the plugin on: no accessors, so this measures what the reprint alone does.
setup_setget setget-fixtures "$fixtures/core/src" tsconfig.json

# The end-to-end line test lays out its own files here, with no plugin package installed: see setget_cli_test.go.
lines="$repo_root/.scratch/diff/setget-lines"
mkdir -p "$lines"
cp "$fixtures/default.project.json" "$fixtures/package.json" "$lines/"
if [ ! -d "$lines/node_modules/@rbxts/types" ]; then
	cp -r "$tests/node_modules" "$lines/node_modules"
fi

# The whole-project comparison and the runtime suite build upstream's tests as one project, without the files in
# tests/src/diagnostics, which each fail by design.
cli="$repo_root/.scratch/diff/tests-cli"
mkdir -p "$cli"
rm -rf "$cli/src"
cp -r "$tests/src" "$cli/src"
rm -rf "$cli/src/diagnostics"
cp "$tests/tsconfig.json" "$tests/default.project.json" "$tests/runTestsWithLune.lua" "$tests/package.json" "$cli/"
if [ ! -d "$cli/node_modules/@rbxts/types" ]; then
	cp -r "$tests/node_modules" "$cli/node_modules"
fi

# TS 7 style configs. JS can't read them, so each project also has a twin with a classic config, for JS to compile.
ts7="$fixtures/ts7style"
setup_ts7_pair() {
	local name="$1" source="$2" modules="$3" ts7_config="$4"
	local classic="$repo_root/.scratch/diff/$name-classic" style="$repo_root/.scratch/diff/$name-ts7"
	for dest in "$classic" "$style"; do
		mkdir -p "$dest"
		rm -rf "$dest/src" "$dest/out"
		cp -r "$source" "$dest/src"
		cp "$fixtures/default.project.json" "$fixtures/package.json" "$dest/"
		if [ ! -d "$dest/node_modules/@rbxts/types" ]; then
			mkdir -p "$dest/node_modules"
			cp -r "$modules/@rbxts" "$dest/node_modules/@rbxts"
		fi
		# A package whose "exports" disagrees with its "main". Classic resolution ignores "exports".
		cp -r "$ts7/packages/exports-pkg" "$dest/node_modules/@rbxts/exports-pkg"
	done
	cp "$fixtures/tsconfig.json" "$classic/tsconfig.json"
	cp "$ts7_config" "$style/tsconfig.json"
}
setup_ts7_pair ts7-core "$ts7/core/src" "$tests/node_modules" "$ts7/tsconfig.json"
setup_ts7_pair ts7-errors "$ts7/errors/src" "$tests/node_modules" "$ts7/tsconfig.json"

# The public rbxts-rs template at a pinned commit. Nothing in its compiler/ folder gets run.
template="$repo_root/.scratch/rbxts-rs-template"
template_commit=e446ad9
if [ ! -d "$template/.git" ]; then
	git clone --quiet https://github.com/rbxts-rs/template "$template"
fi
git -C "$template" checkout --quiet "$template_commit"
if [ ! -d "$template/node_modules/@rbxts/types" ]; then
	(cd "$template" && npm install --ignore-scripts --no-audit --no-fund --silent)
fi
setup_ts7_pair template "$template/src" "$template/node_modules" "$template/tsconfig.json"
cp "$template/default.project.json" "$repo_root/.scratch/diff/template-classic/"
cp "$template/default.project.json" "$repo_root/.scratch/diff/template-ts7/"

# Copies of the upstream tests and the spike fixture with the plugin on.
with_plugin() {
	local name="$1" source="$2"
	local dest="$repo_root/.scratch/diff/$name"
	mkdir -p "$dest"
	rm -rf "$dest/src"
	cp -r "$source/src" "$dest/src"
	# An old tsbuildinfo in out would tell JS that nothing changed.
	rm -rf "$dest/out"
	cp "$source/default.project.json" "$source/package.json" "$dest/"
	sed 's/"compilerOptions": {/"compilerOptions": {\n\t\t"plugins": [{ "transform": "rbxts-transformer-setget" }],/' "$source/tsconfig.json" >"$dest/tsconfig.json"
	if [ ! -d "$dest/node_modules/@rbxts/types" ]; then
		cp -r "$source/node_modules" "$dest/node_modules"
	fi
	add_setget_plugin "$dest"
}
with_plugin setget-upstream "$cli"
with_plugin setget-spike "$spike"
