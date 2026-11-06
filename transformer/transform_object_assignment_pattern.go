package transformer

// Port of src/TSTransformer/nodes/binding/transformObjectAssignmentPattern.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformObjectAssignmentPattern(state *TransformState, assignmentPattern *ast.Node, parentID luau.AnyIdentifier) {
	for _, property := range assignmentPattern.AsObjectLiteralExpression().Properties.Nodes {
		if ast.IsShorthandPropertyAssignment(property) {
			name := property.Name()
			objectAssignmentInitializer := property.AsShorthandPropertyAssignment().ObjectAssignmentInitializer
			value := objectAccessor(
				state,
				parentID,
				state.Checker.GetTypeOfAssignmentPattern(assignmentPattern),
				name,
			)
			id := transformWritableExpression(state, name, objectAssignmentInitializer != nil)
			state.Prereq(luau.Create(&luau.Assignment{
				Left:     id,
				Operator: luau.AssignEqual,
				Right:    value,
			}))
			if !luau.IsAnyIdentifier(id) {
				panic(shared.AssertionMessage(""))
			}
			if objectAssignmentInitializer != nil {
				state.Prereq(transformInitializer(state, id, objectAssignmentInitializer))
			}
		} else if ast.IsSpreadAssignment(property) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoSpreadDestructuring.At(shared.AtNode(property)))
			return
		} else if ast.IsPropertyAssignment(property) {
			name := property.Name()
			propertyInitializer := property.AsPropertyAssignment().Initializer
			init := propertyInitializer
			var initializer *ast.Node
			if ast.IsBinaryExpression(propertyInitializer) {
				binary := propertyInitializer.AsBinaryExpression()
				initializer = SkipDownwards(binary.Right)
				init = SkipDownwards(binary.Left)
			}

			value := objectAccessor(
				state,
				parentID,
				state.Checker.GetTypeOfAssignmentPattern(assignmentPattern),
				name,
			)
			if ast.IsIdentifier(init) || ast.IsElementAccessExpression(init) || ast.IsPropertyAccessExpression(init) {
				id := transformWritableExpression(state, init, initializer != nil)
				state.Prereq(luau.Create(&luau.Assignment{
					Left:     id,
					Operator: luau.AssignEqual,
					Right:    value,
				}))
				if initializer != nil {
					state.Prereq(transformInitializer(state, id, initializer))
				}
			} else if ast.IsArrayLiteralExpression(init) {
				id := state.PushToVar(value, "binding")
				if initializer != nil {
					state.Prereq(transformInitializer(state, id, initializer))
				}
				if !ast.IsIdentifier(name) {
					panic(shared.AssertionMessage(""))
				}
				transformArrayAssignmentPattern(state, init, id)
			} else if ast.IsObjectLiteralExpression(init) {
				id := state.PushToVar(value, "binding")
				if initializer != nil {
					state.Prereq(transformInitializer(state, id, initializer))
				}
				transformObjectAssignmentPattern(state, init, id)
			} else {
				panic(shared.AssertionMessage("transformObjectAssignmentPattern invalid initializer: " + getKindName(init.Kind)))
			}
		} else {
			panic(shared.AssertionMessage("transformObjectAssignmentPattern invalid property: " + getKindName(property.Kind)))
		}
	}
}
