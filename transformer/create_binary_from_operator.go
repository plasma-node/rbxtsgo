package transformer

// Port of src/TSTransformer/util/createBinaryFromOperator.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

var createBinaryOperatorMap = map[ast.Kind]luau.BinaryOperator{
	// comparison
	ast.KindLessThanToken:                luau.BinaryLess,
	ast.KindGreaterThanToken:             luau.BinaryGreater,
	ast.KindLessThanEqualsToken:          luau.BinaryLessEqual,
	ast.KindGreaterThanEqualsToken:       luau.BinaryGreaterEqual,
	ast.KindEqualsEqualsEqualsToken:      luau.BinaryEqual,
	ast.KindExclamationEqualsEqualsToken: luau.BinaryNotEqual,

	// math
	ast.KindMinusToken:            luau.BinarySub,
	ast.KindAsteriskToken:         luau.BinaryMul,
	ast.KindSlashToken:            luau.BinaryDiv,
	ast.KindAsteriskAsteriskToken: luau.BinaryPow,
	ast.KindPercentToken:          luau.BinaryMod,
}

var createBinaryBitwiseOperatorMap = map[ast.Kind]string{
	// bitwise
	ast.KindAmpersandToken:                         "band",
	ast.KindBarToken:                               "bor",
	ast.KindCaretToken:                             "bxor",
	ast.KindLessThanLessThanToken:                  "lshift",
	ast.KindGreaterThanGreaterThanGreaterThanToken: "rshift",
	ast.KindGreaterThanGreaterThanToken:            "arshift",

	// bitwise compound assignment
	ast.KindAmpersandEqualsToken:                         "band",
	ast.KindBarEqualsToken:                               "bor",
	ast.KindCaretEqualsToken:                             "bxor",
	ast.KindLessThanLessThanEqualsToken:                  "lshift",
	ast.KindGreaterThanGreaterThanGreaterThanEqualsToken: "rshift",
	ast.KindGreaterThanGreaterThanEqualsToken:            "arshift",
}

func createBinaryAdd(left luau.Expression, leftType *checker.Type, right luau.Expression, rightType *checker.Type) luau.Expression {
	leftIsString := isDefinitelyType(leftType, isStringType)
	rightIsString := isDefinitelyType(rightType, isStringType)
	if leftIsString || rightIsString {
		leftOperand := left
		if !leftIsString {
			leftOperand = luau.NewCall(luau.Globals.Tostring, left)
		}
		rightOperand := right
		if !rightIsString {
			rightOperand = luau.NewCall(luau.Globals.Tostring, right)
		}
		return luau.NewBinary(leftOperand, luau.BinaryConcat, rightOperand)
	}
	return luau.NewBinary(left, luau.BinaryAdd, right)
}

func createBinaryFromOperator(state *TransformState, node *ast.Node, left luau.Expression, leftType *checker.Type, operatorKind ast.Kind, right luau.Expression, rightType *checker.Type) luau.Expression {
	// simple
	if operator, ok := createBinaryOperatorMap[operatorKind]; ok {
		return luau.NewBinary(left, operator, right)
	}

	// plus
	if operatorKind == ast.KindPlusToken || operatorKind == ast.KindPlusEqualsToken {
		return createBinaryAdd(left, leftType, right, rightType)
	}

	// bitwise
	if bit32Name, ok := createBinaryBitwiseOperatorMap[operatorKind]; ok {
		return luau.NewCall(luau.NewProperty(luau.Globals.Bit32, bit32Name), left, right)
	}

	if operatorKind == ast.KindCommaToken {
		state.PrereqList(wrapExpressionStatement(left))
		return right
	}

	panic(shared.AssertionMessage("createBinaryFromOperator unknown operator: " + getKindName(operatorKind)))
}
