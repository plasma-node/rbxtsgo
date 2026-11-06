package transformer

// Port of src/TSTransformer/util/createHoistDeclaration.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// createHoistDeclaration returns nil where JS returns undefined.
func createHoistDeclaration(state *TransformState, statement *ast.Node) *luau.VariableDeclaration {
	hoists := state.HoistsByStatement[statement]
	if len(hoists) > 0 {
		for _, hoist := range hoists {
			validateIdentifier(state, hoist)
		}
		ids := make([]luau.Node, len(hoists))
		for i, hoistID := range hoists {
			ids[i] = transformIdentifierDefined(state, hoistID)
		}
		return luau.Create(&luau.VariableDeclaration{
			Left:  luau.MakeList(ids...),
			Right: nil,
		})
	}
	return nil
}
