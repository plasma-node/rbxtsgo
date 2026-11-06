package transformer

// Port of src/TSTransformer/util/getConstantValueLiteral.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/jsnum"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// getConstantValueLiteral returns nil where JS returns undefined.
func getConstantValueLiteral(state *TransformState, node *ast.Node) luau.Expression {
	switch constantValue := state.Checker.GetConstantValue(node).(type) {
	case string:
		return luau.NewString(constantValue)
	case jsnum.Number:
		return luau.NewNumber(float64(constantValue))
	}
	return nil
}
