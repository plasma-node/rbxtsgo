package transformer

// Port of src/TSTransformer/nodes/expressions/transformArrayLiteralExpression.ts.

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformArrayLiteralExpression(state *TransformState, node *ast.Node) luau.Expression {
	elements := node.AsArrayLiteralExpression().Elements.Nodes
	if !slices.ContainsFunc(elements, ast.IsSpreadElement) {
		return luau.NewArray(ensureTransformOrder(state, elements, nil)...)
	}

	ptr := createArrayPointer("array")
	lengthId := luau.NewTempID("length")
	lengthInitialized := false
	amtElementsSinceUpdate := 0.0

	updateLengthId := func() {
		right := luau.NewUnary(luau.UnaryLen, ptr.value)
		if lengthInitialized {
			state.Prereq(
				luau.Create(&luau.Assignment{
					Left:     lengthId,
					Operator: luau.AssignEqual,
					Right:    right,
				}),
			)
		} else {
			state.Prereq(
				luau.Create(&luau.VariableDeclaration{
					Left:  lengthId,
					Right: right,
				}),
			)
			lengthInitialized = true
		}
		amtElementsSinceUpdate = 0
	}

	for i := range elements {
		element := elements[i]
		if ast.IsSpreadElement(element) {
			if luau.IsArray(ptr.value) {
				disableArrayInline(state, ptr)
				updateLengthId()
			}
			if !luau.IsAnyIdentifier(ptr.value) {
				panic(shared.AssertionMessage(""))
			}

			t := state.GetType(element.Expression())
			addIterableToArrayBuilder := getAddIterableToArrayBuilder(state, element.Expression(), t)
			spreadExp := transformExpression(state, element.Expression())
			shouldUpdateLengthId := i < len(elements)-1
			state.PrereqList(
				addIterableToArrayBuilder(
					state,
					spreadExp,
					ptr.value.(luau.AnyIdentifier),
					lengthId,
					amtElementsSinceUpdate,
					shouldUpdateLengthId,
				),
			)
		} else {
			expression, prereqs := Capture(state, func() luau.Expression { return transformExpression(state, element) })
			if luau.IsArray(ptr.value) && !prereqs.IsEmpty() {
				disableArrayInline(state, ptr)
				updateLengthId()
			}
			if luau.IsArray(ptr.value) {
				ptr.value.(*luau.Array).Members.Push(expression)
			} else {
				state.PrereqList(prereqs)
				state.Prereq(
					luau.Create(&luau.Assignment{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: ptr.value.(luau.IndexableExpression),
							Index:      luau.NewBinary(lengthId, luau.BinaryAdd, luau.NewNumber(amtElementsSinceUpdate+1)),
						}),
						Operator: luau.AssignEqual,
						Right:    expression,
					}),
				)
			}
			amtElementsSinceUpdate++
		}
	}

	return ptr.value
}
