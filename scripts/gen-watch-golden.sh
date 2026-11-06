#!/usr/bin/env bash
# Purpose: record a session of real JS rbxtsc -w into watch/testdata/js-session.txt, for TestSessionMatchesUpstream.
#          Never edit that file by hand. It runs the same edits as the test, with waits long enough for chokidar.
# Run:     bash scripts/gen-watch-golden.sh, after scripts/setup-diff-corpus.sh.
# Studio:  not needed.
# Date:    2026-11-05, watch mode.
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
root="$repo_root/.scratch/watchtest"
cli="$repo_root/.scratch/golden-deps/node_modules/roblox-ts/out/CLI/cli.js"
modules="$repo_root/.scratch/diff/tests/node_modules/@rbxts"
if [ ! -d "$root/node_modules/@rbxts/types" ]; then
	mkdir -p "$root/node_modules"
	cp -r "$modules" "$root/node_modules/@rbxts"
fi
live="$root/js-session/live"
rm -rf "$root/js-session"
mkdir -p "$live/src/shared" "$live/src/server" "$live/node_modules"
cp -r "$root/node_modules/@rbxts" "$live/node_modules/@rbxts"
echo '{ "name": "watchtest", "version": "0.0.0", "private": true }' >"$live/package.json"
cat >"$live/tsconfig.json" <<'JSON'
{
	"compilerOptions": {
		"allowSyntheticDefaultImports": true,
		"downlevelIteration": true,
		"jsx": "react",
		"jsxFactory": "Roact.jsx",
		"module": "commonjs",
		"moduleResolution": "Node",
		"noLib": true,
		"resolveJsonModule": true,
		"forceConsistentCasingInFileNames": true,
		"moduleDetection": "force",
		"strict": true,
		"target": "ESNext",
		"typeRoots": ["node_modules/@rbxts"],
		"experimentalDecorators": true,
		"rootDir": "src",
		"outDir": "out",
		"baseUrl": "src"
	},
	"include": ["src/**/*"]
}
JSON
cat >"$live/default.project.json" <<'JSON'
{
	"name": "watchtest",
	"tree": {
		"$className": "DataModel",
		"ReplicatedStorage": {
			"$className": "ReplicatedStorage",
			"rbxts_include": {
				"$path": "include",
				"node_modules": { "$className": "Folder", "@rbxts": { "$path": "node_modules/@rbxts" } }
			},
			"TS": { "$path": "out/shared" }
		},
		"ServerScriptService": {
			"$className": "ServerScriptService",
			"TS": { "$path": "out/server" }
		}
	}
}
JSON
printf 'export const a = 1;\n' >"$live/src/shared/a.ts"
printf 'import { a } from "shared/a";\nprint(a);\n' >"$live/src/server/m.ts"
printf 'hello\n' >"$live/src/shared/data.txt"

cd "$live"
log="$repo_root/watch/testdata/js-session.txt"
mkdir -p "$(dirname "$log")"
node "$cli" -w >"$log" 2>&1 &
watcher=$!
stop() {
	# Git Bash starts node directly here, so this PID is the watcher.
	kill "$watcher" 2>/dev/null || true
	wait "$watcher" 2>/dev/null || true
}
trap stop EXIT
sleep 8
printf 'export const a = 2;\n' >src/shared/a.ts
sleep 3
printf 'export const a: string = 2;\n' >src/shared/a.ts
sleep 3
printf 'export const a = 3;\n' >src/shared/a.ts
sleep 3
printf 'bye\n' >src/shared/data.txt
sleep 3
printf 'export const b = 3;\n' >src/shared/b.ts
sleep 3
rm src/shared/b.ts
sleep 3
rm src/shared/data.txt
sleep 3
echo "wrote $log"
