package transformer

// Port of src/TSTransformer/nodes/statements/transformBlock.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformBlock(state *TransformState, node *ast.Node) *luau.List {
	return luau.MakeList(luau.Create(&luau.DoStatement{
		Statements: transformStatementList(state, node, node.AsBlock().Statements.Nodes, nil),
	}))
}
