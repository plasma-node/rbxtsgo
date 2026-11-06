package transformer

// Port of src/TSTransformer/nodes/jsx/transformJsx.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformJsx(state *TransformState, node *ast.Node, tagName *ast.Node, attributes *ast.Node, children []*ast.Node) *luau.CallExpression {
	// jsxFactoryEntity seems to always be defined and will default to `React.createElement`
	jsxFactoryEntity := state.Resolver.GetJsxFactoryEntity(node)
	if jsxFactoryEntity == nil {
		panic(shared.AssertionMessage("Expected jsxFactoryEntity to be defined"))
	}

	createElementExpression := convertToIndexableExpression(transformEntityName(state, jsxFactoryEntity))

	tagNameExp := transformJsxTagName(state, tagName)

	var attributesPtr *mapPointer
	if len(attributes.AsJsxAttributes().Properties.Nodes) > 0 {
		attributesPtr = createMapPointer("attributes")
		transformJsxAttributes(state, attributes, attributesPtr)
	}

	transformedChildren := transformJsxChildren(state, children)

	args := []luau.Expression{tagNameExp}

	if attributesPtr != nil {
		args = append(args, attributesPtr.value)
	} else if len(transformedChildren) > 0 {
		args = append(args, luau.NewNil())
	}

	args = append(args, transformedChildren...)

	return luau.NewCall(createElementExpression, args...)
}
