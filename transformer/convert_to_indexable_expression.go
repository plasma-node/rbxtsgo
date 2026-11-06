package transformer

// Port of src/TSTransformer/util/convertToIndexableExpression.ts.

import "github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"

func convertToIndexableExpression(expression luau.Expression) luau.IndexableExpression {
	if luau.IsIndexableExpression(expression) {
		return expression.(luau.IndexableExpression)
	}
	return luau.Create(&luau.ParenthesizedExpression{Expression: expression})
}
