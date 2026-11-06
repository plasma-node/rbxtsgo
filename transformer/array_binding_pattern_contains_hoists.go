package transformer

// Port of src/TSTransformer/util/arrayBindingPatternContainsHoists.ts.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

func arrayBindingPatternContainsHoists(state *TransformState, arrayBindingPattern *ast.Node) bool {
	for _, element := range arrayBindingPattern.AsBindingPattern().Elements.Nodes {
		// If it's not a ts.BindingElement, it must be a ts.OmittedExpression. In that case, doesn't create a variable.
		// For non-identifiers, element.name must be a nested array/object binding pattern.
		// For those cases, the hoisting logic is handled elsewhere and the variable here will be a tempId.
		// In TS 7 an omitted element is a ts.BindingElement with a nil name.
		if ast.IsBindingElement(element) && element.Name() != nil && ast.IsIdentifier(element.Name()) {
			symbol := state.Checker.GetSymbolAtLocation(element.Name())
			if symbol != nil {
				// isHoisted is marked inside checkVariableHoist
				checkVariableHoist(state, element.Name(), symbol)
				if state.IsHoisted[symbol] {
					return true
				}
			}
		}
	}
	return false
}
