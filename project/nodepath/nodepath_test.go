package nodepath

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type flavorCases struct {
	Resolve    [][]string `json:"resolve"`
	Normalize  [][]string `json:"normalize"`
	IsAbsolute [][]any    `json:"isAbsolute"`
	Join       [][]string `json:"join"`
	Relative   [][]string `json:"relative"`
	Dirname    [][]string `json:"dirname"`
	Basename   [][]string `json:"basename"`
	Extname    [][]string `json:"extname"`
}

func TestAgainstNode(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Node  string      `json:"node"`
		Win32 flavorCases `json:"win32"`
		Posix flavorCases `json:"posix"`
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	for _, flavor := range []struct {
		name  string
		path  *Path
		cases flavorCases
	}{{"win32", Win32, golden.Win32}, {"posix", Posix, golden.Posix}} {
		t.Run(flavor.name, func(t *testing.T) {
			p := flavor.path
			c := flavor.cases
			total := 0
			check := func(fn string, got, want string, args ...string) {
				t.Helper()
				total++
				if got != want {
					t.Errorf("%s(%q) = %q, Node %s gives %q", fn, args, got, golden.Node, want)
				}
			}
			for _, row := range c.Normalize {
				check("normalize", p.Normalize(row[0]), row[1], row[0])
			}
			for _, row := range c.IsAbsolute {
				total++
				if got := p.IsAbsolute(row[0].(string)); got != row[1].(bool) {
					t.Errorf("isAbsolute(%q) = %v", row[0], got)
				}
			}
			for _, row := range c.Dirname {
				check("dirname", p.Dirname(row[0]), row[1], row[0])
			}
			for _, row := range c.Extname {
				check("extname", p.Extname(row[0]), row[1], row[0])
			}
			for _, row := range c.Basename {
				check("basename", p.Basename(row[0], row[1]), row[2], row[0], row[1])
			}
			for _, row := range c.Join {
				check("join", p.Join(row[0], row[1]), row[2], row[0], row[1])
			}
			for _, row := range c.Resolve {
				check("resolve", p.Resolve(row[0], row[1]), row[2], row[0], row[1])
			}
			for _, row := range c.Relative {
				check("relative", p.Relative(row[0], row[1]), row[2], row[0], row[1])
			}
			if total < 1000 {
				t.Errorf("only %d cases", total)
			}
			t.Logf("%d cases against Node %s", total, golden.Node)
		})
	}
}
