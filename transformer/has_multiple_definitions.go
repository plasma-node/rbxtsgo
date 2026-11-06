package transformer

// Port of src/TSTransformer/util/hasMultipleDefinitions.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

func hasMultipleDefinitions(symbol *ast.Symbol, filter func(declaration *ast.Node) bool) bool {
	amtValueDefinitions := 0
	for _, declaration := range symbol.Declarations() {
		if filter(declaration) {
			amtValueDefinitions++
			if amtValueDefinitions > 1 {
				return true
			}
		}
	}
	return false
}
