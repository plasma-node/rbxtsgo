package transformer

// Port of src/TSTransformer/nodes/expressions/transformPropertyAccessExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformPropertyAccessExpressionInner(state *TransformState, node *ast.Node, expression luau.Expression, name string) luau.Expression {
	// a in a.b
	validateNotAnyType(state, node.Expression())

	addIndexDiagnostics(state, node, state.Checker.GetNonOptionalType(state.GetType(node)))

	if ast.IsDeleteExpression(SkipUpwards(node).Parent) {
		state.Prereq(
			luau.Create(&luau.Assignment{
				Left:     luau.NewProperty(convertToIndexableExpression(expression), name),
				Operator: luau.AssignEqual,
				Right:    luau.NewNil(),
			}),
		)
		return luau.NewNone()
	}

	return luau.NewProperty(convertToIndexableExpression(expression), name)
}

func transformPropertyAccessExpression(state *TransformState, node *ast.Node) luau.Expression {
	constantValue := getConstantValueLiteral(state, node)
	if constantValue != nil {
		return constantValue
	}

	return transformOptionalChain(state, node)
}
