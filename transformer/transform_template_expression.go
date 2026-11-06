package transformer

// Port of src/TSTransformer/nodes/expressions/transformTemplateExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformTemplateExpression(state *TransformState, node *ast.Node) luau.Expression {
	template := node.AsTemplateExpression()
	templateSpans := template.TemplateSpans.Nodes
	parts := luau.NewList()

	if template.Head.Text() != "" {
		parts.Push(transformInterpolatedStringPart(state, template.Head))
	}

	spanExpressions := make([]*ast.Node, len(templateSpans))
	for i, templateSpan := range templateSpans {
		spanExpressions[i] = templateSpan.Expression()
	}
	orderedExpressions := ensureTransformOrder(state, spanExpressions, nil)

	for i := range templateSpans {
		parts.Push(orderedExpressions[i])

		templateSpan := templateSpans[i].AsTemplateSpan()
		if templateSpan.Literal.Text() != "" {
			parts.Push(transformInterpolatedStringPart(state, templateSpan.Literal))
		}
	}

	return luau.Create(&luau.InterpolatedString{Parts: parts})
}
