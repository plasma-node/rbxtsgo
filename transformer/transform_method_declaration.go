package transformer

// Port of src/TSTransformer/nodes/transformMethodDeclaration.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformMethodDeclaration(state *TransformState, node *ast.Node, ptr *mapPointer) *luau.List {
	result := luau.NewList()

	body := node.Body()
	if body == nil {
		return luau.NewList()
	}

	nodeName := node.Name()
	if nodeName == nil {
		panic(shared.AssertionMessage(""))
	}
	if ast.IsPrivateIdentifier(nodeName) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoPrivateIdentifier.At(shared.AtNode(nodeName)))
		return luau.NewList()
	}

	transformed := transformParameters(state, node)
	statements, parameters, hasDotDotDot := transformed.statements, transformed.parameters, transformed.hasDotDotDot
	statements.PushList(transformStatementList(state, body, body.Statements(), nil))

	name := transformPropertyName(state, nodeName)

	hasDecorators := ast.HasDecorators(node)
	if !hasDecorators {
		for _, parameter := range node.Parameters() {
			if ast.HasDecorators(parameter) {
				hasDecorators = true
				break
			}
		}
	}
	if hasDecorators {
		if !luau.IsSimplePrimitive(name) {
			tempID := luau.NewTempID("key")
			result.Push(luau.Create(&luau.VariableDeclaration{
				Left:  tempID,
				Right: name,
			}))
			name = tempID
		}
		state.SetClassElementObjectKey(node, name)
	}

	isAsync := ast.HasSyntacticModifier(node, ast.ModifierFlagsAsync)

	if node.AsMethodDeclaration().AsteriskToken != nil {
		if isAsync {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoAsyncGeneratorFunctions.At(shared.AtNode(node)))
		}
		statements = wrapStatementsAsGenerator(state, node, statements)
	}

	// can we use `function class:name() end`?
	if stringName, isString := name.(*luau.StringLiteral); !isAsync && isString && !luau.IsMap(ptr.value) && luau.IsValidIdentifier(stringName.Value) {
		ptrValue := ptr.value.(luau.AnyIdentifier)
		if isMethod(state, node) {
			parameters.Shift() // remove `self`
			result.Push(luau.Create(&luau.MethodDeclaration{
				Expression:   ptrValue,
				Name:         stringName.Value,
				Statements:   statements,
				Parameters:   parameters,
				HasDotDotDot: hasDotDotDot,
			}))
		} else {
			result.Push(luau.Create(&luau.FunctionDeclaration{
				Name:         luau.NewProperty(ptrValue, stringName.Value),
				Localize:     false,
				Statements:   statements,
				Parameters:   parameters,
				HasDotDotDot: hasDotDotDot,
			}))
		}
		return result
	}

	var expression luau.Expression = luau.Create(&luau.FunctionExpression{
		Statements:   statements,
		Parameters:   parameters,
		HasDotDotDot: hasDotDotDot,
	})

	if isAsync {
		expression = luau.NewCall(state.TS(node, "async"), expression)
	}

	// we have to use `class[name] = function()`
	result.PushList(state.CapturePrereqs(func() { assignToMapPointer(state, ptr, name, expression) }))

	return result
}
