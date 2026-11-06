package transformer

// Port of src/TSTransformer/util/isMethod.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func getThisParameter(parameters []*ast.Node) *ast.Node {
	if len(parameters) > 0 {
		name := parameters[0].Name()
		if ast.IsIdentifier(name) && ast.IsThisIdentifier(name) {
			return name
		}
	}
	return nil
}

func isMethodDeclaration(state *TransformState, node *ast.Node) bool {
	if ast.IsFunctionLike(node) {
		thisParam := getThisParameter(node.Parameters())
		if thisParam != nil {
			return state.GetType(thisParam).Flags()&checker.TypeFlagsVoid == 0
		} else {
			// namespace declare functions with `this` arg defined (i.e. utf8)
			if ast.IsFunctionDeclaration(node) {
				return false
			}

			if ast.IsMethodDeclaration(node) || ast.IsMethodSignatureDeclaration(node) {
				return true
			}

			// for some reason, FunctionExpressions within ObjectLiteralExpressions are implicitly methods
			if ast.IsFunctionExpression(node) {
				parent := SkipUpwards(node).Parent
				if ast.IsPropertyAssignment(parent) {
					grandparent := SkipUpwards(parent).Parent
					if ast.IsObjectLiteralExpression(grandparent) {
						return true
					}
				}
			}

			return false
		}
	}
	return false
}

func isMethodInner(state *TransformState, node *ast.Node, t *checker.Type) bool {
	hasMethodDefinition := false
	hasCallbackDefinition := false

	for _, callSignature := range state.Checker.GetCallSignatures(t) {
		var thisValueDeclaration *ast.Node
		if thisParameter := callSignature.ThisParameter(); thisParameter != nil {
			thisValueDeclaration = thisParameter.ValueDeclaration()
		}
		if thisValueDeclaration != nil {
			if state.GetType(thisValueDeclaration).Flags()&checker.TypeFlagsVoid == 0 {
				hasMethodDefinition = true
			} else {
				hasCallbackDefinition = true
			}
		} else if callSignature.Declaration() != nil {
			if isMethodDeclaration(state, callSignature.Declaration()) {
				hasMethodDefinition = true
			} else {
				hasCallbackDefinition = true
			}
		}
	}

	if hasMethodDefinition && hasCallbackDefinition {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoMixedTypeCall.At(shared.AtNode(node)))
	}

	return hasMethodDefinition
}

func isMethodFromType(state *TransformState, node *ast.Node, t *checker.Type) bool {
	result := false

	walkTypes(t, func(u *checker.Type) {
		// `result ||= getOrSetDefault(...)`: once result is true, later members are neither checked nor cached
		if u.Symbol() != nil && !result {
			value, ok := state.Multi.IsMethodCache[u.Symbol()]
			if !ok {
				value = isMethodInner(state, node, u)
				state.Multi.IsMethodCache[u.Symbol()] = value
			}
			result = value
		}
	})

	return result
}

func isMethod(state *TransformState, node *ast.Node) bool {
	return isMethodFromType(state, node, state.GetType(node))
}
