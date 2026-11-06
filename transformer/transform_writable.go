package transformer

// Port of src/TSTransformer/nodes/transformWritable.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformWritableExpression(state *TransformState, node *ast.Node, readAfterWrite bool) luau.WritableExpression {
	if ast.IsPrototypeAccess(node) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoPrototype.At(shared.AtNode(node)))
	}
	if ast.IsPropertyAccessExpression(node) {
		expression := transformExpression(state, node.Expression())
		var indexable luau.IndexableExpression
		if readAfterWrite {
			indexable = state.PushToVarIfNonID(expression, "exp")
		} else {
			indexable = convertToIndexableExpression(expression)
		}
		return luau.NewProperty(indexable, node.Name().Text())
	} else if ast.IsElementAccessExpression(node) {
		ordered := ensureTransformOrder(state, []*ast.Node{node.Expression(), node.AsElementAccessExpression().ArgumentExpression}, nil)
		expression, index := ordered[0], ordered[1]
		indexExp := addOneIfArrayType(state, state.GetType(node.Expression()), index)
		var indexable luau.IndexableExpression
		if readAfterWrite {
			indexable = state.PushToVarIfNonID(expression, "exp")
		} else {
			indexable = convertToIndexableExpression(expression)
		}
		if readAfterWrite {
			indexExp = state.PushToVarIfComplex(indexExp, "index")
		}
		return luau.Create(&luau.ComputedIndexExpression{
			Expression: indexable,
			Index:      indexExp,
		})
	}
	transformed := transformExpression(state, SkipDownwards(node))
	writable, ok := transformed.(luau.WritableExpression)
	if !ok {
		panic(shared.AssertionMessage(""))
	}
	return writable
}

// transformWritableAssignmentResult is the object transformWritableAssignment returns.
type transformWritableAssignmentResult struct {
	writable luau.WritableExpression
	readable luau.WritableExpression
	value    luau.Expression
}

func transformWritableAssignment(state *TransformState, writeNode *ast.Node, valueNode *ast.Node, readAfterWrite bool, readBeforeWrite bool) transformWritableAssignmentResult {
	writable := transformWritableExpression(state, writeNode, readAfterWrite)
	value, prereqs := Capture(state, func() luau.Expression { return transformExpression(state, valueNode) })

	// if !readBeforeWrite, readable won't be used anyways
	readable := writable
	if readBeforeWrite && !prereqs.IsEmpty() {
		readable = state.PushToVar(writable, "readable")
	}
	state.PrereqList(prereqs)

	return transformWritableAssignmentResult{writable: writable, readable: readable, value: value}
}
