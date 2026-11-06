package transformer

// Port of src/TSTransformer/util/addIndexDiagnostics.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func addIndexDiagnostics(state *TransformState, node *ast.Node, expType *checker.Type) {
	symbol := getFirstDefinedSymbol(state, expType)
	if (symbol != nil && getPropertyCallMacro(state, symbol) != nil) ||
		(!isValidMethodIndexWithoutCall(state, SkipUpwards(node)) && isMethod(state, node)) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoIndexWithoutCall.At(shared.AtNode(node)))
	}

	if ast.IsPrototypeAccess(node) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoPrototype.At(shared.AtNode(node)))
	}
}
