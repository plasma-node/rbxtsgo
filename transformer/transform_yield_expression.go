package transformer

// Port of src/TSTransformer/nodes/expressions/transformYieldExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformYieldExpression(state *TransformState, node *ast.Node) luau.Expression {
	yieldExpression := node.AsYieldExpression()
	if yieldExpression.Expression == nil {
		return luau.NewCall(luau.Globals.Coroutine.Yield)
	}

	expression := transformExpression(state, yieldExpression.Expression)
	if yieldExpression.AsteriskToken != nil {
		loopId := luau.NewTempID("result")

		finalizer := luau.MakeList[luau.Statement](luau.Create(&luau.BreakStatement{}))
		var evaluated luau.Expression = luau.NewNone()

		if !isUsedAsStatement(node) {
			returnValue := state.PushToVar(nil, "returnValue")
			finalizer.Unshift(
				luau.Create(&luau.Assignment{
					Left:     returnValue,
					Operator: luau.AssignEqual,
					Right:    luau.NewProperty(loopId, "value"),
				}),
			)
			evaluated = returnValue
		}

		state.Prereq(
			luau.Create(&luau.ForStatement{
				IDs:        luau.MakeList(loopId),
				Expression: luau.NewProperty(convertToIndexableExpression(expression), "next"),
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.IfStatement{
						Condition:  luau.NewProperty(loopId, "done"),
						Statements: finalizer,
						ElseBody:   luau.NewList(),
					}),
					luau.Create(&luau.CallStatement{
						Expression: luau.NewCall(luau.Globals.Coroutine.Yield, luau.NewProperty(loopId, "value")),
					}),
				),
			}),
		)

		return evaluated
	} else {
		return luau.NewCall(luau.Globals.Coroutine.Yield, expression)
	}
}
