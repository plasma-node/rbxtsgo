package transformer

// Port of src/TSTransformer/util/wrapStatementsAsGenerator.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func wrapStatementsAsGenerator(state *TransformState, node *ast.Node, statements *luau.List) *luau.List {
	return luau.MakeList(
		luau.Create(&luau.ReturnStatement{
			Expression: luau.NewCall(state.TS(node, "generator"),
				luau.Create(&luau.FunctionExpression{
					HasDotDotDot: false,
					Parameters:   luau.NewList(),
					Statements:   statements,
				}),
			),
		}),
	)
}
