package transformer

// Port of src/TSTransformer/nodes/expressions/transformThisExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformThisExpression(state *TransformState, node *ast.Node) luau.Expression {
	symbol := state.Checker.GetSymbolAtLocation(node)
	if symbol == getSymbolOrThrow(state, "globalThis") {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoGlobalThis.At(shared.AtNode(node)))
	}

	if symbol != nil {
		container := ast.GetThisContainer(node, false, false)

		// ts.hasStaticModifier doesn't work on static blocks
		isStatic := ast.HasStaticModifier(container) || ast.IsClassStaticBlockDeclaration(container)

		// MethodDeclaration creates it's own implicit this
		if isStatic && !ast.IsMethodDeclaration(container) && ast.IsClassLike(container.Parent) {
			identifier := state.ClassIdentifierMap[container.Parent]
			if identifier != nil {
				return identifier
			}
		}
	}

	return luau.Globals.Self
}
