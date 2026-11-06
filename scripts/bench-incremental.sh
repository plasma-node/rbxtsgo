#!/usr/bin/env bash
# Purpose: time incremental builds on any project with "incremental": true: a clean build, a build with no changes, and
#          an edit of each file given. Prints total time and the --verbose phases for each, as medians. Each file is put
#          back afterwards. The numbers are uncontrolled.
# Run:     bash scripts/bench-incremental.sh <project folder> [runs, default 7] [file to edit...]
#          The project needs "incremental": true in its tsconfig. Each edit appends a print statement.
# Studio:  not needed.
# Date:    2026-11-05, incremental builds.
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
go_bin="$(command -v go || true)"
if [ -z "$go_bin" ]; then
	echo "go not found on PATH. Install Go (the version in go.mod) and make sure 'go' runs in this shell." >&2
	exit 1
fi
exe=""
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) exe=".exe" ;; esac
if [ $# -lt 1 ]; then
	sed -n '2,6p' "$0"
	exit 2
fi
project="$(cd "$1" && pwd)"
runs="${2:-7}"
shift
if [ $# -gt 0 ]; then
	shift
fi
bin_dir="$repo_root/.scratch/bench-watch"
mkdir -p "$bin_dir"
(cd "$repo_root" && "$go_bin" build -o "$bin_dir/rbxtsgo$exe" ./cmd/rbxtsgo)
bin="$bin_dir/rbxtsgo$exe"

# The median of the numbers on stdin.
median() {
	sort -n | awk '{ value[NR] = $1 } END { print value[int((NR + 1) / 2)] }'
}
timed_build() {
	local begin end
	begin=$(date +%s%N)
	"$bin" -p "$project" --verbose >"$bin_dir/last.log" 2>&1 || {
		cat "$bin_dir/last.log"
		echo "the build failed" >&2
		exit 1
	}
	end=$(date +%s%N)
	echo $(((end - begin) / 1000000))
}
report() {
	local label="$1" times="$2" timing
	timing="$(grep '^\[timing\]' "$bin_dir/last.log" | tail -1 | sed 's/^\[timing\] build //')"
	printf '%-28s median %s ms (last run: %s)\n' "$label" "$(echo "$times" | tr ' ' '\n' | median)" "$timing"
}

find_state() {
	find "$project" -name '*.tsbuildinfo' -not -path '*/node_modules/*'
}

current=""
backup=""
restore() {
	if [ -n "$current" ]; then
		cp "$backup" "$current"
		rm -f "$backup"
		current=""
	fi
}
trap restore EXIT

list=""
for _ in $(seq "$runs"); do
	find_state | while read -r state; do rm -f "$state"; done
	list="$list $(timed_build)"
done
report "full (no state file)" "$list"
timed_build >/dev/null
list=""
for _ in $(seq "$runs"); do
	list="$list $(timed_build)"
done
report "no change" "$list"
for file in "$@"; do
	backup="$bin_dir/backup.$$"
	cp "$file" "$backup"
	current="$file"
	list=""
	for i in $(seq "$runs"); do
		printf '\nprint("__rbxtsgo_bench_%s");\n' "$i" >>"$file"
		list="$list $(timed_build)"
	done
	report "edit $(basename "$file")" "$list"
	restore
	timed_build >/dev/null
done
