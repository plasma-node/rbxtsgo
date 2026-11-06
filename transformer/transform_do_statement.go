package transformer

// Port of src/TSTransformer/nodes/statements/transformDoStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformDoStatement(state *TransformState, node *ast.Node) *luau.List {
	doStatement := node.AsDoStatement()
	expression, statement := doStatement.Expression, doStatement.Statement

	statements := transformStatementList(state, statement, getStatements(statement), nil)

	conditionIsInvertedInLuau := true
	if ast.IsPrefixUnaryExpression(expression) && expression.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken {
		expression = expression.AsPrefixUnaryExpression().Operand
		conditionIsInvertedInLuau = false
	}

	condition, conditionPrereqs := Capture(state, func() luau.Expression {
		return createTruthinessChecks(state, transformExpression(state, expression), expression)
	})

	repeatStatements := luau.NewList()
	repeatStatements.Push(luau.Create(&luau.DoStatement{
		Statements: statements,
	}))
	repeatStatements.PushList(conditionPrereqs)

	if conditionIsInvertedInLuau {
		condition = luau.NewUnary(luau.UnaryNot, condition)
	}
	return luau.MakeList(luau.Create(&luau.RepeatStatement{
		Statements: repeatStatements,
		Condition:  condition,
	}))
}
