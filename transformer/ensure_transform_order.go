package transformer

// Port of src/TSTransformer/util/ensureTransformOrder.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// ensureTransformOrder takes an array of `ts.Expression` and transforms each, capturing prereqs. Returns the
// transformed nodes. Ensures the `luau.Expression` nodes execute in the same order as the `ts.Expression` nodes.
// A nil transformer means transformExpression.
func ensureTransformOrder(state *TransformState, nodes []*ast.Node, transformer func(state *TransformState, node *ast.Node) luau.Expression) []luau.Expression {
	if transformer == nil {
		transformer = transformExpression
	}
	expressions := make([]luau.Expression, len(nodes))
	prereqsList := make([]*luau.List, len(nodes))
	for i, node := range nodes {
		expressions[i], prereqsList[i] = Capture(state, func() luau.Expression { return transformer(state, node) })
	}
	lastArgWithPrereqsIndex := -1
	for i := len(prereqsList) - 1; i >= 0; i-- {
		if !prereqsList[i].IsEmpty() {
			lastArgWithPrereqsIndex = i
			break
		}
	}
	result := []luau.Expression{}
	for i := range expressions {
		expression, prereqs := expressions[i], prereqsList[i]
		state.PrereqList(prereqs)

		isConstVar := false
		exp := nodes[i]
		if ast.IsIdentifier(exp) {
			symbol := state.Checker.GetSymbolAtLocation(exp)
			if symbol != nil && !isSymbolMutable(state, symbol) {
				isConstVar = true
			}
		}

		if i < lastArgWithPrereqsIndex &&
			!luau.IsSimplePrimitive(expression) &&
			!luau.IsTemporaryIdentifier(expression) &&
			!isConstVar {
			result = append(result, state.PushToVar(expression, "exp"))
		} else {
			result = append(result, expression)
		}
	}
	return result
}
