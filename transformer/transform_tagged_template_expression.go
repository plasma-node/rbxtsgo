package transformer

// Port of src/TSTransformer/nodes/expressions/transformTaggedTemplateExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformTaggedTemplateExpression(state *TransformState, node *ast.Node) luau.Expression {
	taggedTemplate := node.AsTaggedTemplateExpression()
	tagExp := transformExpression(state, taggedTemplate.Tag)

	if ast.IsTemplateExpression(taggedTemplate.Template) {
		template := taggedTemplate.Template.AsTemplateExpression()
		templateSpans := template.TemplateSpans.Nodes

		strings := []luau.Expression{}
		strings = append(strings, luau.NewString(template.Head.Text()))
		for _, templateSpan := range templateSpans {
			strings = append(strings, luau.NewString(templateSpan.AsTemplateSpan().Literal.Text()))
		}

		spanExpressions := make([]*ast.Node, len(templateSpans))
		for i, templateSpan := range templateSpans {
			spanExpressions[i] = templateSpan.Expression()
		}
		expressions := ensureTransformOrder(state, spanExpressions, nil)

		return luau.NewCall(convertToIndexableExpression(tagExp), append([]luau.Expression{luau.NewArray(strings...)}, expressions...)...)
	} else {
		return luau.NewCall(convertToIndexableExpression(tagExp), luau.NewArray(luau.NewString(taggedTemplate.Template.Text())))
	}
}
