package transformer

// Port of src/TSTransformer/util/getStatements.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

func getStatements(statement *ast.Node) []*ast.Node {
	if ast.IsBlock(statement) {
		return statement.AsBlock().Statements.Nodes
	}
	return []*ast.Node{statement}
}
