package transformer

// Port of src/TSTransformer/nodes/jsx/transformJsxAttributes.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

// getTextOfJsxNamespacedName is ts.getTextOfJsxNamespacedName from TS 5.5.3.
func getTextOfJsxNamespacedName(node *ast.Node) string {
	namespacedName := node.AsJsxNamespacedName()
	return namespacedName.Namespace.Text() + ":" + namespacedName.Name().Text()
}

func createJsxAttributeLoop(state *TransformState, attributesPtrValue luau.AnyIdentifier, expression luau.Expression, tsExpression *ast.Node) luau.Statement {
	definitelyObject := isDefinitelyType(state.GetType(tsExpression), isObjectType)
	if !definitelyObject {
		expression = state.PushToVarIfComplex(expression, "attribute")
	}

	keyID := luau.NewTempID("k")
	valueID := luau.NewTempID("v")
	var statement luau.Statement = luau.Create(&luau.ForStatement{
		IDs:        luau.MakeList(keyID, valueID),
		Expression: expression,
		Statements: luau.MakeList(luau.Create(&luau.Assignment{
			Left: luau.Create(&luau.ComputedIndexExpression{
				Expression: attributesPtrValue,
				Index:      keyID,
			}),
			Operator: luau.AssignEqual,
			Right:    valueID,
		})),
	})

	if !definitelyObject {
		statement = luau.Create(&luau.IfStatement{
			Condition:  createTruthinessChecks(state, expression, tsExpression),
			Statements: luau.MakeList(statement),
			ElseBody:   luau.NewList(),
		})
	}

	return statement
}

func transformJsxAttribute(state *TransformState, attribute *ast.Node, attributesPtr *mapPointer) {
	initializer := attribute.AsJsxAttribute().Initializer
	if initializer != nil && ast.IsJsxExpression(initializer) {
		initializer = initializer.AsJsxExpression().Expression
	}

	var init luau.Expression
	var initPrereqs *luau.List
	if initializer != nil {
		init, initPrereqs = Capture(state, func() luau.Expression { return transformExpression(state, initializer) })
	} else {
		init, initPrereqs = luau.NewBool(true), luau.NewList()
	}

	if !initPrereqs.IsEmpty() {
		disableMapInline(state, attributesPtr)
		state.PrereqList(initPrereqs)
	}

	var text string
	if ast.IsIdentifier(attribute.Name()) {
		text = attribute.Name().Text()
	} else {
		text = getTextOfJsxNamespacedName(attribute.Name())
	}
	name := luau.NewString(text)
	assignToMapPointer(state, attributesPtr, name, init)
}

func transformJsxAttributes(state *TransformState, attributes *ast.Node, attributesPtr *mapPointer) {
	properties := attributes.AsJsxAttributes().Properties.Nodes
	for _, attribute := range properties {
		if ast.IsJsxAttribute(attribute) {
			transformJsxAttribute(state, attribute, attributesPtr)
		} else {
			// spread attributes: `<frame { ...x }/>`

			expType := state.Checker.GetNonOptionalType(state.GetType(attribute.Expression()))
			symbol := getFirstDefinedSymbol(state, expType)
			if symbol != nil && state.Services.MacroManager.IsMacroOnlyClass(symbol) {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoMacroObjectSpread.At(shared.AtNode(attribute)))
			}

			expression := transformExpression(state, attribute.Expression())

			if attribute == properties[0] && isDefinitelyType(expType, isObjectType) {
				attributesPtr.value = state.PushToVar(
					luau.NewCall(luau.Globals.Table.Clone, expression),
					attributesPtr.name,
				)
				state.Prereq(luau.Create(&luau.CallStatement{
					// Explicitly remove metatable because things like classes can be spread
					Expression: luau.NewCall(luau.Globals.Setmetatable, attributesPtr.value, luau.NewNil()),
				}))
				continue
			}

			disableMapInline(state, attributesPtr)
			state.Prereq(createJsxAttributeLoop(state, attributesPtr.value.(luau.AnyIdentifier), expression, attribute.Expression()))
		}
	}
}
