package transformer

// Port of src/TSTransformer/util/assignment.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

var compoundOperatorMap = map[ast.Kind]luau.AssignmentOperator{
	// compound assignment
	ast.KindMinusEqualsToken:            luau.AssignSub,
	ast.KindAsteriskEqualsToken:         luau.AssignMul,
	ast.KindSlashEqualsToken:            luau.AssignDiv,
	ast.KindAsteriskAsteriskEqualsToken: luau.AssignPow,
	ast.KindPercentEqualsToken:          luau.AssignMod,

	// unary
	ast.KindPlusPlusToken:   luau.AssignAdd,
	ast.KindMinusMinusToken: luau.AssignSub,

	// normal assignment
	ast.KindEqualsToken: luau.AssignEqual,
}

// getSimpleAssignmentOperator returns "" where JS returns undefined.
func getSimpleAssignmentOperator(leftType *checker.Type, operatorKind ast.Kind, rightType *checker.Type) luau.AssignmentOperator {
	// plus
	if operatorKind == ast.KindPlusEqualsToken {
		if isDefinitelyType(leftType, isStringType) || isDefinitelyType(rightType, isStringType) {
			return luau.AssignConcat
		}
		return luau.AssignAdd
	}

	return compoundOperatorMap[operatorKind]
}

func createAssignmentExpression(state *TransformState, readable luau.WritableExpression, operator luau.AssignmentOperator, value luau.Expression) luau.WritableExpression {
	state.Prereq(luau.Create(&luau.Assignment{
		Left:     readable,
		Operator: operator,
		Right:    value,
	}))
	return readable
}

func createCompoundAssignmentStatement(state *TransformState, node *ast.Node, writable luau.WritableExpression, writableType *checker.Type, readable luau.WritableExpression, operator ast.Kind, value luau.Expression, valueType *checker.Type) *luau.Assignment {
	return luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: luau.AssignEqual,
		Right:    createBinaryFromOperator(state, node, readable, writableType, operator, value, valueType),
	})
}

func createCompoundAssignmentExpression(state *TransformState, node *ast.Node, writable luau.WritableExpression, writableType *checker.Type, readable luau.WritableExpression, operator ast.Kind, value luau.Expression, valueType *checker.Type) luau.WritableExpression {
	return createAssignmentExpression(
		state,
		writable,
		luau.AssignEqual,
		createBinaryFromOperator(state, node, readable, writableType, operator, value, valueType),
	)
}
