package transformer

// Port of src/TSTransformer/util/isValidMethodIndexWithoutCall.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

func isValidMethodIndexWithoutCall(state *TransformState, node *ast.Node) bool {
	parent := node.Parent
	// a.b !== undefined
	if ast.IsBinaryExpression(parent) {
		return true
	}

	// !a.b
	if ast.IsPrefixUnaryExpression(parent) {
		return true
	}

	// typeIs/typeOf macros
	if ast.IsCallExpression(parent) {
		expType := state.Checker.GetNonOptionalType(state.GetType(parent.Expression()))
		symbol := getFirstDefinedSymbol(state, expType)
		if symbol != nil {
			// Upstream compares against CALL_MACROS.typeIs and typeOf. Go funcs aren't comparable, so this compares names.
			macro := state.Services.MacroManager.CallMacro(symbol)
			if
			// typeIs will be a TypeError if usage is not the first argument
			macro == "typeIs" ||
				macro == "typeOf" {
				return true
			}
		}
	}

	return false
}
