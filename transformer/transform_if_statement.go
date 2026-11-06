package transformer

// Port of src/TSTransformer/nodes/statements/transformIfStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformIfStatementInner(state *TransformState, node *ast.Node) *luau.IfStatement {
	ifStatement := node.AsIfStatement()
	condition := createTruthinessChecks(state, transformExpression(state, ifStatement.Expression), ifStatement.Expression)

	statements := transformStatementList(state, ifStatement.ThenStatement, getStatements(ifStatement.ThenStatement), nil)

	elseStatement := ifStatement.ElseStatement

	var elseBody luau.NodeOrList
	if elseStatement == nil {
		elseBody = luau.NewList()
	} else if ast.IsIfStatement(elseStatement) {
		elseIf, elseIfPrereqs := Capture(state, func() *luau.IfStatement { return transformIfStatementInner(state, elseStatement) })
		if elseIfPrereqs.IsEmpty() {
			elseBody = elseIf
		} else {
			elseIfStatements := luau.NewList()
			elseIfStatements.PushList(elseIfPrereqs)
			elseIfStatements.Push(elseIf)
			elseBody = elseIfStatements
		}
	} else {
		elseBody = transformStatementList(state, elseStatement, getStatements(elseStatement), nil)
	}

	return luau.Create(&luau.IfStatement{
		Condition:  condition,
		Statements: statements,
		ElseBody:   elseBody,
	})
}

func transformIfStatement(state *TransformState, node *ast.Node) *luau.List {
	return luau.MakeList(transformIfStatementInner(state, node))
}
