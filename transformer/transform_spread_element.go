package transformer

// Port of src/TSTransformer/nodes/expressions/transformSpreadElement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformSpreadElement(state *TransformState, node *ast.Node) luau.Expression {
	validateNotAnyType(state, node.Expression())

	// array literal is caught and handled separately in transformArrayLiteralExpression.ts
	if ast.IsArrayLiteralExpression(node.Parent) || node.Parent.ArgumentList() == nil {
		panic(shared.AssertionMessage(""))
	}
	arguments := node.Parent.Arguments()
	if arguments[len(arguments)-1] != node {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoPrecedingSpreadElement.At(shared.AtNode(node)))
	}

	expression := transformExpression(state, node.Expression())

	t := state.GetType(node.Expression())
	if isDefinitelyType(t, isArrayType(state)) {
		return luau.NewCall(luau.Globals.Unpack, expression)
	} else {
		addIterableToArrayBuilder := getAddIterableToArrayBuilder(state, node.Expression(), t)
		arrayId := state.PushToVar(luau.NewArray(), "array")
		lengthId := state.PushToVar(luau.NewNumber(0), "length")
		state.PrereqList(addIterableToArrayBuilder(state, expression, arrayId, lengthId, 0, false))
		return luau.NewCall(luau.Globals.Unpack, arrayId)
	}
}
