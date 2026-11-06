package transformer

// Port of src/TSTransformer/util/getAssignableValue.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func getAssignableValue(operator luau.AssignmentOperator, value luau.Expression, valueType *checker.Type) luau.Expression {
	if operator == luau.AssignConcat && !isDefinitelyType(valueType, isStringType) {
		return luau.NewCall(luau.Globals.Tostring, value)
	}
	return value
}
