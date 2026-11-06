package transformer

// Port of src/TSTransformer/nodes/transformLogical.ts.

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

type logicalChainItem struct {
	node       *ast.Node
	typ        *checker.Type
	expression luau.Expression
	statements *luau.List
	inline     bool
}

// flattenByOperator splits `node` recursively by binary operator `operatorKind` into an Array<ts.Expression>
//
// i.e. using `&&` as our operator, `a && b && c` -> `[a, b, c]`
func flattenByOperator(node *ast.Node, operatorKind ast.Kind) []*ast.Node {
	var result []*ast.Node
	for ast.IsBinaryExpression(node) && node.AsBinaryExpression().OperatorToken.Kind == operatorKind {
		result = slices.Insert(result, 0, node.AsBinaryExpression().Right)
		node = node.AsBinaryExpression().Left
	}
	result = slices.Insert(result, 0, node)
	return result
}

// getLogicalChain builds an Array<LogicalChainItem> using `splitBinaryChain` and
// captures all of the prerequisite statements for each expression
func getLogicalChain(state *TransformState, binaryExp *ast.Node, binaryOperatorKind ast.Kind, enableInlining bool) []*logicalChainItem {
	array := flattenByOperator(binaryExp, binaryOperatorKind)
	chain := make([]*logicalChainItem, len(array))
	for index, node := range array {
		typ := state.GetType(node)
		expression, statements := Capture(state, func() luau.Expression { return transformExpression(state, node) })
		inline := false
		if enableInlining {
			willWrap := index < len(array)-1 && willCreateTruthinessChecks(typ)
			inline = statements.IsEmpty() && !willWrap
		}
		chain[index] = &logicalChainItem{node, typ, expression, statements, inline}
	}
	return chain
}

// buildLogicalChainPrereqs recursively creates prerequisite statements for non-inlined expressions in a logical chain
func buildLogicalChainPrereqs(
	state *TransformState,
	chain []*logicalChainItem,
	conditionID *luau.TemporaryIdentifier,
	buildCondition func(conditionID *luau.TemporaryIdentifier, node *ast.Node) luau.Expression,
	index int,
) {
	expInfo := chain[index]
	state.PrereqList(expInfo.statements)
	if index == 0 {
		state.Prereq(luau.Create(&luau.VariableDeclaration{
			Left:  conditionID,
			Right: expInfo.expression,
		}))
	} else {
		state.Prereq(luau.Create(&luau.Assignment{
			Left:     conditionID,
			Operator: luau.AssignEqual,
			Right:    expInfo.expression,
		}))
	}
	if index+1 < len(chain) {
		state.Prereq(luau.Create(&luau.IfStatement{
			Condition: buildCondition(conditionID, expInfo.node),
			Statements: state.CapturePrereqs(func() {
				buildLogicalChainPrereqs(state, chain, conditionID, buildCondition, index+1)
			}),
			ElseBody: luau.NewList(),
		}))
	}
}

// mergeInlineExpressions merges any neighboring inline LogicalChainItems together, builds new expressions using
// `binaryOperator`
//
// i.e. using `and` as our operator,
//
//	[{ expression: a, inline: true }, { expression: b, inline: true }, { expression: c, inline: false }]
//
// turns into
//
//	[{ expression: a and b, inline: true }, { expression: c, inline: false }]
//
// Upstream splices the array in place. Here it returns the shortened slice.
func mergeInlineExpressions(chain []*logicalChainItem, binaryOperator luau.BinaryOperator) []*logicalChainItem {
	for i := 0; i < len(chain); i++ {
		info := chain[i]
		if info.inline {
			exps := []luau.Expression{info.expression}
			j := i + 1
			for j < len(chain) && chain[j].inline {
				exps = append(exps, chain[j].expression)
				chain = slices.Delete(chain, j, j+1)
			}
			info.expression = binaryExpressionChain(exps, binaryOperator)
		}
	}
	return chain
}

// buildInlineConditionExpression solves the simple case for `&&` / `||` where items can be inlined into `and` / `or`
// expressions.
func buildInlineConditionExpression(
	state *TransformState,
	node *ast.Node,
	tsBinaryOperator ast.Kind,
	luaBinaryOperator luau.BinaryOperator,
	buildCondition func(conditionID *luau.TemporaryIdentifier, node *ast.Node) luau.Expression,
) luau.Expression {
	chain := getLogicalChain(state, node, tsBinaryOperator, true)

	chain = mergeInlineExpressions(chain, luaBinaryOperator)

	// single inline at the end, no temp variable needed
	if len(chain) == 1 && chain[0].inline {
		return chain[0].expression
	}

	conditionID := luau.NewTempID("condition")
	buildLogicalChainPrereqs(state, chain, conditionID, buildCondition, 0)
	return conditionID
}

func transformLogical(state *TransformState, node *ast.Node) luau.Expression {
	operatorKind := node.AsBinaryExpression().OperatorToken.Kind
	if operatorKind == ast.KindAmpersandAmpersandToken {
		return buildInlineConditionExpression(state, node, operatorKind, luau.BinaryAnd, func(conditionID *luau.TemporaryIdentifier, node *ast.Node) luau.Expression {
			return createTruthinessChecks(state, conditionID, node)
		})
	} else if operatorKind == ast.KindBarBarToken {
		return buildInlineConditionExpression(state, node, operatorKind, luau.BinaryOr, func(conditionID *luau.TemporaryIdentifier, node *ast.Node) luau.Expression {
			return luau.NewUnary(luau.UnaryNot, createTruthinessChecks(state, conditionID, node))
		})
	} else if operatorKind == ast.KindQuestionQuestionToken {
		conditionBuilder := func(conditionID *luau.TemporaryIdentifier, _ *ast.Node) luau.Expression {
			return luau.NewBinary(conditionID, luau.BinaryEqual, luau.NewNil())
		}
		if !isPossiblyType(state.GetType(node), isBooleanLiteralType(state, false)) {
			return buildInlineConditionExpression(state, node, operatorKind, luau.BinaryOr, conditionBuilder)
		}
		chain := getLogicalChain(state, node, ast.KindQuestionQuestionToken, false)
		conditionID := luau.NewTempID("condition")
		buildLogicalChainPrereqs(state, chain, conditionID, conditionBuilder, 0)
		return conditionID
	}
	panic(shared.AssertionMessage("Operator not implemented: " + getKindName(operatorKind)))
}
