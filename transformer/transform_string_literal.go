package transformer

// Port of src/TSTransformer/nodes/expressions/transformStringLiteral.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformStringLiteral(state *TransformState, node *ast.Node) luau.Expression {
	return luau.NewString(createStringFromLiteral(node))
}
