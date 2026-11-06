package transformer

// Port of src/TSTransformer/nodes/statements/transformVariableStatement.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

// transformVariable returns the variable's export access or identifier. right can be nil.
func transformVariable(state *TransformState, identifier *ast.Node, right luau.Expression) luau.WritableExpression {
	validateIdentifier(state, identifier)

	symbol := state.Checker.GetSymbolAtLocation(identifier)
	if symbol == nil {
		panic(shared.AssertionMessage(""))
	}

	// export let
	if isSymbolMutable(state, symbol) {
		exportAccess := state.GetModuleIDPropertyAccess(symbol)
		if exportAccess != nil {
			if right != nil {
				state.Prereq(luau.Create(&luau.Assignment{
					Left:     exportAccess,
					Operator: luau.AssignEqual,
					Right:    right,
				}))
			}
			return exportAccess
		}
	}

	var left luau.AnyIdentifier = transformIdentifierDefined(state, identifier)

	checkVariableHoist(state, identifier, symbol)
	if state.IsHoisted[symbol] {
		// no need to do `x = nil` if the variable is already created
		if right != nil {
			state.Prereq(luau.Create(&luau.Assignment{Left: left, Operator: luau.AssignEqual, Right: right}))
		}
	} else {
		state.Prereq(luau.Create(&luau.VariableDeclaration{Left: left, Right: right}))
	}

	return left.(luau.WritableExpression)
}

// transformOptimizedArrayBindingPattern takes rhs as a luau.Expression or a list of them.
//
// TS 7 parses an omitted element of an array binding pattern as a BindingElement with no name, where TS 5.5.3 has an
// OmittedExpression, so an omitted element is also Name() == nil here.
func transformOptimizedArrayBindingPattern(state *TransformState, bindingPattern *ast.Node, rhs luau.NodeOrList) *luau.List {
	return state.CapturePrereqs(func() {
		ids := luau.NewList()
		statements := state.CapturePrereqs(func() {
			for _, element := range bindingPattern.AsBindingPattern().Elements.Nodes {
				if ast.IsOmittedExpression(element) || element.Name() == nil {
					ids.Push(luau.NewTempID(""))
				} else {
					bindingElement := element.AsBindingElement()
					if bindingElement.DotDotDotToken != nil {
						state.Diagnostic.AddDiagnostic(shared.Errors.NoSpreadDestructuring.At(shared.AtNode(element)))
						return
					}
					if ast.IsIdentifier(element.Name()) {
						validateIdentifier(state, element.Name())
						var id luau.AnyIdentifier = transformIdentifierDefined(state, element.Name())
						ids.Push(id)
						if bindingElement.Initializer != nil {
							state.Prereq(transformInitializer(state, id.(luau.WritableExpression), bindingElement.Initializer))
						}
					} else {
						id := luau.NewTempID("binding")
						ids.Push(id)
						if bindingElement.Initializer != nil {
							state.Prereq(transformInitializer(state, id, bindingElement.Initializer))
						}
						if ast.IsArrayBindingPattern(element.Name()) {
							transformArrayBindingPattern(state, element.Name(), id)
						} else {
							transformObjectBindingPattern(state, element.Name(), id)
						}
					}
				}
			}
		})
		if ids.IsEmpty() {
			panic(shared.AssertionMessage(""))
		}
		state.Prereq(luau.Create(&luau.VariableDeclaration{Left: ids, Right: rhs}))
		state.PrereqList(statements)
	})
}

func transformVariableDeclaration(state *TransformState, node *ast.Node) *luau.List {
	declaration := node.AsVariableDeclaration()
	statements := luau.NewList()
	var value luau.Expression
	if declaration.Initializer != nil {
		// must transform right _before_ checking isHoisted, that way references inside of value can be hoisted
		statements.PushList(
			state.CapturePrereqs(func() { value = transformExpression(state, declaration.Initializer) }),
		)
	}

	name := node.Name()
	if ast.IsIdentifier(name) {
		statements.PushList(
			state.CapturePrereqs(func() { transformVariable(state, name, value) }),
		)
	} else {
		// in destructuring, rhs must be executed first
		if !(declaration.Initializer != nil && value != nil) {
			panic(shared.AssertionMessage(""))
		}

		// optimize empty destructure
		if len(name.AsBindingPattern().Elements.Nodes) == 0 {
			if array, isArray := value.(*luau.Array); !isArray || !array.Members.IsEmpty() {
				statements.PushList(wrapExpressionStatement(value))
			}
			return statements
		}

		if ast.IsArrayBindingPattern(name) {
			array, isArray := value.(*luau.Array)
			if luau.IsCall(value) &&
				isLuaTupleType(state)(state.GetType(declaration.Initializer)) &&
				!arrayBindingPatternContainsHoists(state, name) {
				statements.PushList(transformOptimizedArrayBindingPattern(state, name, value))
			} else if isArray &&
				!array.Members.IsEmpty() &&
				// we can't localize multiple variables at the same time if any of them are hoisted
				!arrayBindingPatternContainsHoists(state, name) {
				statements.PushList(transformOptimizedArrayBindingPattern(state, name, array.Members))
			} else {
				statements.PushList(
					state.CapturePrereqs(func() {
						transformArrayBindingPattern(state, name, state.PushToVar(value, "binding"))
					}),
				)
			}
		} else {
			statements.PushList(
				state.CapturePrereqs(func() {
					transformObjectBindingPattern(state, name, state.PushToVar(value, "binding"))
				}),
			)
		}
	}

	return statements
}

func isVarDeclaration(node *ast.Node) bool {
	return node.Flags&ast.NodeFlagsConst == 0 && node.Flags&ast.NodeFlagsLet == 0
}

func transformVariableDeclarationList(state *TransformState, node *ast.Node) *luau.List {
	if isVarDeclaration(node) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoVar.At(shared.AtNode(node)))
	}

	statements := luau.NewList()
	for _, declaration := range node.AsVariableDeclarationList().Declarations.Nodes {
		variableStatements, prereqs := Capture(state, func() *luau.List { return transformVariableDeclaration(state, declaration) })
		statements.PushList(prereqs)
		statements.PushList(variableStatements)
	}

	return statements
}

func transformVariableStatement(state *TransformState, node *ast.Node) *luau.List {
	return transformVariableDeclarationList(state, node.AsVariableStatement().DeclarationList)
}
