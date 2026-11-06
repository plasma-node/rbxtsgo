package transformer

// Port of src/TSTransformer/nodes/transformInitializer.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformInitializer(state *TransformState, id luau.WritableExpression, initializer *ast.Node) *luau.IfStatement {
	return luau.Create(&luau.IfStatement{
		Condition: luau.NewBinary(id, luau.BinaryEqual, luau.NewNil()),
		ElseBody:  luau.NewList(),
		Statements: state.CapturePrereqs(func() {
			state.Prereq(luau.Create(&luau.Assignment{
				Left:     id,
				Operator: luau.AssignEqual,
				Right:    transformExpression(state, initializer),
			}))
		}),
	})
}
