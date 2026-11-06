package transformer

// Port of src/TSTransformer/util/isSymbolOfValue.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

// isSymbolOfValue checks that the symbol represents something with a value at runtime.
// Thus excluding types and const enums.
func isSymbolOfValue(symbol *ast.Symbol) bool {
	return symbol.Flags()&ast.SymbolFlagsValue != 0 && symbol.Flags()&ast.SymbolFlagsConstEnum == 0
}
