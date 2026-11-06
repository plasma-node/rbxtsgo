#!/usr/bin/env bash
# Purpose: ecosystem validation. Builds real @rbxts/* projects with JS rbxtsc 3.0.0 and with rbxtsgo, then compares the
#          out and include folders byte for byte and the printed diagnostics with colors removed.
# Run:     bash scripts/validate-ecosystem.sh fixture <name>...   our fixtures in testdata/ecosystem, installed in .scratch/eco/fx
#          bash scripts/validate-ecosystem.sh fixtures            every fixture
#          bash scripts/validate-ecosystem.sh project <dir> [flags]   an installed project, such as a clone in .scratch/eco/clones
#          bash scripts/validate-ecosystem.sh projects             the pinned clones in testdata/ecosystem/projects.txt
#          bash scripts/validate-ecosystem.sh clone <name> <url> [commit] [flags]   plain depth-1 clone, read-only, no scripts, then compare
#          Setup: bash scripts/setup-diff-corpus.sh installs the JS compiler into .scratch/golden-deps.
#          Safety: npm install --ignore-scripts only. No project script or downloaded binary runs. A project whose
#          tsconfig has a transformer plugin other than setget or Flamework is skipped.
# Studio:  not needed.
# Date:    2026-11-05, ecosystem validation.
set -uo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"
command -v go >/dev/null 2>&1 || {
	echo "go not found on PATH. Install Go (the version in go.mod) and make sure 'go' runs in this shell." >&2
	exit 1
}
exe=""
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) exe=".exe" ;; esac
js_cli="$repo_root/.scratch/golden-deps/node_modules/roblox-ts/out/CLI/cli.js"
eco="$repo_root/.scratch/eco"
store="$eco/store"
bin="$eco/rbxtsgo$exe"
mkdir -p "$eco/fx" "$eco/clones" "$store"

if [ ! -f "$js_cli" ]; then
	echo "missing JS compiler: run bash scripts/setup-diff-corpus.sh" >&2
	exit 1
fi
if [ -z "${SKIP_BUILD:-}" ]; then
	go build -o "$bin" ./cmd/rbxtsgo || exit 1
fi

strip_ansi() { sed -E 's/\x1b\[[0-9;]*[A-Za-z]//g' | tr -d '\r'; }
now_ms() { echo $(($(date +%s%N) / 1000000)); }

# compare_project <dir> [flags...]: dir must be installed. Prints one summary line.
compare_project() {
	local dir="$1"
	shift
	local name
	name="$(basename "$dir")"
	local out="$store/$name"
	rm -rf "$out" 2>/dev/null
	rm -rf "$out/js" "$out/go" "$out/files.txt" 2>/dev/null
	mkdir -p "$out"
	local t0 t1 t1b t2 jsc goc
	rm -rf "$dir/out" "$dir/include"
	t0=$(now_ms)
	# A project with its own roblox-ts (Flamework needs the transformer to see the same TypeScript) is built with that.
	local cli="$js_cli"
	[ -f "$dir/node_modules/roblox-ts/out/CLI/cli.js" ] && cli="$dir/node_modules/roblox-ts/out/CLI/cli.js"
	(cd "$dir" && node "$cli" -p . "$@") >"$out/js.raw" 2>&1
	jsc=$?
	t1=$(now_ms)
	mkdir -p "$out/js"
	cp -r "$dir/out" "$out/js/out" 2>/dev/null
	cp -r "$dir/include" "$out/js/include" 2>/dev/null
	rm -rf "$dir/out" "$dir/include"
	t1b=$(now_ms)
	(cd "$dir" && "$bin" -p . "$@") >"$out/go.raw" 2>&1
	goc=$?
	t2=$(now_ms)
	mkdir -p "$out/go"
	cp -r "$dir/out" "$out/go/out" 2>/dev/null
	cp -r "$dir/include" "$out/go/include" 2>/dev/null
	strip_ansi <"$out/js.raw" >"$out/js.log"
	strip_ansi <"$out/go.raw" >"$out/go.log"
	# JS prints a progress line with its own timing; ignore nothing else.
	local total same differ only dts_total dts_same
	total=$(cd "$out/js" && find . -type f ! -name '*.tsbuildinfo' | wc -l)
	same=0
	differ=0
	dts_total=0
	dts_same=0
	while IFS= read -r f; do
		case "$f" in *.d.ts) dts_total=$((dts_total + 1)) ;; esac
		if [ -f "$out/go/$f" ] && cmp -s "$out/js/$f" "$out/go/$f"; then
			same=$((same + 1))
			case "$f" in *.d.ts) dts_same=$((dts_same + 1)) ;; esac
		else
			differ=$((differ + 1))
			echo "  DIFF $f" >>"$out/files.txt"
		fi
	done < <(cd "$out/js" && find . -type f ! -name '*.tsbuildinfo' | sort)
	only=$(cd "$out/go" && find . -type f ! -name '*.tsbuildinfo' | while read -r f; do [ -f "$out/js/$f" ] || echo "  ONLY-GO $f"; done)
	[ -n "$only" ] && echo "$only" >>"$out/files.txt"
	# Related locations are printed differently by TS 7 (accepted, see docs/known-differences.md): drop those blocks
	# from both logs for the second comparison.
	for side in js go; do
		awk '/^  [^ ].*:[0-9]+:[0-9]+/ { skip = 1 } /^$/ { skip = 0 } !skip { print }' "$out/$side.log" | grep -v -E '^(Built in |No changes, finished in )' | grep -v "^rbxtsgo can.t run the transformer plugin" >"$out/$side.norm"
	done
	local diag="match"
	if ! diff -q "$out/js.log" "$out/go.log" >/dev/null; then
		diag="DIFFER"
		diff -q "$out/js.norm" "$out/go.norm" >/dev/null && diag="match-modulo-known-format"
	fi
	echo "$name: files=$total identical=$same differ=$differ (of which .d.ts: $dts_same/$dts_total identical) exit js=$jsc go=$goc diagnostics=$diag js_ms=$((t1 - t0)) go_ms=$((t2 - t1b))"
	[ -f "$out/files.txt" ] && head -n 10 "$out/files.txt"
	[ "$diag" = DIFFER ] && echo "  see diff of $out/js.log $out/go.log"
	return 0
}

install_fixture() {
	local name="$1" src="$repo_root/testdata/ecosystem/$1"
	local dest="$eco/fx/$name"
	mkdir -p "$dest"
	rm -rf "$dest/src"
	if [ -f "$src/src-from" ]; then from="$(tr -d "[:space:]" <"$src/src-from")"; cp -r "$repo_root/testdata/ecosystem/$from/src" "$dest/src"; else cp -r "$src/src" "$dest/src"; fi
	if ! cmp -s "$src/package.json" "$dest/package.json"; then rm -rf "$dest/node_modules" "$dest/package-lock.json"; fi
	cp "$src/package.json" "$dest/"
	rm -f "$dest/src/extra.d.ts"; [ -f "$src/extra.d.ts" ] && cp "$src/extra.d.ts" "$dest/src/extra.d.ts"
	if [ -f "$src/default.project.json" ]; then cp "$src/default.project.json" "$dest/"; else cp "$repo_root/testdata/ecosystem/_shared/default.project.json" "$dest/"; fi
	if [ -f "$src/tsconfig.json" ]; then cp "$src/tsconfig.json" "$dest/"; else cp "$repo_root/testdata/ecosystem/_shared/tsconfig.json" "$dest/"; fi
	(cd "$dest" && npm install --ignore-scripts --no-audit --no-fund --silent) || echo "npm install failed for $name" >&2
}

cmd="${1:-}"
shift || true
case "$cmd" in
fixture)
	for n in "$@"; do
		install_fixture "$n"
		compare_project "$eco/fx/$n"
	done
	;;
fixtures)
	for d in testdata/ecosystem/*/; do
		n="$(basename "$d")"
		[ "$n" = _shared ] && continue
		install_fixture "$n"
		compare_project "$eco/fx/$n"
	done
	;;
project)
	dir="$1"
	shift
	compare_project "$(cd "$dir" && pwd)" "$@"
	;;
clone)
	name="$1" url="$2" commit="${3:-}"
	dest="$eco/clones/$name"
	if [ ! -d "$dest/.git" ]; then
		git clone --quiet --depth 1 "$url" "$dest" || exit 1
	fi
	if [ -n "$commit" ] && [ "$(git -C "$dest" rev-parse HEAD)" != "$commit" ]; then
		git -C "$dest" fetch --quiet --depth 1 origin "$commit" && git -C "$dest" checkout --quiet FETCH_HEAD || exit 1
	fi
	git -C "$dest" remote remove origin 2>/dev/null
	echo "$name at $(git -C "$dest" rev-parse HEAD)"
	# Only Flamework (and setget) transformer plugins are allowed, because a plugin runs code on this machine.
	prep="$(node scripts/eco-prep-clone.cjs "$dest")" || exit 1
	echo "  prep: $prep"
	bad="$(echo "$prep" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{const p=JSON.parse(s).plugins.filter(x=>x!=="rbxts-transformer-flamework"&&x!=="rbxts-transformer-setget");console.log(p.join(","))})')"
	if [ -n "$bad" ]; then
		echo "$name: SKIPPED, tsconfig has other transformer plugins: $bad"
		exit 3
	fi
	(cd "$dest" && npm install --ignore-scripts --legacy-peer-deps --no-audit --no-fund --silent) || { echo "$name: SKIPPED, npm install failed"; exit 4; }
	compare_project "$dest" "${@:4}"
	;;
projects)
	# testdata/ecosystem/projects.txt: name | url | commit | flags
	while IFS='|' read -r pname purl pcommit pflags; do
		pname="$(echo "$pname" | tr -d '[:space:]')"
		case "$pname" in '' | '#'*) continue ;; esac
		# shellcheck disable=SC2086
		SKIP_BUILD=1 bash "$0" clone "$pname" "$(echo "$purl" | tr -d '[:space:]')" "$(echo "$pcommit" | tr -d '[:space:]')" $pflags
	done <testdata/ecosystem/projects.txt
	;;
*)
	sed -n 2,12p "$0"
	exit 2
	;;
esac
