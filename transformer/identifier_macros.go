package transformer

// Port of src/TSTransformer/macros/identifierMacros.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// identifierMacros is IDENTIFIER_MACROS.
func identifierMacros() map[string]identifierMacro {
	return map[string]identifierMacro{
		"Promise": func(state *TransformState, node *ast.Node) luau.Expression {
			return state.TS(node, "Promise")
		},
	}
}
