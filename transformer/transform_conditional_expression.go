package transformer

// Port of src/TSTransformer/nodes/expressions/transformConditionalExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformConditionalExpression(state *TransformState, node *ast.Node) luau.Expression {
	conditional := node.AsConditionalExpression()
	condition := transformExpression(state, conditional.Condition)
	whenTrue, whenTruePrereqs := Capture(state, func() luau.Expression { return transformExpression(state, conditional.WhenTrue) })
	whenFalse, whenFalsePrereqs := Capture(state, func() luau.Expression { return transformExpression(state, conditional.WhenFalse) })

	if isUsedAsStatement(node) {
		whenTruePrereqs.PushList(wrapExpressionStatement(whenTrue))
		whenFalsePrereqs.PushList(wrapExpressionStatement(whenFalse))
		state.Prereq(
			luau.Create(&luau.IfStatement{
				Condition:  createTruthinessChecks(state, condition, conditional.Condition),
				Statements: whenTruePrereqs,
				ElseBody:   whenFalsePrereqs,
			}),
		)
		return luau.NewNone()
	}

	if whenTruePrereqs.IsEmpty() && whenFalsePrereqs.IsEmpty() {
		return luau.Create(&luau.IfExpression{
			Condition:   createTruthinessChecks(state, condition, conditional.Condition),
			Expression:  whenTrue,
			Alternative: whenFalse,
		})
	}

	tempId := luau.NewTempID("result")
	state.Prereq(
		luau.Create(&luau.VariableDeclaration{
			Left:  tempId,
			Right: nil,
		}),
	)

	whenTruePrereqs.Push(
		luau.Create(&luau.Assignment{
			Left:     tempId,
			Operator: luau.AssignEqual,
			Right:    whenTrue,
		}),
	)

	whenFalsePrereqs.Push(
		luau.Create(&luau.Assignment{
			Left:     tempId,
			Operator: luau.AssignEqual,
			Right:    whenFalse,
		}),
	)

	state.Prereq(
		luau.Create(&luau.IfStatement{
			Condition:  createTruthinessChecks(state, condition, conditional.Condition),
			Statements: whenTruePrereqs,
			ElseBody:   whenFalsePrereqs,
		}),
	)

	return tempId
}
