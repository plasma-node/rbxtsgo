package transformer

// Port of src/TSTransformer/nodes/statements/transformExportAssignment.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformExportEquals(state *TransformState, node *ast.Node) *luau.List {
	state.HasExportEquals = true

	sourceFile := ast.GetSourceFileOfNode(node)
	sourceStatements := sourceFile.Statements.Nodes
	finalStatement := sourceStatements[len(sourceStatements)-1]
	if finalStatement == node {
		return luau.MakeList[luau.Node](
			luau.Create(&luau.ReturnStatement{Expression: transformExpression(state, node.Expression())}),
		)
	}
	return luau.MakeList[luau.Node](
		luau.Create(&luau.VariableDeclaration{
			Left:  state.GetModuleIDFromNode(node),
			Right: transformExpression(state, node.Expression()),
		}),
	)
}

func transformExportDefault(state *TransformState, node *ast.Node) *luau.List {
	statements := luau.NewList()

	expression, prereqs := Capture(state, func() luau.Expression { return transformExpression(state, node.Expression()) })
	statements.PushList(prereqs)
	statements.Push(luau.Create(&luau.VariableDeclaration{
		Left:  luau.NewID("default"),
		Right: expression,
	}))

	return statements
}

func transformExportAssignment(state *TransformState, node *ast.Node) *luau.List {
	symbol := state.Checker.GetSymbolAtLocation(node.Expression())
	if symbol != nil && isSymbolMutable(state, symbol) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoExportAssignmentLet.At(shared.AtNode(node)))
	}

	if symbol != nil && !isSymbolOfValue(state.Checker.SkipAlias(symbol)) {
		return luau.NewList()
	}

	if node.AsExportAssignment().IsExportEquals {
		return transformExportEquals(state, node)
	}
	return transformExportDefault(state, node)
}
