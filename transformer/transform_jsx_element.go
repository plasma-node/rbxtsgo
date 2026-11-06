package transformer

// Port of src/TSTransformer/nodes/expressions/transformJsxElement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformJsxElement(state *TransformState, node *ast.Node) luau.Expression {
	element := node.AsJsxElement()
	openingElement := element.OpeningElement.AsJsxOpeningElement()
	return transformJsx(state, node, openingElement.TagName, openingElement.Attributes, element.Children.Nodes)
}
