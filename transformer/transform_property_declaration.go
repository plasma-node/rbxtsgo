package transformer

// Port of src/TSTransformer/nodes/class/transformPropertyDeclaration.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformPropertyDeclaration(state *TransformState, node *ast.Node, name luau.AnyIdentifier) *luau.List {
	if !ast.HasStaticModifier(node) {
		return luau.NewList()
	}

	if ast.IsPrivateIdentifier(node.Name()) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoPrivateIdentifier.At(shared.AtNode(node)))
		return luau.NewList()
	}

	if node.Initializer() == nil {
		return luau.NewList()
	}

	return luau.MakeList(luau.Create(&luau.Assignment{
		Left: luau.Create(&luau.ComputedIndexExpression{
			Expression: name,
			Index:      transformPropertyName(state, node.Name()),
		}),
		Operator: luau.AssignEqual,
		Right:    transformExpression(state, node.Initializer()),
	}))
}
