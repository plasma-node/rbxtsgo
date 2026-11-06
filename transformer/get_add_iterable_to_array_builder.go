package transformer

// Port of src/TSTransformer/util/getAddIterableToArrayBuilder.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

type addIterableToArrayBuilder = func(
	state *TransformState,
	expression luau.Expression,
	arrayID luau.AnyIdentifier,
	lengthID luau.AnyIdentifier,
	amtElementsSinceUpdate float64,
	shouldUpdateLengthID bool,
) *luau.List

func addArray(
	state *TransformState,
	expression luau.Expression,
	arrayID luau.AnyIdentifier,
	lengthID luau.AnyIdentifier,
	amtElementsSinceUpdate float64,
	shouldUpdateLengthID bool,
) *luau.List {
	result := luau.NewList()

	inputArray := state.PushToVarIfNonID(expression, "array")
	var inputLength luau.Expression = luau.NewUnary(luau.UnaryLen, inputArray)
	if shouldUpdateLengthID {
		inputLength = state.PushToVar(inputLength, valueToIDStr(inputArray)+"Length")
	}

	result.Push(luau.Create(&luau.CallStatement{
		Expression: luau.NewCall(luau.Globals.Table.Move,
			inputArray,
			luau.NewNumber(1),
			inputLength,
			luau.NewBinary(lengthID, luau.BinaryAdd, luau.NewNumber(amtElementsSinceUpdate+1)),
			arrayID,
		),
	}))

	if shouldUpdateLengthID {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthID,
			Operator: luau.AssignAdd,
			Right:    inputLength,
		}))
	}

	return result
}

func addString(
	state *TransformState,
	expression luau.Expression,
	arrayID luau.AnyIdentifier,
	lengthID luau.AnyIdentifier,
	amtElementsSinceUpdate float64,
	shouldUpdateLengthID bool,
) *luau.List {
	result := luau.NewList()

	if amtElementsSinceUpdate > 0 {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthID,
			Operator: luau.AssignAdd,
			Right:    luau.NewNumber(amtElementsSinceUpdate),
		}))
	}

	valueID := luau.NewTempID("char")
	result.Push(luau.Create(&luau.ForStatement{
		IDs:        luau.MakeList(valueID),
		Expression: luau.NewCall(luau.Globals.String.Gmatch, expression, luau.Globals.Utf8.Charpattern),
		Statements: luau.MakeList[luau.Node](
			luau.Create(&luau.Assignment{
				Left:     lengthID,
				Operator: luau.AssignAdd,
				Right:    luau.NewNumber(1),
			}),
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: arrayID,
					Index:      lengthID,
				}),
				Operator: luau.AssignEqual,
				Right:    valueID,
			}),
		),
	}))

	return result
}

func addSet(
	state *TransformState,
	expression luau.Expression,
	arrayID luau.AnyIdentifier,
	lengthID luau.AnyIdentifier,
	amtElementsSinceUpdate float64,
	shouldUpdateLengthID bool,
) *luau.List {
	result := luau.NewList()

	if amtElementsSinceUpdate > 0 {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthID,
			Operator: luau.AssignAdd,
			Right:    luau.NewNumber(amtElementsSinceUpdate),
		}))
	}

	valueID := luau.NewTempID("v")

	result.Push(luau.Create(&luau.ForStatement{
		IDs:        luau.MakeList(valueID),
		Expression: expression,
		Statements: luau.MakeList[luau.Node](
			luau.Create(&luau.Assignment{
				Left:     lengthID,
				Operator: luau.AssignAdd,
				Right:    luau.NewNumber(1),
			}),
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: arrayID,
					Index:      lengthID,
				}),
				Operator: luau.AssignEqual,
				Right:    valueID,
			}),
		),
	}))

	return result
}

func addMap(
	state *TransformState,
	expression luau.Expression,
	arrayID luau.AnyIdentifier,
	lengthID luau.AnyIdentifier,
	amtElementsSinceUpdate float64,
	shouldUpdateLengthID bool,
) *luau.List {
	result := luau.NewList()

	if amtElementsSinceUpdate > 0 {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthID,
			Operator: luau.AssignAdd,
			Right:    luau.NewNumber(amtElementsSinceUpdate),
		}))
	}

	keyID := luau.NewTempID("k")
	valueID := luau.NewTempID("v")
	result.Push(luau.Create(&luau.ForStatement{
		IDs:        luau.MakeList(keyID, valueID),
		Expression: expression,
		Statements: luau.MakeList[luau.Node](
			luau.Create(&luau.Assignment{
				Left:     lengthID,
				Operator: luau.AssignAdd,
				Right:    luau.NewNumber(1),
			}),
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: arrayID,
					Index:      lengthID,
				}),
				Operator: luau.AssignEqual,
				Right:    luau.NewArray(keyID, valueID),
			}),
		),
	}))

	return result
}

func addIterableFunction(
	state *TransformState,
	expression luau.Expression,
	arrayID luau.AnyIdentifier,
	lengthID luau.AnyIdentifier,
	amtElementsSinceUpdate float64,
	shouldUpdateLengthID bool,
) *luau.List {
	result := luau.NewList()

	if amtElementsSinceUpdate > 0 {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthID,
			Operator: luau.AssignAdd,
			Right:    luau.NewNumber(amtElementsSinceUpdate),
		}))
	}

	valueID := luau.NewTempID("result")

	result.Push(luau.Create(&luau.ForStatement{
		IDs:        luau.MakeList[luau.Node](valueID),
		Expression: expression,
		Statements: luau.MakeList[luau.Node](
			luau.Create(&luau.Assignment{
				Left:     lengthID,
				Operator: luau.AssignAdd,
				Right:    luau.NewNumber(1),
			}),
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: arrayID,
					Index:      lengthID,
				}),
				Operator: luau.AssignEqual,
				Right:    valueID,
			}),
		),
	}))

	return result
}

func addIterableFunctionLuaTuple(
	state *TransformState,
	expression luau.Expression,
	arrayID luau.AnyIdentifier,
	lengthID luau.AnyIdentifier,
	amtElementsSinceUpdate float64,
	shouldUpdateLengthID bool,
) *luau.List {
	result := luau.NewList()

	if amtElementsSinceUpdate > 0 {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthID,
			Operator: luau.AssignAdd,
			Right:    luau.NewNumber(amtElementsSinceUpdate),
		}))
	}

	iterFuncID := state.PushToVar(expression, "iterFunc")
	valueID := luau.NewTempID("results")
	result.Push(luau.Create(&luau.WhileStatement{
		Condition: luau.NewBool(true),
		Statements: luau.MakeList[luau.Node](
			luau.Create(&luau.VariableDeclaration{
				Left:  valueID,
				Right: luau.NewArray(luau.NewCall(iterFuncID)),
			}),
			luau.Create(&luau.IfStatement{
				Condition:  luau.NewBinary(luau.NewUnary(luau.UnaryLen, valueID), luau.BinaryEqual, luau.NewNumber(0)),
				Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				ElseBody:   luau.NewList(),
			}),
			luau.Create(&luau.Assignment{
				Left:     lengthID,
				Operator: luau.AssignAdd,
				Right:    luau.NewNumber(1),
			}),
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: arrayID,
					Index:      lengthID,
				}),
				Operator: luau.AssignEqual,
				Right:    valueID,
			}),
		),
	}))

	return result
}

func addGenerator(
	state *TransformState,
	expression luau.Expression,
	arrayID luau.AnyIdentifier,
	lengthID luau.AnyIdentifier,
	amtElementsSinceUpdate float64,
	shouldUpdateLengthID bool,
) *luau.List {
	result := luau.NewList()

	if amtElementsSinceUpdate > 0 {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthID,
			Operator: luau.AssignAdd,
			Right:    luau.NewNumber(amtElementsSinceUpdate),
		}))
	}

	iterID := luau.NewTempID("result")
	result.Push(luau.Create(&luau.ForStatement{
		IDs:        luau.MakeList[luau.Node](iterID),
		Expression: luau.NewProperty(convertToIndexableExpression(expression), "next"),
		Statements: luau.MakeList[luau.Node](
			luau.Create(&luau.IfStatement{
				Condition:  luau.NewProperty(iterID, "done"),
				Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				ElseBody:   luau.NewList(),
			}),
			luau.Create(&luau.Assignment{
				Left:     lengthID,
				Operator: luau.AssignAdd,
				Right:    luau.NewNumber(1),
			}),
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: arrayID,
					Index:      lengthID,
				}),
				Operator: luau.AssignEqual,
				Right:    luau.NewProperty(iterID, "value"),
			}),
		),
	}))

	return result
}

func addNothing(
	state *TransformState,
	expression luau.Expression,
	arrayID luau.AnyIdentifier,
	lengthID luau.AnyIdentifier,
	amtElementsSinceUpdate float64,
	shouldUpdateLengthID bool,
) *luau.List {
	return luau.NewList()
}

func getAddIterableToArrayBuilder(state *TransformState, node *ast.Node, t *checker.Type) addIterableToArrayBuilder {
	if isDefinitelyType(t, isArrayType(state)) {
		return addArray
	} else if isDefinitelyType(t, isStringType) {
		return addString
	} else if isDefinitelyType(t, isSetType(state)) {
		return addSet
	} else if isDefinitelyType(t, isMapType(state)) {
		return addMap
	} else if isDefinitelyType(t, isIterableFunctionLuaTupleType(state)) {
		return addIterableFunctionLuaTuple
	} else if isDefinitelyType(t, isIterableFunctionType(state)) {
		return addIterableFunction
	} else if isDefinitelyType(t, isGeneratorType(state)) {
		return addGenerator
	} else if isDefinitelyType(t, isIterableType(state)) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoIterableIteration.At(shared.AtNode(node)))
		return addNothing
	} else if t.IsUnion() {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoMacroUnion.At(shared.AtNode(node)))
		return addNothing
	} else {
		panic(shared.AssertionMessage("Iteration type not implemented: " + state.Checker.TypeToString(t)))
	}
}
