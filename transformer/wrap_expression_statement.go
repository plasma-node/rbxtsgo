package transformer

// Port of src/TSTransformer/util/wrapExpressionStatement.ts.

import "github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"

func wrapExpressionStatement(node luau.Expression) *luau.List {
	if luau.IsTemporaryIdentifier(node) || luau.IsNone(node) {
		return luau.NewList()
	} else if luau.IsCall(node) {
		return luau.MakeList(luau.Create(&luau.CallStatement{Expression: node.(luau.CallLike)}))
	} else {
		return luau.MakeList(
			luau.Create(&luau.VariableDeclaration{
				Left:  luau.NewTempID(""),
				Right: node,
			}),
		)
	}
}
