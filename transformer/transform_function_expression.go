package transformer

// Port of src/TSTransformer/nodes/expressions/transformFunctionExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformFunctionExpression(state *TransformState, node *ast.Node) luau.Expression {
	if node.Name() != nil {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoFunctionExpressionName.At(shared.AtNode(node.Name())))
	}

	transformed := transformParameters(state, node)
	statements, parameters, hasDotDotDot := transformed.statements, transformed.parameters, transformed.hasDotDotDot

	body := node.Body()
	if ast.IsFunctionBlock(body) {
		statements.PushList(transformStatementList(state, body, body.AsBlock().Statements.Nodes, nil))
	} else {
		returnStatements, prereqs := Capture(state, func() *luau.List { return transformReturnStatementInner(state, body) })
		statements.PushList(prereqs)
		statements.PushList(returnStatements)
	}

	isAsync := ast.HasSyntacticModifier(node, ast.ModifierFlagsAsync)

	if node.BodyData().AsteriskToken != nil {
		if isAsync {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoAsyncGeneratorFunctions.At(shared.AtNode(node)))
		}
		statements = wrapStatementsAsGenerator(state, node, statements)
	}

	var expression luau.Expression = luau.Create(&luau.FunctionExpression{
		HasDotDotDot: hasDotDotDot,
		Parameters:   parameters,
		Statements:   statements,
	})

	if isAsync {
		expression = luau.NewCall(state.TS(node, "async"), expression)
	}

	return expression
}
