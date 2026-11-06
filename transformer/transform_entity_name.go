package transformer

// Port of src/TSTransformer/nodes/transformEntityName.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformEntityName(state *TransformState, node *ast.Node) luau.Expression {
	if ast.IsIdentifier(node) {
		validateIdentifier(state, node)
		return transformIdentifier(state, node)
	}
	return transformQualifiedName(state, node)
}

func transformQualifiedName(state *TransformState, node *ast.Node) *luau.PropertyAccessExpression {
	qualifiedName := node.AsQualifiedName()
	return luau.NewProperty(convertToIndexableExpression(transformEntityName(state, qualifiedName.Left)), qualifiedName.Right.Text())
}
