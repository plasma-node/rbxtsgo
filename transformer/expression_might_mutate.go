package transformer

// Port of src/TSTransformer/util/expressionMightMutate.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func expressionMightMutate(state *TransformState, expression luau.Expression, node *ast.Node) bool {
	if luau.IsTemporaryIdentifier(expression) {
		// Assume tempIds are never re-assigned after being returned
		// TODO: Is this actually safe to assume?
		return false
	} else if parenthesized, ok := expression.(*luau.ParenthesizedExpression); ok {
		return expressionMightMutate(state, parenthesized.Expression, nil)
	} else if luau.IsSimplePrimitive(expression) {
		return false
	} else if luau.IsFunctionExpression(expression) {
		return false
	} else if luau.IsVarArgsLiteral(expression) {
		return false
	} else if ifExpression, ok := expression.(*luau.IfExpression); ok {
		return expressionMightMutate(state, ifExpression.Condition, nil) ||
			expressionMightMutate(state, ifExpression.Expression, nil) ||
			expressionMightMutate(state, ifExpression.Alternative, nil)
	} else if binary, ok := expression.(*luau.BinaryExpression); ok {
		return expressionMightMutate(state, binary.Left, nil) || expressionMightMutate(state, binary.Right, nil)
	} else if unary, ok := expression.(*luau.UnaryExpression); ok {
		return expressionMightMutate(state, unary.Expression, nil)
	} else if array, ok := expression.(*luau.Array); ok {
		return array.Members.Some(func(member luau.Node) bool {
			return expressionMightMutate(state, member.(luau.Expression), nil)
		})
	} else if set, ok := expression.(*luau.Set); ok {
		return set.Members.Some(func(member luau.Node) bool {
			return expressionMightMutate(state, member.(luau.Expression), nil)
		})
	} else if mapExpression, ok := expression.(*luau.Map); ok {
		return mapExpression.Fields.Some(func(value luau.Node) bool {
			field := value.(*luau.MapField)
			return expressionMightMutate(state, field.Index, nil) || expressionMightMutate(state, field.Value, nil)
		})
	}
	if node != nil {
		node = SkipDownwards(node)
		if ast.IsIdentifier(node) {
			symbol := state.Checker.GetSymbolAtLocation(node)
			if symbol != nil && !isSymbolMutable(state, symbol) {
				return false
			}
		}
	}
	// Identifier
	// ComputedIndexExpression
	// PropertyAccessExpression
	// CallExpression
	// MethodCallExpression
	return true
}
