package luau

// Ported from src/LuauRenderer/util/getEnding.ts, getSafeBracketEquals.ts,
// needsParentheses.ts, renderArguments.ts, renderParameters.ts and
// renderStatements.ts.

// ---------------------------------------------------------------------------
// getEnding.ts

func endsWithIndexableExpressionInner(node Node) bool {
	switch n := node.(type) {
	case *BinaryExpression:
		// `a + b`
		return endsWithIndexableExpressionInner(n.Right)
	case *UnaryExpression:
		// `-a`
		return endsWithIndexableExpressionInner(n.Expression)
	case *IfExpression:
		// `if a then b else c`, `if a then b elseif c then d else e`
		return endsWithIndexableExpressionInner(n.Alternative)
	}
	// `a` or `(a)` or `a.b` or `a[b]` or `a()`
	return IsIndexableExpression(node)
}

// furthestRight picks the last expression of a declaration or assignment:
// `local a = b`, `a = b`, `local a`, `local a, b`.
func furthestRight(left, right NodeOrList) Node {
	if right != nil {
		if list, ok := right.(*List); ok {
			Assert(list.IsNonEmpty(), "")
			return list.Tail.Value
		}
		return right.(Node)
	}
	if list, ok := left.(*List); ok {
		Assert(list.IsNonEmpty(), "")
		return list.Tail.Value
	}
	return left.(Node)
}

func endsWithIndexableExpression(node Statement) bool {
	switch n := node.(type) {
	case *CallStatement:
		// `a()`
		return true
	case *VariableDeclaration:
		return endsWithIndexableExpressionInner(furthestRight(n.Left, n.Right))
	case *Assignment:
		return endsWithIndexableExpressionInner(furthestRight(n.Left, n.Right))
	}
	return false
}

func startsWithParenthesisInner(node Node) bool {
	switch n := node.(type) {
	case *ParenthesizedExpression:
		// `(a)`
		return true
	case *CallExpression:
		// `(a)()`
		return startsWithParenthesisInner(n.Expression)
	case *MethodCallExpression:
		// `(a):b()`
		return startsWithParenthesisInner(n.Expression)
	case *PropertyAccessExpression:
		// `(a).b`
		return startsWithParenthesisInner(n.Expression)
	case *ComputedIndexExpression:
		// `(a)[b]`
		return startsWithParenthesisInner(n.Expression)
	}
	return false
}

func startsWithParenthesis(node Node) bool {
	switch n := node.(type) {
	case *CallStatement:
		// `(a)()`
		return startsWithParenthesisInner(n.Expression.Callee())
	case *Assignment:
		if list, ok := n.Left.(*List); ok {
			// `(a).b, c = d`
			Assert(list.IsNonEmpty(), "")
			return startsWithParenthesisInner(list.Head.Value)
		}
		// `(a).b = c`
		return startsWithParenthesisInner(n.Left.(Node))
	}
	return false
}

func getNextNonComment(state *RenderState) Node {
	top := state.PeekListNode()
	if top == nil {
		return nil
	}
	listNode := top.Next
	for listNode != nil && IsComment(listNode.Value) {
		listNode = listNode.Next
	}
	if listNode == nil {
		return nil
	}
	return listNode.Value
}

// needsSemicolon resolves if the given statement needs to end with a `;`
// (upstream getEnding returns ";" or "").
//
// Used to avoid "ambiguous syntax" errors in Luau. This is only necessary in
// statements which can end in an IndexableExpression: CallStatement,
// VariableDeclaration and Assignment.
func needsSemicolon(state *RenderState, node Statement) bool {
	nextStatement := getNextNonComment(state)
	return nextStatement != nil && endsWithIndexableExpression(node) && startsWithParenthesis(nextStatement)
}

// ---------------------------------------------------------------------------
// getSafeBracketEquals.ts

// hasBracketRun reports whether s contains "]" + n*"=" + "]".
func hasBracketRun(s string, n int) bool {
	for i := 0; i+n+1 < len(s); i++ {
		if s[i] != ']' {
			continue
		}
		j := 1
		for j <= n && s[i+j] == '=' {
			j++
		}
		if j == n+1 && s[i+n+1] == ']' {
			return true
		}
	}
	return false
}

// endsWithBracketRun reports whether s ends with "]" + n*"=".
func endsWithBracketRun(s string, n int) bool {
	if len(s) < n+1 || s[len(s)-n-1] != ']' {
		return false
	}
	for i := len(s) - n; i < len(s); i++ {
		if s[i] != '=' {
			return false
		}
	}
	return true
}

// getSafeBracketEquals returns the number of "=" a long bracket needs so that
// str cannot close it early.
func getSafeBracketEquals(str string) int {
	amtEquals := 0
	for hasBracketRun(str, amtEquals) || endsWithBracketRun(str, amtEquals) {
		amtEquals++
	}
	return amtEquals
}

func (s *RenderState) writeEquals(n int) {
	for i := 0; i < n; i++ {
		s.buf = append(s.buf, '=')
	}
}

// ---------------------------------------------------------------------------
// needsParentheses.ts

// https://www.lua.org/manual/5.1/manual.html#2.5.6
//
//	1. or
//	2. and
//	3. <     >     <=    >=    ~=    ==
//	4. ..
//	5. +     -
//	6. *     /     %
//	7. not   #     - (unary)
//	8. ^

const ifExpressionPrecedence = 1

func unaryPrecedence(op UnaryOperator) int {
	switch op {
	case "not", "#", "-":
		return 7
	}
	return 0 // upstream: undefined
}

func binaryPrecedence(op BinaryOperator) int {
	switch op {
	case "or":
		return 1
	case "and":
		return 2
	case "<", ">", "<=", ">=", "~=", "==":
		return 3
	case "..":
		return 4
	case "+", "-":
		return 5
	case "*", "/", "//", "%":
		return 6
	case "^":
		return 8
	}
	return 0 // upstream: undefined
}

func getPrecedence(node Node) int {
	switch n := node.(type) {
	case *IfExpression:
		return ifExpressionPrecedence
	case *BinaryExpression:
		return binaryPrecedence(n.Operator)
	case *UnaryExpression:
		return unaryPrecedence(n.Operator)
	}
	Assert(false, "")
	return 0
}

// needsParentheses takes an IfExpression, UnaryExpression or BinaryExpression.
func needsParentheses(node Expression) bool {
	parent := node.Parent()
	if parent != nil && IsExpressionWithPrecedence(parent) {
		nodePrecedence := getPrecedence(node)
		parentPrecedence := getPrecedence(parent)
		if nodePrecedence < parentPrecedence {
			return true
		} else if nodePrecedence == parentPrecedence {
			if p, ok := parent.(*BinaryExpression); ok {
				return Node(node) == Node(p.Right)
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// renderArguments.ts, renderParameters.ts, renderStatements.ts

// renderArguments renders a list of expressions separated by ", ".
func renderArguments(state *RenderState, expressions *List) {
	for ln := expressions.Head; ln != nil; ln = ln.Next {
		if ln != expressions.Head {
			state.buf = append(state.buf, ", "...)
		}
		renderNode(state, ln.Value)
	}
}

// renderParameters renders the parameters of a function-like node separated
// by ", ", with "..." on the end if hasDotDotDot is set.
func renderParameters(state *RenderState, parameters *List, hasDotDotDot bool) {
	for ln := parameters.Head; ln != nil; ln = ln.Next {
		if ln != parameters.Head {
			state.buf = append(state.buf, ", "...)
		}
		renderNode(state, ln.Value)
	}
	if hasDotDotDot {
		if parameters.Head != nil {
			state.buf = append(state.buf, ", "...)
		}
		state.buf = append(state.buf, "..."...)
	}
}

// renderStatementsTo renders the given list of statements.
//
// Pushes each list node onto the state's list node stack as it gets rendered
// to give context to other statements as they render. Useful for getting the
// next or previous sibling statement.
func renderStatementsTo(state *RenderState, statements *List) {
	hasFinalStatement := false
	for listNode := statements.Head; listNode != nil; listNode = listNode.Next {
		Assert(!hasFinalStatement || IsComment(listNode.Value),
			"Cannot render statement after break, continue, or return!")
		hasFinalStatement = hasFinalStatement || IsFinalStatement(listNode.Value)

		state.PushListNode(listNode)
		renderNode(state, listNode.Value)
		state.PopListNode()
	}
}

// block renders statements one indent level deeper.
func renderBlock(state *RenderState, statements *List) {
	state.indent++
	renderStatementsTo(state, statements)
	state.indent--
}
