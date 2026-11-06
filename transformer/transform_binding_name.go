package transformer

// Port of src/TSTransformer/nodes/binding/transformBindingName.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformBindingName(state *TransformState, name *ast.Node, initializers *luau.List) luau.AnyIdentifier {
	var id luau.AnyIdentifier
	if ast.IsIdentifier(name) {
		id = transformIdentifierDefined(state, name)
	} else {
		id = luau.NewTempID("binding")
		initializers.PushList(state.CapturePrereqs(func() {
			if ast.IsArrayBindingPattern(name) {
				transformArrayBindingPattern(state, name, id)
			} else {
				transformObjectBindingPattern(state, name, id)
			}
		}))
	}
	return id
}
