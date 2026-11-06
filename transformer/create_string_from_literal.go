package transformer

// Port of src/TSTransformer/util/createStringFromLiteral.ts, with the escape handling of upstream's master (#3033,
// issue #2910) split in two. See docs/known-differences.md.
//
// Always on, because JS 3.0.0's Luau for these fails to parse, so no working output can change:
//   - \uXXXX, which Luau rejects because it needs \u{XXXX}, including surrogate pairs, which become one code point
//   - a backslash and a line break in a template, which the renderer turns into a raw line break in a backtick string
//   - \{ in a template, which the renderer turns into \\{ and so starts an interpolation that never ends
//
// A literal with any of those gets all of the rewrite. It couldn't run before, and half a fix would turn its parse error
// into a wrong value.
//
// Only with fixEmit, because the Luau parses and has the wrong value:
//   - a backslash and a line break in a plain string, which keeps the backslash or the line break
//   - a backslash and U+2028 or U+2029
//   - identity escapes that mean something else in Luau, like \a and \z
//   - line breaks in the source of a template, normalized to LF as JS does

import (
	"strconv"
	"strings"
	"unicode/utf8"

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
func createStringFromLiteral(state *TransformState, node *ast.Node) string {
	text := literalText(node)
	if state.Data.ProjectOptions.FixEmit || literalFailsToParse(node) {
		return rewriteStringEscapes(text, ast.IsStringLiteral(node))
	}
	return text
}

func literalText(node *ast.Node) string {
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

// literalFailsToParse says whether JS 3.0.0's Luau for this literal fails to parse. A template's tokens render as one
// string, so if one of them can't, the others have to be fixed along with it.
func literalFailsToParse(node *ast.Node) bool {
	if ast.IsStringLiteral(node) {
		text := literalText(node)
		return !rendersAsLongString(text) && textFailsToParse(text, true)
	}
	if ast.IsNoSubstitutionTemplateLiteral(node) {
		return textFailsToParse(literalText(node), false)
	}
	expression := node.Parent
	if ast.IsTemplateSpan(expression) {
		expression = expression.Parent
	}
	template := expression.AsTemplateExpression()
	if textFailsToParse(literalText(template.Head), false) {
		return true
	}
	for _, span := range template.TemplateSpans.Nodes {
		if textFailsToParse(literalText(span.AsTemplateSpan().Literal), false) {
			return true
		}
	}
	return false
}

// textFailsToParse goes through whole escapes, so that the second backslash of \\u0041 doesn't start one.
func textFailsToParse(text string, isStringLiteral bool) bool {
	for i := 0; i+1 < len(text); i++ {
		if text[i] != '\\' {
			continue
		}
		i++
		switch text[i] {
		case 'u':
			if isHex4At(text, i+1) {
				return true
			}
		case '\n', '\r', '{':
			if !isStringLiteral {
				return true
			}
		}
	}
	return false
}

// trimLiteralEdges is text.slice(start, -end). The edges are ASCII, so bytes and UTF-16 units agree.
func trimLiteralEdges(text string, start int, end int) string {
	if start >= len(text)-end {
		return ""
	}
	return text[start : len(text)-end]
}

// rendersAsLongString says whether renderStringLiteral uses [[...]] for this text, where escapes mean nothing to Luau,
// and so none of them can fail to parse.
func rendersAsLongString(text string) bool {
	return strings.IndexByte(text, '\n') >= 0 || (strings.IndexByte(text, '"') >= 0 && strings.IndexByte(text, '\'') >= 0)
}

// isHex4At says whether four hex digits start at text[i].
func isHex4At(text string, i int) bool {
	if i+4 > len(text) {
		return false
	}
	for _, c := range []byte(text[i : i+4]) {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// surrogatePairAt returns the code point of \uD8XX\uDCXX, whose high half's digits start at text[i], or 0.
func surrogatePairAt(text string, i int) rune {
	if !isHex4At(text, i) || i+10 > len(text) || text[i+4] != '\\' || text[i+5] != 'u' || !isHex4At(text, i+6) {
		return 0
	}
	high, _ := strconv.ParseUint(text[i:i+4], 16, 32)
	low, _ := strconv.ParseUint(text[i+6:i+10], 16, 32)
	if high < 0xD800 || high > 0xDBFF || low < 0xDC00 || low > 0xDFFF {
		return 0
	}
	return rune((high-0xD800)<<10 + (low - 0xDC00) + 0x10000)
}

// luauSharedEscapes are the characters after a backslash that mean the same to TS and Luau. Anything else after a
// backslash is the character itself in TS. Luau has \a for a bell and \z to skip whitespace.
const luauSharedEscapes = "0bfnrtvux\\\"'"

// rewriteStringEscapes is upstream's createStringFromLiteral, as one pass over whole escapes, so that the second backslash
// of \\u0041 doesn't start one.
func rewriteStringEscapes(text string, isStringLiteral bool) string {
	// as TS normalizes physical line endings in templates. A plain string has them only after a backslash, and normalizing
	// first means every kind of line break is one case below.
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	var result strings.Builder
	result.Grow(len(text))
	for i := 0; i < len(text); {
		c := text[i]
		if c != '\\' || i+1 >= len(text) {
			result.WriteByte(c)
			i++
			continue
		}
		next, size := utf8.DecodeRuneInString(text[i+1:])
		escape := text[i : i+1+size]
		end := i + 1 + size
		switch {
		case next == 'u' && isHex4At(text, end):
			if codePoint := surrogatePairAt(text, end); codePoint != 0 {
				result.WriteString(`\u{` + strconv.FormatInt(int64(codePoint), 16) + `}`)
				end += 10
			} else {
				result.WriteString(`\u{` + text[end:end+4] + `}`)
				end += 4
			}
		case next == '\\' && !isStringLiteral && strings.HasPrefix(text[end:], "u{"):
			// the renderer would read the second backslash of \\u{41} as the start of \u{41}, which would make {41} an
			// interpolation.
			result.WriteString(`\x5C`)
		case next == '\n' || next == 0x2028 || next == 0x2029:
			// adds no characters in TS
		case strings.ContainsRune(luauSharedEscapes, next) || (next == '`' && !isStringLiteral):
			result.WriteString(escape)
		default:
			result.WriteRune(next)
		}
		i = end
	}
	return result.String()
}
