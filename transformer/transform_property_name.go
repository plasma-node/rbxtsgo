package transformer

// Port of src/TSTransformer/nodes/transformPropertyName.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformPropertyName(state *TransformState, name *ast.Node) luau.Expression {
	// identifier directly is from `{ a: value }`, so key must be "a"
	if ast.IsIdentifier(name) {
		return luau.NewString(name.Text())
	}
	// `name.expression`, if identifier, is from `{ [a]: value }`
	if ast.IsComputedPropertyName(name) {
		return transformExpression(state, name.Expression())
	}
	return transformExpression(state, name)
}
