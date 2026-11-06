package difftest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// cliDiagnostic is one located message as the binary prints it.
type cliDiagnostic struct {
	File     string
	Line     int
	Column   int
	Severity string
	Code     string
	Message  string
	// Snippet is the source line printed under the message.
	Snippet string
	Related bool
}

func (d cliDiagnostic) String() string {
	return fmt.Sprintf("%s:%d:%d %s %s %q", d.File, d.Line, d.Column, d.Severity, d.Code, d.Message)
}

var (
	cliHeader  = regexp.MustCompile(`^(\S.*?):(\d+):(\d+) - (error|warning)\s+(TS\s?(?:roblox-ts|\d+))?:? ?(.*)$`)
	cliRelated = regexp.MustCompile(`^\s+(\S.*?):(\d+):(\d+)(?: - .*)?$`)
	cliSnippet = regexp.MustCompile(`^\s*(\d+) (.*)$`)
)

// parseCLIOutput reads what TS's pretty formatter prints.
func parseCLIOutput(output string) []cliDiagnostic {
	var list []cliDiagnostic
	lines := strings.Split(strings.ReplaceAll(ansi.ReplaceAllString(output, ""), "\r\n", "\n"), "\n")
	for i, line := range lines {
		var d *cliDiagnostic
		if match := cliHeader.FindStringSubmatch(line); match != nil {
			lineNumber, _ := strconv.Atoi(match[2])
			column, _ := strconv.Atoi(match[3])
			d = &cliDiagnostic{File: filepath.ToSlash(match[1]), Line: lineNumber, Column: column, Severity: match[4], Code: match[5], Message: match[6]}
		} else if match := cliRelated.FindStringSubmatch(line); match != nil {
			lineNumber, _ := strconv.Atoi(match[2])
			column, _ := strconv.Atoi(match[3])
			d = &cliDiagnostic{File: filepath.ToSlash(match[1]), Line: lineNumber, Column: column, Severity: "related", Related: true}
		}
		if d == nil {
			continue
		}
		for j := i + 1; j < len(lines) && j <= i+3; j++ {
			if match := cliSnippet.FindStringSubmatch(lines[j]); match != nil && match[1] == strconv.Itoa(d.Line) {
				d.Snippet = match[2]
				break
			}
		}
		list = append(list, *d)
	}
	return list
}

func runCommand(t *testing.T, dir string, name string, args ...string) (string, int) {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	code := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = exit.ExitCode()
	}
	return string(output), code
}

type lineProject struct {
	dir    string
	common string
	cases  string
}

// put lays out one case in the shared project folder.
func (p *lineProject) put(t *testing.T, name, source, tsconfig string, crlf bool) {
	t.Helper()
	for _, sub := range []string{"src", "out", "include"} {
		if err := os.RemoveAll(filepath.Join(p.dir, sub)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(p.dir, "src"), 0o777); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(filepath.Join(p.cases, source))
	if err != nil {
		t.Fatal(err)
	}
	if crlf {
		text = []byte(strings.ReplaceAll(string(text), "\n", "\r\n"))
	}
	if err := os.WriteFile(filepath.Join(p.dir, "src", name), text, 0o666); err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(name, "rojo") {
		helper, err := os.ReadFile(filepath.Join(p.cases, "helper"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.dir, "src", "helper.ts"), helper, 0o666); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(p.dir, "tsconfig.json"), []byte(tsconfig), 0o666); err != nil {
		t.Fatal(err)
	}
}

func tokenAtColumn(text string, column int) string {
	units := 0
	for i, r := range text {
		if units >= column-1 {
			return tokenAt(text, i)
		}
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}
	}
	return ""
}

// TestSetgetCLILines is the check on what the user sees. For each case, it builds the file with accessors through the
// real binary, with --setget and no plugin package installed. It builds the same code written as methods, on the same
// lines, with setget off, through the binary and through JS rbxtsc. That second file's errors are the ground truth,
// because no reprint can be involved. Each message must have the same line, code and text, and the same column or, on
// a line where a rewrite comes first and moves the column, the same token. The printed source line must be the user's.
func TestSetgetCLILines(t *testing.T) {
	root := repoRoot(t)
	projectDir := filepath.Join(root, ".scratch", "diff", "setget-lines")
	if _, err := os.Stat(filepath.Join(projectDir, "node_modules", "@rbxts", "types")); err != nil {
		t.Skip("run scripts/setup-diff-corpus.sh")
	}
	bin := filepath.Join(t.TempDir(), "rbxtsgo")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if output, code := runCommand(t, root, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", bin, "./cmd/rbxtsgo"); code != 0 {
		t.Fatal(output)
	}
	jsCLI := filepath.Join(root, ".scratch", "golden-deps", "node_modules", "roblox-ts", "out", "CLI", "cli.js")
	if _, err := os.Stat(jsCLI); err != nil {
		t.Skip("run scripts/setup-diff-corpus.sh")
	}
	common, err := os.ReadFile(filepath.Join(root, "testdata", "differential", "fixtures", "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	casesDir := filepath.Join(root, "testdata", "differential", "fixtures", "setget", "lines")
	entries, err := os.ReadDir(casesDir)
	if err != nil {
		t.Fatal(err)
	}
	p := &lineProject{dir: projectDir, cases: casesDir}
	tsconfig := string(common)
	moved, compared := 0, 0
	for _, entry := range entries {
		extension := filepath.Ext(entry.Name())
		base := strings.TrimSuffix(entry.Name(), extension)
		if extension == "" || strings.HasSuffix(base, ".plain") {
			continue
		}
		t.Run(base, func(t *testing.T) {
			flags := []string{"-p", "."}
			if base == "warningTruthy" {
				flags = append(flags, "--logTruthyChanges")
			}
			if base == "directive" {
				flags = append(flags, "--allowCommentDirectives")
			}
			crlf := base == "crlf"
			plainSource := base + ".plain" + extension

			p.put(t, entry.Name(), plainSource, tsconfig, crlf)
			wantOutput, wantCode := runCommand(t, projectDir, bin, flags...)
			want := parseCLIOutput(wantOutput)
			jsOutput, jsCode := runCommand(t, projectDir, "node", append([]string{jsCLI}, flags...)...)
			fromJS := parseCLIOutput(jsOutput)

			p.put(t, entry.Name(), entry.Name(), tsconfig, crlf)
			gotOutput, gotCode := runCommand(t, projectDir, bin, append(flags, "--setget")...)
			got := parseCLIOutput(gotOutput)

			// The reprint can change a syntax error, as it does in JS. These are listed in accepted-diffs.md.
			if base == "syntaxRepaired" {
				if len(got) != 0 || gotCode != 0 || len(want) == 0 {
					t.Errorf("with setget the printer should repair this: %d messages, exit code %d, and %d on the plain file", len(got), gotCode, len(want))
				}
				return
			}
			// Other syntax errors change too: the printed text has a different mistake, so the code and the column can differ
			// from the plain file. What's reported still points at the user's line, with the user's line shown.
			if base == "syntaxKept" {
				kept := []string{"32:1 TS1123", "43:1 TS1211"}
				var reported []string
				for _, d := range got {
					reported = append(reported, fmt.Sprintf("%d:%d %s", d.Line, d.Column, d.Code))
				}
				if !slices.Equal(reported, kept) {
					t.Errorf("reported %v, want %v:\n%s", reported, kept, gotOutput)
				}
				return
			}
			if len(want) == 0 {
				t.Fatalf("the ground truth has no diagnostics, so this case shows nothing:\n%s", wantOutput)
			}
			if wantCode != jsCode {
				t.Errorf("plain file: exit code %d from rbxtsgo, %d from JS", wantCode, jsCode)
			}
			if len(want) != len(fromJS) {
				t.Errorf("plain file: %d messages from rbxtsgo, %d from JS:\n%s\n%s", len(want), len(fromJS), wantOutput, jsOutput)
			} else {
				for i := range want {
					if want[i].Line != fromJS[i].Line || want[i].Column != fromJS[i].Column || want[i].Code != fromJS[i].Code {
						t.Errorf("plain file: rbxtsgo says %v, JS says %v", want[i], fromJS[i])
					}
				}
			}
			if gotCode != wantCode {
				t.Errorf("exit code %d with setget, %d on the plain file", gotCode, wantCode)
			}
			if len(got) != len(want) {
				t.Fatalf("%d messages with setget, %d on the plain file:\n--- setget\n%s\n--- plain\n%s", len(got), len(want), gotOutput, wantOutput)
			}
			read := func(source string) []string {
				text, err := os.ReadFile(filepath.Join(casesDir, source))
				if err != nil {
					t.Fatal(err)
				}
				return strings.Split(string(text), "\n")
			}
			onLines, plainLines := read(entry.Name()), read(plainSource)
			compared += len(want)
			for i := range want {
				g, w := got[i], want[i]
				if g.File != w.File || g.Line != w.Line || g.Severity != w.Severity || g.Code != w.Code || g.Message != w.Message {
					t.Errorf("message %d\n  setget: %v\n  plain:  %v", i, g, w)
					continue
				}
				if g.Line < 1 || g.Line > len(onLines) {
					continue
				}
				source := onLines[g.Line-1]
				// The formatter writes each tab as one space.
				if g.Snippet != "" && strings.TrimRight(g.Snippet, " ") != strings.TrimRight(strings.ReplaceAll(source, "\t", " "), " \r") {
					t.Errorf("message %d shows %q, and the source line is %q", i, g.Snippet, source)
				}
				if g.Column != w.Column {
					moved++
					gotToken := tokenAtColumn(source, g.Column)
					wantToken := tokenAtColumn(plainLines[w.Line-1], w.Column)
					wantToken = strings.TrimPrefix(strings.TrimPrefix(wantToken, "__get"), "__set")
					if gotToken != wantToken {
						t.Errorf("message %d: column %d is %q with setget, column %d is %q on the plain file", i, g.Column, gotToken, w.Column, wantToken)
					}
				}
			}
		})
	}
	t.Logf("%d messages compared. %d had a rewrite earlier on their line, so their column moved, and landed on the same token", compared, moved)
}
