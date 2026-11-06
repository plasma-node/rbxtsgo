package luau

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Differential tests for the ECMAScript Number()/String(number) emulation
// against values recorded from Node (testdata/generate.js).

func TestJSNumberFallbacks(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "golden", "util.json"))
	if err != nil {
		t.Fatal(err)
	}
	var g struct {
		NumberFallbacks [][3]any    `json:"numberFallbacks"`
		NumberToString  [][2]string `json:"numberToString"`
	}
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}
	if len(g.NumberFallbacks) == 0 || len(g.NumberToString) == 0 {
		t.Fatal("no cases")
	}

	for _, c := range g.NumberFallbacks {
		in, want, isOne := c[0].(string), c[1].(string), c[2].(bool)
		// String(Number(s.replace(/_/g, "")))
		if got := jsNumberToString(jsToNumber(replaceUnderscores(in))); got != want {
			t.Errorf("String(Number(%q without _)) = %q, node says %q", in, got, want)
		}
		// Number(s) === 1, used to omit a numeric for step
		if got := jsToNumber(in) == 1; got != isOne {
			t.Errorf("Number(%q) === 1: %v, node says %v", in, got, isOne)
		}
	}

	for _, c := range g.NumberToString {
		bits, err := strconv.ParseUint(c[0], 16, 64)
		if err != nil {
			t.Fatal(err)
		}
		v := math.Float64frombits(bits)
		if got := jsNumberToString(v); got != c[1] {
			t.Errorf("String(%v) = %q, node says %q", v, got, c[1])
		}
	}
}

func replaceUnderscores(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '_' {
			out = append(out, s[i])
		}
	}
	return string(out)
}
