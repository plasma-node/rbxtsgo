package transformer

// Port of src/TSTransformer/util/binding/getAccessorForBindingType.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

// bindingAccessor is the type BindingAccessor. Accessors push onto idStack and the caller's next call must see it, so
// it's a pointer to the slice.
type bindingAccessor func(state *TransformState, parentID luau.AnyIdentifier, index int, idStack *[]luau.AnyIdentifier, isOmitted bool) luau.Expression

func peek[T any](array []T) T {
	var last T
	if len(array) > 0 {
		last = array[len(array)-1]
	}
	return last
}

func arrayAccessor(state *TransformState, parentID luau.AnyIdentifier, index int, idStack *[]luau.AnyIdentifier, isOmitted bool) luau.Expression {
	return luau.Create(&luau.ComputedIndexExpression{
		Expression: parentID,
		Index:      luau.NewNumber(float64(index + 1)),
	})
}

func stringAccessor(state *TransformState, parentID luau.AnyIdentifier, index int, idStack *[]luau.AnyIdentifier, isOmitted bool) luau.Expression {
	var id luau.AnyIdentifier
	if len(*idStack) == 0 {
		id = state.PushToVar(
			luau.NewCall(luau.Globals.String.Gmatch, parentID, luau.Globals.Utf8.Charpattern),
			"matcher",
		)
		*idStack = append(*idStack, id)
	} else {
		id = (*idStack)[0]
	}

	callExp := luau.NewCall(id)

	if isOmitted {
		state.Prereq(luau.Create(&luau.CallStatement{
			Expression: callExp,
		}))
		return luau.NewNone()
	}
	return callExp
}

func setAccessor(state *TransformState, parentID luau.AnyIdentifier, index int, idStack *[]luau.AnyIdentifier, isOmitted bool) luau.Expression {
	args := []luau.Expression{parentID}
	lastID := peek(*idStack)
	if lastID != nil {
		args = append(args, lastID)
	}
	callExp := luau.NewCall(luau.Globals.Next, args...)
	if isOmitted {
		state.Prereq(luau.Create(&luau.CallStatement{
			Expression: callExp,
		}))
		return luau.NewNone()
	}
	id := state.PushToVar(callExp, "value")
	*idStack = append(*idStack, id)
	return id
}

func mapAccessor(state *TransformState, parentID luau.AnyIdentifier, index int, idStack *[]luau.AnyIdentifier, isOmitted bool) luau.Expression {
	args := []luau.Expression{parentID}
	lastID := peek(*idStack)
	if lastID != nil {
		args = append(args, lastID)
	}
	keyID := luau.NewTempID("k")
	valueID := luau.NewTempID("v")
	ids := luau.MakeList(keyID, valueID)
	state.Prereq(luau.Create(&luau.VariableDeclaration{
		Left:  ids,
		Right: luau.NewCall(luau.Globals.Next, args...),
	}))
	*idStack = append(*idStack, keyID)
	return luau.Create(&luau.Array{Members: ids})
}

func iterableFunctionLuaTupleAccessor(state *TransformState, parentID luau.AnyIdentifier, index int, idStack *[]luau.AnyIdentifier, isOmitted bool) luau.Expression {
	callExp := luau.NewCall(parentID)
	if isOmitted {
		state.Prereq(luau.Create(&luau.CallStatement{
			Expression: callExp,
		}))
		return luau.NewNone()
	}
	return luau.NewArray(callExp)
}

func iterableFunctionAccessor(state *TransformState, parentID luau.AnyIdentifier, index int, idStack *[]luau.AnyIdentifier, isOmitted bool) luau.Expression {
	callExp := luau.NewCall(parentID)
	if isOmitted {
		state.Prereq(luau.Create(&luau.CallStatement{
			Expression: callExp,
		}))
		return luau.NewNone()
	}
	return callExp
}

func iterAccessor(state *TransformState, parentID luau.AnyIdentifier, index int, idStack *[]luau.AnyIdentifier, isOmitted bool) luau.Expression {
	callExp := luau.NewCall(luau.NewProperty(parentID, "next"))
	if isOmitted {
		state.Prereq(luau.Create(&luau.CallStatement{Expression: callExp}))
		return luau.NewNone()
	}
	return luau.NewProperty(callExp, "value")
}

// isThis is ts.isThis from TS 5.5.3.
func isThis(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindThisKeyword:
		return true
	case ast.KindIdentifier:
		// 'this' as a parameter
		return ast.IsThisIdentifier(node) && node.Parent.Kind == ast.KindParameter
	default:
		return false
	}
}

func getAccessorForBindingType(state *TransformState, node *ast.Node, t *checker.Type) bindingAccessor {
	if isDefinitelyType(t, isArrayType(state)) {
		return arrayAccessor
	} else if isDefinitelyType(t, isStringType) {
		return stringAccessor
	} else if isDefinitelyType(t, isSetType(state)) {
		return setAccessor
	} else if isDefinitelyType(t, isMapType(state)) {
		return mapAccessor
	} else if isDefinitelyType(t, isIterableFunctionLuaTupleType(state)) {
		return iterableFunctionLuaTupleAccessor
	} else if isDefinitelyType(t, isIterableFunctionType(state)) {
		return iterableFunctionAccessor
	} else if isDefinitelyType(t, isIterableType(state)) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoIterableIteration.At(shared.AtNode(node)))
		return func(*TransformState, luau.AnyIdentifier, int, *[]luau.AnyIdentifier, bool) luau.Expression {
			return luau.NewNone()
		}
	} else if isDefinitelyType(t, isGeneratorType(state)) ||
		isDefinitelyType(t, isObjectType) ||
		isThis(node) {
		return iterAccessor
	}
	panic(shared.AssertionMessage("Destructuring not supported for type: " + state.Checker.TypeToString(t)))
}
