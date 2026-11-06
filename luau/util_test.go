package luau_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

type utilGolden struct {
	LuauAstVersion       string      `json:"luauAstVersion"`
	IsValidIdentifier    [][2]any    `json:"isValidIdentifier"`
	IsValidNumberLiteral [][2]any    `json:"isValidNumberLiteral"`
	IsMetamethod         [][2]any    `json:"isMetamethod"`
	IsReservedClassField [][2]any    `json:"isReservedClassField"`
	IsReservedIdentifier [][2]any    `json:"isReservedIdentifier"`
	GetKindName          [][2]any    `json:"getKindName"`
	NumberLiterals       [][2]any    `json:"numberLiterals"`
	NumberFallbacks      [][3]any    `json:"numberFallbacks"`
	NumberToString       [][2]string `json:"numberToString"`
}

func checkStringPredicate(t *testing.T, name string, cases [][2]any, fn func(string) bool) {
	t.Helper()
	if len(cases) == 0 {
		t.Fatalf("%s: no cases", name)
	}
	for _, c := range cases {
		in, want := c[0].(string), c[1].(bool)
		if got := fn(in); got != want {
			t.Errorf("%s(%q) = %v, upstream says %v", name, in, got, want)
		}
	}
}

func TestUtilPredicates(t *testing.T) {
	var g utilGolden
	readGolden(t, "util.json", &g)
	checkStringPredicate(t, "IsValidIdentifier", g.IsValidIdentifier, luau.IsValidIdentifier)
	checkStringPredicate(t, "IsValidNumberLiteral", g.IsValidNumberLiteral, luau.IsValidNumberLiteral)
	checkStringPredicate(t, "IsMetamethod", g.IsMetamethod, luau.IsMetamethod)
	checkStringPredicate(t, "IsReservedClassField", g.IsReservedClassField, luau.IsReservedClassField)
	checkStringPredicate(t, "IsReservedIdentifier", g.IsReservedIdentifier, luau.IsReservedIdentifier)
}

func TestGetKindName(t *testing.T) {
	var g utilGolden
	readGolden(t, "util.json", &g)
	if len(g.GetKindName) == 0 {
		t.Fatal("no cases")
	}
	for _, c := range g.GetKindName {
		kind := luau.SyntaxKind(int(c[0].(float64)))
		want, _ := c[1].(string) // nil (undefined) becomes ""
		if got := luau.GetKindName(kind); got != want {
			t.Errorf("GetKindName(%d) = %q, upstream says %q", kind, got, want)
		}
	}
}

// NewNumber must build the same literal text as upstream number().
func TestNewNumber(t *testing.T) {
	var g utilGolden
	readGolden(t, "util.json", &g)
	if len(g.NumberLiterals) == 0 {
		t.Fatal("no cases")
	}
	for _, c := range g.NumberLiterals {
		bits, err := strconv.ParseUint(c[0].(string), 16, 64)
		if err != nil {
			t.Fatal(err)
		}
		v := math.Float64frombits(bits)
		want := c[1].(map[string]any)
		neg, value := want["neg"].(bool), want["value"].(string)

		var gotNeg bool
		var gotValue string
		switch n := luau.NewNumber(v).(type) {
		case *luau.NumberLiteral:
			gotValue = n.Value
		case *luau.UnaryExpression:
			gotNeg = true
			if n.Operator != luau.UnaryMinus {
				t.Errorf("NewNumber(%v): operator %q", v, n.Operator)
			}
			gotValue = n.Expression.(*luau.NumberLiteral).Value
		default:
			t.Fatalf("NewNumber(%v): %T", v, n)
		}
		if gotNeg != neg || gotValue != value {
			t.Errorf("NewNumber(%v) = neg %v %q, upstream says neg %v %q", v, gotNeg, gotValue, neg, value)
		}
	}
}

func TestNewNumberNaNPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewNumber(NaN) did not panic")
		}
	}()
	luau.NewNumber(math.NaN())
}
