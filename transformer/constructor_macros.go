package transformer

// Port of src/TSTransformer/macros/constructorMacros.ts.

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func wrapWeak(state *TransformState, node *ast.Node, macro constructorMacro) luau.Expression {
	return luau.NewCall(
		luau.Globals.Setmetatable,
		macro(state, node),
		luau.NewMap(luau.KeyValue{Index: luau.Strings.Mode, Value: luau.Strings.K}),
	)
}

func arrayConstructor(state *TransformState, node *ast.Node) luau.Expression {
	arguments := node.AsNewExpression().Arguments
	if arguments != nil && len(arguments.Nodes) > 0 {
		args := ensureTransformOrder(state, arguments.Nodes, nil)
		return luau.NewCall(luau.Globals.Table.Create, args...)
	}
	return luau.NewArray()
}

func setConstructor(state *TransformState, node *ast.Node) luau.Expression {
	arguments := node.AsNewExpression().Arguments
	if arguments == nil || len(arguments.Nodes) == 0 {
		return luau.NewSet()
	}
	arg := arguments.Nodes[0]
	// spreads cause prereq array, which cannot be optimised like this
	if ast.IsArrayLiteralExpression(arg) && !slices.ContainsFunc(arg.AsArrayLiteralExpression().Elements.Nodes, ast.IsSpreadElement) {
		return luau.NewSet(ensureTransformOrder(state, arg.AsArrayLiteralExpression().Elements.Nodes, nil)...)
	} else {
		id := state.PushToVar(luau.NewSet(), "set")
		valueID := luau.NewTempID("v")
		state.Prereq(luau.Create(&luau.ForStatement{
			IDs:        luau.MakeList[luau.AnyIdentifier](luau.NewTempID(""), valueID),
			Expression: transformExpression(state, arg),
			Statements: luau.MakeList[luau.Statement](
				luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: id,
						Index:      valueID,
					}),
					Operator: luau.AssignEqual,
					Right:    luau.NewBool(true),
				}),
			),
		}))
		return id
	}
}

func mapConstructor(state *TransformState, node *ast.Node) luau.Expression {
	arguments := node.AsNewExpression().Arguments
	if arguments == nil || len(arguments.Nodes) == 0 {
		return luau.NewMap()
	}
	arg := arguments.Nodes[0]
	transformed := transformExpression(state, arg)
	if luau.IsArray(transformed) && transformed.(*luau.Array).Members.Every(luau.IsArray) {
		members := transformed.(*luau.Array).Members.ToArray()
		elements := make([]luau.KeyValue, len(members))
		for i, e := range members {
			// non-null and type assertion because array will always have 2 members,
			// due to map constructor typing
			if !(luau.IsArray(e) && e.(*luau.Array).Members.IsNonEmpty()) {
				panic(shared.AssertionMessage(""))
			}
			head := e.(*luau.Array).Members.Head
			elements[i] = luau.KeyValue{Index: head.Value.(luau.Expression), Value: head.Next.Value.(luau.Expression)}
		}
		return luau.NewMap(elements...)
	} else {
		id := state.PushToVar(luau.NewMap(), "map")
		valueID := luau.NewTempID("v")
		state.Prereq(luau.Create(&luau.ForStatement{
			IDs:        luau.MakeList[luau.AnyIdentifier](luau.NewTempID(""), valueID),
			Expression: transformed,
			Statements: luau.MakeList[luau.Statement](
				luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: id,
						Index: luau.Create(&luau.ComputedIndexExpression{
							Expression: valueID,
							Index:      luau.NewNumber(1),
						}),
					}),
					Operator: luau.AssignEqual,
					Right: luau.Create(&luau.ComputedIndexExpression{
						Expression: valueID,
						Index:      luau.NewNumber(2),
					}),
				}),
			),
		}))
		return id
	}
}

// constructorMacros is CONSTRUCTOR_MACROS.
func constructorMacros() map[string]constructorMacro {
	return map[string]constructorMacro{
		"ArrayConstructor": arrayConstructor,
		"SetConstructor":   setConstructor,
		"MapConstructor":   mapConstructor,
		"WeakSetConstructor": func(state *TransformState, node *ast.Node) luau.Expression {
			return wrapWeak(state, node, setConstructor)
		},
		"WeakMapConstructor": func(state *TransformState, node *ast.Node) luau.Expression {
			return wrapWeak(state, node, mapConstructor)
		},
		"ReadonlyMapConstructor": mapConstructor,
		"ReadonlySetConstructor": setConstructor,
	}
}
