package transformer

// With fixEmit, what `continue` compiles to. Ported from the idea of upstream's open PR #3125 (issue #2899), which is
// not merged. A switch compiles to `repeat ... until true`, so a Luau `continue` in one only leaves the repeat, and the
// code after the switch runs. Unlike the PR, a continue that goes through a try statement gets out too.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// createContinueStatements returns the statements that continue the loop around node. What's next on the way out
// decides them, and each one that isn't the loop passes it on:
//   - a switch: raise its flag and break out of its repeat, and transformSwitchStatement's `if flag` carries on
//   - a try: return TS.TRY_CONTINUE, and transformFlowControl carries on after the TS.try call
//   - the loop: continue
func createContinueStatements(state *TransformState, node *ast.Node) *luau.List {
	ancestor := ast.FindAncestor(node.Parent, func(ancestor *ast.Node) bool {
		return ast.IsSwitchStatement(ancestor) || ast.IsTryStatement(ancestor) || ast.IsIterationStatement(ancestor, false) || ast.IsFunctionLikeDeclaration(ancestor)
	})
	if ancestor != nil && ast.IsTryStatement(ancestor) {
		state.MarkTryUses(func(u *TryUses) { u.UsesContinue = true })
		return luau.MakeList(luau.Create(&luau.ReturnStatement{
			Expression: state.TS(node, "TRY_CONTINUE"),
		}))
	}
	if ancestor == nil || !ast.IsSwitchStatement(ancestor) {
		return luau.MakeList(luau.Create(&luau.ContinueStatement{}))
	}
	// one flag per switch, shared by every continue that crosses it
	flag := state.switchContinueFlags[ancestor]
	if flag == nil {
		flag = luau.NewTempID("continue")
		if state.switchContinueFlags == nil {
			state.switchContinueFlags = map[*ast.Node]*luau.TemporaryIdentifier{}
		}
		state.switchContinueFlags[ancestor] = flag
	}
	return luau.MakeList[luau.Node](
		luau.Create(&luau.Assignment{
			Left:     flag,
			Operator: luau.AssignEqual,
			Right:    luau.NewBool(true),
		}),
		luau.Create(&luau.BreakStatement{}),
	)
}

// wrapSwitchForContinue puts the flag's declaration before the switch's repeat, so that it outlives the break, and
// carries on after it. It leaves a switch that no continue crosses alone.
func wrapSwitchForContinue(state *TransformState, node *ast.Node, repeat luau.Node) *luau.List {
	flag := state.switchContinueFlags[node]
	if flag == nil {
		return luau.MakeList(repeat)
	}
	return luau.MakeList[luau.Node](
		luau.Create(&luau.VariableDeclaration{
			Left:  flag,
			Right: luau.NewBool(false),
		}),
		repeat,
		luau.Create(&luau.IfStatement{
			Condition:  flag,
			Statements: createContinueStatements(state, node),
			ElseBody:   luau.NewList(),
		}),
	)
}
