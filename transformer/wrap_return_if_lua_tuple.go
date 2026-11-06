package transformer

// Port of src/TSTransformer/util/wrapReturnIfLuaTuple.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func shouldWrapLuaTuple(state *TransformState, node *ast.Node, exp luau.Expression) bool {
	if !luau.IsCall(exp) {
		return true
	}

	child := SkipUpwards(node)
	parent := child.Parent

	// `foo();`
	if ast.IsExpressionStatement(parent) {
		return false
	}

	// if part of for statement definition, except if used as the condition
	if ast.IsForStatement(parent) && parent.AsForStatement().Condition != child {
		return false
	}

	// `const [a] = foo()`
	if ast.IsVariableDeclaration(parent) &&
		ast.IsArrayBindingPattern(parent.Name()) &&
		!arrayBindingPatternContainsHoists(state, parent.Name()) {
		return false
	}

	// `[a] = foo()`
	if ast.IsAssignmentExpression(parent, false) && ast.IsArrayLiteralExpression(parent.AsBinaryExpression().Left) {
		return false
	}

	// `foo()[n]`
	if ast.IsElementAccessExpression(parent) {
		return false
	}

	// `return foo()`
	if ast.IsReturnStatement(parent) {
		return false
	}

	// `void foo()`
	if ast.IsVoidExpression(parent) {
		return false
	}

	return true
}

func wrapReturnIfLuaTuple(state *TransformState, node *ast.Node, exp luau.Expression) luau.Expression {
	if isLuaTupleType(state)(state.GetType(node)) && shouldWrapLuaTuple(state, node, exp) {
		return luau.NewArray(exp)
	}
	return exp
}
