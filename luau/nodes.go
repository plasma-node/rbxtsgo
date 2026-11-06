package luau

// Ported from src/LuauAST/types/nodes.ts and src/LuauAST/types/mapping.ts.
//
// Every node is a pointer to a concrete struct. Struct literals are always
// well formed because Kind() is a method of the concrete type, not a stored
// field. Use Create to build a node so that its children get their parent
// pointers (see create.go).
//
// Upstream unions are represented like this:
//
//   - A field that is "node or list of nodes" (Assignment.Left/Right,
//     VariableDeclaration.Left/Right, ReturnStatement.Expression,
//     IfStatement.ElseBody) has type NodeOrList. Both *List and every node
//     implement it; use AsList or a type switch to tell them apart. A nil
//     NodeOrList is upstream's undefined.
//   - A list that holds a mix of kinds (InterpolatedString.Parts,
//     MixedTable.Fields) is an ordinary *List.
//   - Optional node fields (NumericForStatement.Step) are nil when absent.
//
// Lists are untyped (*List of Node). The element types upstream documents are
// noted on each field.

// NodeOrList is implemented by every node and by *List.
type NodeOrList interface {
	isNodeOrList()
}

// Node is any Luau AST node.
type Node interface {
	NodeOrList
	Kind() SyntaxKind
	// Parent returns the node's parent, or nil.
	Parent() Node
	SetParent(Node)

	// shallowClone returns a struct copy: same parent, same children, same
	// lists. Upstream `{ ...node }`.
	shallowClone() Node
	// adopt gives each child field this node as its parent, cloning children
	// that already have one. Used by Create.
	adopt()
}

// Expression is any node in the upstream Expression category (kinds
// KindIdentifier through KindMixedTable, which includes None).
type Expression interface {
	Node
	isExpression()
}

// IndexableExpression is Identifier, TemporaryIdentifier,
// ComputedIndexExpression, PropertyAccessExpression, CallExpression,
// MethodCallExpression or ParenthesizedExpression.
type IndexableExpression interface {
	Expression
	isIndexableExpression()
}

// AnyIdentifier is *Identifier or *TemporaryIdentifier.
type AnyIdentifier interface {
	IndexableExpression
	isAnyIdentifier()
}

// WritableExpression is an AnyIdentifier, *PropertyAccessExpression or
// *ComputedIndexExpression.
type WritableExpression interface {
	IndexableExpression
	isWritableExpression()
}

// CallLike is *CallExpression or *MethodCallExpression.
type CallLike interface {
	IndexableExpression
	isCallLike()
	// Callee returns the `expression` field of either call kind.
	Callee() IndexableExpression
}

// Statement is any node in the upstream Statement category.
type Statement interface {
	Node
	isStatement()
}

// Field is a MapField or InterpolatedStringPart.
type Field interface {
	Node
	isField()
}

// HasStatements is implemented by the nodes that own a statement list
// (upstream guard hasStatements).
type HasStatements interface {
	Node
	GetStatements() *List
}

// HasParameters is implemented by the function-like nodes.
type HasParameters interface {
	Node
	GetParameters() *List
	GetHasDotDotDot() bool
}

// BaseNode holds the state shared by all nodes.
type BaseNode struct {
	parent Node
}

func (b *BaseNode) Parent() Node     { return b.parent }
func (b *BaseNode) SetParent(p Node) { b.parent = p }
func (b *BaseNode) isNodeOrList()    {}

// ---------------------------------------------------------------------------
// Expressions

type None struct{ BaseNode }

type NilLiteral struct{ BaseNode }

type FalseLiteral struct{ BaseNode }

type TrueLiteral struct{ BaseNode }

type NumberLiteral struct {
	BaseNode
	Value string
}

type StringLiteral struct {
	BaseNode
	Value string
}

type VarArgsLiteral struct{ BaseNode }

type FunctionExpression struct {
	BaseNode
	Parameters   *List // of AnyIdentifier
	HasDotDotDot bool
	Statements   *List // of Statement
}

type Identifier struct {
	BaseNode
	Name string
}

type TemporaryIdentifier struct {
	BaseNode
	Name string
	ID   int
}

type ComputedIndexExpression struct {
	BaseNode
	Expression IndexableExpression
	Index      Expression
}

type PropertyAccessExpression struct {
	BaseNode
	Expression IndexableExpression
	Name       string
}

type CallExpression struct {
	BaseNode
	Expression IndexableExpression
	Args       *List // of Expression
}

type MethodCallExpression struct {
	BaseNode
	Name       string
	Expression IndexableExpression
	Args       *List // of Expression
}

type ParenthesizedExpression struct {
	BaseNode
	Expression Expression
}

type BinaryExpression struct {
	BaseNode
	Left     Expression
	Operator BinaryOperator
	Right    Expression
}

type UnaryExpression struct {
	BaseNode
	Operator   UnaryOperator
	Expression Expression
}

type IfExpression struct {
	BaseNode
	Condition   Expression
	Expression  Expression
	Alternative Expression
}

type InterpolatedString struct {
	BaseNode
	Parts *List // of *InterpolatedStringPart or Expression
}

type Array struct {
	BaseNode
	Members *List // of Expression
}

type Map struct {
	BaseNode
	Fields *List // of *MapField
}

type Set struct {
	BaseNode
	Members *List // of Expression
}

type MixedTable struct {
	BaseNode
	Fields *List // of *MapField or Expression
}

// ---------------------------------------------------------------------------
// Statements

type Assignment struct {
	BaseNode
	Left     NodeOrList // WritableExpression or *List of WritableExpression
	Operator AssignmentOperator
	Right    NodeOrList // Expression or *List of Expression
}

type BreakStatement struct{ BaseNode }

type CallStatement struct {
	BaseNode
	Expression CallLike
}

type ContinueStatement struct{ BaseNode }

type DoStatement struct {
	BaseNode
	Statements *List // of Statement
}

type WhileStatement struct {
	BaseNode
	Condition  Expression
	Statements *List // of Statement
}

type RepeatStatement struct {
	BaseNode
	Condition  Expression
	Statements *List // of Statement
}

type IfStatement struct {
	BaseNode
	Condition  Expression
	Statements *List      // of Statement
	ElseBody   NodeOrList // *IfStatement or *List of Statement
}

type NumericForStatement struct {
	BaseNode
	ID         AnyIdentifier
	Start      Expression
	End        Expression
	Step       Expression // optional
	Statements *List      // of Statement
}

type ForStatement struct {
	BaseNode
	IDs        *List // of AnyIdentifier
	Expression Expression
	Statements *List // of Statement
}

type FunctionDeclaration struct {
	BaseNode
	Localize     bool
	Name         WritableExpression // AnyIdentifier or *PropertyAccessExpression
	Parameters   *List              // of AnyIdentifier
	HasDotDotDot bool
	Statements   *List // of Statement
}

type MethodDeclaration struct {
	BaseNode
	Expression   IndexableExpression
	Name         string
	Parameters   *List // of AnyIdentifier
	HasDotDotDot bool
	Statements   *List // of Statement
}

type VariableDeclaration struct {
	BaseNode
	Left  NodeOrList // AnyIdentifier or *List of AnyIdentifier
	Right NodeOrList // Expression, *List of Expression, or nil
}

type ReturnStatement struct {
	BaseNode
	Expression NodeOrList // Expression or *List of Expression
}

type Comment struct {
	BaseNode
	Text string
}

// ---------------------------------------------------------------------------
// Fields

type MapField struct {
	BaseNode
	Index Expression
	Value Expression
}

type InterpolatedStringPart struct {
	BaseNode
	Text string
}

// ---------------------------------------------------------------------------
// Kind() and category markers

func (*Identifier) Kind() SyntaxKind               { return KindIdentifier }
func (*TemporaryIdentifier) Kind() SyntaxKind      { return KindTemporaryIdentifier }
func (*ComputedIndexExpression) Kind() SyntaxKind  { return KindComputedIndexExpression }
func (*PropertyAccessExpression) Kind() SyntaxKind { return KindPropertyAccessExpression }
func (*CallExpression) Kind() SyntaxKind           { return KindCallExpression }
func (*MethodCallExpression) Kind() SyntaxKind     { return KindMethodCallExpression }
func (*ParenthesizedExpression) Kind() SyntaxKind  { return KindParenthesizedExpression }
func (*None) Kind() SyntaxKind                     { return KindNone }
func (*NilLiteral) Kind() SyntaxKind               { return KindNilLiteral }
func (*FalseLiteral) Kind() SyntaxKind             { return KindFalseLiteral }
func (*TrueLiteral) Kind() SyntaxKind              { return KindTrueLiteral }
func (*NumberLiteral) Kind() SyntaxKind            { return KindNumberLiteral }
func (*StringLiteral) Kind() SyntaxKind            { return KindStringLiteral }
func (*VarArgsLiteral) Kind() SyntaxKind           { return KindVarArgsLiteral }
func (*FunctionExpression) Kind() SyntaxKind       { return KindFunctionExpression }
func (*BinaryExpression) Kind() SyntaxKind         { return KindBinaryExpression }
func (*UnaryExpression) Kind() SyntaxKind          { return KindUnaryExpression }
func (*IfExpression) Kind() SyntaxKind             { return KindIfExpression }
func (*InterpolatedString) Kind() SyntaxKind       { return KindInterpolatedString }
func (*Array) Kind() SyntaxKind                    { return KindArray }
func (*Map) Kind() SyntaxKind                      { return KindMap }
func (*Set) Kind() SyntaxKind                      { return KindSet }
func (*MixedTable) Kind() SyntaxKind               { return KindMixedTable }
func (*Assignment) Kind() SyntaxKind               { return KindAssignment }
func (*BreakStatement) Kind() SyntaxKind           { return KindBreakStatement }
func (*CallStatement) Kind() SyntaxKind            { return KindCallStatement }
func (*ContinueStatement) Kind() SyntaxKind        { return KindContinueStatement }
func (*DoStatement) Kind() SyntaxKind              { return KindDoStatement }
func (*WhileStatement) Kind() SyntaxKind           { return KindWhileStatement }
func (*RepeatStatement) Kind() SyntaxKind          { return KindRepeatStatement }
func (*IfStatement) Kind() SyntaxKind              { return KindIfStatement }
func (*NumericForStatement) Kind() SyntaxKind      { return KindNumericForStatement }
func (*ForStatement) Kind() SyntaxKind             { return KindForStatement }
func (*FunctionDeclaration) Kind() SyntaxKind      { return KindFunctionDeclaration }
func (*MethodDeclaration) Kind() SyntaxKind        { return KindMethodDeclaration }
func (*VariableDeclaration) Kind() SyntaxKind      { return KindVariableDeclaration }
func (*ReturnStatement) Kind() SyntaxKind          { return KindReturnStatement }
func (*Comment) Kind() SyntaxKind                  { return KindComment }
func (*MapField) Kind() SyntaxKind                 { return KindMapField }
func (*InterpolatedStringPart) Kind() SyntaxKind   { return KindInterpolatedStringPart }

// Expression
func (*Identifier) isExpression()               {}
func (*TemporaryIdentifier) isExpression()      {}
func (*ComputedIndexExpression) isExpression()  {}
func (*PropertyAccessExpression) isExpression() {}
func (*CallExpression) isExpression()           {}
func (*MethodCallExpression) isExpression()     {}
func (*ParenthesizedExpression) isExpression()  {}
func (*None) isExpression()                     {}
func (*NilLiteral) isExpression()               {}
func (*FalseLiteral) isExpression()             {}
func (*TrueLiteral) isExpression()              {}
func (*NumberLiteral) isExpression()            {}
func (*StringLiteral) isExpression()            {}
func (*VarArgsLiteral) isExpression()           {}
func (*FunctionExpression) isExpression()       {}
func (*BinaryExpression) isExpression()         {}
func (*UnaryExpression) isExpression()          {}
func (*IfExpression) isExpression()             {}
func (*InterpolatedString) isExpression()       {}
func (*Array) isExpression()                    {}
func (*Map) isExpression()                      {}
func (*Set) isExpression()                      {}
func (*MixedTable) isExpression()               {}

// IndexableExpression
func (*Identifier) isIndexableExpression()               {}
func (*TemporaryIdentifier) isIndexableExpression()      {}
func (*ComputedIndexExpression) isIndexableExpression()  {}
func (*PropertyAccessExpression) isIndexableExpression() {}
func (*CallExpression) isIndexableExpression()           {}
func (*MethodCallExpression) isIndexableExpression()     {}
func (*ParenthesizedExpression) isIndexableExpression()  {}

// AnyIdentifier
func (*Identifier) isAnyIdentifier()          {}
func (*TemporaryIdentifier) isAnyIdentifier() {}

// WritableExpression
func (*Identifier) isWritableExpression()               {}
func (*TemporaryIdentifier) isWritableExpression()      {}
func (*PropertyAccessExpression) isWritableExpression() {}
func (*ComputedIndexExpression) isWritableExpression()  {}

// CallLike
func (*CallExpression) isCallLike()       {}
func (*MethodCallExpression) isCallLike() {}

func (n *CallExpression) Callee() IndexableExpression       { return n.Expression }
func (n *MethodCallExpression) Callee() IndexableExpression { return n.Expression }

// Statement
func (*Assignment) isStatement()          {}
func (*BreakStatement) isStatement()      {}
func (*CallStatement) isStatement()       {}
func (*ContinueStatement) isStatement()   {}
func (*DoStatement) isStatement()         {}
func (*WhileStatement) isStatement()      {}
func (*RepeatStatement) isStatement()     {}
func (*IfStatement) isStatement()         {}
func (*NumericForStatement) isStatement() {}
func (*ForStatement) isStatement()        {}
func (*FunctionDeclaration) isStatement() {}
func (*MethodDeclaration) isStatement()   {}
func (*VariableDeclaration) isStatement() {}
func (*ReturnStatement) isStatement()     {}
func (*Comment) isStatement()             {}

// Field
func (*MapField) isField()               {}
func (*InterpolatedStringPart) isField() {}

// HasStatements
func (n *ForStatement) GetStatements() *List        { return n.Statements }
func (n *NumericForStatement) GetStatements() *List { return n.Statements }
func (n *FunctionExpression) GetStatements() *List  { return n.Statements }
func (n *DoStatement) GetStatements() *List         { return n.Statements }
func (n *FunctionDeclaration) GetStatements() *List { return n.Statements }
func (n *IfStatement) GetStatements() *List         { return n.Statements }
func (n *MethodDeclaration) GetStatements() *List   { return n.Statements }
func (n *RepeatStatement) GetStatements() *List     { return n.Statements }
func (n *WhileStatement) GetStatements() *List      { return n.Statements }

// HasParameters
func (n *FunctionExpression) GetParameters() *List   { return n.Parameters }
func (n *FunctionDeclaration) GetParameters() *List  { return n.Parameters }
func (n *MethodDeclaration) GetParameters() *List    { return n.Parameters }
func (n *FunctionExpression) GetHasDotDotDot() bool  { return n.HasDotDotDot }
func (n *FunctionDeclaration) GetHasDotDotDot() bool { return n.HasDotDotDot }
func (n *MethodDeclaration) GetHasDotDotDot() bool   { return n.HasDotDotDot }

// ShallowClone returns a struct copy of node: same parent, same child nodes
// and same lists (upstream `{ ...node }`).
func ShallowClone(node Node) Node { return node.shallowClone() }
