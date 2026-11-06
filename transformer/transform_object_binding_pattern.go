package transformer

// Port of src/TSTransformer/nodes/binding/transformObjectBindingPattern.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformObjectBindingPattern(state *TransformState, bindingPattern *ast.Node, parentID luau.AnyIdentifier) {
	validateNotAnyType(state, bindingPattern)
	for _, element := range bindingPattern.AsBindingPattern().Elements.Nodes {
		bindingElement := element.AsBindingElement()
		if bindingElement.DotDotDotToken != nil {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoSpreadDestructuring.At(shared.AtNode(element)))
			return
		}
		name := element.Name()
		prop := bindingElement.PropertyName
		if ast.IsIdentifier(name) {
			propOrName := prop
			if propOrName == nil {
				propOrName = name
			}
			value := objectAccessor(state, parentID, state.GetType(bindingPattern), propOrName)
			id := transformVariable(state, name, value)
			if bindingElement.Initializer != nil {
				state.Prereq(transformInitializer(state, id, bindingElement.Initializer))
			}
		} else {
			// if name is not identifier, it must be a binding pattern
			// in that case, prop is guaranteed to exist
			if prop == nil {
				panic(shared.AssertionMessage(""))
			}
			value := objectAccessor(state, parentID, state.GetType(bindingPattern), prop)
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
}
