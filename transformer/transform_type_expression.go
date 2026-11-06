package transformer

// Port of src/TSTransformer/nodes/expressions/transformTypeExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformTypeExpression(state *TransformState, node *ast.Node) luau.Expression {
	return transformExpression(state, node.Expression())
}
