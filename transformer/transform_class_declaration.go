package transformer

// Port of src/TSTransformer/nodes/statements/transformClassDeclaration.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformClassDeclaration(state *TransformState, node *ast.Node) *luau.List {
	return transformClassLikeDeclaration(state, node).statements
}
