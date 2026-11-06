package transformer

// Port of src/TSTransformer/macros/propertyCallMacros.ts.
//
// Upstream's tables are nested by class and wrapped in place when the module loads. Here the lists are functions, so
// they don't cause an initialization cycle, and propertyCallMacros flattens them into one table keyed "Class.method".

import (
	"slices"
	"strconv"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func makeMathMethod(operator luau.BinaryOperator) propertyCallMacro {
	return func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		rhs := args[0]
		if !luau.IsSimple(rhs) {
			rhs = luau.Create(&luau.ParenthesizedExpression{Expression: rhs})
		}
		return luau.NewBinary(expression, operator, rhs)
	}
}

var operatorToNameMap = map[luau.BinaryOperator]string{
	luau.BinaryAdd:      "add",
	luau.BinarySub:      "sub",
	luau.BinaryMul:      "mul",
	luau.BinaryDiv:      "div",
	luau.BinaryFloorDiv: "idiv",
}

func makeMathSet(operators ...luau.BinaryOperator) map[string]propertyCallMacro {
	result := map[string]propertyCallMacro{}
	for _, operator := range operators {
		methodName := operatorToNameMap[operator]
		if methodName == "" {
			panic(shared.AssertionMessage(""))
		}
		result[methodName] = makeMathMethod(operator)
	}
	return result
}

func makeStringCallback(strCallback *luau.PropertyAccessExpression) propertyCallMacro {
	return func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		return luau.NewCall(strCallback, append([]luau.Expression{expression}, args...)...)
	}
}

func stringCallbacks() map[string]propertyCallMacro {
	return map[string]propertyCallMacro{
		"size": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.NewUnary(luau.UnaryLen, expression)
		},

		"byte":    makeStringCallback(luau.Globals.String.Byte),
		"find":    makeStringCallback(luau.Globals.String.Find),
		"format":  makeStringCallback(luau.Globals.String.Format),
		"gmatch":  makeStringCallback(luau.Globals.String.Gmatch),
		"gsub":    makeStringCallback(luau.Globals.String.Gsub),
		"lower":   makeStringCallback(luau.Globals.String.Lower),
		"match":   makeStringCallback(luau.Globals.String.Match),
		"rep":     makeStringCallback(luau.Globals.String.Rep),
		"reverse": makeStringCallback(luau.Globals.String.Reverse),
		"split":   makeStringCallback(luau.Globals.String.Split),
		"sub":     makeStringCallback(luau.Globals.String.Sub),
		"upper":   makeStringCallback(luau.Globals.String.Upper),
	}
}

func makeEveryOrSomeMethod(
	callbackArgsListMaker func(keyID *luau.TemporaryIdentifier, valueID *luau.TemporaryIdentifier, expression luau.Expression) []luau.Expression,
	initialState bool,
) propertyCallMacro {
	return func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		expression = state.PushToVarIfComplex(expression, "exp")

		resultID := state.PushToVar(luau.NewBool(initialState), "result")
		callbackID := state.PushToVarIfNonID(args[0], "callback")

		keyID := luau.NewTempID("k")
		valueID := luau.NewTempID("v")

		callCallback := luau.NewCall(callbackID, callbackArgsListMaker(keyID, valueID, expression)...)
		var condition luau.Expression = callCallback
		if initialState {
			condition = luau.NewUnary(luau.UnaryNot, callCallback)
		}
		state.Prereq(luau.Create(&luau.ForStatement{
			IDs:        luau.MakeList(keyID, valueID),
			Expression: expression,
			Statements: luau.MakeList[luau.Statement](
				luau.Create(&luau.IfStatement{
					Condition: condition,
					Statements: luau.MakeList[luau.Statement](
						luau.Create(&luau.Assignment{
							Left:     resultID,
							Operator: luau.AssignEqual,
							Right:    luau.NewBool(!initialState),
						}),
						luau.Create(&luau.BreakStatement{}),
					),
					ElseBody: luau.NewList(),
				}),
			),
		}))

		return resultID
	}
}

func makeEveryMethod(
	callbackArgsListMaker func(keyID *luau.TemporaryIdentifier, valueID *luau.TemporaryIdentifier, expression luau.Expression) []luau.Expression,
) propertyCallMacro {
	return makeEveryOrSomeMethod(callbackArgsListMaker, true)
}

func makeSomeMethod(
	callbackArgsListMaker func(keyID *luau.TemporaryIdentifier, valueID *luau.TemporaryIdentifier, expression luau.Expression) []luau.Expression,
) propertyCallMacro {
	return makeEveryOrSomeMethod(callbackArgsListMaker, false)
}

func argumentsWithDefaults(state *TransformState, args []luau.Expression, defaults []luau.Expression) []luau.Expression {
	// potentially nil arguments
	for i := 0; i < len(args); i++ {
		if !luau.IsSimplePrimitive(args[i]) {
			name := valueToIDStr(args[i])
			if name == "" {
				name = "arg" + strconv.Itoa(i)
			}
			args[i] = state.PushToVar(args[i], name)
			state.Prereq(luau.Create(&luau.IfStatement{
				Condition: luau.NewBinary(args[i], luau.BinaryEqual, luau.NewNil()),
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.Assignment{
						Left:     args[i],
						Operator: luau.AssignEqual,
						Right:    defaults[i],
					}),
				),
				ElseBody: luau.NewList(),
			}))
		}
	}

	// not specified
	for j := len(args); j < len(defaults); j++ {
		args = append(args, defaults[j])
	}

	return args
}

func arrayLikeMethods() map[string]propertyCallMacro {
	return map[string]propertyCallMacro{
		"size": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.NewUnary(luau.UnaryLen, expression)
		},
	}
}

func readonlyArrayMethods() map[string]propertyCallMacro {
	return map[string]propertyCallMacro{
		"isEmpty": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.NewBinary(luau.NewUnary(luau.UnaryLen, expression), luau.BinaryEqual, luau.NewNumber(0))
		},

		"join": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			args = argumentsWithDefaults(state, args, []luau.Expression{luau.Strings.Comma})
			indexType := state.Checker.GetIndexTypeOfType(state.GetType(node.Expression().Expression()), state.Checker.GetNumberType())

			// table.concat only works on string and number types, so call tostring() otherwise
			if indexType != nil && !isDefinitelyType(indexType, isStringType, isNumberType) {
				expression = state.PushToVarIfComplex(expression, "exp")
				id := state.PushToVar(luau.NewCall(luau.Globals.Table.Create, luau.NewUnary(luau.UnaryLen, expression)), "result")
				keyID := luau.NewTempID("k")
				valueID := luau.NewTempID("v")
				state.Prereq(luau.Create(&luau.ForStatement{
					IDs:        luau.MakeList(keyID, valueID),
					Expression: expression,
					Statements: luau.MakeList[luau.Statement](
						luau.Create(&luau.Assignment{
							Left: luau.Create(&luau.ComputedIndexExpression{
								Expression: id,
								Index:      keyID,
							}),
							Operator: luau.AssignEqual,
							Right:    luau.NewCall(luau.Globals.Tostring, valueID),
						}),
					),
				}))

				expression = id
			}

			return luau.NewCall(luau.Globals.Table.Concat, expression, args[0])
		},

		"move": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			moveArgs := []luau.Expression{expression, offset(args[0], 1), offset(args[1], 1), offset(args[2], 1)}
			if len(args) > 3 {
				moveArgs = append(moveArgs, args[3])
			}
			return luau.NewCall(luau.Globals.Table.Move, moveArgs...)
		},

		"includes": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			callArgs := []luau.Expression{expression, args[0]}
			if len(args) > 1 {
				callArgs = append(callArgs, offset(args[1], 1))
			}
			return luau.NewBinary(luau.NewCall(luau.Globals.Table.Find, callArgs...), luau.BinaryNotEqual, luau.NewNil())
		},

		"indexOf": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			findArgs := []luau.Expression{expression, args[0]}

			if len(args) > 1 {
				findArgs = append(findArgs, offset(args[1], 1))
			}

			return offset(
				luau.Create(&luau.BinaryExpression{
					Left:     luau.NewCall(luau.Globals.Table.Find, findArgs...),
					Operator: luau.BinaryOr,
					Right:    luau.NewNumber(0),
				}),
				-1,
			)
		},

		"every": makeEveryMethod(func(keyID *luau.TemporaryIdentifier, valueID *luau.TemporaryIdentifier, expression luau.Expression) []luau.Expression {
			return []luau.Expression{valueID, offset(keyID, -1), expression}
		}),

		"some": makeSomeMethod(func(keyID *luau.TemporaryIdentifier, valueID *luau.TemporaryIdentifier, expression luau.Expression) []luau.Expression {
			return []luau.Expression{valueID, offset(keyID, -1), expression}
		}),

		"forEach": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")

			callbackID := state.PushToVarIfNonID(args[0], "callback")
			keyID := luau.NewTempID("k")
			valueID := luau.NewTempID("v")
			state.Prereq(luau.Create(&luau.ForStatement{
				IDs:        luau.MakeList(keyID, valueID),
				Expression: expression,
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.CallStatement{
						Expression: luau.NewCall(callbackID, valueID, offset(keyID, -1), expression),
					}),
				),
			}))

			if !isUsedAsStatement(node) {
				return luau.NewNil()
			}
			return luau.NewNone()
		},

		"map": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")
			newValueID := state.PushToVar(
				luau.NewCall(luau.Globals.Table.Create, luau.NewUnary(luau.UnaryLen, expression)),
				"newValue",
			)
			callbackID := state.PushToVarIfNonID(args[0], "callback")
			keyID := luau.NewTempID("k")
			valueID := luau.NewTempID("v")
			state.Prereq(luau.Create(&luau.ForStatement{
				IDs:        luau.MakeList(keyID, valueID),
				Expression: expression,
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.Assignment{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: newValueID,
							Index:      keyID,
						}),
						Operator: luau.AssignEqual,
						Right:    luau.NewCall(callbackID, valueID, offset(keyID, -1), expression),
					}),
				),
			}))

			return newValueID
		},

		"mapFiltered": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")

			newValueID := state.PushToVar(luau.NewArray(), "newValue")
			callbackID := state.PushToVarIfNonID(args[0], "callback")
			lengthID := state.PushToVar(luau.NewNumber(0), "length")
			keyID := luau.NewTempID("k")
			valueID := luau.NewTempID("v")
			resultID := luau.NewTempID("result")
			state.Prereq(luau.Create(&luau.ForStatement{
				IDs:        luau.MakeList(keyID, valueID),
				Expression: expression,
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.VariableDeclaration{
						Left:  resultID,
						Right: luau.NewCall(callbackID, valueID, offset(keyID, -1), expression),
					}),
					luau.Create(&luau.IfStatement{
						Condition: luau.NewBinary(resultID, luau.BinaryNotEqual, luau.NewNil()),
						Statements: luau.MakeList[luau.Statement](
							luau.Create(&luau.Assignment{
								Left:     lengthID,
								Operator: luau.AssignAdd,
								Right:    luau.NewNumber(1),
							}),
							luau.Create(&luau.Assignment{
								Left: luau.Create(&luau.ComputedIndexExpression{
									Expression: newValueID,
									Index:      lengthID,
								}),
								Operator: luau.AssignEqual,
								Right:    resultID,
							}),
						),
						ElseBody: luau.NewList(),
					}),
				),
			}))

			return newValueID
		},

		"filterUndefined": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")

			lengthID := state.PushToVar(luau.NewNumber(0), "length")
			indexID1 := luau.NewTempID("i")
			state.Prereq(luau.Create(&luau.ForStatement{
				IDs:        luau.MakeList(indexID1),
				Expression: expression,
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.IfStatement{
						Condition: luau.NewBinary(indexID1, luau.BinaryGreater, lengthID),
						Statements: luau.MakeList[luau.Statement](
							luau.Create(&luau.Assignment{
								Left:     lengthID,
								Operator: luau.AssignEqual,
								Right:    indexID1,
							}),
						),
						ElseBody: luau.NewList(),
					}),
				),
			}))

			resultID := state.PushToVar(luau.NewArray(), "result")
			resultLengthID := state.PushToVar(luau.NewNumber(0), "resultLength")
			indexID2 := luau.NewTempID("i")
			valueID := luau.NewTempID("v")
			state.Prereq(luau.Create(&luau.NumericForStatement{
				ID:    indexID2,
				Start: luau.NewNumber(1),
				End:   lengthID,

				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.VariableDeclaration{
						Left: valueID,
						Right: luau.Create(&luau.ComputedIndexExpression{
							Expression: convertToIndexableExpression(expression),
							Index:      indexID2,
						}),
					}),
					luau.Create(&luau.IfStatement{
						Condition: luau.NewBinary(valueID, luau.BinaryNotEqual, luau.NewNil()),
						Statements: luau.MakeList[luau.Statement](
							luau.Create(&luau.Assignment{
								Left:     resultLengthID,
								Operator: luau.AssignAdd,
								Right:    luau.NewNumber(1),
							}),
							luau.Create(&luau.Assignment{
								Left: luau.Create(&luau.ComputedIndexExpression{
									Expression: resultID,
									Index:      resultLengthID,
								}),
								Operator: luau.AssignEqual,
								Right:    valueID,
							}),
						),
						ElseBody: luau.NewList(),
					}),
				),
			}))

			return resultID
		},

		"filter": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")

			newValueID := state.PushToVar(luau.NewArray(), "newValue")
			callbackID := state.PushToVarIfNonID(args[0], "callback")
			lengthID := state.PushToVar(luau.NewNumber(0), "length")
			keyID := luau.NewTempID("k")
			valueID := luau.NewTempID("v")
			state.Prereq(luau.Create(&luau.ForStatement{
				IDs:        luau.MakeList(keyID, valueID),
				Expression: expression,
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.IfStatement{
						Condition: luau.Create(&luau.BinaryExpression{
							Left:     luau.NewCall(callbackID, valueID, offset(keyID, -1), expression),
							Operator: luau.BinaryEqual,
							Right:    luau.NewBool(true),
						}),
						Statements: luau.MakeList[luau.Statement](
							luau.Create(&luau.Assignment{
								Left:     lengthID,
								Operator: luau.AssignAdd,
								Right:    luau.NewNumber(1),
							}),
							luau.Create(&luau.Assignment{
								Left: luau.Create(&luau.ComputedIndexExpression{
									Expression: newValueID,
									Index:      lengthID,
								}),
								Operator: luau.AssignEqual,
								Right:    valueID,
							}),
						),
						ElseBody: luau.NewList(),
					}),
				),
			}))

			return newValueID
		},

		"reduce": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")

			var start luau.Expression = luau.NewNumber(1)
			end := luau.NewUnary(luau.UnaryLen, expression)
			const step = 1

			lengthExp := luau.NewUnary(luau.UnaryLen, expression)

			var resultID *luau.TemporaryIdentifier
			// if there was no initialValue supplied
			if len(args) < 2 {
				state.Prereq(luau.Create(&luau.IfStatement{
					Condition: luau.NewBinary(lengthExp, luau.BinaryEqual, luau.NewNumber(0)),
					Statements: luau.MakeList[luau.Statement](
						luau.Create(&luau.CallStatement{
							Expression: luau.NewCall(luau.Globals.Error,
								luau.NewString(
									"Attempted to call `ReadonlyArray.reduce()` on an empty array without an initialValue.",
								),
							),
						}),
					),
					ElseBody: luau.NewList(),
				}))
				resultID = state.PushToVar(
					luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(expression),
						Index:      start,
					}),
					"result",
				)
				start = offset(start, step)
			} else {
				resultID = state.PushToVar(args[1], "result")
			}
			callbackID := state.PushToVar(args[0], "callback")

			iteratorID := luau.NewTempID("i")
			var stepExpression luau.Expression
			if step != 1 {
				stepExpression = luau.NewNumber(step)
			}
			state.Prereq(luau.Create(&luau.NumericForStatement{
				ID:    iteratorID,
				Start: start,
				End:   end,
				Step:  stepExpression,
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.Assignment{
						Left:     resultID,
						Operator: luau.AssignEqual,
						Right: luau.NewCall(callbackID,
							resultID,
							luau.Create(&luau.ComputedIndexExpression{
								Expression: convertToIndexableExpression(expression),
								Index:      iteratorID,
							}),
							offset(iteratorID, -1),
							expression,
						),
					}),
				),
			}))

			return resultID
		},

		"find": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")

			callbackID := state.PushToVarIfNonID(args[0], "callback")
			loopID := luau.NewTempID("i")
			valueID := luau.NewTempID("v")
			resultID := state.PushToVar(nil, "result")

			state.Prereq(luau.Create(&luau.ForStatement{
				Expression: expression,
				IDs:        luau.MakeList(loopID, valueID),
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.IfStatement{
						Condition: luau.Create(&luau.BinaryExpression{
							Left:     luau.NewCall(callbackID, valueID, offset(loopID, -1), expression),
							Operator: luau.BinaryEqual,
							Right:    luau.NewBool(true),
						}),
						Statements: luau.MakeList[luau.Statement](
							luau.Create(&luau.Assignment{
								Left:     resultID,
								Operator: luau.AssignEqual,
								Right:    valueID,
							}),
							luau.Create(&luau.BreakStatement{}),
						),
						ElseBody: luau.NewList(),
					}),
				),
			}))

			return resultID
		},

		"findIndex": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")

			callbackID := state.PushToVarIfNonID(args[0], "callback")
			loopID := luau.NewTempID("i")
			valueID := luau.NewTempID("v")
			resultID := state.PushToVar(luau.NewNumber(-1), "result")

			state.Prereq(luau.Create(&luau.ForStatement{
				Expression: expression,
				IDs:        luau.MakeList(loopID, valueID),
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.IfStatement{
						Condition: luau.Create(&luau.BinaryExpression{
							Left:     luau.NewCall(callbackID, valueID, offset(loopID, -1), expression),
							Operator: luau.BinaryEqual,
							Right:    luau.NewBool(true),
						}),
						Statements: luau.MakeList[luau.Statement](
							luau.Create(&luau.Assignment{
								Left:     resultID,
								Operator: luau.AssignEqual,
								Right:    offset(loopID, -1),
							}),
							luau.Create(&luau.BreakStatement{}),
						),
						ElseBody: luau.NewList(),
					}),
				),
			}))

			return resultID
		},
	}
}

func arrayMethods() map[string]propertyCallMacro {
	return map[string]propertyCallMacro{
		"push": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			// for `a.push()` always emit luau.unary so the call doesn't disappear in emit
			if len(args) == 0 {
				return luau.NewUnary(luau.UnaryLen, expression)
			}

			expression = state.PushToVarIfComplex(expression, "exp")

			for i := 0; i < len(args); i++ {
				state.Prereq(luau.Create(&luau.CallStatement{
					Expression: luau.NewCall(luau.Globals.Table.Insert, expression, args[i]),
				}))
			}

			if !isUsedAsStatement(node) {
				return luau.NewUnary(luau.UnaryLen, expression)
			}
			return luau.NewNone()
		},

		"pop": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")

			var lengthExp luau.Expression = luau.NewUnary(luau.UnaryLen, expression)

			returnValueIsUsed := !isUsedAsStatement(node)
			var retValue *luau.TemporaryIdentifier
			if returnValueIsUsed {
				lengthExp = state.PushToVar(lengthExp, "length")
				retValue = state.PushToVar(
					luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(expression),
						Index:      lengthExp,
					}),
					"result",
				)
			}

			state.Prereq(luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: convertToIndexableExpression(expression),
					Index:      lengthExp,
				}),
				Operator: luau.AssignEqual,
				Right:    luau.NewNil(),
			}))

			if returnValueIsUsed {
				return retValue
			}
			return luau.NewNone()
		},

		"shift": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.NewCall(luau.Globals.Table.Remove, expression, luau.NewNumber(1))
		},

		"unshift": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			expression = state.PushToVarIfComplex(expression, "exp")

			for i := len(args) - 1; i >= 0; i-- {
				arg := args[i]
				state.Prereq(luau.Create(&luau.CallStatement{
					Expression: luau.NewCall(luau.Globals.Table.Insert, expression, luau.NewNumber(1), arg),
				}))
			}

			if !isUsedAsStatement(node) {
				return luau.NewUnary(luau.UnaryLen, expression)
			}
			return luau.NewNone()
		},

		"insert": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.NewCall(luau.Globals.Table.Insert, expression, offset(args[0], 1), args[1])
		},

		"remove": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.NewCall(luau.Globals.Table.Remove, expression, offset(args[0], 1))
		},

		"unorderedRemove": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			indexExp := state.PushToVarIfComplex(offset(args[0], 1), "index")

			expression = state.PushToVarIfComplex(expression, "exp")

			lengthID := state.PushToVar(luau.NewUnary(luau.UnaryLen, expression), "length")

			valueIsUsed := !isUsedAsStatement(node)
			valueID := state.PushToVar(
				luau.Create(&luau.ComputedIndexExpression{
					Expression: convertToIndexableExpression(expression),
					Index:      indexExp,
				}),
				"value",
			)

			state.Prereq(luau.Create(&luau.IfStatement{
				Condition: luau.NewBinary(valueID, luau.BinaryNotEqual, luau.NewNil()),
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.Assignment{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: convertToIndexableExpression(expression),
							Index:      indexExp,
						}),
						Operator: luau.AssignEqual,
						Right: luau.Create(&luau.ComputedIndexExpression{
							Expression: convertToIndexableExpression(expression),
							Index:      lengthID,
						}),
					}),
					luau.Create(&luau.Assignment{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: convertToIndexableExpression(expression),
							Index:      lengthID,
						}),
						Operator: luau.AssignEqual,
						Right:    luau.NewNil(),
					}),
				),
				ElseBody: luau.NewList(),
			}))

			if valueIsUsed {
				return valueID
			}
			return luau.NewNone()
		},

		"sort": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			valueIsUsed := !isUsedAsStatement(node)
			if valueIsUsed {
				expression = state.PushToVarIfComplex(expression, "exp")
			}

			args = slices.Insert(args, 0, expression)

			state.Prereq(luau.Create(&luau.CallStatement{
				Expression: luau.NewCall(luau.Globals.Table.Sort, args...),
			}))

			if valueIsUsed {
				return expression
			}
			return luau.NewNone()
		},

		"clear": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			state.Prereq(luau.Create(&luau.CallStatement{
				Expression: luau.NewCall(luau.Globals.Table.Clear, expression),
			}))
			if !isUsedAsStatement(node) {
				return luau.NewNil()
			}
			return luau.NewNone()
		},
	}
}

func readonlySetMapSharedMethods() map[string]propertyCallMacro {
	return map[string]propertyCallMacro{
		"isEmpty": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.NewBinary(luau.NewCall(luau.Globals.Next, expression), luau.BinaryEqual, luau.NewNil())
		},

		"size": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			sizeID := state.PushToVar(luau.NewNumber(0), "size")
			state.Prereq(luau.Create(&luau.ForStatement{
				IDs:        luau.MakeList(luau.NewTempID("")),
				Expression: expression,
				Statements: luau.MakeList[luau.Statement](
					luau.Create(&luau.Assignment{
						Left:     sizeID,
						Operator: luau.AssignAdd,
						Right:    luau.NewNumber(1),
					}),
				),
			}))
			return sizeID
		},

		"has": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			left := luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(expression),
				Index:      args[0],
			})
			return luau.NewBinary(left, luau.BinaryNotEqual, luau.NewNil())
		},
	}
}

func setMapSharedMethods() map[string]propertyCallMacro {
	return map[string]propertyCallMacro{
		"delete": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			arg := state.PushToVarIfComplex(args[0], "value")
			valueIsUsed := !isUsedAsStatement(node)
			var valueExistedID *luau.TemporaryIdentifier
			if valueIsUsed {
				expressionID := state.PushToVarIfNonID(expression, "exp")
				expression = expressionID
				valueExistedID = state.PushToVar(
					luau.Create(&luau.BinaryExpression{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: expressionID,
							Index:      arg,
						}),
						Operator: luau.BinaryNotEqual,
						Right:    luau.NewNil(),
					}),
					"valueExisted",
				)
			}

			state.Prereq(luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: convertToIndexableExpression(expression),
					Index:      arg,
				}),
				Operator: luau.AssignEqual,
				Right:    luau.NewNil(),
			}))

			if valueIsUsed {
				return valueExistedID
			}
			return luau.NewNone()
		},

		"clear": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			state.Prereq(luau.Create(&luau.CallStatement{
				Expression: luau.NewCall(luau.Globals.Table.Clear, expression),
			}))
			if !isUsedAsStatement(node) {
				return luau.NewNil()
			}
			return luau.NewNone()
		},
	}
}

func readonlySetMethods() map[string]propertyCallMacro {
	methods := readonlySetMapSharedMethods()

	methods["forEach"] = func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		expression = state.PushToVarIfComplex(expression, "exp")

		callbackID := state.PushToVarIfNonID(args[0], "callback")
		valueID := luau.NewTempID("v")
		state.Prereq(luau.Create(&luau.ForStatement{
			IDs:        luau.MakeList(valueID),
			Expression: expression,
			Statements: luau.MakeList[luau.Statement](
				luau.Create(&luau.CallStatement{
					Expression: luau.NewCall(callbackID, valueID, valueID, expression),
				}),
			),
		}))

		if !isUsedAsStatement(node) {
			return luau.NewNil()
		}
		return luau.NewNone()
	}

	return methods
}

func setMethods() map[string]propertyCallMacro {
	methods := setMapSharedMethods()

	methods["add"] = func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		valueIsUsed := !isUsedAsStatement(node)
		if valueIsUsed {
			expression = state.PushToVarIfComplex(expression, "exp")
		}
		state.Prereq(luau.Create(&luau.Assignment{
			Left: luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(expression),
				Index:      args[0],
			}),
			Operator: luau.AssignEqual,
			Right:    luau.NewBool(true),
		}))
		if valueIsUsed {
			return expression
		}
		return luau.NewNone()
	}

	return methods
}

func readonlyMapMethods() map[string]propertyCallMacro {
	methods := readonlySetMapSharedMethods()

	methods["forEach"] = func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		expression = state.PushToVarIfComplex(expression, "exp")

		callbackID := state.PushToVarIfNonID(args[0], "callback")
		keyID := luau.NewTempID("k")
		valueID := luau.NewTempID("v")
		state.Prereq(luau.Create(&luau.ForStatement{
			IDs:        luau.MakeList(keyID, valueID),
			Expression: expression,
			Statements: luau.MakeList[luau.Statement](
				luau.Create(&luau.CallStatement{
					Expression: luau.NewCall(callbackID, valueID, keyID, expression),
				}),
			),
		}))

		if !isUsedAsStatement(node) {
			return luau.NewNil()
		}
		return luau.NewNone()
	}

	methods["get"] = func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		return luau.Create(&luau.ComputedIndexExpression{
			Expression: convertToIndexableExpression(expression),
			Index:      args[0],
		})
	}

	return methods
}

func mapMethods() map[string]propertyCallMacro {
	methods := setMapSharedMethods()

	methods["set"] = func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		keyExp, valueExp := args[0], args[1]
		valueIsUsed := !isUsedAsStatement(node)
		if valueIsUsed {
			expression = state.PushToVarIfComplex(expression, "exp")
		}
		state.Prereq(luau.Create(&luau.Assignment{
			Left: luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(expression),
				Index:      keyExp,
			}),
			Operator: luau.AssignEqual,
			Right:    valueExp,
		}))
		if valueIsUsed {
			return expression
		}
		return luau.NewNone()
	}

	return methods
}

func promiseMethods() map[string]propertyCallMacro {
	return map[string]propertyCallMacro{
		"then": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Create(&luau.MethodCallExpression{
				Expression: convertToIndexableExpression(expression),
				Name:       "andThen",
				Args:       luau.MakeList(args...),
			})
		},
	}
}

// propertyCallMacros is PROPERTY_CALL_MACROS with the comment wrapping applied, flattened to one table.
func propertyCallMacros() map[string]propertyCallMacro {
	classes := map[string]map[string]propertyCallMacro{
		// math classes
		"CFrame":       makeMathSet(luau.BinaryAdd, luau.BinarySub, luau.BinaryMul),
		"UDim":         makeMathSet(luau.BinaryAdd, luau.BinarySub),
		"UDim2":        makeMathSet(luau.BinaryAdd, luau.BinarySub),
		"Vector2":      makeMathSet(luau.BinaryAdd, luau.BinarySub, luau.BinaryMul, luau.BinaryDiv, luau.BinaryFloorDiv),
		"Vector2int16": makeMathSet(luau.BinaryAdd, luau.BinarySub, luau.BinaryMul, luau.BinaryDiv),
		"Vector3":      makeMathSet(luau.BinaryAdd, luau.BinarySub, luau.BinaryMul, luau.BinaryDiv, luau.BinaryFloorDiv),
		"Vector3int16": makeMathSet(luau.BinaryAdd, luau.BinarySub, luau.BinaryMul, luau.BinaryDiv),
		"Number":       makeMathSet(luau.BinaryFloorDiv),

		"String":        stringCallbacks(),
		"ArrayLike":     arrayLikeMethods(),
		"ReadonlyArray": readonlyArrayMethods(),
		"Array":         arrayMethods(),
		"ReadonlySet":   readonlySetMethods(),
		"Set":           setMethods(),
		"ReadonlyMap":   readonlyMapMethods(),
		"Map":           mapMethods(),
		"Promise":       promiseMethods(),
	}

	// apply comment wrapping
	macros := map[string]propertyCallMacro{}
	for className, macroList := range classes {
		for methodName, macro := range macroList {
			name := className + "." + methodName
			macros[name] = wrapComments(name, macro)
		}
	}
	return macros
}

// comment logic

func header(text string) *luau.Comment {
	return luau.NewComment(" ▼ " + text + " ▼")
}

func footer(text string) *luau.Comment {
	return luau.NewComment(" ▲ " + text + " ▲")
}

func wasExpressionPushed(statements *luau.List, expression luau.Expression) bool {
	if statements.IsNonEmpty() {
		firstStatement := statements.Head.Value
		if luau.IsVariableDeclaration(firstStatement) {
			declaration := firstStatement.(*luau.VariableDeclaration)
			if !luau.IsList(declaration.Left) && luau.IsTemporaryIdentifier(declaration.Left.(luau.Node)) {
				if declaration.Right == luau.NodeOrList(expression) {
					return true
				}
			}
		}
	}
	return false
}

func wrapComments(methodName string, callback propertyCallMacro) propertyCallMacro {
	return func(state *TransformState, callNode *ast.Node, callExp luau.Expression, args []luau.Expression) luau.Expression {
		expression, prereqs := Capture(state, func() luau.Expression { return callback(state, callNode, callExp, args) })

		size := prereqs.Size()
		if size > 0 {
			// detect the case of `expression = state.pushToVarIfComplex(expression, "exp");` and put header after
			wasPushed := wasExpressionPushed(prereqs, callExp)
			var pushStatement luau.Node
			if wasPushed {
				pushStatement = prereqs.Shift()
				size--
			}
			if size > 1 {
				prereqs.Unshift(header(methodName))
				if wasPushed && pushStatement != nil {
					prereqs.Unshift(pushStatement)
				}
				prereqs.Push(footer(methodName))
			} else {
				if wasPushed && pushStatement != nil {
					prereqs.Unshift(pushStatement)
				}
			}
		}

		state.PrereqList(prereqs)
		return expression
	}
}
