#!/usr/bin/env bash
# Purpose: the scoreboard. Compares rbxtsgo with the JS roblox-ts@3.0.0 goldens, then compares the rbxtsgo binary with
#          the JS CLI on a whole project, then runs upstream's runtime suite on rbxtsgo's output if every file compiled.
#          Setup, once per clone: bash scripts/setup-diff-corpus.sh. Details: testdata/differential/README.md.
# Run:     bash scripts/scoreboard.sh [-v] [--cli] [--runtime]
#          With no --cli or --runtime, only the golden comparison runs, which takes seconds.
# Studio:  not needed.
# Date:    2026-11-05, differential harness for the transformer port.
set -uo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"
export PATH="$PATH:$HOME/.aftman/bin"
command -v go >/dev/null 2>&1 || {
	echo "go not found on PATH. Install Go (the version in go.mod) and make sure 'go' runs in this shell." >&2
	exit 1
}
exe=""
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) exe=".exe" ;; esac

verbose=""
cli=0
runtime=0
for arg in "$@"; do
	case "$arg" in
	-v) verbose="-v" ;;
	--cli) cli=1 ;;
	--runtime) runtime=1 ;;
	*)
		echo "unknown argument: $arg" >&2
		exit 2
		;;
	esac
done

mkdir -p .scratch/bin
go build -o .scratch/bin/scoreboard$exe ./cmd/scoreboard || exit 1
go build -o .scratch/bin/rbxtsgo$exe ./cmd/rbxtsgo || exit 1
.scratch/bin/scoreboard$exe $verbose -report .scratch/diff/report.txt
echo "Failures with first differences: .scratch/diff/report.txt"

if [ "$cli" = 1 ]; then
	bash scripts/scoreboard-cli.sh
fi

if [ "$runtime" = 1 ]; then
	# The runtime suite needs the whole tests project compiled, so it runs on the CLI copy.
	bash scripts/scoreboard-runtime.sh
fi
