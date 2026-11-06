package luau

import "strings"

// Ported from src/LuauRenderer/nodes/statements/*.ts (renderAssignment,
// renderBreakStatement, renderCallStatement, renderComment,
// renderContinueStatement, renderDoStatement, renderForStatement,
// renderFunctionDeclaration, renderIfStatement, renderMethodDeclaration,
// renderNumericForStatement, renderRepeatStatement, renderReturnStatement,
// renderVariableDeclaration, renderWhileStatement).
//
// A line is writeIndent, the content, then writeLineEnd. Upstream builds the
// content first and indents it afterwards, but the indent never changes in
// between, so the bytes are the same.

// renderListOrNode renders a node, or the nodes of a list separated by ", ".
// A list must not be empty when requireNonEmpty is set.
func renderListOrNode(state *RenderState, v NodeOrList, requireNonEmpty bool) {
	if list, ok := v.(*List); ok {
		if requireNonEmpty {
			Assert(!list.IsEmpty(), "")
		}
		for ln := list.Head; ln != nil; ln = ln.Next {
			if ln != list.Head {
				state.writeString(", ")
			}
			renderNode(state, ln.Value)
		}
		return
	}
	renderNode(state, v.(Node))
}

func renderAssignment(state *RenderState, node *Assignment) {
	state.writeIndent()
	renderListOrNode(state, node.Left, true)
	state.writeByte(' ')
	state.writeString(string(node.Operator))
	state.writeByte(' ')
	renderListOrNode(state, node.Right, true)
	state.writeLineEnd(node)
}

func renderBreakStatement(state *RenderState) {
	state.writeLine("break")
}

func renderCallStatement(state *RenderState, node *CallStatement) {
	state.writeIndent()
	renderNode(state, node.Expression)
	state.writeLineEnd(node)
}

func renderComment(state *RenderState, node *Comment) {
	text := node.Text
	if strings.IndexByte(text, '\n') >= 0 {
		eq := getSafeBracketEquals(text)
		state.writeIndent()
		state.writeString("--[")
		state.writeEquals(eq)
		state.writeString("[\n")

		state.indent++
		for {
			i := strings.IndexByte(text, '\n')
			state.writeIndent()
			if i < 0 {
				state.writeString(text)
				state.writeByte('\n')
				break
			}
			state.writeString(text[:i])
			state.writeByte('\n')
			text = text[i+1:]
		}
		state.indent--

		state.writeIndent()
		state.writeByte(']')
		state.writeEquals(eq)
		state.writeString("]\n")
	} else {
		state.writeIndent()
		state.writeString("--")
		state.writeString(text)
		state.writeByte('\n')
	}
}

func renderContinueStatement(state *RenderState) {
	state.writeLine("continue")
}

func renderDoStatement(state *RenderState, node *DoStatement) {
	state.writeLine("do")
	renderBlock(state, node.Statements)
	state.writeLine("end")
}

func renderForStatement(state *RenderState, node *ForStatement) {
	state.writeIndent()
	state.writeString("for ")
	if node.IDs.IsEmpty() {
		state.writeByte('_')
	} else {
		for ln := node.IDs.Head; ln != nil; ln = ln.Next {
			if ln != node.IDs.Head {
				state.writeString(", ")
			}
			renderNode(state, ln.Value)
		}
	}
	state.writeString(" in ")
	renderNode(state, node.Expression)
	state.writeString(" do\n")
	renderBlock(state, node.Statements)
	state.writeLine("end")
}

func renderFunctionDeclaration(state *RenderState, node *FunctionDeclaration) {
	if node.Localize {
		Assert(IsAnyIdentifier(node.Name), "local function cannot be a property")
	}
	state.writeIndent()
	if node.Localize {
		state.writeString("local ")
	}
	state.writeString("function ")
	renderNode(state, node.Name)
	state.writeByte('(')
	renderParameters(state, node.Parameters, node.HasDotDotDot)
	state.writeString(")\n")
	renderBlock(state, node.Statements)
	state.writeLine("end")
}

func renderIfStatement(state *RenderState, node *IfStatement) {
	state.writeIndent()
	state.writeString("if ")
	renderNode(state, node.Condition)
	state.writeString(" then\n")
	renderBlock(state, node.Statements)

	currentElseBody := node.ElseBody
	for {
		elseIf, ok := currentElseBody.(*IfStatement)
		if !ok {
			break
		}
		state.writeIndent()
		state.writeString("elseif ")
		renderNode(state, elseIf.Condition)
		state.writeString(" then\n")
		renderBlock(state, elseIf.Statements)
		currentElseBody = elseIf.ElseBody
	}

	if list, ok := currentElseBody.(*List); ok && list.IsNonEmpty() {
		state.writeLine("else")
		renderBlock(state, list)
	}

	state.writeLine("end")
}

func renderMethodDeclaration(state *RenderState, node *MethodDeclaration) {
	state.writeIndent()
	state.writeString("function ")
	renderNode(state, node.Expression)
	state.writeByte(':')
	state.writeString(node.Name)
	state.writeByte('(')
	renderParameters(state, node.Parameters, node.HasDotDotDot)
	state.writeString(")\n")
	renderBlock(state, node.Statements)
	state.writeLine("end")
}

func renderNumericForStatement(state *RenderState, node *NumericForStatement) {
	state.writeIndent()
	state.writeString("for ")
	renderNode(state, node.ID)
	state.writeString(" = ")
	renderNode(state, node.Start)
	state.writeString(", ")
	renderNode(state, node.End)

	// step of 1 can be omitted
	if node.Step != nil {
		if lit, ok := node.Step.(*NumberLiteral); !ok || jsToNumber(lit.Value) != 1 {
			state.writeString(", ")
			renderNode(state, node.Step)
		}
	}

	state.writeString(" do\n")
	renderBlock(state, node.Statements)
	state.writeLine("end")
}

func renderRepeatStatement(state *RenderState, node *RepeatStatement) {
	state.writeLine("repeat")
	renderBlock(state, node.Statements)
	state.writeIndent()
	state.writeString("until ")
	renderNode(state, node.Condition)
	state.writeByte('\n')
}

func renderReturnStatement(state *RenderState, node *ReturnStatement) {
	state.writeIndent()
	state.writeString("return ")
	renderListOrNode(state, node.Expression, false)
	state.writeByte('\n')
}

func renderVariableDeclaration(state *RenderState, node *VariableDeclaration) {
	state.writeIndent()
	state.writeString("local ")
	renderListOrNode(state, node.Left, true)
	if node.Right != nil {
		state.writeString(" = ")
		renderListOrNode(state, node.Right, true)
	}
	state.writeLineEnd(node)
}

func renderWhileStatement(state *RenderState, node *WhileStatement) {
	state.writeIndent()
	state.writeString("while ")
	renderNode(state, node.Condition)
	state.writeString(" do\n")
	renderBlock(state, node.Statements)
	state.writeLine("end")
}
