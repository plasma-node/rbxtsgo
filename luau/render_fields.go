package luau

// Ported from src/LuauRenderer/nodes/fields/renderMapField.ts and
// renderInterpolatedStringPart.ts.

func renderMapField(state *RenderState, node *MapField) {
	if s, ok := node.Index.(*StringLiteral); ok && IsValidIdentifier(s.Value) {
		state.writeString(s.Value)
		state.writeString(" = ")
		renderNode(state, node.Value)
		return
	}

	// Upstream renders the value before the index. That only shows in the
	// fallback names of temporary identifiers that SolveTempIDs did not see,
	// but keep the order: render the value, park it, write the index, then put
	// the value back.
	start := len(state.buf)
	renderNode(state, node.Value)
	scratchStart := len(state.scratch)
	state.scratch = append(state.scratch, state.buf[start:]...)
	state.buf = state.buf[:start]

	state.writeByte('[')
	renderNode(state, node.Index)
	state.writeString("] = ")
	state.buf = append(state.buf, state.scratch[scratchStart:]...)
	state.scratch = state.scratch[:scratchStart]
}

// renderInterpolatedStringPart escapes braces (but not inside \u{...} escape
// codes) and newlines. Same as the upstream replaces
//
//	.replace(/(\\u{[a-fA-F0-9]+})|([{}])/g, (_, unicodeEscape, brace) => unicodeEscape ?? "\\" + brace)
//	.replace(/(\r\n?|\n)/g, "\\$1")
//
// done in one pass: the second replace only touches CR and LF, which the
// first neither produces nor consumes.
func renderInterpolatedStringPart(state *RenderState, node *InterpolatedStringPart) {
	text := node.Text
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch c {
		case '{', '}':
			state.buf = append(state.buf, '\\', c)
		case '\r':
			state.buf = append(state.buf, '\\', '\r')
			if i+1 < len(text) && text[i+1] == '\n' {
				state.buf = append(state.buf, '\n')
				i++
			}
		case '\n':
			state.buf = append(state.buf, '\\', '\n')
		case '\\':
			// \u{HEX+} is copied through unchanged.
			if end := unicodeEscapeEnd(text, i); end > 0 {
				state.buf = append(state.buf, text[i:end]...)
				i = end - 1
			} else {
				state.buf = append(state.buf, c)
			}
		default:
			state.buf = append(state.buf, c)
		}
	}
}

// unicodeEscapeEnd returns the index after a `\u{HEX+}` starting at text[i], or 0.
func unicodeEscapeEnd(text string, i int) int {
	if i+3 >= len(text) || text[i+1] != 'u' || text[i+2] != '{' {
		return 0
	}
	j := i + 3
	for j < len(text) && isHexDigit(text[j]) {
		j++
	}
	if j == i+3 || j >= len(text) || text[j] != '}' {
		return 0
	}
	return j + 1
}
