package transformer

// Port of src/TSTransformer/nodes/expressions/transformNoSubstitutionTemplateLiteral.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// backtick string literals without interpolation expressions should be preserved
// as they still are valid in luau
func transformNoSubstitutionTemplateLiteral(state *TransformState, node *ast.Node) luau.Expression {
	return luau.Create(&luau.InterpolatedString{
		Parts: luau.MakeList(transformInterpolatedStringPart(node)),
	})
}
