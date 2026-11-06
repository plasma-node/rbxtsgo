package transformer

// Port of src/TSTransformer/nodes/expressions/transformExpression.ts. TRANSFORMER_BY_KIND is a switch, in the same
// order, because a package-level map of these functions would be an initialization cycle.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformExpressionNoEmit() luau.Expression {
	return luau.NewNone()
}

func transformExpressionDiagnostic(state *TransformState, factory *shared.DiagnosticFactory, node *ast.Node) luau.Expression {
	state.Diagnostic.AddDiagnostic(factory.At(shared.AtNode(node)))
	return transformExpressionNoEmit()
}

func transformExpression(state *TransformState, node *ast.Node) luau.Expression {
	switch node.Kind {
	// banned expressions
	case ast.KindBigIntLiteral:
		return transformExpressionDiagnostic(state, shared.Errors.NoBigInt, node)
	case ast.KindNullKeyword:
		return transformExpressionDiagnostic(state, shared.Errors.NoNullLiteral, node)
	case ast.KindPrivateIdentifier:
		return transformExpressionDiagnostic(state, shared.Errors.NoPrivateIdentifier, node)
	case ast.KindRegularExpressionLiteral:
		return transformExpressionDiagnostic(state, shared.Errors.NoRegex, node)
	case ast.KindTypeOfExpression:
		return transformExpressionDiagnostic(state, shared.Errors.NoTypeOfExpression, node)

	// skip transforms
	case ast.KindImportKeyword:
		return transformExpressionNoEmit()

	// regular transforms
	case ast.KindArrayLiteralExpression:
		return transformArrayLiteralExpression(state, node)
	case ast.KindArrowFunction:
		return transformFunctionExpression(state, node)
	case ast.KindAsExpression:
		return transformTypeExpression(state, node)
	case ast.KindAwaitExpression:
		return transformAwaitExpression(state, node)
	case ast.KindBinaryExpression:
		return transformBinaryExpression(state, node)
	case ast.KindCallExpression:
		return transformCallExpression(state, node)
	case ast.KindClassExpression:
		return transformClassExpression(state, node)
	case ast.KindConditionalExpression:
		return transformConditionalExpression(state, node)
	case ast.KindDeleteExpression:
		return transformDeleteExpression(state, node)
	case ast.KindElementAccessExpression:
		return transformElementAccessExpression(state, node)
	case ast.KindExpressionWithTypeArguments:
		return transformTypeExpression(state, node)
	case ast.KindFalseKeyword:
		return transformFalseKeyword()
	case ast.KindFunctionExpression:
		return transformFunctionExpression(state, node)
	case ast.KindIdentifier:
		return transformIdentifier(state, node)
	case ast.KindJsxElement:
		return transformJsxElement(state, node)
	case ast.KindJsxExpression:
		return transformJsxExpression(state, node)
	case ast.KindJsxFragment:
		return transformJsxFragment(state, node)
	case ast.KindJsxSelfClosingElement:
		return transformJsxSelfClosingElement(state, node)
	case ast.KindNewExpression:
		return transformNewExpression(state, node)
	case ast.KindNonNullExpression:
		return transformTypeExpression(state, node)
	case ast.KindNoSubstitutionTemplateLiteral:
		return transformNoSubstitutionTemplateLiteral(state, node)
	case ast.KindNumericLiteral:
		return transformNumericLiteral(state, node)
	case ast.KindObjectLiteralExpression:
		return transformObjectLiteralExpression(state, node)
	case ast.KindOmittedExpression:
		return transformOmittedExpression()
	case ast.KindParenthesizedExpression:
		return transformParenthesizedExpression(state, node)
	case ast.KindPostfixUnaryExpression:
		return transformPostfixUnaryExpression(state, node)
	case ast.KindPrefixUnaryExpression:
		return transformPrefixUnaryExpression(state, node)
	case ast.KindPropertyAccessExpression:
		return transformPropertyAccessExpression(state, node)
	case ast.KindSatisfiesExpression:
		return transformTypeExpression(state, node)
	case ast.KindSpreadElement:
		return transformSpreadElement(state, node)
	case ast.KindStringLiteral:
		return transformStringLiteral(state, node)
	case ast.KindSuperKeyword:
		return transformSuperKeyword()
	case ast.KindTaggedTemplateExpression:
		return transformTaggedTemplateExpression(state, node)
	case ast.KindTemplateExpression:
		return transformTemplateExpression(state, node)
	case ast.KindThisKeyword:
		return transformThisExpression(state, node)
	case ast.KindTrueKeyword:
		return transformTrueKeyword()
	case ast.KindTypeAssertionExpression:
		return transformTypeExpression(state, node)
	case ast.KindVoidExpression:
		return transformVoidExpression(state, node)
	case ast.KindYieldExpression:
		return transformYieldExpression(state, node)
	}
	panic(shared.AssertionMessage("Unknown expression: " + getKindName(node.Kind)))
}
