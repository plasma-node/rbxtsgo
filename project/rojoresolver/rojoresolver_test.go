package rojoresolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

type goldenResult struct {
	Lines  []string `json:"lines"`
	Throws bool     `json:"throws"`
}

type goldenFile struct {
	Node     string                  `json:"node"`
	FromPath map[string]goldenResult `json:"fromPath"`
	Synth    map[string]goldenResult `json:"synthetic"`
	Invalid  map[string]goldenResult `json:"invalid"`
	Find     map[string]struct {
		Path     *string  `json:"path"`
		Warnings []string `json:"warnings"`
	} `json:"find"`
	Types    map[string]int      `json:"types"`
	Relative map[string][]string `json:"relative"`
}

type casesFile struct {
	Configs   []string `json:"configs"`
	Synthetic []string `json:"synthetic"`
	Find      []string `json:"find"`
	Types     []string `json:"types"`
	Lookups   []string `json:"lookups"`
}

func readJSON(t *testing.T, name string, into any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatal(err)
	}
}

func jsArray(values []string) string {
	if values == nil {
		return "[]"
	}
	var b strings.Builder
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(values); err != nil {
		panic(err)
	}
	return strings.TrimRight(b.String(), "\n")
}

func describe(r *Resolver, root string, cases casesFile) []string {
	clean := func(text string) string { return strings.ReplaceAll(text, root, "<ROOT>") }
	var lines []string
	for _, warning := range r.Warnings() {
		lines = append(lines, "warning "+clean(warning))
	}
	lines = append(lines, fmt.Sprintf("isGame %v", r.IsGame))
	for _, partition := range r.Partitions() {
		lines = append(lines, fmt.Sprintf("partition %s %s", clean(partition.FsPath), jsArray(partition.RbxPath)))
	}
	var seen []RbxPath
	for _, rel := range cases.Lookups {
		rbxPath := r.GetRbxPathFromFilePath(filepath.Join(root, filepath.FromSlash(rel)))
		if rbxPath == nil {
			lines = append(lines, fmt.Sprintf("lookup %s undefined", rel))
			continue
		}
		lines = append(lines, fmt.Sprintf("lookup %s %s", rel, jsArray(rbxPath)))
		lines = append(lines, fmt.Sprintf("  isolated %v network %d", r.IsIsolated(rbxPath), r.GetNetworkType(rbxPath)))
		if !slices.ContainsFunc(seen, func(v RbxPath) bool { return slices.Equal(v, rbxPath) }) {
			seen = append(seen, rbxPath)
		}
	}
	for _, from := range seen {
		for _, to := range seen {
			relative := Relative(from, to)
			for i, v := range relative {
				if v == RbxPathParent {
					relative[i] = "<parent>"
				}
			}
			lines = append(lines, fmt.Sprintf("pair %s %s %d %s", jsArray(from), jsArray(to), r.GetFileRelation(from, to), jsArray(relative)))
		}
	}
	return lines
}

func TestAgainstNpmPackage(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "fixtures"))
	if err != nil {
		t.Fatal(err)
	}
	var cases casesFile
	readJSON(t, "cases.json", &cases)
	var golden goldenFile
	if _, err := os.Stat(filepath.Join("testdata", "golden."+runtime.GOOS+".json")); err != nil {
		t.Skip("no golden for this platform: run scripts/gen-goldens.sh here")
	}
	readJSON(t, "golden."+runtime.GOOS+".json", &golden)
	t.Logf("golden from Node %s", golden.Node)

	compare := func(t *testing.T, r *Resolver, err error, want goldenResult) {
		t.Helper()
		if want.Throws {
			if err == nil {
				t.Errorf("JS throws, got no error")
			}
			return
		}
		if err != nil {
			t.Fatalf("JS doesn't throw, got %v", err)
		}
		got := describe(r, root, cases)
		if !slices.Equal(got, want.Lines) {
			t.Errorf("differs from JS: %s", firstDifference(got, want.Lines))
		}
	}

	for _, config := range cases.Configs {
		t.Run("fromPath/"+config, func(t *testing.T) {
			r, err := FromPath(filepath.Join(root, filepath.FromSlash(config)))
			compare(t, r, err, golden.FromPath[config])
		})
	}
	for _, dir := range cases.Synthetic {
		t.Run("synthetic/"+dir, func(t *testing.T) {
			r, err := Synthetic(filepath.Join(root, filepath.FromSlash(dir)))
			compare(t, r, err, golden.Synth[dir])
		})
	}
	if len(golden.Invalid) < 30 {
		t.Errorf("only %d invalid cases", len(golden.Invalid))
	}
	for name, want := range golden.Invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			r, err := FromPath(filepath.Join(root, "invalid", name))
			compare(t, r, err, want)
		})
	}
	for _, dir := range cases.Find {
		t.Run("find/"+dir, func(t *testing.T) {
			found, warnings, err := FindRojoConfigFilePath(filepath.Join(root, filepath.FromSlash(dir)))
			if err != nil {
				t.Fatal(err)
			}
			want := golden.Find[dir]
			wantPath := ""
			if want.Path != nil {
				wantPath = *want.Path
			}
			if got := strings.ReplaceAll(found, root, "<ROOT>"); got != wantPath {
				t.Errorf("path = %q, JS gives %q", got, wantPath)
			}
			var cleaned []string
			for _, warning := range warnings {
				cleaned = append(cleaned, strings.ReplaceAll(warning, root, "<ROOT>"))
			}
			if !slices.Equal(cleaned, want.Warnings) && len(cleaned)+len(want.Warnings) != 0 {
				t.Errorf("warnings = %q, JS gives %q", cleaned, want.Warnings)
			}
		})
	}
	synthetic, err := Synthetic(filepath.Join(root, "model", "out"))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range golden.Types {
		if got := int(synthetic.GetRbxTypeFromFilePath(name)); got != want {
			t.Errorf("getRbxTypeFromFilePath(%q) = %d, JS gives %d", name, got, want)
		}
	}
	rbx := [][]string{{}, {"a"}, {"a", "b"}, {"a", "c"}, {"a", "b", "c"}, {"d"}, {"ReplicatedStorage", "x"}, {"StarterGui", "x"}, {"StarterPlayer", "StarterPlayerScripts", "y"}}
	for _, from := range rbx {
		for _, to := range rbx {
			relative := Relative(from, to)
			for i, v := range relative {
				if v == RbxPathParent {
					relative[i] = "<parent>"
				}
			}
			want := golden.Relative[jsArray(from)+jsArray(to)]
			if !slices.Equal(relative, want) {
				t.Errorf("relative(%v, %v) = %v, JS gives %v", from, to, relative, want)
			}
		}
	}
}

func TestObjectKeyOrderMatchesJS(t *testing.T) {
	value, err := parseJSON(`{"b": 1, "10": 2, "a": 3, "2": 4, "01": 5, "4294967295": 6, "4294967294": 7, "b": 8}`)
	if err != nil {
		t.Fatal(err)
	}
	got := value.(*object).keys
	want := []string{"2", "10", "4294967294", "b", "a", "01", "4294967295"}
	if !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

// firstDifference keeps failures readable, because the full listings are long.
func firstDifference(got, want []string) string {
	for i := range max(len(got), len(want)) {
		g, w := "<none>", "<none>"
		if i < len(got) {
			g = got[i]
		}
		if i < len(want) {
			w = want[i]
		}
		if g != w {
			return fmt.Sprintf("line %d of %d/%d\n got %s\nwant %s", i, len(got), len(want), g, w)
		}
	}
	return "no difference"
}
