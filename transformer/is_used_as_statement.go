package transformer

// Port of src/TSTransformer/util/isUsedAsStatement.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

func isUsedAsStatement(expression *ast.Node) bool {
	child := SkipUpwards(expression)
	parent := child.Parent

	if ast.IsExpressionStatement(parent) {
		return true
	}

	// if part of for statement definition, except if used as the condition
	if ast.IsForStatement(parent) && parent.AsForStatement().Condition != child {
		return true
	}

	if ast.IsDeleteExpression(parent) && isUsedAsStatement(parent) {
		return true
	}

	return false
}
