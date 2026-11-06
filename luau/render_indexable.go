package luau

// Ported from src/LuauRenderer/nodes/expressions/indexable/*.ts
// (renderCallExpression, renderComputedIndexExpression, renderIdentifier,
// renderMethodCallExpression, renderParenthesizedExpression,
// renderPropertyAccessExpression, renderTemporaryIdentifier).

func renderCallExpression(state *RenderState, node *CallExpression) {
	renderNode(state, node.Expression)
	state.writeByte('(')
	renderArguments(state, node.Args)
	state.writeByte(')')
}

func renderComputedIndexExpression(state *RenderState, node *ComputedIndexExpression) {
	renderNode(state, node.Expression)
	if s, ok := node.Index.(*StringLiteral); ok && IsValidIdentifier(s.Value) {
		state.writeByte('.')
		state.writeString(s.Value)
	} else {
		state.writeByte('[')
		renderNode(state, node.Index)
		state.writeByte(']')
	}
}

func renderIdentifier(state *RenderState, node *Identifier) {
	if !IsValidIdentifier(node.Name) {
		Assert(false, `Invalid Luau Identifier: "`+node.Name+`"`)
	}
	state.writeString(node.Name)
}

func renderMethodCallExpression(state *RenderState, node *MethodCallExpression) {
	Assert(IsValidIdentifier(node.Name), "")
	renderNode(state, node.Expression)
	state.writeByte(':')
	state.writeString(node.Name)
	state.writeByte('(')
	renderArguments(state, node.Args)
	state.writeByte(')')
}

func renderParenthesizedExpression(state *RenderState, node *ParenthesizedExpression) {
	// skip nested parentheses
	expression := node.Expression
	for {
		p, ok := expression.(*ParenthesizedExpression)
		if !ok {
			break
		}
		expression = p.Expression
	}
	if IsSimple(expression) {
		renderNode(state, node.Expression)
	} else {
		state.writeByte('(')
		renderNode(state, node.Expression)
		state.writeByte(')')
	}
}

func renderPropertyAccessExpression(state *RenderState, node *PropertyAccessExpression) {
	renderNode(state, node.Expression)
	if IsValidIdentifier(node.Name) {
		state.writeByte('.')
		state.writeString(node.Name)
	} else {
		state.writeString(`["`)
		state.writeString(node.Name)
		state.writeString(`"]`)
	}
}

func renderTemporaryIdentifier(state *RenderState, node *TemporaryIdentifier) {
	name := state.GetTempName(node)
	if !IsValidIdentifier(name) {
		Assert(false, `Invalid Temporary Identifier: "`+name+`"`)
	}
	state.writeString(name)
}
