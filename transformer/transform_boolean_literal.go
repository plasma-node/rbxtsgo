package transformer

// Port of src/TSTransformer/nodes/expressions/transformBooleanLiteral.ts.

import "github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"

func transformTrueKeyword() luau.Expression {
	return luau.Create(&luau.TrueLiteral{})
}

func transformFalseKeyword() luau.Expression {
	return luau.Create(&luau.FalseLiteral{})
}
