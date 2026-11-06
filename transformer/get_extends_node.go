package transformer

// Port of src/TSTransformer/util/getExtendsNode.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

// getExtendsNode returns nil where JS returns undefined.
func getExtendsNode(node *ast.Node) *ast.Node {
	heritageClauses := node.ClassLikeData().HeritageClauses
	if heritageClauses == nil {
		return nil
	}
	for _, clause := range heritageClauses.Nodes {
		heritageClause := clause.AsHeritageClause()
		if heritageClause.Token == ast.KindExtendsKeyword {
			if len(heritageClause.Types.Nodes) == 0 {
				return nil
			}
			return heritageClause.Types.Nodes[0]
		}
	}
	return nil
}
