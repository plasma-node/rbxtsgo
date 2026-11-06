package transformer

// Port of src/TSTransformer/util/validateNotAny.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func validateNotAnyType(state *TransformState, node *ast.Node) {
	if ast.IsSpreadElement(node) {
		node = SkipDownwards(node.Expression())
	}

	t := state.GetType(node)

	if isDefinitelyType(t, isArrayType(state)) {
		// Array<T> -> T
		indexType := state.Checker.GetIndexTypeOfType(t, state.Checker.GetNumberType())
		if indexType != nil {
			t = indexType
		}
	}

	if isDefinitelyType(t, isAnyType(state)) {
		// given a type like `a: { [index: string]: any }`, `a["b"]` will not have a symbol
		symbol := getOriginalSymbolOfNode(state.Checker, node)
		if symbol != nil {
			if !state.Multi.IsReportedByNoAnyCache[symbol] {
				state.Multi.IsReportedByNoAnyCache[symbol] = true
				state.Diagnostic.AddDiagnostic(shared.Errors.NoAny.At(shared.AtNode(node)))
			}
		} else {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoAny.At(shared.AtNode(node)))
		}
	}
}
