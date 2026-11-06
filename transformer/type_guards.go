package transformer

// Port of src/TSTransformer/typeGuards.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
)

func isBlockLike(node *ast.Node) bool {
	return node.Kind == ast.KindSourceFile ||
		node.Kind == ast.KindBlock ||
		node.Kind == ast.KindModuleBlock ||
		node.Kind == ast.KindCaseClause ||
		node.Kind == ast.KindDefaultClause
}

func isUnaryAssignmentOperator(operator ast.Kind) bool {
	return operator == ast.KindPlusPlusToken || operator == ast.KindMinusMinusToken
}

func isTemplateLiteralType(t *checker.Type) bool {
	return t.Flags()&checker.TypeFlagsTemplateLiteral != 0
}

// isNamespace tests the keyword, because TS 7 has no NodeFlags.Namespace. As with the flag, the inner parts of
// `namespace A.B {}` count too, and `module A {}` doesn't.
func isNamespace(node *ast.Node) bool {
	return ast.IsModuleDeclaration(node) && node.AsModuleDeclaration().Keyword == ast.KindNamespaceKeyword
}
