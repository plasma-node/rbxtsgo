package transformer

// Port of src/TSTransformer/nodes/binding/transformArrayBindingPattern.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformArrayBindingPattern(state *TransformState, bindingPattern *ast.Node, parentID luau.AnyIdentifier) {
	validateNotAnyType(state, bindingPattern)

	index := 0
	idStack := []luau.AnyIdentifier{}
	accessor := getAccessorForBindingType(state, bindingPattern, state.GetType(bindingPattern))
	for _, element := range bindingPattern.AsBindingPattern().Elements.Nodes {
		// ts.isOmittedExpression(element). TS 7's parser makes a BindingElement with no name for a hole, where TS 5.5.3's
		// makes an OmittedExpression.
		if element.Name() == nil {
			accessor(state, parentID, index, &idStack, true)
		} else {
			bindingElement := element.AsBindingElement()
			if bindingElement.DotDotDotToken != nil {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoSpreadDestructuring.At(shared.AtNode(element)))
				return
			}
			name := element.Name()
			value := accessor(state, parentID, index, &idStack, false)
			if ast.IsIdentifier(name) {
				id := transformVariable(state, name, value)
				if bindingElement.Initializer != nil {
					state.Prereq(transformInitializer(state, id, bindingElement.Initializer))
				}
			} else {
				id := state.PushToVar(value, "binding")
				if bindingElement.Initializer != nil {
					state.Prereq(transformInitializer(state, id, bindingElement.Initializer))
				}
				if ast.IsArrayBindingPattern(name) {
					transformArrayBindingPattern(state, name, id)
				} else {
					transformObjectBindingPattern(state, name, id)
				}
			}
		}
		index++
	}
}
