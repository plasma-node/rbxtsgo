#!/usr/bin/env bash
# Purpose: whole-project comparison. Builds upstream's runtime specs (without tests/src/diagnostics, which can't build as
#          one project) with the real JS CLI and with the rbxtsgo binary at 1 and 8 workers, and diffs the out and include
#          folders. Also checks --writeOnlyChanged rewrites nothing on a second run, and prints rough timings. Those are
#          uncontrolled: one run each, on a machine doing other things.
# Run:     bash scripts/scoreboard-cli.sh. It uses .scratch/bin/rbxtsgo (rbxtsgo.exe on Windows), so run scripts/scoreboard.sh first.
# Studio:  not needed.
# Date:    2026-11-05, differential harness for the transformer port.
set -uo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"
dir=.scratch/diff/tests-cli
if [ ! -d "$dir/node_modules" ]; then
	echo "$dir is missing. Run: bash scripts/setup-diff-corpus.sh" >&2
	exit 1
fi
js_cli=.scratch/golden-deps/node_modules/roblox-ts/out/CLI/cli.js
exe=""
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) exe=".exe" ;; esac
bin="$PWD/.scratch/bin/rbxtsgo$exe"
out_store=.scratch/diff/cli-out
rm -rf "$out_store"
mkdir -p "$out_store"

timed() {
	local started ended
	started=$(date +%s%N)
	"$@"
	local code=$?
	ended=$(date +%s%N)
	echo "  exit $code, $(((ended - started) / 1000000)) ms"
	return 0
}
capture() {
	rm -rf "$out_store/$1"
	mkdir -p "$out_store/$1"
	cp -r "$dir/out" "$out_store/$1/out" 2>/dev/null
	cp -r "$dir/include" "$out_store/$1/include" 2>/dev/null
}

rm -rf "$dir/out" "$dir/include"
echo "JS rbxtsc:"
timed node "$js_cli" -p "$dir" --allowCommentDirectives >"$out_store/js.log" 2>&1
tail -n 3 "$out_store/js.log"
capture js

for workers in 1 8; do
	rm -rf "$dir/out" "$dir/include"
	echo "rbxtsgo, $workers workers:"
	timed "$bin" -p "$dir" --allowCommentDirectives --checkers "$workers" >"$out_store/go$workers.log" 2>&1
	tail -n 3 "$out_store/go$workers.log"
	capture "go$workers"
done

status=0
for workers in 1 8; do
	if diff -rq "$out_store/js" "$out_store/go$workers" >"$out_store/diff$workers.txt"; then
		echo "JS and rbxtsgo at $workers workers: out and include are byte-identical ($(find "$out_store/js" -type f | wc -l) files)"
	else
		status=1
		echo "JS and rbxtsgo at $workers workers differ: $(wc -l <"$out_store/diff$workers.txt") paths. See $out_store/diff$workers.txt"
	fi
done
if diff -rq "$out_store/go1" "$out_store/go8" >/dev/null; then
	echo "1 worker and 8 workers: byte-identical"
else
	status=1
	echo "1 worker and 8 workers differ"
fi

# write-only-changed: a second run must leave every timestamp alone.
stamp="$out_store/stamp"
touch "$stamp"
sleep 1
"$bin" -p "$dir" --allowCommentDirectives --writeOnlyChanged >/dev/null 2>&1
changed=$(find "$dir/out" -name '*.luau' -newer "$stamp" | wc -l)
echo "--writeOnlyChanged on an unchanged project rewrote $changed .luau files"
bash scripts/scoreboard-plugins.sh || status=1
exit $status
