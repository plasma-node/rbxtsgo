package transformer

// Port of src/TSTransformer/nodes/binding/transformArrayAssignmentPattern.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformArrayAssignmentPattern(state *TransformState, assignmentPattern *ast.Node, parentID luau.AnyIdentifier) {
	index := 0
	idStack := []luau.AnyIdentifier{}
	accessor := getAccessorForBindingType(
		state,
		assignmentPattern,
		state.Checker.GetTypeOfAssignmentPattern(assignmentPattern),
	)
	for _, element := range assignmentPattern.AsArrayLiteralExpression().Elements.Nodes {
		if ast.IsOmittedExpression(element) {
			accessor(state, parentID, index, &idStack, true)
		} else if ast.IsSpreadElement(element) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoSpreadDestructuring.At(shared.AtNode(element)))
		} else {
			var initializer *ast.Node
			// JS 3.0.0 crashes on [(a)] = b. From upstream's master, in transformAssignmentPatternElement.
			element = SkipDownwards(element)
			if ast.IsBinaryExpression(element) {
				binary := element.AsBinaryExpression()
				initializer = SkipDownwards(binary.Right)
				element = SkipDownwards(binary.Left)
			}

			value := accessor(state, parentID, index, &idStack, false)
			if ast.IsIdentifier(element) ||
				ast.IsElementAccessExpression(element) ||
				ast.IsPropertyAccessExpression(element) {
				id := transformWritableExpression(state, element, initializer != nil)
				state.Prereq(luau.Create(&luau.Assignment{
					Left:     id,
					Operator: luau.AssignEqual,
					Right:    value,
				}))
				if initializer != nil {
					state.Prereq(transformInitializer(state, id, initializer))
				}
			} else if ast.IsArrayLiteralExpression(element) {
				id := state.PushToVar(value, "binding")
				if initializer != nil {
					state.Prereq(transformInitializer(state, id, initializer))
				}
				transformArrayAssignmentPattern(state, element, id)
			} else if ast.IsObjectLiteralExpression(element) {
				id := state.PushToVar(value, "binding")
				if initializer != nil {
					state.Prereq(transformInitializer(state, id, initializer))
				}
				transformObjectAssignmentPattern(state, element, id)
			} else {
				panic(shared.AssertionMessage("transformArrayAssignmentPattern invalid element: " + getKindName(element.Kind)))
			}
		}
		index++
	}
}
