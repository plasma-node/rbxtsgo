package transformer

// Port of src/TSTransformer/nodes/expressions/transformStringLiteral.ts.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformStringLiteral(state *TransformState, node *ast.Node) luau.Expression {
	result := luau.NewString(createStringFromLiteral(state, node))
	// Text with both quote characters renders as [[...]], where its escapes mean nothing to Luau. It came from a string
	// with one of them as its delimiter, escaped inside, so that delimiter works. Upstream's master (#3033) keeps the
	// source's quote on every string. This keeps it only where the output was wrong.
	if state.Data.ProjectOptions.FixEmit && strings.IndexByte(result.Value, '"') >= 0 && strings.IndexByte(result.Value, '\'') >= 0 {
		result.Quote = scanner.GetSourceTextOfNodeFromSourceFile(ast.GetSourceFileOfNode(node), node, false)[0]
	}
	return result
}
