package transformer

// Port of src/TSTransformer/util/createTypeCheck.ts.

import "github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"

func createTypeCheck(expression luau.Expression, typeName *luau.StringLiteral) *luau.BinaryExpression {
	return luau.NewBinary(luau.NewCall(luau.Globals.Type, expression), luau.BinaryEqual, typeName)
}
