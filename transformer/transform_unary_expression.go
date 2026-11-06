package transformer

// Port of src/TSTransformer/nodes/expressions/transformUnaryExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformPostfixUnaryExpression(state *TransformState, node *ast.Node) luau.Expression {
	unary := node.AsPostfixUnaryExpression()
	validateNotAnyType(state, unary.Operand)

	writable := transformWritableExpression(state, unary.Operand, true)
	origValue := luau.NewTempID("original")

	state.Prereq(
		luau.Create(&luau.VariableDeclaration{
			Left:  origValue,
			Right: writable,
		}),
	)

	var operator luau.AssignmentOperator
	if unary.Operator == ast.KindPlusPlusToken {
		operator = luau.AssignAdd
	} else if unary.Operator == ast.KindMinusMinusToken {
		operator = luau.AssignSub
	} else {
		assertNever(unary.Operator, "transformPostfixUnaryExpression")
	}

	state.Prereq(
		luau.Create(&luau.Assignment{
			Left:     writable,
			Operator: operator,
			Right:    luau.NewNumber(1),
		}),
	)

	return origValue
}

func transformPrefixUnaryExpression(state *TransformState, node *ast.Node) luau.Expression {
	unary := node.AsPrefixUnaryExpression()
	validateNotAnyType(state, unary.Operand)

	if unary.Operator == ast.KindPlusPlusToken || unary.Operator == ast.KindMinusMinusToken {
		writable := transformWritableExpression(state, unary.Operand, true)
		operator := luau.AssignSub
		if unary.Operator == ast.KindPlusPlusToken {
			operator = luau.AssignAdd
		}
		state.Prereq(
			luau.Create(&luau.Assignment{
				Left:     writable,
				Operator: operator,
				Right:    luau.NewNumber(1),
			}),
		)
		return writable
	} else if unary.Operator == ast.KindPlusToken {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoUnaryPlus.At(shared.AtNode(node)))
		return transformExpression(state, unary.Operand)
	} else if unary.Operator == ast.KindMinusToken {
		if !isDefinitelyType(state.GetType(unary.Operand), isNumberType) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoNonNumberUnaryMinus.At(shared.AtNode(node)))
		}
		return luau.NewUnary(luau.UnaryMinus, transformExpression(state, unary.Operand))
	} else if unary.Operator == ast.KindExclamationToken {
		checks := createTruthinessChecks(state, transformExpression(state, unary.Operand), unary.Operand)
		return luau.NewUnary(luau.UnaryNot, checks)
	} else if unary.Operator == ast.KindTildeToken {
		return luau.NewCall(luau.NewProperty(luau.Globals.Bit32, "bnot"), transformExpression(state, unary.Operand))
	}
	assertNever(unary.Operator, "transformPrefixUnaryExpression")
	panic("unreachable")
}
