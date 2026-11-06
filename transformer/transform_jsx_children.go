package transformer

// Port of src/TSTransformer/nodes/jsx/transformJsxChildren.ts.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformJsxChildren(state *TransformState, children []*ast.Node) []luau.Expression {
	lastJsxChildIndex := -1
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		if !ast.IsJsxText(child) || !child.AsJsxText().ContainsOnlyTriviaWhiteSpaces {
			lastJsxChildIndex = i
			break
		}
	}

	for i := 0; i < lastJsxChildIndex; i++ {
		child := children[i]
		if ast.IsJsxExpression(child) && child.AsJsxExpression().DotDotDotToken != nil {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoPrecedingJsxSpreadElement.At(shared.AtNode(child)))
		}
	}

	var filtered []*ast.Node
	for _, v := range children {
		// ignore jsx text that only contains whitespace
		if ast.IsJsxText(v) && v.AsJsxText().ContainsOnlyTriviaWhiteSpaces {
			continue
		}
		// ignore empty jsx expressions, i.e. `{}`
		if ast.IsJsxExpression(v) && v.AsJsxExpression().Expression == nil {
			continue
		}
		filtered = append(filtered, v)
	}

	return ensureTransformOrder(state, filtered, func(state *TransformState, node *ast.Node) luau.Expression {
		if ast.IsJsxText(node) {
			text := fixupWhitespaceAndDecodeEntities(node.AsJsxText().Text)
			return luau.NewString(strings.ReplaceAll(text, `\`, `\\`))
		}
		return transformExpression(state, node)
	})
}
