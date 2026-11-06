package transformer

// Port of src/TSTransformer/util/findConstructor.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

// findConstructor returns nil where JS returns undefined.
func findConstructor(node *ast.Node) *ast.Node {
	for _, member := range node.Members() {
		if ast.IsConstructorDeclaration(member) && member.Body() != nil {
			return member
		}
	}
	return nil
}
