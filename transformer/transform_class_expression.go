package transformer

// Port of src/TSTransformer/nodes/expressions/transformClassExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformClassExpression(state *TransformState, node *ast.Node) luau.Expression {
	result := transformClassLikeDeclaration(state, node)
	state.PrereqList(result.statements)
	return result.name
}
