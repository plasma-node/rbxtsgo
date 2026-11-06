package transformer

// Port of src/TSTransformer/nodes/transformInterpolatedStringPart.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformInterpolatedStringPart(node *ast.Node) *luau.InterpolatedStringPart {
	return luau.Create(&luau.InterpolatedStringPart{Text: createStringFromLiteral(node)})
}
