package luau

// Ported from src/LuauRenderer/render.ts and the LuauRenderer public API
// (src/LuauRenderer/index.ts).

// renderNode appends the Luau for node to state's buffer. It is the Go
// counterpart of KIND_TO_RENDERER in upstream render().
func renderNode(state *RenderState, node Node) {
	switch n := node.(type) {
	// indexable expressions
	case *Identifier:
		renderIdentifier(state, n)
	case *TemporaryIdentifier:
		renderTemporaryIdentifier(state, n)
	case *ComputedIndexExpression:
		renderComputedIndexExpression(state, n)
	case *PropertyAccessExpression:
		renderPropertyAccessExpression(state, n)
	case *CallExpression:
		renderCallExpression(state, n)
	case *MethodCallExpression:
		renderMethodCallExpression(state, n)
	case *ParenthesizedExpression:
		renderParenthesizedExpression(state, n)

	// expressions
	case *None:
		Assert(false, "Cannot render None")
	case *NilLiteral:
		state.writeString("nil")
	case *FalseLiteral:
		state.writeString("false")
	case *TrueLiteral:
		state.writeString("true")
	case *NumberLiteral:
		renderNumberLiteral(state, n)
	case *StringLiteral:
		renderStringLiteral(state, n)
	case *VarArgsLiteral:
		state.writeString("...")
	case *FunctionExpression:
		renderFunctionExpression(state, n)
	case *BinaryExpression:
		renderBinaryExpression(state, n)
	case *UnaryExpression:
		renderUnaryExpression(state, n)
	case *IfExpression:
		renderIfExpression(state, n)
	case *InterpolatedString:
		renderInterpolatedString(state, n)
	case *Array:
		renderArray(state, n)
	case *Map:
		renderMap(state, n)
	case *Set:
		renderSet(state, n)
	case *MixedTable:
		renderMixedTable(state, n)

	// statements
	case *Assignment:
		renderAssignment(state, n)
	case *BreakStatement:
		renderBreakStatement(state)
	case *CallStatement:
		renderCallStatement(state, n)
	case *ContinueStatement:
		renderContinueStatement(state)
	case *DoStatement:
		renderDoStatement(state, n)
	case *WhileStatement:
		renderWhileStatement(state, n)
	case *RepeatStatement:
		renderRepeatStatement(state, n)
	case *IfStatement:
		renderIfStatement(state, n)
	case *NumericForStatement:
		renderNumericForStatement(state, n)
	case *ForStatement:
		renderForStatement(state, n)
	case *FunctionDeclaration:
		renderFunctionDeclaration(state, n)
	case *MethodDeclaration:
		renderMethodDeclaration(state, n)
	case *VariableDeclaration:
		renderVariableDeclaration(state, n)
	case *ReturnStatement:
		renderReturnStatement(state, n)
	case *Comment:
		renderComment(state, n)

	// fields
	case *MapField:
		renderMapField(state, n)
	case *InterpolatedStringPart:
		renderInterpolatedStringPart(state, n)

	default:
		// Upstream would throw "KIND_TO_RENDERER[node.kind] is not a function".
		Assert(false, "Cannot render a nil or unknown node")
	}
}

// Render returns a string that represents the given syntax node as Luau
// code. Recursively called until the node is completely rendered.
func Render(state *RenderState, node Node) string {
	start := len(state.buf)
	renderNode(state, node)
	result := string(state.buf[start:])
	state.buf = state.buf[:start]
	return result
}

// RenderStatements renders the given list of statements. SolveTempIDs should
// have been run on the list first.
func RenderStatements(state *RenderState, statements *List) string {
	start := len(state.buf)
	renderStatementsTo(state, statements)
	result := string(state.buf[start:])
	state.buf = state.buf[:start]
	return result
}

// RenderAST returns a string that represents the given syntax tree as Luau
// code.
func RenderAST(ast *List) string {
	state := NewRenderState()
	state.buf = make([]byte, 0, 4096)

	SolveTempIDs(state, ast)

	renderStatementsTo(state, ast)
	return string(state.buf)
}
