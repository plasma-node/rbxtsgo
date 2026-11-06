package transformer

// Port of src/TSTransformer/util/validateMethodAssignment.ts.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func hasCallSignatures(t *checker.Type) bool {
	hasCallSignatures := false
	walkTypes(t, func(u *checker.Type) {
		hasCallSignatures = hasCallSignatures || len(checkerOfType(u).GetCallSignatures(u)) > 0
	})
	return hasCallSignatures
}

func validateTypes(state *TransformState, node *ast.Node, baseType *checker.Type, assignmentType *checker.Type) {
	if hasCallSignatures(baseType) && hasCallSignatures(assignmentType) {
		assignmentIsMethod := isMethodFromType(state, node, assignmentType)
		if isMethodFromType(state, node, baseType) != assignmentIsMethod {
			if assignmentIsMethod {
				state.Diagnostic.AddDiagnostic(shared.Errors.ExpectedMethodGotFunction.At(shared.AtNode(node)))
			} else {
				state.Diagnostic.AddDiagnostic(shared.Errors.ExpectedFunctionGotMethod.At(shared.AtNode(node)))
			}
		}
	}
}

func validateObjectLiteralElement(state *TransformState, node *ast.Node) {
	t := state.GetType(node)
	contextualType := state.Checker.GetContextualTypeForObjectLiteralElement(node, checker.ContextFlagsNone)
	if contextualType != nil && contextualType != t {
		validateTypes(state, node, t, contextualType)
	}
}

func validateHeritageClause(state *TransformState, node *ast.Node, typeNode *ast.Node) {
	name := ast.GetPropertyNameForPropertyNameNode(node.Name())
	// JS returns undefined for a computed name that isn't a literal, where Go returns InternalSymbolNameMissing
	if name == "" || name == ast.InternalSymbolNameMissing {
		return
	}

	// In TS 5.5.3, getPropertyNameForPropertyNameNode returns an escaped name and the public getTypeOfPropertyOfType
	// escapes it again. So a name starting with `__` looks up one with an extra `_`. Go's names are never escaped.
	if strings.HasPrefix(name, "__") {
		name = "_" + name
	}

	t := state.GetType(node)
	propertyType := state.Checker.GetTypeOfPropertyOfType(state.GetType(typeNode), name)
	if propertyType == nil {
		return
	}

	validateTypes(state, node, t, propertyType)
}

func validateSpread(state *TransformState, node *ast.Node) {
	t := state.GetType(node.Expression())
	contextualType := state.Checker.GetContextualType(node.Expression(), checker.ContextFlagsNone)
	if contextualType == nil {
		return
	}

	// This order can differ from TS 5.5.3's on spread types, see "Property order on spread types" in the guide. It
	// can only change the order of the diagnostics below.
	for _, property := range state.Checker.GetPropertiesOfType(t) {
		// JS passes property.name to the public getTypeOfPropertyOfType, which escapes it. For a symbol-keyed or
		// private property that finds nothing, so upstream always reaches the `continue` below. Go's name finds it.
		if strings.HasPrefix(property.Name(), ast.InternalSymbolNamePrefix) {
			continue
		}
		basePropertyType := state.Checker.GetTypeOfPropertyOfType(t, property.Name())
		assignmentPropertyType := state.Checker.GetTypeOfPropertyOfType(contextualType, property.Name())
		if basePropertyType == nil {
			continue
		}
		if assignmentPropertyType == nil {
			continue
		}

		validateTypes(state, node, basePropertyType, assignmentPropertyType)
	}
}

func validateMethodAssignment(state *TransformState, node *ast.Node) {
	if ast.IsClassElement(node) && ast.IsClassLike(node.Parent) && node.Name() != nil {
		for _, typeNode := range getAllSuperTypeNodes(node.Parent) {
			validateHeritageClause(state, node, typeNode)
		}
	} else if ast.IsObjectLiteralElement(node) {
		if ast.IsSpreadAssignment(node) {
			if !ast.IsObjectLiteralExpression(node.Expression()) {
				validateSpread(state, node)
			}
		} else {
			validateObjectLiteralElement(state, node)
		}
	}
}
