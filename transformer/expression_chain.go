package transformer

// Port of src/TSTransformer/util/expressionChain.ts.

import "github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"

// binaryExpressionChain combines multiple luau.Expression objects into a chain of luau.BinaryExpressions
//
// i.e. using `and` as our operator, `[a, b, c]` -> `a and b and c`
func binaryExpressionChain(expressions []luau.Expression, operator luau.BinaryOperator) luau.Expression {
	acc := expressions[0]
	for _, current := range expressions[1:] {
		acc = luau.NewBinary(acc, operator, current)
	}
	return acc
}

// propertyAccessExpressionChain combines multiple strings into a chain of property accesses
//
// i.e. `["a", "b", "c"]` -> `exp.a.b.c`
func propertyAccessExpressionChain(expression luau.Expression, names []string) luau.IndexableExpression {
	acc := convertToIndexableExpression(expression)
	for _, current := range names {
		acc = luau.NewProperty(acc, current)
	}
	return acc
}
