package transformer

// Port of src/TSTransformer/util/addOneIfArrayType.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func addOneIfArrayType(state *TransformState, t *checker.Type, expression luau.Expression) luau.Expression {
	if isDefinitelyType(t, isArrayType(state), isUndefinedType) {
		return offset(expression, 1)
	}
	return expression
}
