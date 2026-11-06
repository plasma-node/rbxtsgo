package transformer

// Port of src/TSTransformer/nodes/transformOptionalChain.ts.

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

type optionalChainItemKind int

const (
	optionalChainItemKindPropertyAccess optionalChainItemKind = iota
	optionalChainItemKindElementAccess
	optionalChainItemKindCall
	optionalChainItemKindPropertyCall
	optionalChainItemKindElementCall
)

// chainItem is the union of upstream's PropertyAccessItem, ElementAccessItem, CallItem, PropertyCallItem and
// ElementCallItem. kind says which fields are in use.
type chainItem struct {
	kind     optionalChainItemKind
	node     *ast.Node
	optional bool
	typ      *checker.Type

	name               string        // PropertyAccess, PropertyCall
	expression         *ast.Node     // ElementAccess (the argument), PropertyCall, ElementCall
	argumentExpression *ast.Node     // ElementCall
	callOptional       bool          // PropertyCall, ElementCall
	callType           *checker.Type // PropertyCall, ElementCall
	args               []*ast.Node   // Call, PropertyCall, ElementCall
}

func createPropertyAccessItem(state *TransformState, node *ast.Node) *chainItem {
	return &chainItem{
		node:     node,
		kind:     optionalChainItemKindPropertyAccess,
		optional: node.AsPropertyAccessExpression().QuestionDotToken != nil,
		typ:      state.GetType(node.Expression()),
		name:     node.Name().Text(),
	}
}

func createElementAccessItem(state *TransformState, node *ast.Node) *chainItem {
	return &chainItem{
		node:       node,
		kind:       optionalChainItemKindElementAccess,
		optional:   node.AsElementAccessExpression().QuestionDotToken != nil,
		typ:        state.GetType(node.Expression()),
		expression: node.AsElementAccessExpression().ArgumentExpression,
	}
}

func createCallItem(state *TransformState, node *ast.Node) *chainItem {
	return &chainItem{
		node:     node,
		kind:     optionalChainItemKindCall,
		optional: node.AsCallExpression().QuestionDotToken != nil,
		typ:      state.GetType(node.Expression()),
		args:     node.AsCallExpression().Arguments.Nodes,
	}
}

func createPropertyCallItem(state *TransformState, node *ast.Node, expression *ast.Node) *chainItem {
	return &chainItem{
		node:         node,
		expression:   expression,
		kind:         optionalChainItemKindPropertyCall,
		optional:     expression.AsPropertyAccessExpression().QuestionDotToken != nil,
		typ:          state.GetType(node.Expression()),
		name:         expression.Name().Text(),
		callType:     state.GetType(node),
		callOptional: node.AsCallExpression().QuestionDotToken != nil,
		args:         node.AsCallExpression().Arguments.Nodes,
	}
}

func createElementCallItem(state *TransformState, node *ast.Node, expression *ast.Node) *chainItem {
	return &chainItem{
		node:               node,
		expression:         expression,
		kind:               optionalChainItemKindElementCall,
		optional:           expression.AsElementAccessExpression().QuestionDotToken != nil,
		typ:                state.GetType(expression),
		argumentExpression: expression.AsElementAccessExpression().ArgumentExpression,
		callType:           state.GetType(node),
		callOptional:       node.AsCallExpression().QuestionDotToken != nil,
		args:               node.AsCallExpression().Arguments.Nodes,
	}
}

type flattenOptionalChainResult struct {
	chain      []*chainItem
	expression *ast.Node
}

func flattenOptionalChain(state *TransformState, expression *ast.Node) flattenOptionalChainResult {
	var chain []*chainItem
	for {
		if ast.IsPropertyAccessExpression(expression) {
			chain = slices.Insert(chain, 0, createPropertyAccessItem(state, expression))
			expression = expression.Expression()
		} else if ast.IsElementAccessExpression(expression) {
			chain = slices.Insert(chain, 0, createElementAccessItem(state, expression))
			expression = expression.Expression()
		} else if ast.IsCallExpression(expression) {
			// this is a bit of a mess..
			subExp := SkipDownwards(expression.Expression())
			if ast.IsPropertyAccessExpression(subExp) {
				chain = slices.Insert(chain, 0, createPropertyCallItem(state, expression, subExp))
				expression = subExp.Expression()
			} else if ast.IsElementAccessExpression(subExp) {
				chain = slices.Insert(chain, 0, createElementCallItem(state, expression, subExp))
				expression = subExp.Expression()
			} else {
				chain = slices.Insert(chain, 0, createCallItem(state, expression))
				expression = subExp
			}
		} else {
			break
		}
	}
	return flattenOptionalChainResult{chain, expression}
}

func transformChainItem(state *TransformState, baseExpression luau.Expression, item *chainItem) luau.Expression {
	if item.kind == optionalChainItemKindPropertyAccess {
		return transformPropertyAccessExpressionInner(state, item.node, baseExpression, item.name)
	} else if item.kind == optionalChainItemKindElementAccess {
		return transformElementAccessExpressionInner(state, item.node, baseExpression, item.expression)
	} else if item.kind == optionalChainItemKindCall {
		return transformCallExpressionInner(state, item.node, baseExpression, item.args)
	} else if item.kind == optionalChainItemKindPropertyCall {
		return transformPropertyCallExpressionInner(
			state,
			item.node,
			item.expression,
			baseExpression,
			item.name,
			item.args,
		)
	} else {
		return transformElementCallExpressionInner(
			state,
			item.node,
			item.expression,
			baseExpression,
			item.argumentExpression,
			item.args,
		)
	}
}

func createOrSetTempID(state *TransformState, tempID *luau.TemporaryIdentifier, expression luau.Expression, node *ast.Node) *luau.TemporaryIdentifier {
	if tempID == nil {
		name := "result"
		if node.Parent != nil && ast.IsVariableDeclaration(node.Parent) && ast.IsIdentifier(node.Parent.Name()) {
			name = node.Parent.Name().Text()
		}
		tempID = state.PushToVar(expression, name)
	} else {
		if luau.Expression(tempID) != expression {
			state.Prereq(luau.Create(&luau.Assignment{
				Left:     tempID,
				Operator: luau.AssignEqual,
				Right:    expression,
			}))
		}
	}
	return tempID
}

func createNilCheck(tempID *luau.TemporaryIdentifier, statements *luau.List) *luau.IfStatement {
	return luau.Create(&luau.IfStatement{
		Condition:  luau.NewBinary(tempID, luau.BinaryNotEqual, luau.NewNil()),
		Statements: statements,
		ElseBody:   luau.NewList(),
	})
}

func isCompoundCall(item *chainItem) bool {
	return item.kind == optionalChainItemKindPropertyCall || item.kind == optionalChainItemKindElementCall
}

func transformOptionalChainInner(
	state *TransformState,
	chain []*chainItem,
	baseExpression luau.Expression,
	tempID *luau.TemporaryIdentifier,
	index int,
) luau.Expression {
	if index >= len(chain) {
		return baseExpression
	}
	item := chain[index]
	if item.optional || (isCompoundCall(item) && item.callOptional) {
		isMethodCall := false
		isSuperCall := false
		var selfParam *luau.TemporaryIdentifier

		if isCompoundCall(item) {
			isMethodCall = isMethod(state, item.expression)
			isSuperCall = ast.IsSuperProperty(item.expression)

			if item.callOptional && isMethodCall && !isSuperCall {
				selfParam = state.PushToVar(baseExpression, "self")
				baseExpression = selfParam
			}

			if item.optional {
				tempID = createOrSetTempID(state, tempID, baseExpression, chain[len(chain)-1].node)
				baseExpression = tempID
			}

			if item.callOptional {
				if item.kind == optionalChainItemKindPropertyCall {
					baseExpression = luau.NewProperty(convertToIndexableExpression(baseExpression), item.name)
				} else {
					expType := state.GetType(item.expression.Expression())

					baseExpression = luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(baseExpression),
						Index:      addOneIfArrayType(state, expType, transformExpression(state, item.argumentExpression)),
					})
				}
			}
		}

		// capture so we can wrap later if necessary
		result, prereqStatements := Capture(state, func() luau.Expression {
			tempID = createOrSetTempID(state, tempID, baseExpression, chain[len(chain)-1].node)

			newValue, ifStatements := Capture(state, func() luau.Expression {
				var newExpression luau.Expression
				if isCompoundCall(item) && item.callOptional {
					expType := state.Checker.GetNonOptionalType(state.GetType(item.node.Expression()))
					symbol := getFirstDefinedSymbol(state, expType)
					if symbol != nil {
						macro := getPropertyCallMacro(state, symbol)
						if macro != nil {
							state.Diagnostic.AddDiagnostic(shared.Errors.NoOptionalMacroCall.At(shared.AtNode(item.node)))
							return luau.NewNone()
						}
					}

					args := ensureTransformOrder(state, item.args, nil)
					if isMethodCall {
						if isSuperCall {
							args = slices.Insert(args, 0, luau.Expression(luau.Globals.Self))
						} else {
							args = slices.Insert(args, 0, luau.Expression(selfParam))
						}
					}
					newExpression = wrapReturnIfLuaTuple(state, item.node, luau.NewCall(tempID, args...))
				} else {
					newExpression = transformChainItem(state, tempID, item)
				}
				return transformOptionalChainInner(state, chain, newExpression, tempID, index+1)
			})

			isUsed := !luau.IsNone(newValue) && !isUsedAsStatement(item.node)

			if luau.Expression(tempID) != newValue && isUsed {
				ifStatements.Push(luau.Create(&luau.Assignment{
					Left:     tempID,
					Operator: luau.AssignEqual,
					Right:    newValue,
				}))
			} else {
				if luau.IsCall(newValue) {
					ifStatements.Push(luau.Create(&luau.CallStatement{
						Expression: newValue.(luau.CallLike),
					}))
				}
			}

			state.Prereq(createNilCheck(tempID, ifStatements))

			if isUsed {
				return tempID
			}
			return luau.NewNone()
		})

		if isCompoundCall(item) && item.optional && item.callOptional {
			state.Prereq(createNilCheck(tempID, prereqStatements))
		} else {
			state.PrereqList(prereqStatements)
		}

		return result
	} else {
		return transformOptionalChainInner(
			state,
			chain,
			transformChainItem(state, baseExpression, item),
			tempID,
			index+1,
		)
	}
}

func transformOptionalChain(state *TransformState, node *ast.Node) luau.Expression {
	flattened := flattenOptionalChain(state, node)
	return transformOptionalChainInner(state, flattened.chain, transformExpression(state, flattened.expression), nil, 0)
}
