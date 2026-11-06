package transformer

// Port of src/TSTransformer/nodes/expressions/transformJsxExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformJsxExpression(state *TransformState, node *ast.Node) luau.Expression {
	jsxExpression := node.AsJsxExpression()
	if jsxExpression.Expression != nil {
		expression := transformExpression(state, jsxExpression.Expression)
		if jsxExpression.DotDotDotToken != nil {
			return luau.NewCall(luau.Globals.Unpack, expression)
		}
		return expression
	}
	return luau.NewNone()
}
