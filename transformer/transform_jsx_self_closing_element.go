package transformer

// Port of src/TSTransformer/nodes/expressions/transformJsxSelfClosingElement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformJsxSelfClosingElement(state *TransformState, node *ast.Node) luau.Expression {
	element := node.AsJsxSelfClosingElement()
	return transformJsx(state, node, element.TagName, element.Attributes, []*ast.Node{})
}
