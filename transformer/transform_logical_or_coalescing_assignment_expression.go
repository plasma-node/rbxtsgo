package transformer

// Port of src/TSTransformer/nodes/transformLogicalOrCoalescingAssignmentExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformCoalescingAssignmentExpression(state *TransformState, left *ast.Node, right *ast.Node) luau.WritableExpression {
	writable := transformWritableExpression(state, left, true)
	value, valuePreqreqs := Capture(state, func() luau.Expression { return transformExpression(state, right) })

	ifStatements := luau.NewList()
	ifStatements.PushList(valuePreqreqs)
	ifStatements.Push(luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: luau.AssignEqual,
		Right:    value,
	}))

	state.Prereq(luau.Create(&luau.IfStatement{
		Condition:  luau.NewBinary(writable, luau.BinaryEqual, luau.NewNil()),
		Statements: ifStatements,
		ElseBody:   luau.NewList(),
	}))

	return writable
}

func transformLogicalAndAssignmentExpression(state *TransformState, left *ast.Node, right *ast.Node) luau.WritableExpression {
	writable := transformWritableExpression(state, left, true)
	value, valuePreqreqs := Capture(state, func() luau.Expression { return transformExpression(state, right) })

	conditionID := state.PushToVar(writable, "condition")

	ifStatements := luau.NewList()
	ifStatements.PushList(valuePreqreqs)
	ifStatements.Push(luau.Create(&luau.Assignment{
		Left:     conditionID,
		Operator: luau.AssignEqual,
		Right:    value,
	}))

	state.Prereq(luau.Create(&luau.IfStatement{
		Condition:  createTruthinessChecks(state, writable, left),
		Statements: ifStatements,
		ElseBody:   luau.NewList(),
	}))

	state.Prereq(luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: luau.AssignEqual,
		Right:    conditionID,
	}))

	return writable
}

func transformLogicalOrAssignmentExpression(state *TransformState, left *ast.Node, right *ast.Node) luau.WritableExpression {
	writable := transformWritableExpression(state, left, true)
	value, valuePreqreqs := Capture(state, func() luau.Expression { return transformExpression(state, right) })

	conditionID := state.PushToVar(writable, "condition")

	ifStatements := luau.NewList()
	ifStatements.PushList(valuePreqreqs)
	ifStatements.Push(luau.Create(&luau.Assignment{
		Left:     conditionID,
		Operator: luau.AssignEqual,
		Right:    value,
	}))

	state.Prereq(luau.Create(&luau.IfStatement{
		Condition:  luau.NewUnary(luau.UnaryNot, createTruthinessChecks(state, writable, left)),
		Statements: ifStatements,
		ElseBody:   luau.NewList(),
	}))

	state.Prereq(luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: luau.AssignEqual,
		Right:    conditionID,
	}))

	return writable
}

func transformLogicalOrCoalescingAssignmentExpression(state *TransformState, node *ast.Node) luau.WritableExpression {
	binary := node.AsBinaryExpression()
	operator := binary.OperatorToken.Kind
	if operator == ast.KindQuestionQuestionEqualsToken {
		return transformCoalescingAssignmentExpression(state, binary.Left, binary.Right)
	} else if operator == ast.KindAmpersandAmpersandEqualsToken {
		return transformLogicalAndAssignmentExpression(state, binary.Left, binary.Right)
	} else {
		return transformLogicalOrAssignmentExpression(state, binary.Left, binary.Right)
	}
}

func transformLogicalOrCoalescingAssignmentExpressionStatement(state *TransformState, node *ast.Node) *luau.List {
	return state.CapturePrereqs(func() { transformLogicalOrCoalescingAssignmentExpression(state, node) })
}
