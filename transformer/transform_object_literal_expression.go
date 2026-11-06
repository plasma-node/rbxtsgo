package transformer

// Port of src/TSTransformer/nodes/expressions/transformObjectLiteralExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformPropertyAssignment(state *TransformState, ptr *mapPointer, name *ast.Node, initializer *ast.Node) {
	left, leftPrereqs := Capture(state, func() luau.Expression { return transformPropertyName(state, name) })
	right, rightPrereqs := Capture(state, func() luau.Expression { return transformExpression(state, initializer) })

	if !leftPrereqs.IsEmpty() || !rightPrereqs.IsEmpty() {
		disableMapInline(state, ptr)
		state.PrereqList(leftPrereqs)
		left = state.PushToVar(left, "left")
	}

	state.PrereqList(rightPrereqs)

	assignToMapPointer(state, ptr, left, right)
}

func transformSpreadAssignment(state *TransformState, ptr *mapPointer, property *ast.Node) {
	expType := state.Checker.GetNonOptionalType(state.GetType(property.Expression()))
	symbol := getFirstDefinedSymbol(state, expType)
	if symbol != nil && state.Services.MacroManager.IsMacroOnlyClass(symbol) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoMacroObjectSpread.At(shared.AtNode(property)))
	}

	t := state.GetType(property.Expression())
	definitelyObject := isDefinitelyType(t, isObjectType)

	if definitelyObject && luau.IsMap(ptr.value) && ptr.value.(*luau.Map).Fields.IsEmpty() {
		ptr.value = state.PushToVar(
			luau.NewCall(luau.Globals.Table.Clone, transformExpression(state, property.Expression())),
			ptr.name,
		)
		state.Prereq(
			luau.Create(&luau.CallStatement{
				// Explicitly remove metatable because things like classes can be spread
				Expression: luau.NewCall(luau.Globals.Setmetatable, ptr.value, luau.NewNil()),
			}),
		)
		return
	}

	disableMapInline(state, ptr)
	spreadExp := transformExpression(state, property.Expression())
	if !definitelyObject {
		spreadExp = state.PushToVarIfComplex(spreadExp, "spread")
	}

	keyId := luau.NewTempID("k")
	valueId := luau.NewTempID("v")
	var statement luau.Statement = luau.Create(&luau.ForStatement{
		IDs:        luau.MakeList(keyId, valueId),
		Expression: spreadExp,
		Statements: luau.MakeList(
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: ptr.value.(luau.IndexableExpression),
					Index:      keyId,
				}),
				Operator: luau.AssignEqual,
				Right:    valueId,
			}),
		),
	})

	if !definitelyObject {
		statement = luau.Create(&luau.IfStatement{
			Condition:  createTruthinessChecks(state, spreadExp, property.Expression()),
			Statements: luau.MakeList(statement),
			ElseBody:   luau.NewList(),
		})
	}

	state.Prereq(statement)
}

func transformObjectLiteralExpression(state *TransformState, node *ast.Node) luau.Expression {
	// starts as luau.Map, becomes luau.TemporaryIdentifier when `disableInline` is called
	ptr := createMapPointer("object")
	for _, property := range node.AsObjectLiteralExpression().Properties.Nodes {
		validateMethodAssignment(state, property)
		if ast.IsPropertyAssignment(property) {
			if ast.IsPrivateIdentifier(property.Name()) {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoPrivateIdentifier.At(shared.AtNode(property.Name())))
				continue
			}
			transformPropertyAssignment(state, ptr, property.Name(), property.AsPropertyAssignment().Initializer)
		} else if ast.IsShorthandPropertyAssignment(property) {
			transformPropertyAssignment(state, ptr, property.Name(), property.Name())
		} else if ast.IsSpreadAssignment(property) {
			transformSpreadAssignment(state, ptr, property)
		} else if ast.IsMethodDeclaration(property) {
			state.PrereqList(transformMethodDeclaration(state, property, ptr))
		} else {
			// must be ts.AccessorDeclaration, which is banned
			state.Diagnostic.AddDiagnostic(shared.Errors.NoGetterSetter.At(shared.AtNode(property)))
		}
	}
	return ptr.value
}
