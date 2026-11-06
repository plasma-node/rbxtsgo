package transformer

// Port of src/TSTransformer/nodes/statements/transformStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func diagnosticStatement(state *TransformState, factory *shared.DiagnosticFactory, node *ast.Node) *luau.List {
	state.Diagnostic.AddDiagnostic(factory.At(shared.AtNode(node)))
	return luau.NewList()
}

// transformStatement transforms a singular ts.Statement in a luau.list<...>. TRANSFORMER_BY_KIND is this switch: as a
// package-level table it would be an initialization cycle.
func transformStatement(state *TransformState, node *ast.Node) *luau.List {
	// if any modifiers of the node include the `declare` keyword we do not transform
	// `declare` tells us that the identifier of the node is defined somewhere else and we should trust it
	if ast.CanHaveModifiers(node) && hasDeclareModifier(node) {
		return luau.NewList()
	}
	switch node.Kind {
	// no emit
	case ast.KindInterfaceDeclaration, ast.KindTypeAliasDeclaration, ast.KindEmptyStatement:
		return luau.NewList()

	// banned statements
	case ast.KindForInStatement:
		return diagnosticStatement(state, shared.Errors.NoForInStatement, node)
	case ast.KindLabeledStatement:
		return diagnosticStatement(state, shared.Errors.NoLabeledStatement, node)
	case ast.KindDebuggerStatement:
		return diagnosticStatement(state, shared.Errors.NoDebuggerStatement, node)

	// regular transforms
	case ast.KindBlock:
		return transformBlock(state, node)
	case ast.KindBreakStatement:
		return transformBreakStatement(state, node)
	case ast.KindClassDeclaration:
		return transformClassDeclaration(state, node)
	case ast.KindContinueStatement:
		return transformContinueStatement(state, node)
	case ast.KindDoStatement:
		return transformDoStatement(state, node)
	case ast.KindEnumDeclaration:
		return transformEnumDeclaration(state, node)
	case ast.KindExportAssignment:
		return transformExportAssignment(state, node)
	case ast.KindExportDeclaration:
		return transformExportDeclaration(state, node)
	case ast.KindExpressionStatement:
		return transformExpressionStatement(state, node)
	case ast.KindForOfStatement:
		return transformForOfStatement(state, node)
	case ast.KindForStatement:
		return transformForStatement(state, node)
	case ast.KindFunctionDeclaration:
		return transformFunctionDeclaration(state, node)
	case ast.KindIfStatement:
		return transformIfStatement(state, node)
	case ast.KindImportDeclaration:
		return transformImportDeclaration(state, node)
	case ast.KindImportEqualsDeclaration:
		return transformImportEqualsDeclaration(state, node)
	case ast.KindModuleDeclaration:
		return transformModuleDeclaration(state, node)
	case ast.KindReturnStatement:
		return transformReturnStatement(state, node)
	case ast.KindSwitchStatement:
		return transformSwitchStatement(state, node)
	case ast.KindThrowStatement:
		return transformThrowStatement(state, node)
	case ast.KindTryStatement:
		return transformTryStatement(state, node)
	case ast.KindVariableStatement:
		return transformVariableStatement(state, node)
	case ast.KindWhileStatement:
		return transformWhileStatement(state, node)
	}
	panic(shared.AssertionMessage("Unknown statement: " + getKindName(node.Kind)))
}
