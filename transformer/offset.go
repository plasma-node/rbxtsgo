package transformer

// Port of src/TSTransformer/util/offset.ts.

import (
	"math"

	"github.com/microsoft/TypeScript/tsc/internal/jsnum"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// getLiteralNumberValue returns false where JS returns undefined. NaN is a value here, as in JS.
func getLiteralNumberValue(expression luau.Expression) (float64, bool) {
	switch e := expression.(type) {
	case *luau.NumberLiteral:
		return float64(jsnum.FromString(e.Value)), true
	case *luau.UnaryExpression:
		if e.Operator == luau.UnaryMinus {
			if innerValue, ok := getLiteralNumberValue(e.Expression); ok {
				return -innerValue, true
			}
		}
	}
	return 0, false
}

func offset(expression luau.Expression, value float64) luau.Expression {
	if value == 0 {
		return expression
	}

	// this special case handles when the expression is a binary expression with a number literal on its right
	// i.e. array[offset - 1 + 1] -> array[offset]
	if binary, ok := expression.(*luau.BinaryExpression); ok && (binary.Operator == luau.BinaryAdd || binary.Operator == luau.BinarySub) {
		if rightValue, ok := getLiteralNumberValue(binary.Right); ok {
			sign := 1.0
			if binary.Operator == luau.BinarySub {
				sign = -1
			}
			newRightValue := rightValue + value*sign
			if newRightValue == 0 {
				return binary.Left
			} else {
				return luau.NewBinary(binary.Left, binary.Operator, luau.NewNumber(newRightValue))
			}
		}
	}

	if literalValue, ok := getLiteralNumberValue(expression); ok {
		return luau.NewNumber(literalValue + value)
	} else {
		operator := luau.BinarySub
		if value > 0 {
			operator = luau.BinaryAdd
		}
		return luau.NewBinary(expression, operator, luau.NewNumber(math.Abs(value)))
	}
}
