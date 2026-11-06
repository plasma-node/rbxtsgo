package transformer

// Port of src/TSTransformer/nodes/expressions/transformAwaitExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformAwaitExpression(state *TransformState, node *ast.Node) luau.Expression {
	return luau.NewCall(state.TS(node, "await"), transformExpression(state, SkipDownwards(node.Expression())))
}
