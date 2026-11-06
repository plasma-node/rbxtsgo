package transformer

// Port of src/TSTransformer/nodes/statements/transformThrowStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformThrowStatement(state *TransformState, node *ast.Node) *luau.List {
	var args []luau.Expression
	if expression := node.AsThrowStatement().Expression; expression != nil {
		args = append(args, transformExpression(state, expression))
	}
	return luau.MakeList(luau.Create(&luau.CallStatement{
		Expression: luau.NewCall(luau.Globals.Error, args...),
	}))
}
