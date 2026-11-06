package plugins

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const jsCompilerPath = "node_modules/roblox-ts/out/CLI/cli.js"

// flagsOfOurs are the flags only rbxtsgo has that take a value, and boolFlagsOfOurs the ones that don't.
var (
	flagsOfOurs     = []string{"checkers", "cpuprofile", "setgetPrefix"}
	boolFlagsOfOurs = []string{"setget", "no-setget"}
)

// passAlongArgs removes our own flags from the command line, because yargs in the JS compiler rejects unknown ones.
func passAlongArgs(args []string) []string {
	result := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name := strings.TrimLeft(arg, "-")
		if name == arg {
			result = append(result, arg)
			continue
		}
		name, _, hasValue := strings.Cut(name, "=")
		if slices.Contains(boolFlagsOfOurs, name) {
			continue
		}
		ours := slices.Contains(flagsOfOurs, name)
		if !ours {
			result = append(result, arg)
		} else if !hasValue {
			i++
		}
	}
	return result
}

// findJSCompiler looks in the tsconfig's folder and above, as node_modules lookups do.
func findJSCompiler(directory string) (string, bool) {
	for current := directory; ; {
		candidate := filepath.Join(current, filepath.FromSlash(jsCompilerPath))
		if exists(candidate) {
			return candidate, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

// Delegate runs the whole build with the project's JS compiler, with the command line it was given, and returns the
// exit code. It only returns an error when the JS compiler can't start, with what to do about it.
func Delegate(reason, tsConfigPath string, args []string, stderr io.Writer) (int, error) {
	directory := filepath.Dir(tsConfigPath)
	compiler, found := findJSCompiler(directory)
	if !found {
		return 1, fmt.Errorf("%s, and the JS compiler isn't installed at %s.\nTo build this project, install it with `npm install --save-dev roblox-ts@3.0.0` in %s, or remove the plugin from compilerOptions.plugins in %s.", reason, filepath.Join(directory, filepath.FromSlash(jsCompilerPath)), directory, tsConfigPath)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return 1, fmt.Errorf("%s, and the JS compiler at %s needs Node.js, which isn't on PATH.\nTo build this project, install Node.js, or remove the plugin from compilerOptions.plugins in %s.", reason, compiler, tsConfigPath)
	}
	fmt.Fprintf(stderr, "%s. Handing the whole build to the JS compiler at %s.\n", reason, compiler)
	command := exec.Command(node, append([]string{compiler}, passAlongArgs(args)...)...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode(), nil
		}
		return 1, fmt.Errorf("%s, and the JS compiler at %s didn't start: %w", reason, compiler, err)
	}
	return 0, nil
}
