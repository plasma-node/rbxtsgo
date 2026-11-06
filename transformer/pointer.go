package transformer

// Port of src/TSTransformer/util/pointer.ts.

import "github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"

// pointer is Pointer<T>. value holds a *luau.Map, a *luau.Array or an identifier. disableMapInline and
// disableArrayInline change it, so pointers are passed by reference, as in JS.
type pointer struct {
	name  string
	value luau.Expression
}

type mapPointer = pointer
type arrayPointer = pointer

func createMapPointer(name string) *mapPointer {
	return &mapPointer{name, luau.NewMap()}
}

func createArrayPointer(name string) *arrayPointer {
	return &arrayPointer{name, luau.NewArray()}
}

func assignToMapPointer(state *TransformState, ptr *pointer, left luau.Expression, right luau.Expression) {
	if value, ok := ptr.value.(*luau.Map); ok {
		value.Fields.Push(luau.Create(&luau.MapField{
			Index: left,
			Value: right,
		}))
	} else {
		state.Prereq(luau.Create(&luau.Assignment{
			Left: luau.Create(&luau.ComputedIndexExpression{
				Expression: ptr.value.(luau.IndexableExpression),
				Index:      left,
			}),
			Operator: luau.AssignEqual,
			Right:    right,
		}))
	}
}

func disableMapInline(state *TransformState, ptr *mapPointer) {
	if luau.IsMap(ptr.value) {
		ptr.value = state.PushToVar(ptr.value, ptr.name)
	}
}

func disableArrayInline(state *TransformState, ptr *arrayPointer) {
	if luau.IsArray(ptr.value) {
		ptr.value = state.PushToVar(ptr.value, ptr.name)
	}
}
