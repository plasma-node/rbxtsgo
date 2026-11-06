package transformer

// Port of src/TSTransformer/nodes/jsx/transformJsxTagName.ts.

import (
	"unicode"
	"unicode/utf8"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

// isFirstCharLowerCase is `firstChar === firstChar.toLowerCase()` with `firstChar = text[0]`. text[0] is one UTF-16
// unit. For a character outside the BMP, that's a lone high surrogate, which toLowerCase leaves alone, so such a name
// always counts as lower case, even when the whole character has a lower case form.
func isFirstCharLowerCase(text string) bool {
	firstChar, _ := utf8.DecodeRuneInString(text)
	if firstChar > 0xFFFF {
		return true
	}
	return unicode.ToLower(firstChar) == firstChar
}

func transformJsxTagNameExpression(state *TransformState, node *ast.Node) luau.Expression {
	// host component
	if ast.IsIdentifier(node) {
		if isFirstCharLowerCase(node.Text()) {
			return luau.NewString(node.Text())
		}
	}

	if ast.IsPropertyAccessExpression(node) {
		if ast.IsPrivateIdentifier(node.Name()) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoPrivateIdentifier.At(shared.AtNode(node.Name())))
		}
		return luau.NewProperty(convertToIndexableExpression(transformExpression(state, node.Expression())), node.Name().Text())
	} else if ast.IsJsxNamespacedName(node) {
		return luau.NewString(getTextOfJsxNamespacedName(node))
	} else {
		return transformExpression(state, node)
	}
}

func transformJsxTagName(state *TransformState, tagName *ast.Node) luau.Expression {
	expression, prereqs := Capture(state, func() luau.Expression { return transformJsxTagNameExpression(state, tagName) })
	tagNameExp := expression
	if !prereqs.IsEmpty() {
		state.PrereqList(prereqs)
		tagNameExp = state.PushToVarIfComplex(tagNameExp, "tagName")
	}
	return tagNameExp
}
