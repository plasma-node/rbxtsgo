package transformer

// Port of src/TSTransformer/util/isBlockedByTryStatement.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

func isReturnBlockedByTryStatement(node *ast.Node) bool {
	ancestor := ast.FindAncestor(node, func(ancestor *ast.Node) bool {
		return ast.IsTryStatement(ancestor) || ast.IsFunctionLikeDeclaration(ancestor)
	})
	return ancestor != nil && ast.IsTryStatement(ancestor)
}

func isBreakBlockedByTryStatement(node *ast.Node) bool {
	ancestor := ast.FindAncestor(node, func(ancestor *ast.Node) bool {
		return ast.IsTryStatement(ancestor) || ast.IsIterationStatement(ancestor, false) || ast.IsSwitchStatement(ancestor)
	})
	return ancestor != nil && ast.IsTryStatement(ancestor)
}
