#!/usr/bin/env bash
# Purpose: reproduce third_party/ from the pins below, then apply patches/typescript/.
#          Plain clones only, never forks. Push URLs are disabled. Safe to rerun: it resets each
#          clone to its pin, which discards local changes in third_party/.
# Run:     bash scripts/setup-third-party.sh [target-dir]    (default: third_party)
# Studio:  not needed.
# Date:    2026-11-05, foundation work for the port.
set -euo pipefail

# Pins. Changing a pin means re-running the differential tests (scripts/scoreboard.sh).
TYPESCRIPT_TAG=v7.1-beta
TYPESCRIPT_COMMIT=3e9236361fa97a9e6c685f4833ac457ae2d8c50a
ROBLOX_TS_COMMIT=d1d5486094ac1d1821b7eb03ce0de0890d30b82e      # npm roblox-ts@3.0.0 gitHead, tag v3.0.0
ROJO_RESOLVER_COMMIT=1b3b479bc074007abbf1cf81c3082cf20cf2aad3  # npm @roblox-ts/rojo-resolver@1.1.0 gitHead, tag v1.1.0
PATH_TRANSLATOR_COMMIT=2e6f341b51f5ecc5b4f152903e7b5cfb60c4ad7e # npm @roblox-ts/path-translator@1.1.0 gitHead, tag v1.1.0
LUAU_AST_COMMIT=ef18d8bccbe5a31435ab56683a81817a7f22ac51       # npm @roblox-ts/luau-ast@2.0.0 gitHead

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
target="${1:-$repo_root/third_party}"
mkdir -p "$target"
target="$(cd "$target" && pwd)"

# Git options for every clone. Line endings must match upstream exactly on Windows too.
g() { git -c core.autocrlf=false -c advice.detachedHead=false "$@"; }

# pin <dir> <url> <commit>: shallow clone of one commit, with the push URL disabled.
pin() {
	local dir="$target/$1" url="$2" commit="$3"
	if [ ! -d "$dir/.git" ]; then
		mkdir -p "$dir"
		g -C "$dir" init -q
		g -C "$dir" remote add origin "$url"
	fi
	g -C "$dir" config core.autocrlf false
	g -C "$dir" remote set-url --push origin DISABLED
	if ! g -C "$dir" cat-file -e "$commit^{commit}" 2>/dev/null; then
		g -C "$dir" fetch -q --depth 1 origin "$commit"
	fi
	g -C "$dir" checkout -q -f --detach "$commit"
	echo "$1: $(g -C "$dir" rev-parse HEAD)"
}

# TypeScript has about 136k tracked files, and a full checkout fails on Windows. Keep it sparse.
ts="$target/typescript"
if [ ! -d "$ts/.git" ]; then
	g clone -q --no-checkout --depth 1 --branch "$TYPESCRIPT_TAG" --filter=blob:none --sparse \
		https://github.com/microsoft/TypeScript.git "$ts"
fi
g -C "$ts" config core.autocrlf false
g -C "$ts" remote set-url --push origin DISABLED
g -C "$ts" sparse-checkout set --cone tsc/cmd tsc/internal tsc/_scripts
if ! g -C "$ts" cat-file -e "$TYPESCRIPT_COMMIT^{commit}" 2>/dev/null; then
	g -C "$ts" fetch -q --depth 1 --filter=blob:none origin "refs/tags/$TYPESCRIPT_TAG:refs/tags/$TYPESCRIPT_TAG"
fi
g -C "$ts" checkout -q -f --detach "$TYPESCRIPT_COMMIT"
rm -f "$ts/tsc/internal/checker/rbxtsgo_exports.go"
for patch in "$repo_root"/patches/typescript/*.patch; do
	g -C "$ts" apply "$patch"
	echo "typescript: applied $(basename "$patch")"
done
echo "typescript: $(g -C "$ts" rev-parse HEAD)"
if [ "$(g -C "$ts" rev-parse HEAD)" != "$TYPESCRIPT_COMMIT" ]; then
	echo "typescript tag $TYPESCRIPT_TAG moved, expected $TYPESCRIPT_COMMIT" >&2
	exit 1
fi

pin roblox-ts https://github.com/roblox-ts/roblox-ts.git "$ROBLOX_TS_COMMIT"
pin rojo-resolver https://github.com/roblox-ts/rojo-resolver.git "$ROJO_RESOLVER_COMMIT"
pin path-translator https://github.com/roblox-ts/path-translator.git "$PATH_TRANSLATOR_COMMIT"
pin luau-ast https://github.com/roblox-ts/luau-ast.git "$LUAU_AST_COMMIT"
