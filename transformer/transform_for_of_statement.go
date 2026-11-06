package transformer

// Port of src/TSTransformer/nodes/statements/transformForOfStatement.ts.
//
// The builders are functions and not package-level variables as upstream, because they'd be an initialization cycle.
//
// TS 7 parses an omitted element of an array binding pattern as a BindingElement with no name, where TS 5.5.3 has an
// OmittedExpression, so an omitted element is also Name() == nil in transformInLineArrayBindingPattern.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

type loopBuilder func(state *TransformState, statements *luau.List, initializer *ast.Node, exp luau.Expression) *luau.List

func makeForLoopBuilder(callback func(state *TransformState, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression) loopBuilder {
	return func(state *TransformState, statements *luau.List, name *ast.Node, exp luau.Expression) *luau.List {
		ids := luau.NewList()
		initializers := luau.NewList()
		expression := callback(state, name, exp, ids, initializers)
		statements.UnshiftList(initializers)
		return luau.MakeList(luau.Create(&luau.ForStatement{IDs: ids, Expression: expression, Statements: statements}))
	}
}

func transformForInitializerExpressionDirect(state *TransformState, initializer *ast.Node, initializers *luau.List, value luau.Expression) luau.AnyIdentifier {
	if ast.IsArrayLiteralExpression(initializer) {
		parentID, prereqs := Capture(state, func() *luau.TemporaryIdentifier {
			parentID := state.PushToVar(value, "binding")
			transformArrayAssignmentPattern(state, initializer, parentID)
			return parentID
		})
		initializers.PushList(prereqs)
		return parentID
	} else if ast.IsObjectLiteralExpression(initializer) {
		parentID, prereqs := Capture(state, func() *luau.TemporaryIdentifier {
			parentID := state.PushToVar(value, "binding")
			transformObjectAssignmentPattern(state, initializer, parentID)
			return parentID
		})
		initializers.PushList(prereqs)
		return parentID
	} else {
		expression := transformWritableExpression(state, initializer, false)
		initializers.Push(luau.Create(&luau.Assignment{
			Left:     expression,
			Operator: luau.AssignEqual,
			Right:    value,
		}))
	}
	return nil
}

func transformForInitializer(state *TransformState, initializer *ast.Node, initializers *luau.List) luau.AnyIdentifier {
	if ast.IsVariableDeclarationList(initializer) {
		return transformBindingName(state, initializer.AsVariableDeclarationList().Declarations.Nodes[0].Name(), initializers)
	} else if ast.IsArrayLiteralExpression(initializer) {
		parentID := luau.NewTempID("binding")
		initializers.PushList(
			state.CapturePrereqs(func() { transformArrayAssignmentPattern(state, initializer, parentID) }),
		)
		return parentID
	} else if ast.IsObjectLiteralExpression(initializer) {
		parentID := luau.NewTempID("binding")
		initializers.PushList(
			state.CapturePrereqs(func() { transformObjectAssignmentPattern(state, initializer, parentID) }),
		)
		return parentID
	} else {
		valueID := luau.NewTempID("v")
		expression := transformWritableExpression(state, initializer, false)
		initializers.Push(luau.Create(&luau.Assignment{
			Left:     expression,
			Operator: luau.AssignEqual,
			Right:    valueID,
		}))
		return valueID
	}
}

func buildArrayLoop(state *TransformState, statements *luau.List, name *ast.Node, exp luau.Expression) *luau.List {
	return makeForLoopBuilder(func(state *TransformState, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
		ids.Push(luau.NewTempID(""))
		ids.Push(transformForInitializer(state, initializer, initializers))
		return exp
	})(state, statements, name, exp)
}

func buildSetLoop(state *TransformState, statements *luau.List, name *ast.Node, exp luau.Expression) *luau.List {
	return makeForLoopBuilder(func(state *TransformState, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
		ids.Push(transformForInitializer(state, initializer, initializers))
		return exp
	})(state, statements, name, exp)
}

func transformInLineArrayBindingPattern(state *TransformState, pattern *ast.Node, ids *luau.List, initializers *luau.List) {
	for _, element := range pattern.AsBindingPattern().Elements.Nodes {
		if ast.IsOmittedExpression(element) || element.Name() == nil {
			ids.Push(luau.NewTempID(""))
		} else if ast.IsSpreadElement(element) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoSpreadDestructuring.At(shared.AtNode(element)))
		} else {
			id := transformBindingName(state, element.Name(), initializers)
			if elementInitializer := element.AsBindingElement().Initializer; elementInitializer != nil {
				initializers.Push(transformInitializer(state, id.(luau.WritableExpression), elementInitializer))
			}
			ids.Push(id)
		}
	}
}

func transformInLineArrayAssignmentPattern(state *TransformState, assignmentPattern *ast.Node, ids *luau.List, initializers *luau.List) {
	initializers.PushList(
		state.CapturePrereqs(func() {
			for _, element := range assignmentPattern.AsArrayLiteralExpression().Elements.Nodes {
				if ast.IsOmittedExpression(element) {
					ids.Push(luau.NewTempID(""))
				} else if ast.IsSpreadElement(element) {
					state.Diagnostic.AddDiagnostic(shared.Errors.NoSpreadDestructuring.At(shared.AtNode(element)))
				} else {
					var initializer *ast.Node
					if ast.IsBinaryExpression(element) {
						binary := element.AsBinaryExpression()
						initializer = SkipDownwards(binary.Right)
						element = SkipDownwards(binary.Left)
					}

					valueID := luau.NewTempID("binding")
					if ast.IsIdentifier(element) ||
						ast.IsElementAccessExpression(element) ||
						ast.IsPropertyAccessExpression(element) {
						id := transformWritableExpression(state, element, initializer != nil)
						state.Prereq(luau.Create(&luau.Assignment{
							Left:     id,
							Operator: luau.AssignEqual,
							Right:    valueID,
						}))
						if initializer != nil {
							state.Prereq(transformInitializer(state, id, initializer))
						}
					} else if ast.IsArrayLiteralExpression(element) {
						if initializer != nil {
							state.Prereq(transformInitializer(state, valueID, initializer))
						}
						transformArrayAssignmentPattern(state, element, valueID)
					} else if ast.IsObjectLiteralExpression(element) {
						if initializer != nil {
							state.Prereq(transformInitializer(state, valueID, initializer))
						}
						transformObjectAssignmentPattern(state, element, valueID)
					} else {
						panic(shared.AssertionMessage("transformInLineArrayAssignmentPattern invalid element: " + getKindName(element.Kind)))
					}

					ids.Push(valueID)
				}
			}
		}),
	)
}

func buildMapLoop(state *TransformState, statements *luau.List, name *ast.Node, exp luau.Expression) *luau.List {
	return makeForLoopBuilder(func(state *TransformState, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
		// TEST
		if ast.IsVariableDeclarationList(initializer) {
			name := initializer.AsVariableDeclarationList().Declarations.Nodes[0].Name()
			if ast.IsArrayBindingPattern(name) {
				transformInLineArrayBindingPattern(state, name, ids, initializers)
				return exp
			}
		} else if ast.IsArrayLiteralExpression(initializer) {
			transformInLineArrayAssignmentPattern(state, initializer, ids, initializers)
			return exp
		}

		keyID := luau.NewTempID("k")
		valueID := luau.NewTempID("v")
		ids.Push(keyID)
		ids.Push(valueID)

		if ast.IsVariableDeclarationList(initializer) {
			bindingList := luau.NewList()
			initializers.Push(luau.Create(&luau.VariableDeclaration{
				Left:  transformForInitializer(state, initializer, bindingList),
				Right: luau.NewArray(keyID, valueID),
			}))
			initializers.PushList(bindingList)
		} else {
			transformForInitializerExpressionDirect(state, initializer, initializers, luau.NewArray(keyID, valueID))
		}

		return exp
	})(state, statements, name, exp)
}

func buildStringLoop(state *TransformState, statements *luau.List, name *ast.Node, exp luau.Expression) *luau.List {
	return makeForLoopBuilder(func(state *TransformState, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
		ids.Push(transformForInitializer(state, initializer, initializers))
		return luau.NewCall(luau.Globals.String.Gmatch, exp, luau.Globals.Utf8.Charpattern)
	})(state, statements, name, exp)
}

func buildIterableFunctionLoop(state *TransformState, statements *luau.List, name *ast.Node, exp luau.Expression) *luau.List {
	return makeForLoopBuilder(func(state *TransformState, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
		ids.Push(transformForInitializer(state, initializer, initializers))
		return exp
	})(state, statements, name, exp)
}

func makeIterableFunctionLuaTupleShorthand(state *TransformState, array *ast.Node, statements *luau.List, expression luau.Expression) *luau.List {
	ids := luau.NewList()
	initializers := luau.NewList()
	if ast.IsArrayBindingPattern(array) {
		transformInLineArrayBindingPattern(state, array, ids, initializers)
	} else {
		transformInLineArrayAssignmentPattern(state, array, ids, initializers)
	}
	statements.UnshiftList(initializers)
	return luau.MakeList(luau.Create(&luau.ForStatement{IDs: ids, Expression: expression, Statements: statements}))
}

func buildIterableFunctionLuaTupleLoop(t *checker.Type) loopBuilder {
	return func(state *TransformState, statements *luau.List, initializer *ast.Node, exp luau.Expression) *luau.List {
		if ast.IsVariableDeclarationList(initializer) {
			// for (const [a, b] of iter())
			name := initializer.AsVariableDeclarationList().Declarations.Nodes[0].Name()
			if ast.IsArrayBindingPattern(name) {
				return makeIterableFunctionLuaTupleShorthand(state, name, statements, exp)
			}
		} else if ast.IsArrayLiteralExpression(initializer) {
			// for ([a, b] of iter())
			return makeIterableFunctionLuaTupleShorthand(state, initializer, statements, exp)
		}

		var iteratorReturnIDs []luau.Expression

		// get call signature of IterableFunction<T> which is `(): T`
		// and get return type of call signature which is `T`
		luaTupleType := state.Checker.GetReturnTypeOfSignature(state.Checker.GetCallSignatures(t)[0])
		if !(luaTupleType != nil && luaTupleType.Alias() != nil && len(luaTupleType.Alias().TypeArguments()) == 1) {
			panic(shared.AssertionMessage("Incorrect LuaTuple<T> type arguments"))
		}
		tupleArgType := luaTupleType.Alias().TypeArguments()[0]
		// if initializer is a variable declaration `for (const a of iter())`
		// and LuaTuple has defined element amount, and no rest elements
		// then use lua for-in loop, specifying all elements and putting them in table
		if ast.IsVariableDeclarationList(initializer) &&
			checker.IsTupleType(tupleArgType) &&
			!tupleHasRestElement(tupleArgType) {
			tupleType := tupleArgType.TargetTupleType()
			elementInfos := tupleType.ElementInfos()
			for i := range elementInfos {
				name := "element"
				label := elementInfos[i].LabeledDeclaration()
				if label != nil && ast.IsIdentifier(label.Name()) && luau.IsValidIdentifier(label.Name().Text()) {
					name = label.Name().Text()
				}
				iteratorReturnIDs = append(iteratorReturnIDs, luau.NewTempID(name))
			}
		} else {
			iterFuncName := valueToIDStr(exp)
			if iterFuncName == "" {
				iterFuncName = "iterFunc"
			}
			iterFuncID := state.PushToVar(exp, iterFuncName)
			loopStatements := luau.NewList()

			initializerStatements := luau.NewList()
			valueID := transformForInitializer(state, initializer, initializerStatements)

			loopStatements.Push(luau.Create(&luau.VariableDeclaration{
				Left:  valueID,
				Right: luau.NewArray(luau.NewCall(iterFuncID)),
			}))

			loopStatements.Push(luau.Create(&luau.IfStatement{
				Condition:  luau.NewBinary(luau.NewUnary(luau.UnaryLen, valueID), luau.BinaryEqual, luau.NewNumber(0)),
				Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				ElseBody:   luau.NewList(),
			}))

			loopStatements.PushList(initializerStatements)

			loopStatements.PushList(statements)

			return luau.MakeList(luau.Create(&luau.WhileStatement{
				Condition:  luau.NewBool(true),
				Statements: loopStatements,
			}))
		}

		tupleID := transformForInitializer(state, initializer, statements)

		builder := makeForLoopBuilder(func(state *TransformState, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
			for _, id := range iteratorReturnIDs {
				ids.Push(id)
			}

			initializers.Push(luau.Create(&luau.VariableDeclaration{
				Left:  tupleID,
				Right: luau.NewArray(iteratorReturnIDs...),
			}))
			return exp
		})

		return builder(state, statements, initializer, exp)
	}
}

// tupleHasRestElement is `target.combinedFlags & ts.ElementFlags.Rest`. combinedFlags isn't exported, and it's the OR
// of the element flags.
func tupleHasRestElement(tupleType *checker.Type) bool {
	var combinedFlags checker.ElementFlags
	for _, flags := range tupleType.TargetTupleType().ElementFlags() {
		combinedFlags |= flags
	}
	return combinedFlags&checker.ElementFlagsRest != 0
}

func buildGeneratorLoop(state *TransformState, statements *luau.List, name *ast.Node, exp luau.Expression) *luau.List {
	return makeForLoopBuilder(func(state *TransformState, initializer *ast.Node, exp luau.Expression, ids *luau.List, initializers *luau.List) luau.Expression {
		loopID := luau.NewTempID("result")
		ids.Push(loopID)

		initializers.Push(luau.Create(&luau.IfStatement{
			Condition:  luau.NewProperty(loopID, "done"),
			Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
			ElseBody:   luau.NewList(),
		}))

		if ast.IsVariableDeclarationList(initializer) {
			bindingList := luau.NewList()
			initializers.Push(luau.Create(&luau.VariableDeclaration{
				Left:  transformForInitializer(state, initializer, bindingList),
				Right: luau.NewProperty(loopID, "value"),
			}))
			initializers.PushList(bindingList)
		} else {
			transformForInitializerExpressionDirect(state, initializer, initializers, luau.NewProperty(loopID, "value"))
		}

		return luau.NewProperty(convertToIndexableExpression(exp), "next")
	})(state, statements, name, exp)
}

func emptyLoopBuilder(*TransformState, *luau.List, *ast.Node, luau.Expression) *luau.List {
	return luau.NewList()
}

func getLoopBuilder(state *TransformState, node *ast.Node, t *checker.Type) loopBuilder {
	if isDefinitelyType(t, isArrayType(state)) {
		return buildArrayLoop
	} else if isDefinitelyType(t, isSetType(state)) {
		return buildSetLoop
	} else if isDefinitelyType(t, isMapType(state)) {
		return buildMapLoop
	} else if isDefinitelyType(t, isStringType) {
		return buildStringLoop
	} else if isDefinitelyType(t, isIterableFunctionLuaTupleType(state)) {
		return buildIterableFunctionLuaTupleLoop(t)
	} else if isDefinitelyType(t, isIterableFunctionType(state)) {
		return buildIterableFunctionLoop
	} else if isDefinitelyType(t, isGeneratorType(state)) {
		return buildGeneratorLoop
	} else if isDefinitelyType(t, isIterableType(state)) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoIterableIteration.At(shared.AtNode(node)))
		return emptyLoopBuilder
	} else if t.IsUnion() {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoMacroUnion.At(shared.AtNode(node)))
		return emptyLoopBuilder
	} else {
		panic(shared.AssertionMessage("ForOf iteration type not implemented: " + state.Checker.TypeToString(t)))
	}
}

func findRangeMacro(state *TransformState, node *ast.Node) *ast.Node {
	expression := SkipDownwards(node.AsForInOrOfStatement().Expression)
	if ast.IsCallExpression(expression) {
		symbol := getFirstDefinedSymbol(state, state.GetType(expression.Expression()))
		if symbol != nil && symbol == getSymbolOrThrow(state, "$range") {
			return expression
		}
	}
	return nil
}

func transformForOfRangeMacro(state *TransformState, node *ast.Node, macroCall *ast.Node) *luau.List {
	forOf := node.AsForInOrOfStatement()
	result := luau.NewList()

	statements := luau.NewList()
	id := transformForInitializer(state, forOf.Initializer, statements)

	args, prereqs := Capture(state, func() []luau.Expression {
		return ensureTransformOrder(state, macroCall.AsCallExpression().Arguments.Nodes, nil)
	})
	var start, end, step luau.Expression
	if len(args) > 0 {
		start = args[0]
	}
	if len(args) > 1 {
		end = args[1]
	}
	if len(args) > 2 {
		step = args[2]
	}
	result.PushList(prereqs)

	statements.PushList(transformStatementList(state, forOf.Statement, getStatements(forOf.Statement), nil))

	if !(step == nil || luau.IsNumberLiteral(step)) {
		step = luau.NewBinary(step, luau.BinaryOr, luau.NewNumber(1))
	}
	result.Push(luau.Create(&luau.NumericForStatement{
		ID:         id,
		Start:      start,
		End:        end,
		Step:       step,
		Statements: statements,
	}))

	return result
}

func transformForOfStatement(state *TransformState, node *ast.Node) *luau.List {
	forOf := node.AsForInOrOfStatement()
	if forOf.AwaitModifier != nil {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoAwaitForOf.At(shared.AtNode(node)))
	}

	if ast.IsVariableDeclarationList(forOf.Initializer) {
		name := forOf.Initializer.AsVariableDeclarationList().Declarations.Nodes[0].Name()
		if ast.IsIdentifier(name) {
			validateIdentifier(state, name)
		}
	}

	rangeMacroCall := findRangeMacro(state, node)
	if rangeMacroCall != nil {
		return transformForOfRangeMacro(state, node, rangeMacroCall)
	}

	result := luau.NewList()

	exp, expPrereqs := Capture(state, func() luau.Expression { return transformExpression(state, forOf.Expression) })
	result.PushList(expPrereqs)

	expType := state.GetType(forOf.Expression)
	statements := transformStatementList(state, forOf.Statement, getStatements(forOf.Statement), nil)

	builder := getLoopBuilder(state, forOf.Expression, expType)
	result.PushList(builder(state, statements, forOf.Initializer, exp))

	return result
}
