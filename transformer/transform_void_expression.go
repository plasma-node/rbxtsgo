package transformer

// Port of src/TSTransformer/nodes/expressions/transformVoidExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformVoidExpression(state *TransformState, node *ast.Node) luau.Expression {
	state.PrereqList(transformExpressionStatementInner(state, SkipDownwards(node.Expression())))
	return luau.Create(&luau.NilLiteral{})
}
