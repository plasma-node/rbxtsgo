package transformer

// Port of src/TSTransformer/nodes/statements/transformReturnStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func isTupleReturningCall(state *TransformState, tsExpression *ast.Node, luaExpression luau.Expression) bool {
	// intentionally NOT using state.getType() here, because that uses skipUpwards
	return luau.IsCall(luaExpression) &&
		isLuaTupleType(state)(state.Checker.GetTypeAtLocation(SkipDownwards(tsExpression)))
}

func isTupleMacro(state *TransformState, expression *ast.Node) bool {
	if ast.IsCallExpression(expression) {
		symbol := getFirstDefinedSymbol(state, state.GetType(expression.Expression()))
		if symbol != nil && symbol == getSymbolOrThrow(state, "$tuple") {
			return true
		}
	}
	return false
}

func transformReturnStatementInner(state *TransformState, returnExp *ast.Node) *luau.List {
	result := luau.NewList()

	var expression luau.NodeOrList

	if ast.IsCallExpression(returnExp) && isTupleMacro(state, returnExp) {
		args, prereqs := Capture(state, func() []luau.Expression {
			return ensureTransformOrder(state, returnExp.AsCallExpression().Arguments.Nodes, nil)
		})
		result.PushList(prereqs)
		expression = luau.MakeList(args...)
	} else {
		transformed := transformExpression(state, SkipDownwards(returnExp))
		expression = transformed
		if isLuaTupleType(state)(state.GetType(returnExp)) && !isTupleReturningCall(state, returnExp, transformed) {
			if array, ok := transformed.(*luau.Array); ok {
				expression = array.Members
			} else {
				expression = luau.NewCall(luau.Globals.Unpack, transformed)
			}
		}
	}

	if isReturnBlockedByTryStatement(returnExp) {
		state.MarkTryUses(func(u *TryUses) { u.UsesReturn = true })
		tryReturn := state.TS(returnExp, "TRY_RETURN")
		members, isList := luau.AsList(expression)
		if !isList {
			members = luau.MakeList(expression.(luau.Node))
		}
		result.Push(luau.Create(&luau.ReturnStatement{
			Expression: luau.MakeList[luau.Node](
				tryReturn,
				luau.Create(&luau.Array{
					Members: members,
				}),
			),
		}))
	} else {
		result.Push(luau.Create(&luau.ReturnStatement{Expression: expression}))
	}

	return result
}

func transformReturnStatement(state *TransformState, node *ast.Node) *luau.List {
	expression := node.AsReturnStatement().Expression
	if expression == nil {
		if isReturnBlockedByTryStatement(node) {
			state.MarkTryUses(func(u *TryUses) { u.UsesReturn = true })
			return luau.MakeList(luau.Create(&luau.ReturnStatement{
				Expression: luau.MakeList[luau.Node](state.TS(node, "TRY_RETURN"), luau.NewArray()),
			}))
		}
		return luau.MakeList(luau.Create(&luau.ReturnStatement{Expression: luau.NewNil()}))
	}
	return transformReturnStatementInner(state, expression)
}
