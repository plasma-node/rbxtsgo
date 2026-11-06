package transformer

// Port of src/TSTransformer/util/isSymbolMutable.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

func isSymbolMutable(state *TransformState, idSymbol *ast.Symbol) bool {
	if value, ok := state.Multi.IsDefinedAsLetCache[idSymbol]; ok {
		return value
	}
	value := func() bool {
		if idSymbol.ValueDeclaration() != nil {
			if ast.IsParameterDeclaration(idSymbol.ValueDeclaration()) {
				return true
			}

			varDecList := GetAncestor(idSymbol.ValueDeclaration(), ast.IsVariableDeclarationList)
			if varDecList != nil {
				return varDecList.Flags&ast.NodeFlagsLet != 0
			}
		}
		return false
	}()
	state.Multi.IsDefinedAsLetCache[idSymbol] = value
	return value
}
