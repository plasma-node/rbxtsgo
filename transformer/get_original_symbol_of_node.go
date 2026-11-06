package transformer

// Port of src/TSTransformer/util/getOriginalSymbolOfNode.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
)

func getOriginalSymbolOfNode(typeChecker *checker.Checker, node *ast.Node) *ast.Symbol {
	symbol := typeChecker.GetSymbolAtLocation(node)
	if symbol != nil {
		return typeChecker.SkipAlias(symbol)
	}
	return symbol
}
