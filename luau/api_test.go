package luau_test

// Tests of the API surface around the renderer: kind guards (differential),
// Create/ShallowClone on every node kind, RenderState helpers, assertions.

import (
	"sync"
	"testing"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// guardFuncs maps upstream guard names to the Go functions.
var guardFuncs = map[string]func(luau.Node) bool{
	"isAnyIdentifier":            luau.IsAnyIdentifier,
	"isIdentifier":               luau.IsIdentifier,
	"isTemporaryIdentifier":      luau.IsTemporaryIdentifier,
	"isComputedIndexExpression":  luau.IsComputedIndexExpression,
	"isPropertyAccessExpression": luau.IsPropertyAccessExpression,
	"isCallExpression":           luau.IsCallExpression,
	"isMethodCallExpression":     luau.IsMethodCallExpression,
	"isParenthesizedExpression":  luau.IsParenthesizedExpression,
	"isIndexableExpression":      luau.IsIndexableExpression,
	"isNone":                     luau.IsNone,
	"isNilLiteral":               luau.IsNilLiteral,
	"isFalseLiteral":             luau.IsFalseLiteral,
	"isTrueLiteral":              luau.IsTrueLiteral,
	"isNumberLiteral":            luau.IsNumberLiteral,
	"isStringLiteral":            luau.IsStringLiteral,
	"isVarArgsLiteral":           luau.IsVarArgsLiteral,
	"isFunctionExpression":       luau.IsFunctionExpression,
	"isBinaryExpression":         luau.IsBinaryExpression,
	"isUnaryExpression":          luau.IsUnaryExpression,
	"isIfExpression":             luau.IsIfExpression,
	"isInterpolatedString":       luau.IsInterpolatedString,
	"isArray":                    luau.IsArray,
	"isMap":                      luau.IsMap,
	"isSet":                      luau.IsSet,
	"isMixedTable":               luau.IsMixedTable,
	"isExpression":               luau.IsExpression,
	"isAssignment":               luau.IsAssignment,
	"isBreakStatement":           luau.IsBreakStatement,
	"isCallStatement":            luau.IsCallStatement,
	"isContinueStatement":        luau.IsContinueStatement,
	"isDoStatement":              luau.IsDoStatement,
	"isWhileStatement":           luau.IsWhileStatement,
	"isRepeatStatement":          luau.IsRepeatStatement,
	"isIfStatement":              luau.IsIfStatement,
	"isNumericForStatement":      luau.IsNumericForStatement,
	"isForStatement":             luau.IsForStatement,
	"isFunctionDeclaration":      luau.IsFunctionDeclaration,
	"isMethodDeclaration":        luau.IsMethodDeclaration,
	"isVariableDeclaration":      luau.IsVariableDeclaration,
	"isReturnStatement":          luau.IsReturnStatement,
	"isComment":                  luau.IsComment,
	"isStatement":                luau.IsStatement,
	"isMapField":                 luau.IsMapField,
	"isInterpolatedStringPart":   luau.IsInterpolatedStringPart,
	"isField":                    luau.IsField,
	"isSimple":                   luau.IsSimple,
	"isSimplePrimitive":          luau.IsSimplePrimitive,
	"isTable":                    luau.IsTable,
	"isFinalStatement":           luau.IsFinalStatement,
	"isCall":                     luau.IsCall,
	"isWritableExpression":       luau.IsWritableExpression,
	"isFunctionLike":             luau.IsFunctionLike,
	"hasStatements":              luau.HasStatementsKind,
	"isExpressionWithPrecedence": luau.IsExpressionWithPrecedence,
}

func TestKindGuardsMatchUpstream(t *testing.T) {
	var g struct {
		Guards struct {
			Kinds  []string          `json:"kinds"`
			Guards map[string][]bool `json:"guards"`
			IsNode []bool            `json:"isNode"`
		} `json:"guards"`
	}
	readGolden(t, "util.json", &g)
	if len(g.Guards.Guards) != len(guardFuncs) {
		t.Errorf("upstream has %d kind guards, Go test table has %d", len(g.Guards.Guards), len(guardFuncs))
	}
	for name, results := range g.Guards.Guards {
		fn, ok := guardFuncs[name]
		if !ok {
			t.Errorf("guard %s has no Go counterpart", name)
			continue
		}
		for kind, want := range results {
			node := kindDefs[g.Guards.Kinds[kind]].make()
			if got := fn(node); got != want {
				t.Errorf("%s(%s) = %v, upstream says %v", name, g.Guards.Kinds[kind], got, want)
			}
		}
		if fn(nil) {
			t.Errorf("%s(nil) is true", name)
		}
	}
	if luau.IsNode(nil) || !luau.IsNode(luau.NewNil()) || luau.IsNode(luau.NewList()) {
		t.Error("IsNode")
	}
}

// children lists the child nodes and lists of a node through the test field table.
func children(n luau.Node) (nodes []luau.Node, lists []*luau.List) {
	for _, fd := range kindDefs[kindNameByKind[n.Kind()]].fields {
		v := fd.get(n)
		switch c := v.(type) {
		case luau.Node:
			nodes = append(nodes, c)
		case *luau.List:
			lists = append(lists, c)
		}
	}
	return
}

// Every node kind must shallow-clone, and Create must adopt and clone its
// children exactly like upstream create().
func TestCreateAndShallowCloneEveryKind(t *testing.T) {
	var f goldenFile
	readGolden(t, "handwritten.json", &f)
	seenKinds := map[string]bool{}
	for ci := range f.Cases {
		c := &f.Cases[ci]
		l := loadGraph(t, &c.Graph)
		for _, n := range l.nodes {
			kind := kindNameByKind[n.Kind()]
			if seenKinds[kind] {
				continue
			}
			kids, kidLists := children(n)
			seenKinds[kind] = true

			// ShallowClone: distinct struct, same parent, same children.
			cl := luau.ShallowClone(n)
			if cl == n || cl.Kind() != n.Kind() || cl.Parent() != n.Parent() {
				t.Errorf("%s: ShallowClone", kind)
			}
			ckids, clists := children(cl)
			for i := range kids {
				if ckids[i] != kids[i] {
					t.Errorf("%s: clone child %d is not shared", kind, i)
				}
			}
			for i := range kidLists {
				if clists[i] != kidLists[i] {
					t.Errorf("%s: clone list %d is not shared", kind, i)
				}
			}

			// Create adopts parentless children in place.
			for _, k := range kids {
				k.SetParent(nil)
			}
			for _, kl := range kidLists {
				kl.ForEachListNode(func(ln *luau.ListNode) { ln.Value.SetParent(nil) })
			}
			first := luau.ShallowClone(n)
			if luau.Create(first) != first {
				t.Errorf("%s: Create must return its argument", kind)
			}
			fkids, flists := children(first)
			for i, k := range fkids {
				if k != kids[i] || k.Parent() != first {
					t.Errorf("%s: child %d not adopted in place", kind, i)
				}
			}
			var before [][]luau.Node
			for i, fl := range flists {
				before = append(before, fl.ToArray())
				for _, v := range before[i] {
					if v.Parent() != first {
						t.Errorf("%s: list %d element not adopted", kind, i)
					}
				}
			}

			// A second Create on a copy finds parents everywhere and clones.
			second := luau.ShallowClone(first)
			luau.Create(second)
			skids, slists := children(second)
			for i, k := range skids {
				if k == fkids[i] || k.Kind() != fkids[i].Kind() || k.Parent() != second || fkids[i].Parent() != first {
					t.Errorf("%s: child %d not cloned on second adopt", kind, i)
				}
			}
			for i, sl := range slists {
				if sl != flists[i] {
					t.Errorf("%s: list %d should still be shared", kind, i)
				}
				for j, v := range sl.ToArray() {
					if v == before[i][j] || v.Kind() != before[i][j].Kind() || v.Parent() != second || before[i][j].Parent() != first {
						t.Errorf("%s: list %d element %d not cloned on second adopt", kind, i, j)
					}
				}
			}
		}
	}
	if len(seenKinds) != len(kindDefs) {
		t.Errorf("exercised %d kinds, expected %d", len(seenKinds), len(kindDefs))
	}
}

func TestRenderStateHelpers(t *testing.T) {
	s := luau.NewRenderState()
	if s.Newline("x") != "x\n" || s.Indented("x") != "x" || s.Line("x", nil) != "x\n" {
		t.Error("indent 0")
	}
	s.Block(func() {
		if s.Indented("x") != "\tx" || s.Line("x", nil) != "\tx\n" {
			t.Error("indent 1")
		}
		s.Block(func() {
			if s.Indented("x") != "\t\tx" {
				t.Error("indent 2")
			}
		})
	})
	if s.Indented("x") != "x" {
		t.Error("indent restored")
	}

	// Line adds ";" only when the stack says the next statement starts with "(".
	call := luau.Create(&luau.CallStatement{Expression: luau.NewCall(id("f"))})
	paren := luau.Create(&luau.CallStatement{Expression: luau.NewCall(luau.Create(&luau.ParenthesizedExpression{Expression: id("g")}))})
	stmts := luau.MakeList[luau.Node](call, paren)
	if s.Line("f()", call) != "f()\n" {
		t.Error("no stack, no semicolon")
	}
	s.PushListNode(stmts.Head)
	if got := s.Line("f()", call); got != "f();\n" {
		t.Errorf("Line = %q", got)
	}
	if s.PeekListNode() != stmts.Head || s.PopListNode() != stmts.Head || s.PeekListNode() != nil || s.PopListNode() != nil {
		t.Error("list node stack")
	}

	// GetTempName falls back to _0, _1, ... and remembers.
	a, b := luau.NewTempID("a"), luau.NewTempID("b")
	if s.GetTempName(a) != "_0" || s.GetTempName(b) != "_1" || s.GetTempName(a) != "_0" {
		t.Error("GetTempName")
	}
	zero := &luau.RenderState{}
	if zero.GetTempName(a) != "_0" {
		t.Error("zero RenderState")
	}
}

func TestNewTempIDIncrements(t *testing.T) {
	a, b := luau.NewTempID("x"), luau.NewTempID("")
	if b.ID != a.ID+1 || a.Name != "x" || b.Name != "" {
		t.Errorf("ids %d %d", a.ID, b.ID)
	}
	// safe to call from several goroutines
	var wg sync.WaitGroup
	ids := make([]int, 64)
	for i := range ids {
		wg.Add(1)
		go func() { defer wg.Done(); ids[i] = luau.NewTempID("p").ID }()
	}
	wg.Wait()
	seen := map[int]bool{}
	for _, v := range ids {
		if seen[v] {
			t.Fatalf("duplicate temp id %d", v)
		}
		seen[v] = true
	}
}

func TestAssertionError(t *testing.T) {
	msg := assertionOf(func() { luau.Assert(false, "boom") })
	want := "Assertion Failed! boom\nThis is a compiler bug! Please submit a bug report here:\nhttps://github.com/roblox-ts/roblox-ts/issues"
	if msg != want {
		t.Errorf("got %q", msg)
	}
	luau.Assert(true, "fine")
}

func TestRenderPanicsOnNilAndUnsolvedState(t *testing.T) {
	if assertionOf(func() { luau.Render(luau.NewRenderState(), nil) }) == "" {
		t.Error("nil node should assert")
	}
	// Render and RenderStatements leave the state reusable.
	s := luau.NewRenderState()
	l := luau.MakeList[luau.Node](luau.Create(&luau.CallStatement{Expression: luau.NewCall(id("f"))}))
	one := luau.RenderStatements(s, l)
	two := luau.RenderStatements(s, l)
	if one != "f()\n" || one != two {
		t.Errorf("%q %q", one, two)
	}
	if got := luau.Render(s, luau.NewBinary(id("a"), "+", id("b"))); got != "a + b" {
		t.Errorf("Render = %q", got)
	}
}

func TestSolveTempIDsOnNode(t *testing.T) {
	// ast may be a single node as well as a list
	tmp := luau.NewTempID("v")
	fn := luau.Create(&luau.FunctionExpression{Parameters: luau.MakeList[luau.Node](id("_v")), Statements: luau.MakeList[luau.Node](
		luau.Create(&luau.VariableDeclaration{Left: tmp, Right: luau.NewNil()}))})
	s := luau.NewRenderState()
	luau.SolveTempIDs(s, fn)
	if s.SeenTempNodes[tmp.ID] != "_v_1" {
		t.Errorf("got %v", s.SeenTempNodes)
	}
	// already named ids are kept
	s2 := luau.NewRenderState()
	s2.SeenTempNodes[tmp.ID] = "_kept"
	luau.SolveTempIDs(s2, fn)
	if s2.SeenTempNodes[tmp.ID] != "_kept" {
		t.Error("existing name overwritten")
	}
	luau.SolveTempIDs(&luau.RenderState{}, nil) // nothing to visit
}

func TestListHelpers(t *testing.T) {
	a, b := id("a"), id("b")
	l := luau.NewList()
	if l.Shift() != nil || l.Size() != 0 || !l.IsEmpty() {
		t.Error("empty list")
	}
	n := luau.MakeListNode(a)
	if n.Value != a {
		t.Error("MakeListNode")
	}
	if assertionOf(func() { luau.MakeListNode(nil) }) == "" {
		t.Error("MakeListNode(nil) should assert")
	}
	l = luau.MakeList[luau.Node](a, b)
	if got, ok := luau.AsList(l); !ok || got != l {
		t.Error("AsList")
	}
	if _, ok := luau.AsList(a); ok {
		t.Error("AsList(node)")
	}
	var seen []luau.Node
	l.ForEachListNode(func(ln *luau.ListNode) { seen = append(seen, ln.Value) })
	if len(seen) != 2 || seen[0] != a || seen[1] != b {
		t.Error("ForEachListNode")
	}
	names := luau.MapToArray(l, func(n luau.Node) string { return n.(*luau.Identifier).Name })
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Error("MapToArray")
	}
}

func TestVisitOrder(t *testing.T) {
	// before/after pairs wrap children in tree order
	a, b := id("a"), id("b")
	root := luau.Create(&luau.BinaryExpression{Left: a, Operator: "+", Right: b})
	var trace []string
	luau.Visit(root, &luau.Visitor{
		Before: func(n luau.Node) { trace = append(trace, "<"+luau.GetKindName(n.Kind())) },
		After:  func(n luau.Node) { trace = append(trace, ">"+luau.GetKindName(n.Kind())) },
	})
	want := []string{"<BinaryExpression", "<Identifier", ">Identifier", "<Identifier", ">Identifier", ">BinaryExpression"}
	if len(trace) != len(want) {
		t.Fatalf("%v", trace)
	}
	for i := range want {
		if trace[i] != want[i] {
			t.Fatalf("%v", trace)
		}
	}
	luau.Visit(nil, &luau.Visitor{})
	luau.Visit(luau.MakeList[luau.Node](a), &luau.Visitor{})
}

func TestCreateToleratesNilLists(t *testing.T) {
	// A nil list field is skipped rather than crashing; rendering it would.
	arr := luau.Create(&luau.Array{})
	if arr.Members != nil {
		t.Error("Members should stay nil")
	}
}

// HasStatements and HasParameters must be implemented by exactly the kinds
// upstream's hasStatements and isFunctionLike guards select.
func TestHasStatementsAndParameters(t *testing.T) {
	for name, kd := range kindDefs {
		n := kd.make()
		hs, isHS := n.(luau.HasStatements)
		if isHS != luau.HasStatementsKind(n) {
			t.Errorf("%s: HasStatements %v, guard %v", name, isHS, luau.HasStatementsKind(n))
		}
		hp, isHP := n.(luau.HasParameters)
		if isHP != luau.IsFunctionLike(n) {
			t.Errorf("%s: HasParameters %v, guard %v", name, isHP, luau.IsFunctionLike(n))
		}
		if isHS {
			if hs.GetStatements() != nil {
				t.Errorf("%s: zero value has statements", name)
			}
			list := luau.NewList()
			for _, fd := range kd.fields {
				if fd.name == "statements" {
					fd.set(n, list)
				}
			}
			if hs.GetStatements() != list {
				t.Errorf("%s: GetStatements", name)
			}
		}
		if isHP {
			for _, fd := range kd.fields {
				switch fd.name {
				case "hasDotDotDot":
					fd.set(n, true)
				case "parameters":
					fd.set(n, luau.NewList())
				}
			}
			if !hp.GetHasDotDotDot() || hp.GetParameters() == nil {
				t.Errorf("%s: GetHasDotDotDot/GetParameters", name)
			}
		}
	}
}
