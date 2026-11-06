package transformer

// Port of src/TSTransformer/nodes/statements/transformContinueStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformContinueStatement(state *TransformState, node *ast.Node) *luau.List {
	if label := node.AsContinueStatement().Label; label != nil {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoLabeledStatement.At(shared.AtNode(label)))
		return luau.NewList()
	}

	if state.Data.ProjectOptions.FixEmit {
		return createContinueStatements(state, node)
	}

	if isBreakBlockedByTryStatement(node) {
		state.MarkTryUses(func(u *TryUses) { u.UsesContinue = true })

		return luau.MakeList(luau.Create(&luau.ReturnStatement{
			Expression: state.TS(node, "TRY_CONTINUE"),
		}))
	}

	return luau.MakeList(luau.Create(&luau.ContinueStatement{}))
}
