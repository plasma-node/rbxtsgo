package transformer

// Port of src/TSTransformer/nodes/statements/transformTryStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformCatchClause(state *TransformState, node *ast.Node) *luau.FunctionExpression {
	catchClause := node.AsCatchClause()
	parameters := luau.NewList()
	statements := luau.NewList()

	if catchClause.VariableDeclaration != nil {
		parameters.Push(transformBindingName(state, catchClause.VariableDeclaration.Name(), statements))
	}

	statements.PushList(transformStatementList(state, catchClause.Block, catchClause.Block.AsBlock().Statements.Nodes, nil))

	catchFunction := luau.Create(&luau.FunctionExpression{
		Parameters:   parameters,
		HasDotDotDot: false,
		Statements:   statements,
	})

	return catchFunction
}

func transformIntoTryCall(state *TransformState, node *ast.Node, exitTypeID *luau.TemporaryIdentifier, returnsID *luau.TemporaryIdentifier, tryUses *TryUses) luau.Statement {
	tryStatement := node.AsTryStatement()
	var tryCallArgs []luau.Expression

	tryCallArgs = append(tryCallArgs, luau.Create(&luau.FunctionExpression{
		Parameters:   luau.NewList(),
		HasDotDotDot: false,
		Statements:   transformStatementList(state, tryStatement.TryBlock, tryStatement.TryBlock.AsBlock().Statements.Nodes, nil),
	}))

	if tryStatement.CatchClause != nil {
		tryCallArgs = append(tryCallArgs, transformCatchClause(state, tryStatement.CatchClause))
	} else {
		if tryStatement.FinallyBlock == nil {
			panic(shared.AssertionMessage(""))
		}
		tryCallArgs = append(tryCallArgs, luau.NewNil())
	}

	if tryStatement.FinallyBlock != nil {
		tryCallArgs = append(tryCallArgs, luau.Create(&luau.FunctionExpression{
			Parameters:   luau.NewList(),
			HasDotDotDot: false,
			Statements:   transformStatementList(state, tryStatement.FinallyBlock, tryStatement.FinallyBlock.AsBlock().Statements.Nodes, nil),
		}))
	}

	if !tryUses.UsesReturn && !tryUses.UsesBreak && !tryUses.UsesContinue {
		return luau.Create(&luau.CallStatement{
			Expression: luau.NewCall(state.TS(node, "try"), tryCallArgs...),
		})
	}

	return luau.Create(&luau.VariableDeclaration{
		Left:  luau.MakeList(exitTypeID, returnsID),
		Right: luau.NewCall(state.TS(node, "try"), tryCallArgs...),
	})
}

func createFlowControlCondition(state *TransformState, node *ast.Node, exitTypeID *luau.TemporaryIdentifier, flowControlConstant string) luau.Expression {
	return luau.NewBinary(exitTypeID, luau.BinaryEqual, state.TS(node, flowControlConstant))
}

type flowControlCase struct {
	condition  luau.Expression
	statements *luau.List
}

func collapseFlowControlCases(exitTypeID *luau.TemporaryIdentifier, cases []flowControlCase) *luau.List {
	if !(len(cases) > 0) {
		panic(shared.AssertionMessage(""))
	}

	nextStatements := luau.Create(&luau.IfStatement{
		Condition:  exitTypeID,
		Statements: cases[len(cases)-1].statements,
		ElseBody:   luau.NewList(),
	})

	for i := len(cases) - 2; i >= 0; i-- {
		condition := cases[i].condition
		if condition == nil {
			condition = exitTypeID
		}
		nextStatements = luau.Create(&luau.IfStatement{
			Condition:  condition,
			Statements: cases[i].statements,
			ElseBody:   nextStatements,
		})
	}

	return luau.MakeList(nextStatements)
}

func transformFlowControl(state *TransformState, node *ast.Node, exitTypeID *luau.TemporaryIdentifier, returnsID *luau.TemporaryIdentifier, tryUses *TryUses) *luau.List {
	var flowControlCases []flowControlCase

	if !tryUses.UsesReturn && !tryUses.UsesBreak && !tryUses.UsesContinue {
		return luau.NewList()
	}

	returnBlocked := isReturnBlockedByTryStatement(node.Parent)
	breakBlocked := isBreakBlockedByTryStatement(node.Parent)

	if tryUses.UsesReturn && returnBlocked {
		state.MarkTryUses(func(u *TryUses) { u.UsesReturn = true })
	}
	if tryUses.UsesBreak && breakBlocked {
		state.MarkTryUses(func(u *TryUses) { u.UsesBreak = true })
	}
	if tryUses.UsesContinue && breakBlocked {
		state.MarkTryUses(func(u *TryUses) { u.UsesContinue = true })
	}

	if tryUses.UsesReturn {
		if returnBlocked {
			flowControlCases = append(flowControlCases, flowControlCase{
				condition: createFlowControlCondition(state, node, exitTypeID, "TRY_RETURN"),
				statements: luau.MakeList(luau.Create(&luau.ReturnStatement{
					Expression: luau.MakeList(exitTypeID, returnsID),
				})),
			})
			if breakBlocked {
				return collapseFlowControlCases(exitTypeID, flowControlCases)
			}
		} else {
			flowControlCases = append(flowControlCases, flowControlCase{
				condition: createFlowControlCondition(state, node, exitTypeID, "TRY_RETURN"),
				statements: luau.MakeList(luau.Create(&luau.ReturnStatement{
					Expression: luau.NewCall(luau.Globals.Unpack, returnsID),
				})),
			})
		}
	}

	if tryUses.UsesBreak || tryUses.UsesContinue {
		if breakBlocked {
			flowControlCases = append(flowControlCases, flowControlCase{
				statements: luau.MakeList(luau.Create(&luau.ReturnStatement{
					Expression: exitTypeID,
				})),
			})
		} else {
			if tryUses.UsesBreak {
				flowControlCases = append(flowControlCases, flowControlCase{
					condition:  createFlowControlCondition(state, node, exitTypeID, "TRY_BREAK"),
					statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				})
			}
			if tryUses.UsesContinue {
				flowControlCases = append(flowControlCases, flowControlCase{
					condition:  createFlowControlCondition(state, node, exitTypeID, "TRY_CONTINUE"),
					statements: createFlowControlContinue(state, node),
				})
			}
		}
	}

	return collapseFlowControlCases(exitTypeID, flowControlCases)
}

// createFlowControlContinue is the continue after a TS.try call, which may be inside a switch.
func createFlowControlContinue(state *TransformState, node *ast.Node) *luau.List {
	if state.Data.ProjectOptions.FixEmit {
		return createContinueStatements(state, node)
	}
	return luau.MakeList(luau.Create(&luau.ContinueStatement{}))
}

func transformTryStatement(state *TransformState, node *ast.Node) *luau.List {
	statements := luau.NewList()
	exitTypeID := luau.NewTempID("exitType")
	returnsID := luau.NewTempID("returns")

	tryUses := state.PushTryUsesStack()
	statements.Push(transformIntoTryCall(state, node, exitTypeID, returnsID, tryUses))
	state.PopTryUsesStack()

	statements.PushList(transformFlowControl(state, node, exitTypeID, returnsID, tryUses))

	return statements
}
