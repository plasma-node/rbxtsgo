package transformer

// Port of src/TSTransformer/nodes/expressions/transformJsxFragment.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/parser"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformJsxFragment(state *TransformState, node *ast.Node) luau.Expression {
	jsxFactoryEntity := state.Resolver.GetJsxFactoryEntity(node)
	if jsxFactoryEntity == nil {
		panic(shared.AssertionMessage("Expected jsxFactoryEntity to be defined"))
	}

	createElementExpression := convertToIndexableExpression(transformEntityName(state, jsxFactoryEntity))

	// getJsxFragmentFactoryEntity() doesn't seem to default to "Fragment"..
	// but the typechecker does, so we should follow that behavior
	jsxFragmentFactoryEntity := state.Resolver.GetJsxFragmentFactoryEntity(node)
	if jsxFragmentFactoryEntity == nil {
		jsxFragmentFactoryEntity = parser.ParseIsolatedEntityName("Fragment")
	}
	if jsxFragmentFactoryEntity == nil {
		panic(shared.AssertionMessage("Unable to find valid jsxFragmentFactoryEntity"))
	}

	args := []luau.Expression{transformEntityName(state, jsxFragmentFactoryEntity)}

	transformedChildren := transformJsxChildren(state, node.AsJsxFragment().Children.Nodes)

	// props parameter
	if len(transformedChildren) > 0 {
		args = append(args, luau.NewNil())
	}

	args = append(args, transformedChildren...)

	return luau.NewCall(createElementExpression, args...)
}
