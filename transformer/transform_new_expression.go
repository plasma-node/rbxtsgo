package transformer

// Port of src/TSTransformer/nodes/expressions/transformNewExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformNewExpression(state *TransformState, node *ast.Node) luau.Expression {
	validateNotAnyType(state, node.Expression())

	symbol := getFirstConstructSymbol(state, node.Expression())
	if symbol != nil {
		macro := getConstructorMacro(state, symbol)
		if macro != nil {
			return macro(state, node)
		}
	}

	expression := convertToIndexableExpression(transformExpression(state, node.Expression()))
	var args []luau.Expression
	if node.ArgumentList() != nil {
		args = ensureTransformOrder(state, node.Arguments(), nil)
	}
	return luau.NewCall(luau.NewProperty(expression, "new"), args...)
}
