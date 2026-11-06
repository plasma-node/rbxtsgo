package transformer

// Port of src/TSTransformer/nodes/expressions/transformOmittedExpression.ts.

import "github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"

func transformOmittedExpression() luau.Expression {
	return luau.NewNil()
}
