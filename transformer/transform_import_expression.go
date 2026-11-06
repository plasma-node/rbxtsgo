package transformer

// Port of src/TSTransformer/nodes/expressions/transformImportExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformImportExpression(state *TransformState, node *ast.Node) luau.Expression {
	var moduleSpecifier *ast.Node
	if arguments := node.Arguments(); len(arguments) > 0 {
		moduleSpecifier = arguments[0]
	}

	if moduleSpecifier == nil || !ast.IsStringLiteral(moduleSpecifier) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoNonStringModuleSpecifier.At(shared.AtNode(node)))
		return luau.NewNone()
	}

	importExpression := createImportExpression(state, ast.GetSourceFileOfNode(node), moduleSpecifier)
	resolveId := luau.NewID("resolve")

	return luau.NewCall(luau.NewProperty(state.TS(node, "Promise"), "new"),
		luau.Create(&luau.FunctionExpression{
			HasDotDotDot: false,
			Parameters:   luau.MakeList(resolveId),
			Statements: luau.MakeList(
				luau.Create(&luau.CallStatement{
					Expression: luau.NewCall(resolveId, importExpression),
				}),
			),
		}),
	)
}
