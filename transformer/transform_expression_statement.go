package transformer

// Port of src/TSTransformer/nodes/statements/transformExpressionStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// unaryExpressionParts returns the operand and operator of a ts.PrefixUnaryExpression or ts.PostfixUnaryExpression.
func unaryExpressionParts(node *ast.Node) (*ast.Node, ast.Kind) {
	if ast.IsPrefixUnaryExpression(node) {
		prefix := node.AsPrefixUnaryExpression()
		return prefix.Operand, prefix.Operator
	}
	postfix := node.AsPostfixUnaryExpression()
	return postfix.Operand, postfix.Operator
}

func transformUnaryExpressionStatement(state *TransformState, node *ast.Node) *luau.Assignment {
	operand, nodeOperator := unaryExpressionParts(node)
	writable := transformWritableExpression(state, operand, false)
	operator := luau.AssignSub
	if nodeOperator == ast.KindPlusPlusToken {
		operator = luau.AssignAdd
	}
	return luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: operator,
		Right:    luau.NewNumber(1),
	})
}

func transformExpressionStatementInner(state *TransformState, expression *ast.Node) *luau.List {
	if ast.IsBinaryExpression(expression) {
		binary := expression.AsBinaryExpression()
		operatorKind := binary.OperatorToken.Kind
		if ast.IsLogicalOrCoalescingAssignmentExpression(expression) {
			return transformLogicalOrCoalescingAssignmentExpressionStatement(state, expression)
		} else if ast.IsAssignmentOperator(operatorKind) &&
			!ast.IsArrayLiteralExpression(binary.Left) &&
			!ast.IsObjectLiteralExpression(binary.Left) {
			writableType := state.GetType(binary.Left)
			valueType := state.GetType(binary.Right)
			// "" is undefined
			operator := getSimpleAssignmentOperator(writableType, operatorKind, valueType)
			assignment := transformWritableAssignment(state, binary.Left, binary.Right, operator == "", operator == "")
			writable, readable, value := assignment.writable, assignment.readable, assignment.value
			if operator != "" {
				return luau.MakeList(luau.Create(&luau.Assignment{
					Left:     writable,
					Operator: operator,
					Right:    getAssignableValue(operator, value, valueType),
				}))
			} else {
				return luau.MakeList[luau.Node](createCompoundAssignmentStatement(state, expression, writable, writableType, readable, operatorKind, value, valueType))
			}
		}
	} else if ast.IsPrefixUnaryExpression(expression) || ast.IsPostfixUnaryExpression(expression) {
		if _, operator := unaryExpressionParts(expression); isUnaryAssignmentOperator(operator) {
			return luau.MakeList(transformUnaryExpressionStatement(state, expression))
		}
	}

	return wrapExpressionStatement(transformExpression(state, expression))
}

func transformExpressionStatement(state *TransformState, node *ast.Node) *luau.List {
	expression := SkipDownwards(node.AsExpressionStatement().Expression)
	return transformExpressionStatementInner(state, expression)
}
