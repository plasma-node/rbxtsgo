package transformer

// Port of src/TSTransformer/nodes/statements/transformWhileStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformWhileStatement(state *TransformState, node *ast.Node) *luau.List {
	whileStatement := node.AsWhileStatement()
	whileStatements := luau.NewList()

	conditionExp, conditionPrereqs := Capture(state, func() luau.Expression {
		return createTruthinessChecks(state, transformExpression(state, whileStatement.Expression), whileStatement.Expression)
	})

	if !conditionPrereqs.IsEmpty() {
		whileStatements.PushList(conditionPrereqs)
		whileStatements.Push(luau.Create(&luau.IfStatement{
			Condition:  luau.NewUnary(luau.UnaryNot, conditionExp),
			Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
			ElseBody:   luau.NewList(),
		}))
		conditionExp = luau.NewBool(true)
	}

	whileStatements.PushList(transformStatementList(state, whileStatement.Statement, getStatements(whileStatement.Statement), nil))

	return luau.MakeList(luau.Create(&luau.WhileStatement{
		Condition:  conditionExp,
		Statements: whileStatements,
	}))
}
