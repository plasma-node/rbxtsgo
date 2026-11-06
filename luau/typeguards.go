package luau

// Ported from src/LuauAST/impl/typeGuards.ts.
//
// Guards take a Node and test its Kind, like upstream. Go code can usually
// use a type assertion or switch instead; these exist so call sites can be
// ported one to one. A nil node is never any kind.

func kindOf(node Node) SyntaxKind {
	if node == nil {
		return -1
	}
	return node.Kind()
}

// indexable expressions
func IsAnyIdentifier(node Node) bool {
	k := kindOf(node)
	return k == KindIdentifier || k == KindTemporaryIdentifier
}
func IsIdentifier(node Node) bool               { return kindOf(node) == KindIdentifier }
func IsTemporaryIdentifier(node Node) bool      { return kindOf(node) == KindTemporaryIdentifier }
func IsComputedIndexExpression(node Node) bool  { return kindOf(node) == KindComputedIndexExpression }
func IsPropertyAccessExpression(node Node) bool { return kindOf(node) == KindPropertyAccessExpression }
func IsCallExpression(node Node) bool           { return kindOf(node) == KindCallExpression }
func IsMethodCallExpression(node Node) bool     { return kindOf(node) == KindMethodCallExpression }
func IsParenthesizedExpression(node Node) bool  { return kindOf(node) == KindParenthesizedExpression }

func IsIndexableExpression(node Node) bool {
	k := kindOf(node)
	return k >= KindFirstIndexableExpression && k <= KindLastIndexableExpression
}

// expressions
func IsNone(node Node) bool               { return kindOf(node) == KindNone }
func IsNilLiteral(node Node) bool         { return kindOf(node) == KindNilLiteral }
func IsFalseLiteral(node Node) bool       { return kindOf(node) == KindFalseLiteral }
func IsTrueLiteral(node Node) bool        { return kindOf(node) == KindTrueLiteral }
func IsNumberLiteral(node Node) bool      { return kindOf(node) == KindNumberLiteral }
func IsStringLiteral(node Node) bool      { return kindOf(node) == KindStringLiteral }
func IsVarArgsLiteral(node Node) bool     { return kindOf(node) == KindVarArgsLiteral }
func IsFunctionExpression(node Node) bool { return kindOf(node) == KindFunctionExpression }
func IsBinaryExpression(node Node) bool   { return kindOf(node) == KindBinaryExpression }
func IsUnaryExpression(node Node) bool    { return kindOf(node) == KindUnaryExpression }
func IsIfExpression(node Node) bool       { return kindOf(node) == KindIfExpression }
func IsInterpolatedString(node Node) bool { return kindOf(node) == KindInterpolatedString }
func IsArray(node Node) bool              { return kindOf(node) == KindArray }
func IsMap(node Node) bool                { return kindOf(node) == KindMap }
func IsSet(node Node) bool                { return kindOf(node) == KindSet }
func IsMixedTable(node Node) bool         { return kindOf(node) == KindMixedTable }

func IsExpression(node Node) bool {
	k := kindOf(node)
	return k >= KindFirstExpression && k <= KindLastExpression
}

// statements
func IsAssignment(node Node) bool          { return kindOf(node) == KindAssignment }
func IsBreakStatement(node Node) bool      { return kindOf(node) == KindBreakStatement }
func IsCallStatement(node Node) bool       { return kindOf(node) == KindCallStatement }
func IsContinueStatement(node Node) bool   { return kindOf(node) == KindContinueStatement }
func IsDoStatement(node Node) bool         { return kindOf(node) == KindDoStatement }
func IsWhileStatement(node Node) bool      { return kindOf(node) == KindWhileStatement }
func IsRepeatStatement(node Node) bool     { return kindOf(node) == KindRepeatStatement }
func IsIfStatement(node Node) bool         { return kindOf(node) == KindIfStatement }
func IsNumericForStatement(node Node) bool { return kindOf(node) == KindNumericForStatement }
func IsForStatement(node Node) bool        { return kindOf(node) == KindForStatement }
func IsFunctionDeclaration(node Node) bool { return kindOf(node) == KindFunctionDeclaration }
func IsMethodDeclaration(node Node) bool   { return kindOf(node) == KindMethodDeclaration }
func IsVariableDeclaration(node Node) bool { return kindOf(node) == KindVariableDeclaration }
func IsReturnStatement(node Node) bool     { return kindOf(node) == KindReturnStatement }
func IsComment(node Node) bool             { return kindOf(node) == KindComment }

func IsStatement(node Node) bool {
	k := kindOf(node)
	return k >= KindFirstStatement && k <= KindLastStatement
}

// fields
func IsMapField(node Node) bool               { return kindOf(node) == KindMapField }
func IsInterpolatedStringPart(node Node) bool { return kindOf(node) == KindInterpolatedStringPart }

func IsField(node Node) bool {
	k := kindOf(node)
	return k >= KindFirstField && k <= KindLastField
}

// IsNode reports whether v is a node (upstream checks for a numeric kind in
// range; here that is any non-nil Node).
func IsNode(v NodeOrList) bool {
	n, ok := v.(Node)
	return ok && n != nil
}

// IsSimple is Identifier, TemporaryIdentifier, nil, true, false, number or string literals.
func IsSimple(node Node) bool {
	switch kindOf(node) {
	case KindIdentifier, KindTemporaryIdentifier, KindNilLiteral, KindTrueLiteral, KindFalseLiteral,
		KindNumberLiteral, KindStringLiteral:
		return true
	}
	return false
}

// IsSimplePrimitive is nil, true, false, number or string literals.
func IsSimplePrimitive(node Node) bool {
	switch kindOf(node) {
	case KindNilLiteral, KindTrueLiteral, KindFalseLiteral, KindNumberLiteral, KindStringLiteral:
		return true
	}
	return false
}

// IsTable is Array, Set, Map or MixedTable.
func IsTable(node Node) bool {
	switch kindOf(node) {
	case KindArray, KindSet, KindMap, KindMixedTable:
		return true
	}
	return false
}

// IsFinalStatement is break, return or continue.
func IsFinalStatement(node Node) bool {
	switch kindOf(node) {
	case KindBreakStatement, KindReturnStatement, KindContinueStatement:
		return true
	}
	return false
}

// IsCall is CallExpression or MethodCallExpression.
func IsCall(node Node) bool {
	k := kindOf(node)
	return k == KindCallExpression || k == KindMethodCallExpression
}

// IsWritableExpression is Identifier, TemporaryIdentifier, PropertyAccessExpression or ComputedIndexExpression.
func IsWritableExpression(node Node) bool {
	switch kindOf(node) {
	case KindIdentifier, KindTemporaryIdentifier, KindPropertyAccessExpression, KindComputedIndexExpression:
		return true
	}
	return false
}

// IsFunctionLike is FunctionDeclaration, FunctionExpression or MethodDeclaration.
func IsFunctionLike(node Node) bool {
	switch kindOf(node) {
	case KindFunctionDeclaration, KindFunctionExpression, KindMethodDeclaration:
		return true
	}
	return false
}

// HasStatementsKind reports whether node is a kind that owns a statement
// list (upstream hasStatements).
func HasStatementsKind(node Node) bool {
	switch kindOf(node) {
	case KindForStatement, KindNumericForStatement, KindFunctionExpression, KindDoStatement,
		KindFunctionDeclaration, KindIfStatement, KindMethodDeclaration, KindRepeatStatement,
		KindWhileStatement:
		return true
	}
	return false
}

// IsExpressionWithPrecedence is IfExpression, UnaryExpression or BinaryExpression.
func IsExpressionWithPrecedence(node Node) bool {
	switch kindOf(node) {
	case KindIfExpression, KindUnaryExpression, KindBinaryExpression:
		return true
	}
	return false
}
