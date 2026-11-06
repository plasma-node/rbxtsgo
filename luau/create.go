package luau

import "sync/atomic"

// Ported from src/LuauAST/impl/create.ts.

// Create finishes a node built as a struct literal: it gives every child
// (including list elements) this node as its parent and returns the node.
//
// Same rule as upstream `create`: a child that already has a parent is
// replaced by a shallow clone whose parent is the new node (so the original
// keeps its old parent); a child without a parent is adopted in place.
//
//	luau.Create(&luau.BinaryExpression{Left: a, Operator: "+", Right: b})
func Create[T Node](node T) T {
	node.adopt()
	return node
}

// adopt for a single child node field.
func adopt[T Node](parent Node, child T) T {
	var c Node = child
	if c == nil {
		return child
	}
	if c.Parent() != nil {
		clone := c.shallowClone()
		clone.SetParent(parent)
		return clone.(T)
	}
	c.SetParent(parent)
	return child
}

// adoptList adopts every element of a list field. A nil list is skipped.
func adoptList(parent Node, list *List) {
	if list == nil {
		return
	}
	for ln := list.Head; ln != nil; ln = ln.Next {
		if ln.Value.Parent() != nil {
			clone := ln.Value.shallowClone()
			clone.SetParent(parent)
			ln.Value = clone
		} else {
			ln.Value.SetParent(parent)
		}
	}
}

// adoptNodeOrList adopts a field that is either a node or a list of nodes.
func adoptNodeOrList(parent Node, v NodeOrList) NodeOrList {
	switch x := v.(type) {
	case nil:
		return nil
	case *List:
		adoptList(parent, x)
		return x
	case Node:
		return adopt(parent, x)
	}
	return v
}

// ---------------------------------------------------------------------------
// shallowClone: `{ ...node }`

func (n *None) shallowClone() Node                     { c := *n; return &c }
func (n *NilLiteral) shallowClone() Node               { c := *n; return &c }
func (n *FalseLiteral) shallowClone() Node             { c := *n; return &c }
func (n *TrueLiteral) shallowClone() Node              { c := *n; return &c }
func (n *NumberLiteral) shallowClone() Node            { c := *n; return &c }
func (n *StringLiteral) shallowClone() Node            { c := *n; return &c }
func (n *VarArgsLiteral) shallowClone() Node           { c := *n; return &c }
func (n *FunctionExpression) shallowClone() Node       { c := *n; return &c }
func (n *Identifier) shallowClone() Node               { c := *n; return &c }
func (n *TemporaryIdentifier) shallowClone() Node      { c := *n; return &c }
func (n *ComputedIndexExpression) shallowClone() Node  { c := *n; return &c }
func (n *PropertyAccessExpression) shallowClone() Node { c := *n; return &c }
func (n *CallExpression) shallowClone() Node           { c := *n; return &c }
func (n *MethodCallExpression) shallowClone() Node     { c := *n; return &c }
func (n *ParenthesizedExpression) shallowClone() Node  { c := *n; return &c }
func (n *BinaryExpression) shallowClone() Node         { c := *n; return &c }
func (n *UnaryExpression) shallowClone() Node          { c := *n; return &c }
func (n *IfExpression) shallowClone() Node             { c := *n; return &c }
func (n *InterpolatedString) shallowClone() Node       { c := *n; return &c }
func (n *Array) shallowClone() Node                    { c := *n; return &c }
func (n *Map) shallowClone() Node                      { c := *n; return &c }
func (n *Set) shallowClone() Node                      { c := *n; return &c }
func (n *MixedTable) shallowClone() Node               { c := *n; return &c }
func (n *Assignment) shallowClone() Node               { c := *n; return &c }
func (n *BreakStatement) shallowClone() Node           { c := *n; return &c }
func (n *CallStatement) shallowClone() Node            { c := *n; return &c }
func (n *ContinueStatement) shallowClone() Node        { c := *n; return &c }
func (n *DoStatement) shallowClone() Node              { c := *n; return &c }
func (n *WhileStatement) shallowClone() Node           { c := *n; return &c }
func (n *RepeatStatement) shallowClone() Node          { c := *n; return &c }
func (n *IfStatement) shallowClone() Node              { c := *n; return &c }
func (n *NumericForStatement) shallowClone() Node      { c := *n; return &c }
func (n *ForStatement) shallowClone() Node             { c := *n; return &c }
func (n *FunctionDeclaration) shallowClone() Node      { c := *n; return &c }
func (n *MethodDeclaration) shallowClone() Node        { c := *n; return &c }
func (n *VariableDeclaration) shallowClone() Node      { c := *n; return &c }
func (n *ReturnStatement) shallowClone() Node          { c := *n; return &c }
func (n *Comment) shallowClone() Node                  { c := *n; return &c }
func (n *MapField) shallowClone() Node                 { c := *n; return &c }
func (n *InterpolatedStringPart) shallowClone() Node   { c := *n; return &c }

// ---------------------------------------------------------------------------
// adopt: one per kind, children in declaration order (see doc.go)

func (n *None) adopt()                {}
func (n *NilLiteral) adopt()          {}
func (n *FalseLiteral) adopt()        {}
func (n *TrueLiteral) adopt()         {}
func (n *NumberLiteral) adopt()       {}
func (n *StringLiteral) adopt()       {}
func (n *VarArgsLiteral) adopt()      {}
func (n *Identifier) adopt()          {}
func (n *TemporaryIdentifier) adopt() {}
func (n *BreakStatement) adopt()      {}
func (n *ContinueStatement) adopt()   {}
func (n *Comment) adopt()             {}

func (n *InterpolatedStringPart) adopt() {}

func (n *FunctionExpression) adopt() {
	adoptList(n, n.Parameters)
	adoptList(n, n.Statements)
}

func (n *ComputedIndexExpression) adopt() {
	n.Expression = adopt(n, n.Expression)
	n.Index = adopt(n, n.Index)
}

func (n *PropertyAccessExpression) adopt() {
	n.Expression = adopt(n, n.Expression)
}

func (n *CallExpression) adopt() {
	n.Expression = adopt(n, n.Expression)
	adoptList(n, n.Args)
}

func (n *MethodCallExpression) adopt() {
	n.Expression = adopt(n, n.Expression)
	adoptList(n, n.Args)
}

func (n *ParenthesizedExpression) adopt() {
	n.Expression = adopt(n, n.Expression)
}

func (n *BinaryExpression) adopt() {
	n.Left = adopt(n, n.Left)
	n.Right = adopt(n, n.Right)
}

func (n *UnaryExpression) adopt() {
	n.Expression = adopt(n, n.Expression)
}

func (n *IfExpression) adopt() {
	n.Condition = adopt(n, n.Condition)
	n.Expression = adopt(n, n.Expression)
	n.Alternative = adopt(n, n.Alternative)
}

func (n *InterpolatedString) adopt() { adoptList(n, n.Parts) }
func (n *Array) adopt()              { adoptList(n, n.Members) }
func (n *Map) adopt()                { adoptList(n, n.Fields) }
func (n *Set) adopt()                { adoptList(n, n.Members) }
func (n *MixedTable) adopt()         { adoptList(n, n.Fields) }

func (n *Assignment) adopt() {
	n.Left = adoptNodeOrList(n, n.Left)
	n.Right = adoptNodeOrList(n, n.Right)
}

func (n *CallStatement) adopt() {
	n.Expression = adopt(n, n.Expression)
}

func (n *DoStatement) adopt() { adoptList(n, n.Statements) }

func (n *WhileStatement) adopt() {
	n.Condition = adopt(n, n.Condition)
	adoptList(n, n.Statements)
}

func (n *RepeatStatement) adopt() {
	n.Condition = adopt(n, n.Condition)
	adoptList(n, n.Statements)
}

func (n *IfStatement) adopt() {
	n.Condition = adopt(n, n.Condition)
	adoptList(n, n.Statements)
	n.ElseBody = adoptNodeOrList(n, n.ElseBody)
}

func (n *NumericForStatement) adopt() {
	n.ID = adopt(n, n.ID)
	n.Start = adopt(n, n.Start)
	n.End = adopt(n, n.End)
	n.Step = adopt(n, n.Step)
	adoptList(n, n.Statements)
}

func (n *ForStatement) adopt() {
	adoptList(n, n.IDs)
	n.Expression = adopt(n, n.Expression)
	adoptList(n, n.Statements)
}

func (n *FunctionDeclaration) adopt() {
	n.Name = adopt(n, n.Name)
	adoptList(n, n.Parameters)
	adoptList(n, n.Statements)
}

func (n *MethodDeclaration) adopt() {
	n.Expression = adopt(n, n.Expression)
	adoptList(n, n.Parameters)
	adoptList(n, n.Statements)
}

func (n *VariableDeclaration) adopt() {
	n.Left = adoptNodeOrList(n, n.Left)
	n.Right = adoptNodeOrList(n, n.Right)
}

func (n *ReturnStatement) adopt() {
	n.Expression = adoptNodeOrList(n, n.Expression)
}

func (n *MapField) adopt() {
	n.Index = adopt(n, n.Index)
	n.Value = adopt(n, n.Value)
}

// ---------------------------------------------------------------------------
// Helper constructors

var lastTempID atomic.Int64

// NewTempID creates a new temporary identifier. Each call takes the next id
// from a process-wide counter (upstream tempId; the name may be "").
func NewTempID(name string) *TemporaryIdentifier {
	return Create(&TemporaryIdentifier{Name: name, ID: int(lastTempID.Add(1) - 1)})
}

// NewNone creates a new None node.
func NewNone() *None { return Create(&None{}) }

// NewNil creates a new nil literal node.
func NewNil() *NilLiteral { return Create(&NilLiteral{}) }

// NewBool creates a *TrueLiteral or *FalseLiteral.
func NewBool(value bool) Expression {
	if value {
		return Create(&TrueLiteral{})
	}
	return Create(&FalseLiteral{})
}

// NewNumber creates a number literal, or a negated one for negative values,
// using the JavaScript String(number) text (upstream number). NaN panics (upstream
// recurses forever).
func NewNumber(value float64) Expression {
	if value >= 0 {
		return Create(&NumberLiteral{Value: jsNumberToString(value)})
	}
	if value != value {
		panic("luau.NewNumber: NaN")
	}
	abs := -value
	return Create(&UnaryExpression{Operator: UnaryMinus, Expression: NewNumber(abs)})
}

// NewString creates a string literal node. The value is rendered verbatim, so
// it must already be escaped.
func NewString(value string) *StringLiteral { return Create(&StringLiteral{Value: value}) }

// NewID creates a new identifier node.
func NewID(name string) *Identifier { return Create(&Identifier{Name: name}) }

// NewComment creates a new comment node.
func NewComment(text string) *Comment { return Create(&Comment{Text: text}) }

// NewArray creates a new array node.
func NewArray(members ...Expression) *Array {
	return Create(&Array{Members: MakeList(members...)})
}

// NewSet creates a new set node.
func NewSet(members ...Expression) *Set {
	return Create(&Set{Members: MakeList(members...)})
}

// KeyValue is one key/value mapping for NewMap.
type KeyValue struct {
	Index Expression
	Value Expression
}

// NewMap creates a new map node from key/value mappings.
func NewMap(fields ...KeyValue) *Map {
	nodes := make([]Node, len(fields))
	for i, f := range fields {
		nodes[i] = Create(&MapField{Index: f.Index, Value: f.Value})
	}
	return Create(&Map{Fields: MakeList(nodes...)})
}

// MixedField is one entry for NewMixedTable: a key/value mapping when Key is
// not nil, otherwise a positional value.
type MixedField struct {
	Key   Expression
	Value Expression
}

// NewMixedTable creates a new mixed table node.
func NewMixedTable(fields ...MixedField) *MixedTable {
	nodes := make([]Node, len(fields))
	for i, f := range fields {
		if f.Key != nil {
			nodes[i] = Create(&MapField{Index: f.Key, Value: f.Value})
		} else {
			nodes[i] = f.Value
		}
	}
	return Create(&MixedTable{Fields: MakeList(nodes...)})
}

func NewBinary(left Expression, operator BinaryOperator, right Expression) *BinaryExpression {
	return Create(&BinaryExpression{Left: left, Operator: operator, Right: right})
}

func NewUnary(operator UnaryOperator, expression Expression) *UnaryExpression {
	return Create(&UnaryExpression{Operator: operator, Expression: expression})
}

func NewProperty(expression IndexableExpression, name string) *PropertyAccessExpression {
	return Create(&PropertyAccessExpression{Expression: expression, Name: name})
}

func NewCall(expression IndexableExpression, args ...Expression) *CallExpression {
	return Create(&CallExpression{Expression: expression, Args: MakeList(args...)})
}
