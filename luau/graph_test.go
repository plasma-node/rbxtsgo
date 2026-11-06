package luau_test

// Test-only (de)serialization of Luau AST object graphs. The JSON format is
// produced by testdata/graph.js from the real npm package: nodes, lists and
// list nodes keep their identity, parent pointers and sharing, so the Go
// renderer sees exactly the structure the JS renderer saw.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

type fieldType int

const (
	fNode fieldType = iota
	fList
	fNodeOrList
	fString
	fBool
	fInt
)

type fieldDef struct {
	name string
	typ  fieldType
	set  func(n luau.Node, v any)
	get  func(n luau.Node) any
}

type kindDef struct {
	make   func() luau.Node
	fields []fieldDef
}

// converters from decoded graph values to the typed fields.
func asExpr(v any) luau.Expression {
	if v == nil {
		return nil
	}
	return v.(luau.Expression)
}
func asIndexable(v any) luau.IndexableExpression {
	if v == nil {
		return nil
	}
	return v.(luau.IndexableExpression)
}
func asAnyID(v any) luau.AnyIdentifier {
	if v == nil {
		return nil
	}
	return v.(luau.AnyIdentifier)
}
func asWritable(v any) luau.WritableExpression {
	if v == nil {
		return nil
	}
	return v.(luau.WritableExpression)
}
func asCall(v any) luau.CallLike {
	if v == nil {
		return nil
	}
	return v.(luau.CallLike)
}
func asNL(v any) luau.NodeOrList {
	if v == nil {
		return nil
	}
	return v.(luau.NodeOrList)
}
func asList(v any) *luau.List {
	if v == nil {
		return nil
	}
	return v.(*luau.List)
}

// nodeAny converts a typed interface field to a plain `any` that is nil when
// the field is unset.
func nodeAny[T comparable](v T) any {
	var zero T
	if v == zero {
		return nil
	}
	return v
}

func nodeField[N luau.Node](name string, set func(n N, v any), get func(n N) any) fieldDef {
	return fieldDef{name: name, typ: fNode, set: func(n luau.Node, v any) { set(n.(N), v) }, get: func(n luau.Node) any { return get(n.(N)) }}
}
func listField[N luau.Node](name string, p func(n N) **luau.List) fieldDef {
	return fieldDef{name: name, typ: fList,
		set: func(n luau.Node, v any) { *p(n.(N)) = asList(v) },
		get: func(n luau.Node) any {
			l := *p(n.(N))
			if l == nil {
				return nil
			}
			return l
		}}
}
func nlField[N luau.Node](name string, p func(n N) *luau.NodeOrList) fieldDef {
	return fieldDef{name: name, typ: fNodeOrList,
		set: func(n luau.Node, v any) { *p(n.(N)) = asNL(v) },
		get: func(n luau.Node) any { return nodeAny(*p(n.(N))) }}
}
func stringField[N luau.Node](name string, p func(n N) *string) fieldDef {
	return fieldDef{name: name, typ: fString,
		set: func(n luau.Node, v any) { *p(n.(N)) = v.(string) },
		get: func(n luau.Node) any { return *p(n.(N)) }}
}
func boolField[N luau.Node](name string, p func(n N) *bool) fieldDef {
	return fieldDef{name: name, typ: fBool,
		set: func(n luau.Node, v any) { *p(n.(N)) = v.(bool) },
		get: func(n luau.Node) any { return *p(n.(N)) }}
}

func def[N luau.Node](make func() N, fields ...fieldDef) kindDef {
	return kindDef{make: func() luau.Node { return make() }, fields: fields}
}

var kindDefs = map[string]kindDef{
	"Identifier": def(func() *luau.Identifier { return &luau.Identifier{} },
		stringField("name", func(n *luau.Identifier) *string { return &n.Name })),
	"TemporaryIdentifier": def(func() *luau.TemporaryIdentifier { return &luau.TemporaryIdentifier{} },
		stringField("name", func(n *luau.TemporaryIdentifier) *string { return &n.Name }),
		fieldDef{name: "id", typ: fInt,
			set: func(n luau.Node, v any) { n.(*luau.TemporaryIdentifier).ID = int(v.(float64)) },
			get: func(n luau.Node) any { return n.(*luau.TemporaryIdentifier).ID }}),
	"ComputedIndexExpression": def(func() *luau.ComputedIndexExpression { return &luau.ComputedIndexExpression{} },
		nodeField("expression", func(n *luau.ComputedIndexExpression, v any) { n.Expression = asIndexable(v) }, func(n *luau.ComputedIndexExpression) any { return nodeAny(n.Expression) }),
		nodeField("index", func(n *luau.ComputedIndexExpression, v any) { n.Index = asExpr(v) }, func(n *luau.ComputedIndexExpression) any { return nodeAny(n.Index) })),
	"PropertyAccessExpression": def(func() *luau.PropertyAccessExpression { return &luau.PropertyAccessExpression{} },
		nodeField("expression", func(n *luau.PropertyAccessExpression, v any) { n.Expression = asIndexable(v) }, func(n *luau.PropertyAccessExpression) any { return nodeAny(n.Expression) }),
		stringField("name", func(n *luau.PropertyAccessExpression) *string { return &n.Name })),
	"CallExpression": def(func() *luau.CallExpression { return &luau.CallExpression{} },
		nodeField("expression", func(n *luau.CallExpression, v any) { n.Expression = asIndexable(v) }, func(n *luau.CallExpression) any { return nodeAny(n.Expression) }),
		listField("args", func(n *luau.CallExpression) **luau.List { return &n.Args })),
	"MethodCallExpression": def(func() *luau.MethodCallExpression { return &luau.MethodCallExpression{} },
		stringField("name", func(n *luau.MethodCallExpression) *string { return &n.Name }),
		nodeField("expression", func(n *luau.MethodCallExpression, v any) { n.Expression = asIndexable(v) }, func(n *luau.MethodCallExpression) any { return nodeAny(n.Expression) }),
		listField("args", func(n *luau.MethodCallExpression) **luau.List { return &n.Args })),
	"ParenthesizedExpression": def(func() *luau.ParenthesizedExpression { return &luau.ParenthesizedExpression{} },
		nodeField("expression", func(n *luau.ParenthesizedExpression, v any) { n.Expression = asExpr(v) }, func(n *luau.ParenthesizedExpression) any { return nodeAny(n.Expression) })),
	"None":           def(func() *luau.None { return &luau.None{} }),
	"NilLiteral":     def(func() *luau.NilLiteral { return &luau.NilLiteral{} }),
	"FalseLiteral":   def(func() *luau.FalseLiteral { return &luau.FalseLiteral{} }),
	"TrueLiteral":    def(func() *luau.TrueLiteral { return &luau.TrueLiteral{} }),
	"VarArgsLiteral": def(func() *luau.VarArgsLiteral { return &luau.VarArgsLiteral{} }),
	"NumberLiteral": def(func() *luau.NumberLiteral { return &luau.NumberLiteral{} },
		stringField("value", func(n *luau.NumberLiteral) *string { return &n.Value })),
	"StringLiteral": def(func() *luau.StringLiteral { return &luau.StringLiteral{} },
		stringField("value", func(n *luau.StringLiteral) *string { return &n.Value })),
	"FunctionExpression": def(func() *luau.FunctionExpression { return &luau.FunctionExpression{} },
		listField("parameters", func(n *luau.FunctionExpression) **luau.List { return &n.Parameters }),
		boolField("hasDotDotDot", func(n *luau.FunctionExpression) *bool { return &n.HasDotDotDot }),
		listField("statements", func(n *luau.FunctionExpression) **luau.List { return &n.Statements })),
	"BinaryExpression": def(func() *luau.BinaryExpression { return &luau.BinaryExpression{} },
		nodeField("left", func(n *luau.BinaryExpression, v any) { n.Left = asExpr(v) }, func(n *luau.BinaryExpression) any { return nodeAny(n.Left) }),
		fieldDef{name: "operator", typ: fString,
			set: func(n luau.Node, v any) { n.(*luau.BinaryExpression).Operator = luau.BinaryOperator(v.(string)) },
			get: func(n luau.Node) any { return string(n.(*luau.BinaryExpression).Operator) }},
		nodeField("right", func(n *luau.BinaryExpression, v any) { n.Right = asExpr(v) }, func(n *luau.BinaryExpression) any { return nodeAny(n.Right) })),
	"UnaryExpression": def(func() *luau.UnaryExpression { return &luau.UnaryExpression{} },
		fieldDef{name: "operator", typ: fString,
			set: func(n luau.Node, v any) { n.(*luau.UnaryExpression).Operator = luau.UnaryOperator(v.(string)) },
			get: func(n luau.Node) any { return string(n.(*luau.UnaryExpression).Operator) }},
		nodeField("expression", func(n *luau.UnaryExpression, v any) { n.Expression = asExpr(v) }, func(n *luau.UnaryExpression) any { return nodeAny(n.Expression) })),
	"IfExpression": def(func() *luau.IfExpression { return &luau.IfExpression{} },
		nodeField("condition", func(n *luau.IfExpression, v any) { n.Condition = asExpr(v) }, func(n *luau.IfExpression) any { return nodeAny(n.Condition) }),
		nodeField("expression", func(n *luau.IfExpression, v any) { n.Expression = asExpr(v) }, func(n *luau.IfExpression) any { return nodeAny(n.Expression) }),
		nodeField("alternative", func(n *luau.IfExpression, v any) { n.Alternative = asExpr(v) }, func(n *luau.IfExpression) any { return nodeAny(n.Alternative) })),
	"InterpolatedString": def(func() *luau.InterpolatedString { return &luau.InterpolatedString{} },
		listField("parts", func(n *luau.InterpolatedString) **luau.List { return &n.Parts })),
	"Array": def(func() *luau.Array { return &luau.Array{} },
		listField("members", func(n *luau.Array) **luau.List { return &n.Members })),
	"Map": def(func() *luau.Map { return &luau.Map{} },
		listField("fields", func(n *luau.Map) **luau.List { return &n.Fields })),
	"Set": def(func() *luau.Set { return &luau.Set{} },
		listField("members", func(n *luau.Set) **luau.List { return &n.Members })),
	"MixedTable": def(func() *luau.MixedTable { return &luau.MixedTable{} },
		listField("fields", func(n *luau.MixedTable) **luau.List { return &n.Fields })),
	"Assignment": def(func() *luau.Assignment { return &luau.Assignment{} },
		nlField("left", func(n *luau.Assignment) *luau.NodeOrList { return &n.Left }),
		fieldDef{name: "operator", typ: fString,
			set: func(n luau.Node, v any) { n.(*luau.Assignment).Operator = luau.AssignmentOperator(v.(string)) },
			get: func(n luau.Node) any { return string(n.(*luau.Assignment).Operator) }},
		nlField("right", func(n *luau.Assignment) *luau.NodeOrList { return &n.Right })),
	"BreakStatement": def(func() *luau.BreakStatement { return &luau.BreakStatement{} }),
	"CallStatement": def(func() *luau.CallStatement { return &luau.CallStatement{} },
		nodeField("expression", func(n *luau.CallStatement, v any) { n.Expression = asCall(v) }, func(n *luau.CallStatement) any { return nodeAny(n.Expression) })),
	"ContinueStatement": def(func() *luau.ContinueStatement { return &luau.ContinueStatement{} }),
	"DoStatement": def(func() *luau.DoStatement { return &luau.DoStatement{} },
		listField("statements", func(n *luau.DoStatement) **luau.List { return &n.Statements })),
	"WhileStatement": def(func() *luau.WhileStatement { return &luau.WhileStatement{} },
		nodeField("condition", func(n *luau.WhileStatement, v any) { n.Condition = asExpr(v) }, func(n *luau.WhileStatement) any { return nodeAny(n.Condition) }),
		listField("statements", func(n *luau.WhileStatement) **luau.List { return &n.Statements })),
	"RepeatStatement": def(func() *luau.RepeatStatement { return &luau.RepeatStatement{} },
		nodeField("condition", func(n *luau.RepeatStatement, v any) { n.Condition = asExpr(v) }, func(n *luau.RepeatStatement) any { return nodeAny(n.Condition) }),
		listField("statements", func(n *luau.RepeatStatement) **luau.List { return &n.Statements })),
	"IfStatement": def(func() *luau.IfStatement { return &luau.IfStatement{} },
		nodeField("condition", func(n *luau.IfStatement, v any) { n.Condition = asExpr(v) }, func(n *luau.IfStatement) any { return nodeAny(n.Condition) }),
		listField("statements", func(n *luau.IfStatement) **luau.List { return &n.Statements }),
		nlField("elseBody", func(n *luau.IfStatement) *luau.NodeOrList { return &n.ElseBody })),
	"NumericForStatement": def(func() *luau.NumericForStatement { return &luau.NumericForStatement{} },
		nodeField("id", func(n *luau.NumericForStatement, v any) { n.ID = asAnyID(v) }, func(n *luau.NumericForStatement) any { return nodeAny(n.ID) }),
		nodeField("start", func(n *luau.NumericForStatement, v any) { n.Start = asExpr(v) }, func(n *luau.NumericForStatement) any { return nodeAny(n.Start) }),
		nodeField("end", func(n *luau.NumericForStatement, v any) { n.End = asExpr(v) }, func(n *luau.NumericForStatement) any { return nodeAny(n.End) }),
		nodeField("step", func(n *luau.NumericForStatement, v any) { n.Step = asExpr(v) }, func(n *luau.NumericForStatement) any { return nodeAny(n.Step) }),
		listField("statements", func(n *luau.NumericForStatement) **luau.List { return &n.Statements })),
	"ForStatement": def(func() *luau.ForStatement { return &luau.ForStatement{} },
		listField("ids", func(n *luau.ForStatement) **luau.List { return &n.IDs }),
		nodeField("expression", func(n *luau.ForStatement, v any) { n.Expression = asExpr(v) }, func(n *luau.ForStatement) any { return nodeAny(n.Expression) }),
		listField("statements", func(n *luau.ForStatement) **luau.List { return &n.Statements })),
	"FunctionDeclaration": def(func() *luau.FunctionDeclaration { return &luau.FunctionDeclaration{} },
		boolField("localize", func(n *luau.FunctionDeclaration) *bool { return &n.Localize }),
		nodeField("name", func(n *luau.FunctionDeclaration, v any) { n.Name = asWritable(v) }, func(n *luau.FunctionDeclaration) any { return nodeAny(n.Name) }),
		listField("parameters", func(n *luau.FunctionDeclaration) **luau.List { return &n.Parameters }),
		boolField("hasDotDotDot", func(n *luau.FunctionDeclaration) *bool { return &n.HasDotDotDot }),
		listField("statements", func(n *luau.FunctionDeclaration) **luau.List { return &n.Statements })),
	"MethodDeclaration": def(func() *luau.MethodDeclaration { return &luau.MethodDeclaration{} },
		nodeField("expression", func(n *luau.MethodDeclaration, v any) { n.Expression = asIndexable(v) }, func(n *luau.MethodDeclaration) any { return nodeAny(n.Expression) }),
		stringField("name", func(n *luau.MethodDeclaration) *string { return &n.Name }),
		listField("parameters", func(n *luau.MethodDeclaration) **luau.List { return &n.Parameters }),
		boolField("hasDotDotDot", func(n *luau.MethodDeclaration) *bool { return &n.HasDotDotDot }),
		listField("statements", func(n *luau.MethodDeclaration) **luau.List { return &n.Statements })),
	"VariableDeclaration": def(func() *luau.VariableDeclaration { return &luau.VariableDeclaration{} },
		nlField("left", func(n *luau.VariableDeclaration) *luau.NodeOrList { return &n.Left }),
		nlField("right", func(n *luau.VariableDeclaration) *luau.NodeOrList { return &n.Right })),
	"ReturnStatement": def(func() *luau.ReturnStatement { return &luau.ReturnStatement{} },
		nlField("expression", func(n *luau.ReturnStatement) *luau.NodeOrList { return &n.Expression })),
	"Comment": def(func() *luau.Comment { return &luau.Comment{} },
		stringField("text", func(n *luau.Comment) *string { return &n.Text })),
	"MapField": def(func() *luau.MapField { return &luau.MapField{} },
		nodeField("index", func(n *luau.MapField, v any) { n.Index = asExpr(v) }, func(n *luau.MapField) any { return nodeAny(n.Index) }),
		nodeField("value", func(n *luau.MapField, v any) { n.Value = asExpr(v) }, func(n *luau.MapField) any { return nodeAny(n.Value) })),
	"InterpolatedStringPart": def(func() *luau.InterpolatedStringPart { return &luau.InterpolatedStringPart{} },
		stringField("text", func(n *luau.InterpolatedStringPart) *string { return &n.Text })),
}

// kindName looks up the JSON kind name of a node (upstream SyntaxKind names).
var kindNameByKind = map[luau.SyntaxKind]string{
	luau.KindIdentifier: "Identifier", luau.KindTemporaryIdentifier: "TemporaryIdentifier",
	luau.KindComputedIndexExpression: "ComputedIndexExpression", luau.KindPropertyAccessExpression: "PropertyAccessExpression",
	luau.KindCallExpression: "CallExpression", luau.KindMethodCallExpression: "MethodCallExpression",
	luau.KindParenthesizedExpression: "ParenthesizedExpression", luau.KindNone: "None", luau.KindNilLiteral: "NilLiteral",
	luau.KindFalseLiteral: "FalseLiteral", luau.KindTrueLiteral: "TrueLiteral", luau.KindNumberLiteral: "NumberLiteral",
	luau.KindStringLiteral: "StringLiteral", luau.KindVarArgsLiteral: "VarArgsLiteral", luau.KindFunctionExpression: "FunctionExpression",
	luau.KindBinaryExpression: "BinaryExpression", luau.KindUnaryExpression: "UnaryExpression", luau.KindIfExpression: "IfExpression",
	luau.KindInterpolatedString: "InterpolatedString", luau.KindArray: "Array", luau.KindMap: "Map", luau.KindSet: "Set",
	luau.KindMixedTable: "MixedTable", luau.KindAssignment: "Assignment", luau.KindBreakStatement: "BreakStatement",
	luau.KindCallStatement: "CallStatement", luau.KindContinueStatement: "ContinueStatement", luau.KindDoStatement: "DoStatement",
	luau.KindWhileStatement: "WhileStatement", luau.KindRepeatStatement: "RepeatStatement", luau.KindIfStatement: "IfStatement",
	luau.KindNumericForStatement: "NumericForStatement", luau.KindForStatement: "ForStatement",
	luau.KindFunctionDeclaration: "FunctionDeclaration", luau.KindMethodDeclaration: "MethodDeclaration",
	luau.KindVariableDeclaration: "VariableDeclaration", luau.KindReturnStatement: "ReturnStatement", luau.KindComment: "Comment",
	luau.KindMapField: "MapField", luau.KindInterpolatedStringPart: "InterpolatedStringPart",
}

// ---------------------------------------------------------------------------
// Loading

type graphJSON struct {
	Nodes     []map[string]any `json:"nodes"`
	Lists     []map[string]any `json:"lists"`
	ListNodes []map[string]any `json:"listNodes"`
	Roots     map[string]any   `json:"roots"`
}

type loaded struct {
	nodes []luau.Node
	lists []*luau.List
	lns   []*luau.ListNode
}

func (l *loaded) ref(v any) any {
	switch r := v.(type) {
	case nil:
		return nil
	case map[string]any:
		if n, ok := r["$n"]; ok {
			return l.nodes[int(n.(float64))]
		}
		if n, ok := r["$l"]; ok {
			return l.lists[int(n.(float64))]
		}
	}
	panic(fmt.Sprintf("bad ref %v", v))
}

func (l *loaded) listNode(v any) *luau.ListNode {
	if v == nil {
		return nil
	}
	return l.lns[int(v.(float64))]
}

func loadGraph(t testing.TB, g *graphJSON) *loaded {
	t.Helper()
	l := &loaded{}
	for _, nd := range g.Nodes {
		kind := nd["kind"].(string)
		kd, ok := kindDefs[kind]
		if !ok {
			t.Fatalf("unknown kind %q", kind)
		}
		l.nodes = append(l.nodes, kd.make())
	}
	for range g.Lists {
		l.lists = append(l.lists, &luau.List{})
	}
	for range g.ListNodes {
		l.lns = append(l.lns, &luau.ListNode{})
	}
	for i, nd := range g.Nodes {
		n := l.nodes[i]
		kd := kindDefs[nd["kind"].(string)]
		if p, ok := nd["parent"]; ok {
			n.SetParent(l.ref(p).(luau.Node))
		}
		for _, fd := range kd.fields {
			v, ok := nd[fd.name]
			if !ok {
				continue
			}
			switch fd.typ {
			case fNode, fList, fNodeOrList:
				fd.set(n, l.ref(v))
			default:
				fd.set(n, v)
			}
		}
	}
	for i, ld := range g.Lists {
		list := l.lists[i]
		list.Head = l.listNode(ld["head"])
		list.Tail = l.listNode(ld["tail"])
		list.ReadOnly = ld["readonly"].(bool)
	}
	for i, ln := range g.ListNodes {
		node := l.lns[i]
		node.Next = l.listNode(ln["next"])
		node.Prev = l.listNode(ln["prev"])
		node.Value = l.ref(ln["value"]).(luau.Node)
	}
	return l
}

func (l *loaded) root(v any) luau.NodeOrList {
	return l.ref(v).(luau.NodeOrList)
}

// ---------------------------------------------------------------------------
// Dumping (same algorithm and ordering as testdata/graph.js)

type ordered struct {
	name string
	v    luau.NodeOrList
}

type dumper struct {
	nodes     []map[string]any
	lists     []map[string]any
	listNodes []map[string]any
	nodeIDs   map[luau.Node]int
	listIDs   map[*luau.List]int
	lnIDs     map[*luau.ListNode]int
	queue     []func()
}

func dumpGraph(roots []ordered) graphJSON {
	d := &dumper{nodes: []map[string]any{}, lists: []map[string]any{}, listNodes: []map[string]any{}, nodeIDs: map[luau.Node]int{}, listIDs: map[*luau.List]int{}, lnIDs: map[*luau.ListNode]int{}}
	rootRefs := map[string]any{}
	for _, r := range roots {
		rootRefs[r.name] = d.ref(r.v)
	}
	for i := 0; i < len(d.queue); i++ {
		d.queue[i]()
	}
	return graphJSON{Nodes: d.nodes, Lists: d.lists, ListNodes: d.listNodes, Roots: rootRefs}
}

func (d *dumper) ref(v luau.NodeOrList) any {
	switch x := v.(type) {
	case *luau.List:
		id, ok := d.listIDs[x]
		if !ok {
			id = len(d.lists)
			d.listIDs[x] = id
			d.lists = append(d.lists, nil)
			d.queue = append(d.queue, func() {
				out := map[string]any{}
				out["head"] = d.lnRef(x.Head)
				out["readonly"] = x.ReadOnly
				out["tail"] = d.lnRef(x.Tail)
				d.lists[id] = out
			})
		}
		return map[string]any{"$l": id}
	case luau.Node:
		id, ok := d.nodeIDs[x]
		if !ok {
			id = len(d.nodes)
			d.nodeIDs[x] = id
			d.nodes = append(d.nodes, nil)
			d.queue = append(d.queue, func() { d.nodes[id] = d.dumpNode(x) })
		}
		return map[string]any{"$n": id}
	}
	panic("bad ref")
}

func (d *dumper) lnRef(ln *luau.ListNode) any {
	if ln == nil {
		return nil
	}
	id, ok := d.lnIDs[ln]
	if !ok {
		id = len(d.listNodes)
		d.lnIDs[ln] = id
		d.listNodes = append(d.listNodes, nil)
		d.queue = append(d.queue, func() {
			out := map[string]any{}
			out["next"] = d.lnRef(ln.Next)
			out["prev"] = d.lnRef(ln.Prev)
			out["value"] = d.ref(ln.Value)
			d.listNodes[id] = out
		})
	}
	return id
}

func (d *dumper) dumpNode(n luau.Node) map[string]any {
	kd := kindDefs[kindNameByKind[n.Kind()]]
	vals := map[string]func() any{}
	vals["kind"] = func() any { return kindNameByKind[n.Kind()] }
	vals["parent"] = func() any {
		if p := n.Parent(); p != nil {
			return d.ref(p)
		}
		return nil
	}
	for _, fd := range kd.fields {
		fd := fd
		vals[fd.name] = func() any {
			v := fd.get(n)
			if v == nil {
				return nil
			}
			switch fd.typ {
			case fNode, fList, fNodeOrList:
				return d.ref(v.(luau.NodeOrList))
			}
			return v
		}
	}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := map[string]any{}
	for _, k := range keys {
		if v := vals[k](); v != nil {
			out[k] = v
		}
	}
	return out
}

// normalize round-trips through JSON so numbers and maps have the types a
// decoded golden file has.
func normalize(t testing.TB, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func readGolden(t testing.TB, name string, v any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "golden", name))
	if err != nil {
		t.Fatalf("%v (run `node generate.js` in luau/testdata)", err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

func jsonIndent(t testing.TB, v any) (string, error) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", " ")
	return string(b), err
}
