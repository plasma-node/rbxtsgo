package transformer

// Port of src/TSTransformer/nodes/expressions/transformNumericLiteral.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformNumericLiteral(state *TransformState, node *ast.Node) luau.Expression {
	return luau.Create(&luau.NumberLiteral{
		Value: scanner.GetSourceTextOfNodeFromSourceFile(ast.GetSourceFileOfNode(node), node, false),
	})
}
