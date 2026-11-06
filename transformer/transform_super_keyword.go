package transformer

// Port of src/TSTransformer/nodes/expressions/transformSuperKeyword.ts.

import "github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"

func transformSuperKeyword() luau.Expression {
	return luau.Globals.Super
}
