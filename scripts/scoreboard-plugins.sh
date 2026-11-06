#!/usr/bin/env bash
# Purpose: whole-project plugin checks with the real binaries. rbxts-transformer-setget through the JS CLI and through
#          rbxtsgo at 1 and 8 workers, the plugin's own exits, then the fallback to the JS compiler for another plugin.
# Run:     bash scripts/scoreboard-plugins.sh. It uses .scratch/bin/rbxtsgo (rbxtsgo.exe on Windows), so run scripts/scoreboard.sh first.
# Studio:  not needed.
# Date:    2026-11-05, setget and the plugin fallback.
set -uo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"
deps="$PWD/.scratch/golden-deps/node_modules"
js_cli="$deps/roblox-ts/out/CLI/cli.js"
exe=""
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) exe=".exe" ;; esac
bin="$PWD/.scratch/bin/rbxtsgo$exe"
store="$PWD/.scratch/diff/plugin-out"
rm -rf "$store"
mkdir -p "$store"
# The plugin copy in each project finds typescript here, see setup-diff-corpus.sh.
export NODE_PATH="$deps"
status=0

check() {
	if [ "$1" = "$2" ]; then
		echo "  ok: $3"
	else
		status=1
		echo "  FAIL: $3: '$1' and '$2'"
	fi
}

for project in setget-core setget-upstream; do
	dir=".scratch/diff/$project"
	flags=""
	[ "$project" = setget-upstream ] && flags="--allowCommentDirectives"
	echo "$project:"
	rm -rf "$dir/out" "$dir/include"
	node "$js_cli" -p "$dir" $flags --writeTransformedFiles >"$store/$project-js.log" 2>&1
	code_js=$?
	cp -r "$dir/out" "$store/$project-js"
	for workers in 1 8; do
		rm -rf "$dir/out" "$dir/include"
		"$bin" -p "$dir" $flags --writeTransformedFiles --checkers "$workers" >"$store/$project-go$workers.log" 2>&1
		code_go=$?
		check "$code_js" "$code_go" "exit code, $workers workers"
		cp -r "$dir/out" "$store/$project-go$workers"
		if diff -r "$store/$project-js" "$store/$project-go$workers" >"$store/$project-diff$workers.txt"; then
			echo "  ok: out is byte-identical at $workers workers, with the reprints ($(find "$dir/out" -type f | wc -l) files)"
		else
			status=1
			echo "  FAIL: out differs at $workers workers. See $store/$project-diff$workers.txt"
		fi
	done
done

# The built-in option, with no package anywhere: each way of turning it on must give what JS with the plugin gave.
echo "built-in setget:"
builtin="$PWD/.scratch/diff/setget-builtin"
rm -rf "$builtin/src" "$builtin/out" "$builtin/include"
mkdir -p "$builtin"
cp -r testdata/differential/fixtures/setget/core/src "$builtin/src"
cp testdata/differential/fixtures/default.project.json testdata/differential/fixtures/package.json "$builtin/"
if [ ! -d "$builtin/node_modules/@rbxts/types" ]; then
	cp -r .scratch/diff/tests/node_modules "$builtin/node_modules"
fi
plain=testdata/differential/fixtures/tsconfig.json
build_builtin() {
	local label="$1"
	shift
	rm -rf "$builtin/out" "$builtin/include"
	"$bin" -p "$builtin" "$@" >"$store/builtin-$label.log" 2>&1
	check 0 $? "exit code, $label"
	# The JS run above wrote reprints too.
	find "$builtin/out" -name '*.transformed.*' -delete
	if diff -r -x '*.transformed.*' "$store/setget-core-js" "$builtin/out" >"$store/builtin-$label.diff"; then
		echo "  ok: $label is byte-identical to JS with the plugin"
	else
		status=1
		echo "  FAIL: $label differs. See $store/builtin-$label.diff"
	fi
}
cp "$plain" "$builtin/tsconfig.json"
build_builtin flag --setget
sed 's/^{/{ "rbxtsgo": { "setget": true },/' "$plain" >"$builtin/tsconfig.json"
build_builtin block
cp testdata/differential/fixtures/setget/tsconfig.json "$builtin/tsconfig.json"
build_builtin entry-without-package
# TS and the JS compiler must not mind the block.
sed 's/^{/{ "rbxtsgo": { "setget": true },/' "$plain" >"$builtin/tsconfig.json"
node "$deps/typescript/lib/tsc.js" --showConfig -p "$builtin" >"$store/tsc-showconfig.log" 2>&1
check 0 $? "TS 5.5.3 reads a tsconfig with the block"
rm -rf "$builtin/src"
mkdir -p "$builtin/src"
printf 'print(1);\n' >"$builtin/src/a.ts"
node "$js_cli" -p "$builtin" >"$store/js-block.log" 2>&1
check 0 $? "JS rbxtsc builds with the block"

# The plugin exits on the first file in order that has a problem, with no diagnostics.
echo "setget-fatal:"
dir=.scratch/diff/setget-fatal
node "$js_cli" -p "$dir" >"$store/fatal-js.log" 2>&1
code_js=$?
for workers in 1 8; do
	"$bin" -p "$dir" --checkers "$workers" >"$store/fatal-go$workers.log" 2>&1
	code_go=$?
	check "$code_js" "$code_go" "exit code, $workers workers"
	check "$(cat "$store/fatal-js.log")" "$(cat "$store/fatal-go$workers.log")" "plugin output, $workers workers"
done

# Another plugin: a local identity transformer, and the real JS compiler copied into the project.
echo "fallback:"
fallback="$PWD/.scratch/diff/plugin-fallback"
rm -rf "$fallback/src" "$fallback/out" "$fallback/include"
mkdir -p "$fallback/src"
cp testdata/differential/fixtures/default.project.json testdata/differential/fixtures/package.json "$fallback/"
if [ ! -d "$fallback/node_modules/@rbxts/types" ]; then
	cp -r .scratch/diff/tests/node_modules "$fallback/node_modules"
fi
sed 's#"compilerOptions": {#"compilerOptions": { "plugins": [{ "transform": "./other-plugin.js" }],#' testdata/differential/fixtures/tsconfig.json >"$fallback/tsconfig.json"
cat >"$fallback/other-plugin.js" <<'EOF'
const ts = require("typescript");
module.exports = () => context => file => ts.visitEachChild(file, node => node, context);
EOF
printf '/*\n *      kept through the reprint\n */\nprint(1_000);\n' >"$fallback/src/a.ts"

rm -rf "$fallback/node_modules/roblox-ts"
"$bin" -p "$fallback" --checkers 2 >"$store/missing.log" 2>&1
check 1 $? "exit code without the JS compiler"
grep -q "npm install" "$store/missing.log" && grep -q "other-plugin" "$store/missing.log"
check 0 $? "the error names the plugin and what to do"

cp -r "$deps/roblox-ts" "$fallback/node_modules/roblox-ts"
node "$js_cli" -p "$fallback" >"$store/direct.log" 2>&1
code_direct=$?
cp -r "$fallback/out" "$store/fallback-direct"
rm -rf "$fallback/out" "$fallback/include"
"$bin" -p "$fallback" --checkers 2 >"$store/delegated.log" 2>"$store/delegated.err"
check "$code_direct" $? "exit code, delegated and direct"
grep -q "other-plugin" "$store/delegated.err"
check 0 $? "the message names the plugin"
diff -r "$store/fallback-direct" "$fallback/out" >/dev/null
check 0 $? "delegated output matches the JS compiler"
"$bin" -p "$fallback" --verbose >"$store/verbose.log" 2>/dev/null
grep -q "compiling as" "$store/verbose.log"
check 0 $? "flags reach the JS compiler"
printf 'const wrong: string = 1;\n' >"$fallback/src/a.ts"
"$bin" -p "$fallback" >"$store/failed.log" 2>/dev/null
check 1 $? "the JS compiler's exit code comes back"
exit $status
