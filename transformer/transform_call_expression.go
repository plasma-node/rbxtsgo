package transformer

// Port of src/TSTransformer/nodes/expressions/transformCallExpression.ts.

import (
	"fmt"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func runCallMacro(macro callMacro, state *TransformState, node *ast.Node, expression luau.Expression, nodeArguments []*ast.Node) luau.Expression {
	var args []luau.Expression
	prereqs := state.CapturePrereqs(func() {
		args = ensureTransformOrder(state, nodeArguments, nil)
		var lastArg *ast.Node
		if len(nodeArguments) > 0 {
			lastArg = nodeArguments[len(nodeArguments)-1]
		}
		if lastArg != nil && ast.IsSpreadElement(lastArg) {
			signature := state.Checker.GetSignaturesOfType(
				state.GetType(node.Expression()),
				checker.SignatureKindCall,
			)[0]

			parameters := signature.Parameters()
			lastParameter := parameters[len(parameters)-1].ValueDeclaration()
			if lastParameter != nil && ast.IsParameterDeclaration(lastParameter) && lastParameter.AsParameterDeclaration().DotDotDotToken != nil {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoVarArgsMacroSpread.At(shared.AtNode(lastArg)))
				return
			}

			// use .expression for the tuple type, simply `lastArg` would give the tuple's element type
			tupleArgType := state.GetType(lastArg.Expression())
			// Since we've excluded vararg macros, TS will have ensured that the spread is from a tuple type
			if !checker.IsTupleType(tupleArgType) {
				panic(shared.AssertionMessage(""))
			}
			argumentCount := len(tupleArgType.TargetTupleType().ElementFlags())

			spread := args[len(args)-1]
			args = args[:len(args)-1]
			tempIds := luau.NewList()
			for i := len(args); i < argumentCount; i++ {
				tempId := luau.NewTempID(fmt.Sprintf("spread%d", i))
				args = append(args, tempId)
				tempIds.Push(tempId)
			}
			state.Prereq(
				luau.Create(&luau.VariableDeclaration{
					Left:  tempIds,
					Right: spread,
				}),
			)
		}

		for i := range args {
			// a spread can make args longer than nodeArguments, and JS reads undefined past the end
			var nodeArgument *ast.Node
			if i < len(nodeArguments) {
				nodeArgument = nodeArguments[i]
			}
			if expressionMightMutate(state, args[i], nodeArgument) {
				name := valueToIDStr(args[i])
				if name == "" {
					name = fmt.Sprintf("arg%d", i)
				}
				args[i] = state.PushToVar(args[i], name)
			}
		}
	})

	nodeExpression := node.Expression()
	if ast.IsPropertyAccessExpression(nodeExpression) || ast.IsElementAccessExpression(nodeExpression) {
		nodeExpression = nodeExpression.Expression()
	}

	if !prereqs.IsEmpty() && expressionMightMutate(state, expression, nodeExpression) {
		name := valueToIDStr(expression)
		if name == "" {
			name = "exp"
		}
		expression = state.PushToVar(expression, name)
	}
	state.PrereqList(prereqs)

	return wrapReturnIfLuaTuple(state, node, macro(state, node, expression, args))
}

// fixVoidArgumentsForRobloxFunctions: Some C functions like `tonumber()` will error if the given argument is a function
// that returns nothing.
// i.e.
//
//	local function foo()
//	end
//	local x = tonumber(foo()) -- error!
//
// To protect against this, we can wrap possibly-undefined arguments with `()` to coerce the values to `nil`
func fixVoidArgumentsForRobloxFunctions(state *TransformState, t *checker.Type, args []luau.Expression, nodeArguments []*ast.Node) {
	if isPossiblyType(t, isRobloxType(state)) {
		for i := range args {
			arg := args[i]
			nodeArg := nodeArguments[i]
			if ast.IsCallExpression(nodeArg) && isPossiblyType(state.GetType(nodeArg), isUndefinedType) {
				args[i] = luau.Create(&luau.ParenthesizedExpression{
					Expression: arg,
				})
			}
		}
	}
}

func transformCallExpressionInner(state *TransformState, node *ast.Node, expression luau.Expression, nodeArguments []*ast.Node) luau.Expression {
	if ast.IsImportCall(node) {
		return transformImportExpression(state, node)
	}

	// a in a()
	validateNotAnyType(state, node.Expression())

	if ast.IsSuperCall(node) {
		return luau.NewCall(luau.NewProperty(convertToIndexableExpression(expression), "constructor"),
			append([]luau.Expression{luau.Globals.Self}, ensureTransformOrder(state, node.Arguments(), nil)...)...,
		)
	}

	expType := state.Checker.GetNonOptionalType(state.GetType(node.Expression()))
	symbol := getFirstDefinedSymbol(state, expType)
	if symbol != nil {
		macro := getCallMacro(state, symbol)
		if macro != nil {
			return runCallMacro(macro, state, node, expression, nodeArguments)
		}
	}

	args, prereqs := Capture(state, func() []luau.Expression { return ensureTransformOrder(state, nodeArguments, nil) })
	fixVoidArgumentsForRobloxFunctions(state, expType, args, nodeArguments)

	if !prereqs.IsEmpty() && expressionMightMutate(state, expression, node.Expression()) {
		expression = state.PushToVar(expression, "fn")
	}
	state.PrereqList(prereqs)

	exp := luau.NewCall(convertToIndexableExpression(expression), args...)

	return wrapReturnIfLuaTuple(state, node, exp)
}

func transformPropertyCallExpressionInner(state *TransformState, node *ast.Node, expression *ast.Node, baseExpression luau.Expression, name string, nodeArguments []*ast.Node) luau.Expression {
	// a in a.b()
	validateNotAnyType(state, expression.Expression())
	// a.b in a.b()
	validateNotAnyType(state, node.Expression())

	if ast.IsSuperProperty(expression) {
		return luau.NewCall(luau.NewProperty(convertToIndexableExpression(baseExpression), expression.Name().Text()),
			append([]luau.Expression{luau.Globals.Self}, ensureTransformOrder(state, node.Arguments(), nil)...)...,
		)
	}

	expType := state.Checker.GetNonOptionalType(state.GetType(node.Expression()))
	symbol := getFirstDefinedSymbol(state, expType)
	if symbol != nil {
		macro := getPropertyCallMacro(state, symbol)
		if macro != nil {
			return runCallMacro(callMacro(macro), state, node, baseExpression, nodeArguments)
		}
	}

	args, prereqs := Capture(state, func() []luau.Expression { return ensureTransformOrder(state, nodeArguments, nil) })
	fixVoidArgumentsForRobloxFunctions(state, expType, args, nodeArguments)

	if !prereqs.IsEmpty() && expressionMightMutate(state, baseExpression, expression.Expression()) {
		baseExpression = state.PushToVar(baseExpression, "")
	}
	state.PrereqList(prereqs)

	var exp luau.Expression
	if isMethod(state, expression) {
		// check that the name isn't a Luau keyword
		// if it is, we need to use PropertyAccessExpression and manually add the self argument
		if luau.IsValidIdentifier(name) {
			exp = luau.Create(&luau.MethodCallExpression{
				Name:       name,
				Expression: convertToIndexableExpression(baseExpression),
				Args:       luau.MakeList(args...),
			})
		} else {
			baseExpression = state.PushToVarIfComplex(baseExpression, "")
			args = append([]luau.Expression{baseExpression}, args...)
			exp = luau.NewCall(luau.NewProperty(convertToIndexableExpression(baseExpression), name), args...)
		}
	} else {
		// PropertyAccessExpression will wrap the identifier for us if necessary
		exp = luau.NewCall(luau.NewProperty(convertToIndexableExpression(baseExpression), name), args...)
	}

	return wrapReturnIfLuaTuple(state, node, exp)
}

func transformElementCallExpressionInner(state *TransformState, node *ast.Node, expression *ast.Node, baseExpression luau.Expression, argumentExpression *ast.Node, nodeArguments []*ast.Node) luau.Expression {
	elementAccess := expression.AsElementAccessExpression()

	// a in a[b]()
	validateNotAnyType(state, elementAccess.Expression)
	// b in a[b]()
	validateNotAnyType(state, elementAccess.ArgumentExpression)
	// a[b] in a[b]()
	validateNotAnyType(state, node.Expression())

	if ast.IsSuperProperty(expression) {
		return luau.NewCall(
			luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(baseExpression),
				Index:      transformExpression(state, elementAccess.ArgumentExpression),
			}),
			append([]luau.Expression{luau.Globals.Self}, ensureTransformOrder(state, node.Arguments(), nil)...)...,
		)
	}

	expType := state.Checker.GetNonOptionalType(state.GetType(node.Expression()))
	symbol := getFirstDefinedSymbol(state, expType)
	if symbol != nil {
		macro := getPropertyCallMacro(state, symbol)
		if macro != nil {
			return runCallMacro(callMacro(macro), state, node, baseExpression, nodeArguments)
		}
	}

	ordered, prereqs := Capture(state, func() []luau.Expression {
		return ensureTransformOrder(state, append([]*ast.Node{argumentExpression}, nodeArguments...), nil)
	})
	argumentExp, args := ordered[0], ordered[1:]

	fixVoidArgumentsForRobloxFunctions(state, expType, args, nodeArguments)

	if !prereqs.IsEmpty() && expressionMightMutate(state, baseExpression, elementAccess.Expression) {
		baseExpression = state.PushToVar(baseExpression, "")
	}
	state.PrereqList(prereqs)

	if isMethod(state, expression) {
		baseExpression = state.PushToVarIfComplex(baseExpression, "")
		args = append([]luau.Expression{baseExpression}, args...)
	}

	exp := luau.NewCall(
		luau.Create(&luau.ComputedIndexExpression{
			Expression: convertToIndexableExpression(baseExpression),
			Index: addOneIfArrayType(
				state,
				state.Checker.GetNonOptionalType(state.GetType(elementAccess.Expression)),
				argumentExp,
			),
		}),
		args...,
	)

	return wrapReturnIfLuaTuple(state, node, exp)
}

func transformCallExpression(state *TransformState, node *ast.Node) luau.Expression {
	return transformOptionalChain(state, node)
}
