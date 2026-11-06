package luau_test

// Replays testdata/scenarios.js with the Go API and compares the resulting
// object graphs (parent pointers, clones, list node links, read-only flags)
// with the ones the real upstream package produced. Keep in sync with
// scenarios.js; the globals scenarios must stay first. globals-initial is dumped at
// package init; globals-first-use needs Globals.Getmetatable and Strings.Mode to
// be untouched, so no other test may use those two.

import (
	"reflect"
	"sort"
	"testing"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

type scenarioGolden struct {
	Name   string         `json:"name"`
	Graph  *graphJSON     `json:"graph"`
	Values map[string]any `json:"values"`
	Error  *string        `json:"error"`
}

type scenarioFile struct {
	Cases []scenarioGolden `json:"cases"`
}

func id(name string) *luau.Identifier { return luau.NewID(name) }

func list(nodes ...luau.Node) *luau.List { return luau.MakeList(nodes...) }

func callStmt(e luau.CallLike) *luau.CallStatement {
	return luau.Create(&luau.CallStatement{Expression: e})
}

func r(name string, v luau.NodeOrList) ordered { return ordered{name, v} }

type scenarioResult struct {
	roots    []ordered
	values   map[string]any
	snapshot *graphJSON // an already dumped graph, instead of roots
}

// initialGlobals is dumped at package init, before any test can adopt a global.
var initialGlobals = dumpGraph(globalsRoots())

func globalsRoots() []ordered {
	g := luau.Globals
	all := map[string]luau.Node{
		"_G": g.G, "TS": g.TS, "assert": g.Assert, "bit32": g.Bit32, "coroutine.yield": g.Coroutine.Yield, "error": g.Error,
		"exports": g.Exports, "getmetatable": g.Getmetatable, "ipairs": g.Ipairs, "next": g.Next, "pairs": g.Pairs,
		"pcall": g.Pcall, "require": g.Require, "script": g.Script, "select": g.Select, "self": g.Self,
		"setmetatable": g.Setmetatable, "super": g.Super, "tostring": g.Tostring, "type": g.Type, "typeof": g.Typeof,
		"unpack": g.Unpack, "game": g.Game,
		"string.byte": g.String.Byte, "string.find": g.String.Find, "string.format": g.String.Format,
		"string.gmatch": g.String.Gmatch, "string.gsub": g.String.Gsub, "string.lower": g.String.Lower,
		"string.match": g.String.Match, "string.rep": g.String.Rep, "string.reverse": g.String.Reverse,
		"string.split": g.String.Split, "string.sub": g.String.Sub, "string.upper": g.String.Upper,
		"table.clear": g.Table.Clear, "table.clone": g.Table.Clone, "table.concat": g.Table.Concat,
		"table.create": g.Table.Create, "table.find": g.Table.Find, "table.freeze": g.Table.Freeze,
		"table.insert": g.Table.Insert, "table.isfrozen": g.Table.Isfrozen, "table.maxn": g.Table.Maxn,
		"table.move": g.Table.Move, "table.pack": g.Table.Pack, "table.remove": g.Table.Remove,
		"table.sort": g.Table.Sort, "table.unpack": g.Table.Unpack,
		"utf8.charpattern": g.Utf8.Charpattern, "utf8.codes": g.Utf8.Codes, "math.min": g.Math.Min,
		"strings.__index": luau.Strings.Index, "strings.__tostring": luau.Strings.Tostring,
		"strings.__mode": luau.Strings.Mode, "strings.k": luau.Strings.K, "strings.number": luau.Strings.Number,
		"strings.table": luau.Strings.Table, "strings.success": luau.Strings.Success, "strings.value": luau.Strings.Value,
		"strings.error": luau.Strings.Error, "strings., ": luau.Strings.Comma,
	}
	names := make([]string, 0, len(all))
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)
	out := make([]ordered, len(names))
	for i, k := range names {
		out[i] = r(k, all[k])
	}
	return out
}

// goScenarios mirrors scenarios.js entry by entry, in the same order.
func goScenarios() []struct {
	name  string
	build func() scenarioResult
} {
	type S = struct {
		name  string
		build func() scenarioResult
	}
	roots := func(rs ...ordered) scenarioResult { return scenarioResult{roots: rs} }
	return []S{
		{"globals-initial", func() scenarioResult { return scenarioResult{snapshot: &initialGlobals} }},
		{"globals-first-use", func() scenarioResult {
			first := luau.NewCall(luau.Globals.Getmetatable, id("a"))
			second := luau.NewCall(luau.Globals.Getmetatable, id("b"))
			str1 := luau.NewArray(luau.Strings.Mode)
			str2 := luau.NewArray(luau.Strings.Mode)
			return roots(r("first", first), r("second", second), r("str1", str1), r("str2", str2),
				r("getmetatable", luau.Globals.Getmetatable), r("__mode", luau.Strings.Mode))
		}},
		{"create-adopts-children", func() scenarioResult {
			a, b := id("a"), id("b")
			call := luau.NewCall(a, b, id("c"))
			return roots(r("call", call), r("a", a), r("b", b))
		}},
		{"create-clones-reused-child", func() scenarioResult {
			a := id("a")
			sum := luau.NewBinary(a, "+", a)
			return roots(r("sum", sum), r("a", a))
		}},
		{"create-clones-child-with-parent", func() scenarioResult {
			shared := luau.NewBinary(id("a"), "+", id("b"))
			p1 := luau.NewUnary("-", shared)
			p2 := luau.NewUnary("not", shared)
			return roots(r("p1", p1), r("p2", p2), r("shared", shared))
		}},
		{"create-clone-keeps-grandchildren-parent", func() scenarioResult {
			inner := luau.NewBinary(id("a"), "+", id("b"))
			outer := luau.NewBinary(inner, "*", id("c"))
			s1 := luau.Create(&luau.Assignment{Left: id("x"), Operator: "=", Right: outer})
			s2 := luau.Create(&luau.Assignment{Left: id("y"), Operator: "=", Right: outer})
			return roots(r("s1", s1), r("s2", s2), r("outer", outer), r("inner", inner))
		}},
		{"create-list-elements", func() scenarioResult {
			shared := id("s")
			owner := luau.Create(&luau.DoStatement{Statements: list(
				callStmt(luau.NewCall(shared)), callStmt(luau.NewCall(shared)))})
			return roots(r("owner", owner), r("shared", shared))
		}},
		{"create-list-element-with-parent", func() scenarioResult {
			stmt := callStmt(luau.NewCall(id("f")))
			first := luau.Create(&luau.DoStatement{Statements: list(stmt)})
			second := luau.Create(&luau.DoStatement{Statements: list(stmt)})
			return roots(r("first", first), r("second", second), r("stmt", stmt))
		}},
		{"create-node-or-list-fields", func() scenarioResult {
			assign := luau.Create(&luau.Assignment{Left: list(id("a"), id("b")), Operator: "=", Right: list(id("c"), id("d"))})
			decl := luau.Create(&luau.VariableDeclaration{Left: id("e")})
			decl2 := luau.Create(&luau.VariableDeclaration{Left: list(id("f")), Right: luau.NewNil()})
			ret := luau.Create(&luau.ReturnStatement{Expression: list(id("g"))})
			ifs := luau.Create(&luau.IfStatement{Condition: id("h"), Statements: luau.NewList(),
				ElseBody: luau.Create(&luau.IfStatement{Condition: id("i"), Statements: luau.NewList(),
					ElseBody: list(luau.Create(&luau.BreakStatement{}))})})
			return roots(r("assign", assign), r("decl", decl), r("decl2", decl2), r("ret", ret), r("ifs", ifs))
		}},
		{"create-optional-step", func() scenarioResult {
			withStep := luau.Create(&luau.NumericForStatement{ID: id("i"), Start: luau.NewNumber(1), End: luau.NewNumber(2),
				Step: luau.NewNumber(3), Statements: luau.NewList()})
			withoutStep := luau.Create(&luau.NumericForStatement{ID: id("i"), Start: luau.NewNumber(1), End: luau.NewNumber(2),
				Statements: luau.NewList()})
			return roots(r("withStep", withStep), r("withoutStep", withoutStep))
		}},
		{"helpers-constructors", func() scenarioResult {
			return roots(
				r("none", luau.NewNone()),
				r("nil", luau.NewNil()),
				r("t", luau.NewBool(true)),
				r("f", luau.NewBool(false)),
				r("n0", luau.NewNumber(0)),
				r("n1", luau.NewNumber(1.5)),
				r("n2", luau.NewNumber(-2)),
				r("s", luau.NewString("s")),
				r("i", luau.NewID("i")),
				r("cm", luau.NewComment("c")),
				r("arr", luau.NewArray(luau.NewNumber(1), luau.NewNumber(2))),
				r("emptyArr", luau.NewArray()),
				r("set", luau.NewSet(luau.NewString("a"))),
				r("map", luau.NewMap(luau.KeyValue{Index: luau.NewString("k"), Value: luau.NewNumber(1)},
					luau.KeyValue{Index: luau.NewNumber(2), Value: luau.NewNil()})),
				r("mixed", luau.NewMixedTable(luau.MixedField{Value: luau.NewNil()},
					luau.MixedField{Key: luau.NewString("k"), Value: luau.NewNumber(1)})),
				r("bin", luau.NewBinary(luau.NewID("a"), "..", luau.NewID("b"))),
				r("un", luau.NewUnary("#", luau.NewID("a"))),
				r("prop", luau.NewProperty(luau.NewID("a"), "b")),
				r("call", luau.NewCall(luau.NewID("f"), luau.NewID("a"))),
				r("callNoArgs", luau.NewCall(luau.NewID("f"))),
			)
		}},
		{"list-make-and-push", func() scenarioResult {
			a := list(id("1"), id("2"))
			a.Push(id("3"))
			e := luau.NewList()
			e.Push(id("only"))
			return roots(r("a", a), r("e", e))
		}},
		{"list-shift-unshift", func() scenarioResult {
			a := list(id("1"), id("2"), id("3"))
			shifted := a.Shift()
			a.Unshift(id("0"))
			one := list(id("solo"))
			soloShifted := one.Shift()
			empty := luau.NewList()
			emptyShifted := empty.Shift()
			empty.Unshift(id("u"))
			vals := map[string]any{"emptyShifted": nil}
			if emptyShifted != nil {
				vals["emptyShifted"] = "node"
			}
			return scenarioResult{roots: []ordered{r("a", a), r("shifted", shifted), r("one", one),
				r("soloShifted", soloShifted), r("empty", empty)}, values: vals}
		}},
		{"list-shift-dangling-next", func() scenarioResult {
			a := list(id("1"), id("2"))
			shifted := a.Shift()
			return roots(r("a", a), r("shifted", shifted))
		}},
		{"list-pushlist", func() scenarioResult {
			a := list(id("a1"), id("a2"))
			b := list(id("b1"), id("b2"))
			a.PushList(b)
			e1, e2 := luau.NewList(), list(id("e2"))
			e1.PushList(e2)
			e3, e4 := list(id("e3")), luau.NewList()
			e3.PushList(e4)
			return roots(r("a", a), r("b", b), r("e1", e1), r("e2", e2), r("e3", e3), r("e4", e4))
		}},
		{"list-pushlist-then-push", func() scenarioResult {
			a, b := list(id("a1")), list(id("b1"))
			a.PushList(b)
			a.Push(id("a2"))
			return roots(r("a", a), r("b", b))
		}},
		{"list-unshiftlist", func() scenarioResult {
			a := list(id("a1"), id("a2"))
			b := list(id("b1"), id("b2"))
			a.UnshiftList(b)
			e1, e2 := luau.NewList(), list(id("e2"))
			e1.UnshiftList(e2)
			return roots(r("a", a), r("b", b), r("e1", e1), r("e2", e2))
		}},
		{"list-unshiftlist-then-unshift", func() scenarioResult {
			a, b := list(id("a1")), list(id("b1"))
			a.UnshiftList(b)
			a.Unshift(id("a0"))
			return roots(r("a", a), r("b", b))
		}},
		{"list-clone", func() scenarioResult {
			owner := luau.Create(&luau.DoStatement{Statements: list(callStmt(luau.NewCall(id("f"))), luau.Create(&luau.BreakStatement{}))})
			cp := owner.Statements.Clone()
			return roots(r("owner", owner), r("copy", cp))
		}},
		{"list-queries", func() scenarioResult {
			a := list(id("1"), id("2"), id("3"))
			arr := a.ToArray()
			name := func(n luau.Node) string { return n.(*luau.Identifier).Name }
			mapped := luau.MapToArray(a, func(n luau.Node) any { return name(n) })
			return scenarioResult{
				roots: []ordered{r("a", a), r("arr0", arr[0]), r("arr2", arr[2])},
				values: map[string]any{
					"size":            a.Size(),
					"empty":           a.IsEmpty(),
					"nonEmpty":        a.IsNonEmpty(),
					"emptyList":       luau.NewList().IsEmpty(),
					"every":           a.Every(func(n luau.Node) bool { return name(n) != "x" }),
					"everyFalse":      a.Every(func(n luau.Node) bool { return name(n) == "1" }),
					"some":            a.Some(func(n luau.Node) bool { return name(n) == "3" }),
					"someFalse":       a.Some(func(n luau.Node) bool { return name(n) == "x" }),
					"mapped":          mapped,
					"arrLength":       len(arr),
					"isList":          luau.IsList(a),
					"isListNode":      luau.IsList(id("a")),
					"isListUndefined": luau.IsList(nil),
				},
			}
		}},
		{"list-foreach-sees-appends", func() scenarioResult {
			a := list(id("1"))
			seen := []any{}
			a.ForEach(func(n luau.Node) {
				name := n.(*luau.Identifier).Name
				seen = append(seen, name)
				if len(name) < 3 {
					a.Push(id(name + "x"))
				}
			})
			return scenarioResult{roots: []ordered{r("a", a)}, values: map[string]any{"seen": seen}}
		}},
	}
}

func TestScenarios(t *testing.T) {
	var f scenarioFile
	readGolden(t, "scenarios.json", &f)

	goCases := goScenarios()
	byName := map[string]*scenarioGolden{}
	for i := range f.Cases {
		byName[f.Cases[i].Name] = &f.Cases[i]
	}
	// Run in golden order so the globals scenarios come first.
	i := 0
	for _, g := range f.Cases {
		if g.Graph == nil {
			continue
		}
		if i >= len(goCases) || goCases[i].name != g.Name {
			t.Fatalf("scenario order mismatch at %d: golden %q", i, g.Name)
		}
		res := goCases[i].build()
		var got graphJSON
		if res.snapshot != nil {
			got = *res.snapshot
		} else {
			got = dumpGraph(res.roots)
		}
		if !reflect.DeepEqual(normalize(t, got), normalize(t, g.Graph)) {
			gb, _ := jsonIndent(t, normalize(t, got))
			wb, _ := jsonIndent(t, normalize(t, g.Graph))
			t.Errorf("scenario %s: graph differs from upstream\n--- want\n%s\n--- got\n%s", g.Name, wb, gb)
		}
		var wantValues any
		if g.Values != nil {
			wantValues = g.Values
		}
		if res.values != nil || wantValues != nil {
			if !reflect.DeepEqual(normalize(t, res.values), normalize(t, wantValues)) {
				t.Errorf("scenario %s: values: want %v, got %v", g.Name, wantValues, res.values)
			}
		}
		i++
	}
	if i != len(goCases) {
		t.Errorf("ran %d scenarios, Go side has %d", i, len(goCases))
	}
}

// assertionOf returns the AssertionError message raised by fn, or "".
func assertionOf(fn func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			ae, ok := r.(*luau.AssertionError)
			if !ok {
				panic(r)
			}
			msg = ae.Error()
		}
	}()
	fn()
	return ""
}

func TestScenarioErrors(t *testing.T) {
	var f scenarioFile
	readGolden(t, "scenarios.json", &f)
	fns := map[string]func(){
		"list-push-readonly": func() {
			a, b := list(id("a")), luau.NewList()
			a.PushList(b)
			b.Push(id("x"))
		},
		"list-pushlist-to-readonly": func() {
			a, b := luau.NewList(), luau.NewList()
			a.PushList(b)
			luau.NewList().PushList(b)
		},
		"list-pushlist-from-readonly-target": func() {
			a, b := luau.NewList(), luau.NewList()
			a.PushList(b)
			b.PushList(luau.NewList())
		},
		"list-shift-readonly": func() {
			a := list(id("a"))
			luau.NewList().PushList(a)
			a.Shift()
		},
		"list-unshift-readonly": func() {
			a := list(id("a"))
			luau.NewList().PushList(a)
			a.Unshift(id("b"))
		},
		"list-unshiftlist-readonly": func() {
			a := list(id("a"))
			luau.NewList().UnshiftList(a)
			luau.NewList().UnshiftList(a)
		},
		"list-push-non-node": func() { luau.NewList().Push(nil) },
		"list-make-non-node": func() { luau.MakeList[luau.Node](id("a"), nil) },
	}
	n := 0
	for _, g := range f.Cases {
		if g.Error == nil {
			continue
		}
		fn, ok := fns[g.Name]
		if !ok {
			t.Errorf("no Go version of error scenario %s", g.Name)
			continue
		}
		if got := assertionOf(fn); got != *g.Error {
			t.Errorf("%s: want %q, got %q", g.Name, *g.Error, got)
		}
		n++
	}
	if n != len(fns) {
		t.Errorf("golden has %d error scenarios, Go has %d", n, len(fns))
	}
}
