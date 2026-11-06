package transformer

// Port of src/TSTransformer/util/createStringFromLiteral.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/internal/stringutil"
)

const (
	templateEdge     = len("`")
	templateExpStart = len("${")
	templateExpEnd   = len("}")
)

// createStringFromLiteral returns the transformed text from a `ts.TemplateLiteralToken | ts.StringLiteral`.
// Cannot just use `node.text` because that converts `\\n` to be `\n`.
func createStringFromLiteral(node *ast.Node) string {
	text := scanner.GetSourceTextOfNodeFromSourceFile(ast.GetSourceFileOfNode(node), node, false)
	if ast.IsStringLiteral(node) || ast.IsNoSubstitutionTemplateLiteral(node) {
		text = stringutil.StripQuotes(text)
	} else if ast.IsTemplateHead(node) {
		// remove starting ` and ending ${
		text = trimLiteralEdges(text, templateEdge, templateExpStart)
	} else if ast.IsTemplateMiddle(node) {
		// remove starting } and ending ${
		text = trimLiteralEdges(text, templateExpEnd, templateExpStart)
	} else if ast.IsTemplateTail(node) {
		// remove starting } and ending `
		text = trimLiteralEdges(text, templateExpEnd, templateEdge)
	}
	return text
}

// trimLiteralEdges is text.slice(start, -end). The edges are ASCII, so bytes and UTF-16 units agree.
func trimLiteralEdges(text string, start int, end int) string {
	if start >= len(text)-end {
		return ""
	}
	return text[start : len(text)-end]
}
