package transformer

// Port of src/TSTransformer/nodes/statements/transformSwitchStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

type transformCaseClauseExpressionResult struct {
	condition        luau.Expression
	prereqStatements *luau.List
}

func transformCaseClauseExpression(state *TransformState, caseClauseExpression *ast.Node, switchExpression luau.Expression, fallThroughFlagID *luau.TemporaryIdentifier, canFallThroughTo bool) transformCaseClauseExpressionResult {
	expression, prereqStatements := Capture(state, func() luau.Expression { return transformExpression(state, caseClauseExpression) })

	expression = luau.Create(&luau.ParenthesizedExpression{Expression: expression})

	var condition luau.Expression = luau.NewBinary(switchExpression, luau.BinaryEqual, expression)

	if canFallThroughTo {
		if !prereqStatements.IsEmpty() {
			noFallThroughCondition := luau.NewUnary(luau.UnaryNot, fallThroughFlagID)

			prereqStatements.Push(luau.Create(&luau.Assignment{
				Left:     fallThroughFlagID,
				Operator: luau.AssignEqual,
				Right:    condition,
			}))

			prereqStatements = luau.MakeList(luau.Create(&luau.IfStatement{
				Condition:  noFallThroughCondition,
				Statements: prereqStatements,
				ElseBody:   luau.NewList(),
			}))

			condition = fallThroughFlagID
		} else {
			condition = luau.NewBinary(fallThroughFlagID, luau.BinaryOr, condition)
		}
	}

	return transformCaseClauseExpressionResult{
		condition:        condition,
		prereqStatements: prereqStatements,
	}
}

type transformCaseClauseResult struct {
	canFallThroughFrom bool
	prereqs            *luau.List
	clauseStatements   *luau.List
}

func transformCaseClause(state *TransformState, node *ast.Node, switchExpression luau.Expression, fallThroughFlagID *luau.TemporaryIdentifier, canFallThroughTo bool, shouldUpdateFallThroughFlag bool) transformCaseClauseResult {
	clause := node.AsCaseOrDefaultClause()
	caseClauseExpression := transformCaseClauseExpression(state, clause.Expression, switchExpression, fallThroughFlagID, canFallThroughTo)
	condition, prereqStatements := caseClauseExpression.condition, caseClauseExpression.prereqStatements

	var nonEmptyStatements []*ast.Node
	for _, v := range clause.Statements.Nodes {
		if !ast.IsEmptyStatement(v) {
			nonEmptyStatements = append(nonEmptyStatements, v)
		}
	}
	var statements *luau.List
	if len(nonEmptyStatements) == 1 && ast.IsBlock(nonEmptyStatements[0]) {
		firstStatement := nonEmptyStatements[0]
		statements = transformStatementList(state, firstStatement, firstStatement.AsBlock().Statements.Nodes, nil)
	} else {
		statements = transformStatementList(state, node, clause.Statements.Nodes, nil)
	}

	canFallThroughFrom := statements.Tail == nil || !luau.IsFinalStatement(statements.Tail.Value)
	if canFallThroughFrom && shouldUpdateFallThroughFlag {
		statements.Push(luau.Create(&luau.Assignment{
			Left:     fallThroughFlagID,
			Operator: luau.AssignEqual,
			Right:    luau.NewBool(true),
		}))
	}

	clauseStatements := luau.NewList()

	if hoistDeclaration := createHoistDeclaration(state, node); hoistDeclaration != nil {
		clauseStatements.Push(hoistDeclaration)
	}

	clauseStatements.Push(luau.Create(&luau.IfStatement{
		Condition:  condition,
		Statements: statements,
		ElseBody:   luau.NewList(),
	}))

	return transformCaseClauseResult{
		canFallThroughFrom: canFallThroughFrom,
		prereqs:            prereqStatements,
		clauseStatements:   clauseStatements,
	}
}

func transformSwitchStatement(state *TransformState, node *ast.Node) *luau.List {
	switchStatement := node.AsSwitchStatement()
	expression := state.PushToVarIfComplex(transformExpression(state, switchStatement.Expression), "exp")
	fallThroughFlagID := luau.NewTempID("fallthrough")

	isFallThroughFlagNeeded := false

	statements := luau.NewList()
	canFallThroughTo := false
	clauses := switchStatement.CaseBlock.AsCaseBlock().Clauses.Nodes
	for i := 0; i < len(clauses); i++ {
		caseClauseNode := clauses[i]

		if ast.IsCaseClause(caseClauseNode) {
			shouldUpdateFallThroughFlag := i < len(clauses)-1 && ast.IsCaseClause(clauses[i+1])
			clause := transformCaseClause(state, caseClauseNode, expression, fallThroughFlagID, canFallThroughTo, shouldUpdateFallThroughFlag)

			statements.PushList(clause.prereqs)
			statements.PushList(clause.clauseStatements)

			canFallThroughTo = clause.canFallThroughFrom

			if clause.canFallThroughFrom {
				isFallThroughFlagNeeded = true
			}
		} else {
			statements.PushList(transformStatementList(state, caseClauseNode, caseClauseNode.AsCaseOrDefaultClause().Statements.Nodes, nil))
			break
		}
	}

	if isFallThroughFlagNeeded {
		statements.Unshift(luau.Create(&luau.VariableDeclaration{
			Left:  fallThroughFlagID,
			Right: luau.NewBool(false),
		}))
	}

	return luau.MakeList(luau.Create(&luau.RepeatStatement{
		Condition:  luau.NewBool(true),
		Statements: statements,
	}))
}
