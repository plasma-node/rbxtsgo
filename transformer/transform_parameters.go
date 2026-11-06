package transformer

// Port of src/TSTransformer/nodes/transformParameters.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

// optimizeArraySpreadParameter optimizes parameters in the form `...[a, b, c]: [A, B, C]` to be just `(a, b, c)`
func optimizeArraySpreadParameter(state *TransformState, parameters *luau.List, bindingPattern *ast.Node) {
	for _, element := range bindingPattern.AsBindingPattern().Elements.Nodes {
		// ts.isOmittedExpression(element). TS 7's parser makes a BindingElement with no name for a hole, where TS 5.5.3's
		// makes an OmittedExpression.
		if element.Name() == nil {
			parameters.Push(luau.NewTempID(""))
		} else {
			bindingElement := element.AsBindingElement()
			if bindingElement.DotDotDotToken != nil {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoSpreadDestructuring.At(shared.AtNode(element)))
				return
			}
			name := element.Name()
			if ast.IsIdentifier(name) {
				paramID := transformIdentifierDefined(state, name)
				validateIdentifier(state, name)
				parameters.Push(paramID)
				if bindingElement.Initializer != nil {
					state.Prereq(transformInitializer(state, paramID.(luau.WritableExpression), bindingElement.Initializer))
				}
			} else {
				paramID := luau.NewTempID("param")
				parameters.Push(paramID)
				if bindingElement.Initializer != nil {
					state.Prereq(transformInitializer(state, paramID, bindingElement.Initializer))
				}
				if ast.IsArrayBindingPattern(name) {
					transformArrayBindingPattern(state, name, paramID)
				} else {
					transformObjectBindingPattern(state, name, paramID)
				}
			}
		}
	}
}

// transformParametersResult is the object transformParameters returns.
type transformParametersResult struct {
	parameters   *luau.List
	statements   *luau.List
	hasDotDotDot bool
}

func transformParameters(state *TransformState, node *ast.Node) transformParametersResult {
	parameters := luau.NewList()
	statements := luau.NewList()
	hasDotDotDot := false

	if isMethod(state, node) {
		parameters.Push(luau.Globals.Self)
	}

	for _, parameter := range node.Parameters() {
		parameterName := parameter.Name()
		if ast.IsThisIdentifier(parameterName) {
			continue
		}

		declaration := parameter.AsParameterDeclaration()

		if declaration.DotDotDotToken != nil && ast.IsArrayBindingPattern(parameterName) {
			prereqs := state.CapturePrereqs(func() {
				optimizeArraySpreadParameter(state, parameters, parameterName)
			})
			statements.PushList(prereqs)
			continue
		}

		var paramID luau.AnyIdentifier
		if ast.IsIdentifier(parameterName) {
			paramID = transformIdentifierDefined(state, parameterName)
			validateIdentifier(state, parameterName)
		} else {
			paramID = luau.NewTempID("param")
		}

		if declaration.DotDotDotToken != nil {
			hasDotDotDot = true
			statements.Push(luau.Create(&luau.VariableDeclaration{
				Left: paramID,
				Right: luau.Create(&luau.Array{
					Members: luau.MakeList(luau.Create(&luau.VarArgsLiteral{})),
				}),
			}))
		} else {
			parameters.Push(paramID)
		}

		if declaration.Initializer != nil {
			statements.Push(transformInitializer(state, paramID.(luau.WritableExpression), declaration.Initializer))
		}

		// destructuring
		if !ast.IsIdentifier(parameterName) {
			bindingPattern := parameterName
			if ast.IsArrayBindingPattern(bindingPattern) {
				statements.PushList(state.CapturePrereqs(func() {
					transformArrayBindingPattern(state, bindingPattern, paramID)
				}))
			} else {
				statements.PushList(state.CapturePrereqs(func() {
					transformObjectBindingPattern(state, bindingPattern, paramID)
				}))
			}
		}
	}

	return transformParametersResult{
		parameters:   parameters,
		statements:   statements,
		hasDotDotDot: hasDotDotDot,
	}
}
