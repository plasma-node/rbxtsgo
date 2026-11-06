package transformer

// Port of src/TSTransformer/nodes/class/transformClassConstructor.ts.

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

const constructorName = "constructor"

func transformPropertyInitializers(state *TransformState, node *ast.Node) *luau.List {
	statements := luau.NewList()
	for _, member := range node.Members() {
		if !ast.IsPropertyDeclaration(member) {
			continue
		}
		if ast.HasStaticModifier(member) {
			continue
		}

		name := member.Name()
		if ast.IsPrivateIdentifier(name) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoPrivateIdentifier.At(shared.AtNode(node)))
			continue
		}

		initializer := member.Initializer()
		if initializer == nil {
			continue
		}

		index, indexPrereqs := Capture(state, func() luau.Expression { return transformPropertyName(state, name) })
		statements.PushList(indexPrereqs)

		right, rightPrereqs := Capture(state, func() luau.Expression { return transformExpression(state, initializer) })
		statements.PushList(rightPrereqs)

		statements.Push(luau.Create(&luau.Assignment{
			Left: luau.Create(&luau.ComputedIndexExpression{
				Expression: luau.Globals.Self,
				Index:      index,
			}),
			Operator: luau.AssignEqual,
			Right:    right,
		}))
	}
	return statements
}

func transformImplicitClassConstructor(state *TransformState, node *ast.Node, name luau.AnyIdentifier) *luau.List {
	statements := luau.NewList()

	hasDotDotDot := false

	// if extends + no constructor:
	// - add ... to params
	// - add super.constructor(self, ...)
	if getExtendsNode(node) != nil {
		hasDotDotDot = true
		statements.Push(luau.Create(&luau.CallStatement{
			Expression: luau.NewCall(luau.NewProperty(luau.Globals.Super, constructorName),
				luau.Globals.Self,
				luau.Create(&luau.VarArgsLiteral{}),
			),
		}))
	}

	statements.PushList(transformPropertyInitializers(state, node))

	return luau.MakeList(luau.Create(&luau.MethodDeclaration{
		Expression:   name,
		Name:         constructorName,
		Statements:   statements,
		Parameters:   luau.NewList(),
		HasDotDotDot: hasDotDotDot,
	}))
}

func transformClassConstructor(state *TransformState, node *ast.Node, name luau.AnyIdentifier) *luau.List {
	parametersResult := transformParameters(state, node)
	statements, parameters, hasDotDotDot := parametersResult.statements, parametersResult.parameters, parametersResult.hasDotDotDot
	body := node.Body()
	bodyStatements := getStatements(body)

	// property parameters must come after the first super() call
	superIndex := slices.IndexFunc(bodyStatements, func(v *ast.Node) bool {
		return ast.IsExpressionStatement(v) && ast.IsSuperCall(v.Expression())
	})

	statements.PushList(transformStatementList(state, body, bodyStatements[:superIndex+1], nil))

	for _, parameter := range node.Parameters() {
		if ast.IsParameterPropertyDeclaration(parameter, parameter.Parent) {
			paramID := transformIdentifierDefined(state, parameter.Name())
			var paramName string
			switch id := paramID.(type) {
			case *luau.Identifier:
				paramName = id.Name
			case *luau.TemporaryIdentifier:
				paramName = id.Name
			}
			statements.Push(luau.Create(&luau.Assignment{
				Left:     luau.NewProperty(luau.Globals.Self, paramName),
				Operator: luau.AssignEqual,
				Right:    paramID,
			}))
		}
	}

	statements.PushList(transformPropertyInitializers(state, node.Parent))

	statements.PushList(transformStatementList(state, body, bodyStatements[superIndex+1:], nil))

	return luau.MakeList(luau.Create(&luau.MethodDeclaration{
		Expression:   name,
		Name:         constructorName,
		Statements:   statements,
		Parameters:   parameters,
		HasDotDotDot: hasDotDotDot,
	}))
}
