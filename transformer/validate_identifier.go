package transformer

// Port of src/TSTransformer/util/validateIdentifier.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func validateIdentifier(state *TransformState, node *ast.Node) {
	if !luau.IsValidIdentifier(node.Text()) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoInvalidIdentifier.At(shared.AtNode(node)))
	} else if luau.IsReservedIdentifier(node.Text()) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoReservedIdentifier.At(shared.AtNode(node)))
	}
}
