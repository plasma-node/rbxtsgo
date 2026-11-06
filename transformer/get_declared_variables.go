package transformer

// Port of src/TSTransformer/util/getDeclaredVariables.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

func getDeclaredVariablesFromBindingName(node *ast.Node, list *[]*ast.Node) {
	if ast.IsIdentifier(node) {
		*list = append(*list, node)
	} else if ast.IsObjectBindingPattern(node) {
		for _, element := range node.AsBindingPattern().Elements.Nodes {
			getDeclaredVariablesFromBindingName(element.Name(), list)
		}
	} else if ast.IsArrayBindingPattern(node) {
		for _, element := range node.AsBindingPattern().Elements.Nodes {
			// In TS 7 an omitted element is a ts.BindingElement with a nil name.
			if element.Name() != nil {
				getDeclaredVariablesFromBindingName(element.Name(), list)
			}
		}
	}
}

func getDeclaredVariables(node *ast.Node) []*ast.Node {
	list := []*ast.Node{}
	if ast.IsVariableDeclarationList(node) {
		for _, declaration := range node.AsVariableDeclarationList().Declarations.Nodes {
			getDeclaredVariablesFromBindingName(declaration.Name(), &list)
		}
	} else {
		getDeclaredVariablesFromBindingName(node.Name(), &list)
	}
	return list
}
