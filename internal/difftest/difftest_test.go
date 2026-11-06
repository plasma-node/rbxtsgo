package difftest

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const baselineFile = "testdata/differential/baseline.txt"

func repoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := RepoRoot(cwd)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestDifferential is the gate. It fails if a case in baseline.txt stops matching JS roblox-ts. The baseline is a
// ratchet: every case that isn't in it yet is logged as a failure to fix, not hidden, and UPDATE_BASELINE=1 adds the
// cases that pass now. Removing a case from the baseline needs a reason in docs/known-differences.md.
func TestDifferential(t *testing.T) {
	root := repoRoot(t)
	report, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(baselineFile)))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	baseline := map[string]bool{}
	for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if line != "" {
			baseline[line] = true
		}
	}

	var passing []string
	for _, result := range report.Results {
		if result.Pass {
			passing = append(passing, result.Key())
			continue
		}
		if baseline[result.Key()] {
			t.Errorf("regression: %s: %s", result.Key(), result.Detail)
		} else {
			t.Logf("not passing yet: %s", result.Key())
		}
	}
	known := map[string]bool{}
	for _, result := range report.Results {
		known[result.Key()] = true
	}
	for key := range baseline {
		if !known[key] {
			t.Errorf("baseline case no longer exists: %s", key)
		}
	}

	outputPass, outputTotal := report.Counts("output")
	diagnosticPass, diagnosticTotal := report.Counts("diagnostics")
	t.Logf("scoreboard: %d/%d files byte-identical, %d/%d diagnostics matched", outputPass, outputTotal, diagnosticPass, diagnosticTotal)

	if os.Getenv("UPDATE_BASELINE") == "1" {
		slices.Sort(passing)
		content := strings.Join(passing, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(baselineFile)), []byte(content), 0o666); err != nil {
			t.Fatal(err)
		}
	}
}

// TestWorkerCounts checks that the number of workers doesn't change a byte of output, or which cases pass.
func TestWorkerCounts(t *testing.T) {
	root := repoRoot(t)
	one, err := Run(root, Options{Checkers: 1})
	if err != nil {
		t.Fatal(err)
	}
	many, err := Run(root, Options{Checkers: 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(one.Outputs) != len(many.Outputs) {
		t.Errorf("1 worker wrote %d files, 8 workers wrote %d", len(one.Outputs), len(many.Outputs))
	}
	for name, text := range one.Outputs {
		other, ok := many.Outputs[name]
		if !ok {
			t.Errorf("%s: only 1 worker wrote it", name)
		} else if text != other {
			t.Errorf("%s differs between 1 and 8 workers: %s", name, FirstDifference([]byte(text), []byte(other)))
		}
	}
	for i, result := range one.Results {
		if i >= len(many.Results) {
			break
		}
		other := many.Results[i]
		if result.Key() != other.Key() || result.Pass != other.Pass {
			t.Errorf("1 worker: %s pass=%v, 8 workers: %s pass=%v\n%s\n%s", result.Key(), result.Pass, other.Key(), other.Pass, result.Detail, other.Detail)
		}
	}
}
