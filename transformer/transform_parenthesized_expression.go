package transformer

// Port of src/TSTransformer/nodes/expressions/transformParenthesizedExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformParenthesizedExpression(state *TransformState, node *ast.Node) luau.Expression {
	expression := transformExpression(state, SkipDownwards(node.Expression()))
	if luau.IsSimple(expression) {
		return expression
	} else {
		return luau.Create(&luau.ParenthesizedExpression{Expression: expression})
	}
}
