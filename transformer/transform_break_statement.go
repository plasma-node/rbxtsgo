package transformer

// Port of src/TSTransformer/nodes/statements/transformBreakStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformBreakStatement(state *TransformState, node *ast.Node) *luau.List {
	if label := node.AsBreakStatement().Label; label != nil {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoLabeledStatement.At(shared.AtNode(label)))
		return luau.NewList()
	}

	if isBreakBlockedByTryStatement(node) {
		state.MarkTryUses(func(u *TryUses) { u.UsesBreak = true })

		return luau.MakeList(luau.Create(&luau.ReturnStatement{
			Expression: state.TS(node, "TRY_BREAK"),
		}))
	}

	return luau.MakeList(luau.Create(&luau.BreakStatement{}))
}
