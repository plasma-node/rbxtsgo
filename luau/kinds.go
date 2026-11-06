package luau

// Ported from src/LuauAST/impl/enums.ts and src/LuauAST/util/getKindName.ts.

// SyntaxKind identifies a node type. The numeric values and their ordering
// are the same as upstream because category checks use ranges.
type SyntaxKind int

const (
	// indexable expressions
	KindIdentifier SyntaxKind = iota
	KindTemporaryIdentifier
	KindComputedIndexExpression
	KindPropertyAccessExpression
	KindCallExpression
	KindMethodCallExpression
	KindParenthesizedExpression

	// expressions
	KindNone
	KindNilLiteral
	KindFalseLiteral
	KindTrueLiteral
	KindNumberLiteral
	KindStringLiteral
	KindVarArgsLiteral
	KindFunctionExpression
	KindBinaryExpression
	KindUnaryExpression
	KindIfExpression
	KindInterpolatedString
	KindArray
	KindMap
	KindSet
	KindMixedTable

	// statements
	KindAssignment
	KindBreakStatement
	KindCallStatement
	KindContinueStatement
	KindDoStatement
	KindWhileStatement
	KindRepeatStatement
	KindIfStatement
	KindNumericForStatement
	KindForStatement
	KindFunctionDeclaration
	KindMethodDeclaration
	KindVariableDeclaration
	KindReturnStatement
	KindComment

	// fields
	KindMapField
	KindInterpolatedStringPart
)

// Used to detect what category a given kind falls into.
const (
	KindFirstIndexableExpression = KindIdentifier
	KindLastIndexableExpression  = KindParenthesizedExpression
	KindFirstExpression          = KindIdentifier
	KindLastExpression           = KindMixedTable
	KindFirstStatement           = KindAssignment
	KindLastStatement            = KindComment
	KindFirstField               = KindMapField
	KindLastField                = KindInterpolatedStringPart
)

// kindNames is the TypeScript reverse mapping SyntaxKind[kind] of the
// upstream enum. TypeScript enums with aliased members keep the last name
// assigned to a value, so MixedTable (== LastExpression) and
// InterpolatedStringPart (== LastField) come out as the "Last" aliases. The
// other aliased values are special-cased in upstream getKindName, so they are
// listed here under their real names.
var kindNames = [...]string{
	KindIdentifier:               "Identifier",
	KindTemporaryIdentifier:      "TemporaryIdentifier",
	KindComputedIndexExpression:  "ComputedIndexExpression",
	KindPropertyAccessExpression: "PropertyAccessExpression",
	KindCallExpression:           "CallExpression",
	KindMethodCallExpression:     "MethodCallExpression",
	KindParenthesizedExpression:  "ParenthesizedExpression",
	KindNone:                     "None",
	KindNilLiteral:               "NilLiteral",
	KindFalseLiteral:             "FalseLiteral",
	KindTrueLiteral:              "TrueLiteral",
	KindNumberLiteral:            "NumberLiteral",
	KindStringLiteral:            "StringLiteral",
	KindVarArgsLiteral:           "VarArgsLiteral",
	KindFunctionExpression:       "FunctionExpression",
	KindBinaryExpression:         "BinaryExpression",
	KindUnaryExpression:          "UnaryExpression",
	KindIfExpression:             "IfExpression",
	KindInterpolatedString:       "InterpolatedString",
	KindArray:                    "Array",
	KindMap:                      "Map",
	KindSet:                      "Set",
	KindMixedTable:               "LastExpression",
	KindAssignment:               "Assignment",
	KindBreakStatement:           "BreakStatement",
	KindCallStatement:            "CallStatement",
	KindContinueStatement:        "ContinueStatement",
	KindDoStatement:              "DoStatement",
	KindWhileStatement:           "WhileStatement",
	KindRepeatStatement:          "RepeatStatement",
	KindIfStatement:              "IfStatement",
	KindNumericForStatement:      "NumericForStatement",
	KindForStatement:             "ForStatement",
	KindFunctionDeclaration:      "FunctionDeclaration",
	KindMethodDeclaration:        "MethodDeclaration",
	KindVariableDeclaration:      "VariableDeclaration",
	KindReturnStatement:          "ReturnStatement",
	KindComment:                  "Comment",
	KindMapField:                 "MapField",
	KindInterpolatedStringPart:   "LastField",
}

// GetKindName returns the name upstream getKindName gives a kind, including
// its quirks ("LastExpression" for MixedTable, "LastField" for
// InterpolatedStringPart). Unknown kinds return "" (upstream: undefined).
func GetKindName(kind SyntaxKind) string {
	if kind < 0 || int(kind) >= len(kindNames) {
		return ""
	}
	return kindNames[kind]
}
