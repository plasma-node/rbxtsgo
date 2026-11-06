package luau

import "strings"

// Ported from src/LuauRenderer/nodes/expressions/*.ts (renderArray,
// renderBinaryExpression, renderFunctionExpression, renderIfExpression,
// renderInterpolatedString, renderMap, renderMixedTable, renderNumberLiteral,
// renderSet, renderStringLiteral, renderUnaryExpression).
//
// Upstream evaluates render() calls in the same order they appear in the
// output everywhere in this file, so streaming keeps temp-id fallback naming
// identical.

func renderArray(state *RenderState, node *Array) {
	if node.Members.IsEmpty() {
		state.writeString("{}")
		return
	}
	state.writeString("{ ")
	for ln := node.Members.Head; ln != nil; ln = ln.Next {
		if ln != node.Members.Head {
			state.writeString(", ")
		}
		renderNode(state, ln.Value)
	}
	state.writeString(" }")
}

func renderBinaryExpression(state *RenderState, node *BinaryExpression) {
	parens := needsParentheses(node)
	if parens {
		state.writeByte('(')
	}
	renderNode(state, node.Left)
	state.writeByte(' ')
	state.writeString(string(node.Operator))
	state.writeByte(' ')
	renderNode(state, node.Right)
	if parens {
		state.writeByte(')')
	}
}

func renderFunctionExpression(state *RenderState, node *FunctionExpression) {
	if node.Statements.IsEmpty() {
		state.writeString("function(")
		renderParameters(state, node.Parameters, node.HasDotDotDot)
		state.writeString(") end")
		return
	}

	state.writeString("function(")
	renderParameters(state, node.Parameters, node.HasDotDotDot)
	state.writeString(")\n")
	renderBlock(state, node.Statements)
	state.writeIndent()
	state.writeString("end")
}

func renderIfExpression(state *RenderState, node *IfExpression) {
	parens := needsParentheses(node)
	if parens {
		state.writeByte('(')
	}

	state.writeString("if ")
	renderNode(state, node.Condition)
	state.writeString(" then ")
	renderNode(state, node.Expression)
	state.writeByte(' ')

	currentAlternative := node.Alternative
	for {
		alt, ok := currentAlternative.(*IfExpression)
		if !ok {
			break
		}
		state.writeString("elseif ")
		renderNode(state, alt.Condition)
		state.writeString(" then ")
		renderNode(state, alt.Expression)
		state.writeByte(' ')
		currentAlternative = alt.Alternative
	}

	state.writeString("else ")
	renderNode(state, currentAlternative)

	if parens {
		state.writeByte(')')
	}
}

func renderInterpolatedString(state *RenderState, node *InterpolatedString) {
	state.writeByte('`')
	for ln := node.Parts.Head; ln != nil; ln = ln.Next {
		part := ln.Value
		if _, ok := part.(*InterpolatedStringPart); ok {
			renderNode(state, part)
		} else {
			state.writeByte('{')
			// `{{}}` is invalid, so we wrap it in parenthesis
			if IsTable(part) {
				state.writeByte('(')
				renderNode(state, part)
				state.writeByte(')')
			} else {
				renderNode(state, part)
			}
			state.writeByte('}')
		}
	}
	state.writeByte('`')
}

func renderMap(state *RenderState, node *Map) {
	if node.Fields.IsEmpty() {
		state.writeString("{}")
		return
	}

	state.writeString("{\n")
	state.indent++
	for ln := node.Fields.Head; ln != nil; ln = ln.Next {
		state.writeIndent()
		renderNode(state, ln.Value)
		state.writeString(",\n")
	}
	state.indent--
	state.writeIndent()
	state.writeByte('}')
}

func renderMixedTable(state *RenderState, node *MixedTable) {
	if node.Fields.IsEmpty() {
		state.writeString("{}")
		return
	}

	state.writeString("{\n")
	state.indent++
	for ln := node.Fields.Head; ln != nil; ln = ln.Next {
		state.writeIndent()
		renderNode(state, ln.Value)
		state.writeString(",\n")
	}
	state.indent--
	state.writeIndent()
	state.writeByte('}')
}

func renderNumberLiteral(state *RenderState, node *NumberLiteral) {
	if IsValidNumberLiteral(node.Value) {
		state.writeString(node.Value)
	} else {
		state.writeString(jsNumberToString(jsToNumber(strings.ReplaceAll(node.Value, "_", ""))))
	}
}

func renderSet(state *RenderState, node *Set) {
	if node.Members.IsEmpty() {
		state.writeString("{}")
		return
	}

	state.writeString("{\n")
	state.indent++
	for ln := node.Members.Head; ln != nil; ln = ln.Next {
		state.writeIndent()
		state.writeByte('[')
		renderNode(state, ln.Value)
		state.writeString("] = true,\n")
	}
	state.indent--
	state.writeIndent()
	state.writeByte('}')
}

func needsBracketSpacing(node *StringLiteral) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}

	switch p := parent.(type) {
	case *MapField:
		if Node(node) == Node(p.Index) {
			return true
		}
	case *ComputedIndexExpression:
		if Node(node) == Node(p.Index) {
			return true
		}
	case *Set:
		return true
	}
	return false
}

func renderStringLiteral(state *RenderState, node *StringLiteral) {
	value := node.Value
	isMultiline := strings.IndexByte(value, '\n') >= 0
	if !isMultiline && strings.IndexByte(value, '"') < 0 {
		state.writeByte('"')
		state.writeString(value)
		state.writeByte('"')
	} else if !isMultiline && strings.IndexByte(value, '\'') < 0 {
		state.writeByte('\'')
		state.writeString(value)
		state.writeByte('\'')
	} else {
		eq := getSafeBracketEquals(value)
		spacing := needsBracketSpacing(node)
		if spacing {
			state.writeByte(' ')
		}
		state.writeByte('[')
		state.writeEquals(eq)
		state.writeByte('[')
		state.writeString(value)
		state.writeByte(']')
		state.writeEquals(eq)
		state.writeByte(']')
		if spacing {
			state.writeByte(' ')
		}
	}
}

func needsSpace(node *UnaryExpression) bool {
	// not always needs a space
	if node.Operator == UnaryNot {
		return true
	}

	// "--" will create a comment!
	if inner, ok := node.Expression.(*UnaryExpression); ok && inner.Operator == UnaryMinus {
		// previous expression was also "-"
		return true
	}

	return false
}

func renderUnaryExpression(state *RenderState, node *UnaryExpression) {
	parens := needsParentheses(node)
	if parens {
		state.writeByte('(')
	}
	state.writeString(string(node.Operator))
	if needsSpace(node) {
		state.writeByte(' ')
	}
	renderNode(state, node.Expression)
	if parens {
		state.writeByte(')')
	}
}
