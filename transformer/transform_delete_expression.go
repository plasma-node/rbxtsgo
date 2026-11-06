package transformer

// Port of src/TSTransformer/nodes/expressions/transformDeleteExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformDeleteExpression(state *TransformState, node *ast.Node) luau.Expression {
	// we just want the prereqs, deleting is done in the index expression transforms
	transformExpression(state, node.Expression())
	if !isUsedAsStatement(node) {
		return luau.NewBool(true)
	}
	return luau.NewNone()
}
