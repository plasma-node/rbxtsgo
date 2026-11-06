#!/usr/bin/env bash
# Purpose: measure watch mode's save-to-output latency on any project: starts the watcher, edits a file N times, and
#          reports the median, min and max, with phase medians. The file is restored afterwards, and the watcher is
#          killed. Each edit appends a print statement, so the output has to change. The numbers are uncontrolled.
# Run:     bash scripts/bench-watch.sh <project folder or tsconfig> <file to edit> [N, default 20] [benchwatch flags]
#          bash scripts/bench-watch.sh --generate <new folder> <modules> [setget]   writes a synthetic public project. Run
#          scripts/setup-diff-corpus.sh before this, because it takes node_modules from there.
# Studio:  not needed.
# Date:    2026-11-05, watch mode.
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
go_bin="$(command -v go || true)"
if [ -z "$go_bin" ]; then
	echo "go not found on PATH. Install Go (the version in go.mod) and make sure 'go' runs in this shell." >&2
	exit 1
fi
exe=""
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) exe=".exe" ;; esac
bin_dir="$repo_root/.scratch/bench-watch"
mkdir -p "$bin_dir"

if [ "${1:-}" = "--generate" ]; then
	dest="$2"
	modules="$3"
	variant="${4:-}"
	mkdir -p "$dest"
	dest="$(cd "$dest" && pwd)"
	(cd "$repo_root" && "$go_bin" run ./cmd/genbench "$dest" "$modules" $variant)
	if [ ! -d "$dest/node_modules/@rbxts/types" ]; then
		mkdir -p "$dest/node_modules"
		cp -r "$repo_root/.scratch/diff/tests-cli/node_modules/@rbxts" "$dest/node_modules/@rbxts"
	fi
	echo "generated $modules modules in $dest"
	exit 0
fi

if [ $# -lt 2 ]; then
	sed -n '2,8p' "$0"
	exit 2
fi
project="$1"
file="$2"
count="${3:-20}"
shift 2
if [ $# -gt 0 ]; then
	shift
fi

backup="$bin_dir/backup.$$"
cp "$file" "$backup"
restore() {
	if ! cmp -s "$file" "$backup"; then
		cp "$backup" "$file"
		echo "restored $file"
	fi
	rm -f "$backup"
}
trap restore EXIT

(cd "$repo_root" && "$go_bin" build -o "$bin_dir/rbxtsgo$exe" ./cmd/rbxtsgo && "$go_bin" build -o "$bin_dir/benchwatch$exe" ./cmd/benchwatch)
"$bin_dir/benchwatch$exe" -bin "$bin_dir/rbxtsgo$exe" -project "$project" -file "$file" -n "$count" "$@"
