package transformer

// Port of src/TSTransformer/nodes/statements/transformForStatement.ts.

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/astnav"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/jsnum"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

// forStatementPossibleReferencePositions is getPossibleSymbolReferencePositions in TS 5.5.3. The checker's copy can't
// be used: it only searches a whole file correctly, excludes a match at container.end, and reads the characters around
// a match as single bytes, so it drops a match that's next to non-ASCII whitespace.
func forStatementPossibleReferencePositions(sourceFile *ast.SourceFile, symbolName string, container *ast.Node) []int {
	var positions []int
	if symbolName == "" {
		return positions
	}
	text := sourceFile.Text()
	sourceLength := len(text)
	symbolNameLength := len(symbolName)
	position := forStatementIndexOf(text, symbolName, container.Pos())
	for position >= 0 {
		if position > container.End() {
			break
		}
		endPosition := position + symbolNameLength
		before, _ := utf8.DecodeLastRuneInString(text[:position])
		after, _ := utf8.DecodeRuneInString(text[endPosition:])
		if (position == 0 || !scanner.IsIdentifierPart(before)) && (endPosition == sourceLength || !scanner.IsIdentifierPart(after)) {
			positions = append(positions, position)
		}
		position = forStatementIndexOf(text, symbolName, position+symbolNameLength+1)
	}
	return positions
}

func forStatementIndexOf(text string, search string, from int) int {
	if from > len(text) {
		return -1
	}
	index := strings.Index(text[from:], search)
	if index < 0 {
		return -1
	}
	return from + index
}

// forStatementLocalSymbolForExportSpecifier is getLocalSymbolForExportSpecifier in TS 5.5.3.
func forStatementLocalSymbolForExportSpecifier(c *checker.Checker, referenceLocation *ast.Node, referenceSymbol *ast.Symbol, exportSpecifier *ast.Node) *ast.Symbol {
	specifier := exportSpecifier.AsExportSpecifier()
	var isAlias bool
	if specifier.PropertyName != nil {
		isAlias = specifier.PropertyName == referenceLocation
	} else {
		isAlias = exportSpecifier.Parent.Parent.AsExportDeclaration().ModuleSpecifier == nil
	}
	if isAlias {
		if symbol := c.GetExportSpecifierLocalTargetSymbol(exportSpecifier); symbol != nil {
			return symbol
		}
	}
	return referenceSymbol
}

// forStatementEachSymbolReferenceInFile is ts.FindAllReferences.Core.eachSymbolReferenceInFile in TS 5.5.3. It stops at
// the first true from cb. The checker's GetReferencesToSymbolInFile has no searchContainer and doesn't skip the
// definition, and upstream relies on both.
func forStatementEachSymbolReferenceInFile(state *TransformState, definition *ast.Node, sourceFile *ast.SourceFile, cb func(token *ast.Node) bool, searchContainer *ast.Node) bool {
	c := state.Checker
	var symbol *ast.Symbol
	if ast.IsParameterPropertyDeclaration(definition.Parent, definition.Parent.Parent) {
		symbol, _ = c.GetSymbolsOfParameterPropertyDeclaration(definition.Parent, definition.Text())
	} else {
		symbol = c.GetSymbolAtLocation(definition)
	}
	if symbol == nil {
		return false
	}
	for _, position := range forStatementPossibleReferencePositions(sourceFile, symbol.Name(), searchContainer) {
		token := astnav.GetTouchingPropertyName(sourceFile, position)
		if token == sourceFile.AsNode() {
			continue
		}
		if !ast.IsIdentifier(token) || token == definition || token.Text() != definition.Text() {
			continue
		}
		referenceSymbol := c.GetSymbolAtLocation(token)
		if referenceSymbol == symbol ||
			(token.Parent.Kind == ast.KindShorthandPropertyAssignment && c.GetShorthandAssignmentValueSymbol(token.Parent) == symbol) ||
			(ast.IsExportSpecifier(token.Parent) && forStatementLocalSymbolForExportSpecifier(c, token, referenceSymbol, token.Parent) == symbol) {
			if cb(token) {
				return true
			}
		}
	}
	return false
}

// forStatementNodeText is node.getText().
func forStatementNodeText(node *ast.Node) string {
	return scanner.GetSourceTextOfNodeFromSourceFile(ast.GetSourceFileOfNode(node), node, false)
}

// forStatementNumberIsInteger is Number.isInteger.
func forStatementNumberIsInteger(value jsnum.Number) bool {
	return !value.IsNaN() && !value.IsInf() && float64(value) == math.Trunc(float64(value))
}

// isUnaryExpressionWithWrite is ts.isUnaryExpressionWithWrite from TS 5.5.3.
func isUnaryExpressionWithWrite(expr *ast.Node) bool {
	switch expr.Kind {
	case ast.KindPostfixUnaryExpression:
		return true
	case ast.KindPrefixUnaryExpression:
		operator := expr.AsPrefixUnaryExpression().Operator
		return operator == ast.KindPlusPlusToken || operator == ast.KindMinusMinusToken
	default:
		return false
	}
}

func addFinalizersToIfStatement(node *luau.IfStatement, finalizers *luau.List) {
	if node.Statements.IsNonEmpty() {
		addFinalizers(node.Statements, node.Statements.Head, finalizers)
	}
	if elseBody, isList := luau.AsList(node.ElseBody); isList {
		if elseBody.IsNonEmpty() {
			addFinalizers(elseBody, elseBody.Head, finalizers)
		}
	} else {
		addFinalizersToIfStatement(node.ElseBody.(*luau.IfStatement), finalizers)
	}
}

func addFinalizers(list *luau.List, node *luau.ListNode, finalizers *luau.List) {
	if list.IsEmpty() {
		panic(shared.AssertionMessage(""))
	}

	statement := node.Value
	if luau.IsContinueStatement(statement) {
		finalizersClone := finalizers.Clone()

		// fix node parents
		finalizersClone.ForEach(func(node luau.Node) { node.SetParent(statement.Parent()) })

		if node.Prev != nil {
			node.Prev.Next = finalizersClone.Head
		} else if node == list.Head {
			list.Head = finalizersClone.Head
		}

		node.Prev = finalizersClone.Tail

		finalizersClone.Tail.Next = node
	}

	if luau.IsDoStatement(statement) {
		doStatement := statement.(*luau.DoStatement)
		if doStatement.Statements.IsNonEmpty() {
			addFinalizers(doStatement.Statements, doStatement.Statements.Head, finalizers)
		}
	} else if luau.IsIfStatement(statement) {
		addFinalizersToIfStatement(statement.(*luau.IfStatement), finalizers)
	}

	if node.Next != nil {
		addFinalizers(list, node.Next, finalizers)
	}
}

func canSkipClone(state *TransformState, initializer *ast.Node, id *ast.Node) bool {
	// is symbol used in initializer (besides its definition)
	return !forStatementEachSymbolReferenceInFile(state, id, ast.GetSourceFileOfNode(id), func(*ast.Node) bool { return true }, initializer)
}

func isIDWriteOrAsyncRead(state *TransformState, forStatement *ast.Node, id *ast.Node) bool {
	incrementor := forStatement.AsForStatement().Incrementor
	return forStatementEachSymbolReferenceInFile(
		state,
		id,
		ast.GetSourceFileOfNode(id),
		func(token *ast.Node) bool {
			// write
			if ast.IsWriteAccess(token) &&
				(incrementor == nil || !isAncestorOf(incrementor, token)) {
				return true
			}

			// async read
			ancestor := GetAncestor(token, func(v *ast.Node) bool { return v == forStatement || ast.IsFunctionLike(v) })
			if ancestor != nil && ancestor != forStatement {
				return true
			}
			return false
		},
		forStatement,
	)
}

func transformForStatementFallback(state *TransformState, node *ast.Node) *luau.List {
	forStatement := node.AsForStatement()
	initializer, condition, incrementor, statement := forStatement.Initializer, forStatement.Condition, forStatement.Incrementor, forStatement.Statement

	result := luau.NewList()
	whileStatements := luau.NewList()
	finalizerStatements := luau.NewList()

	var variables []*ast.Node
	if initializer != nil && ast.IsVariableDeclarationList(initializer) {
		variables = getDeclaredVariables(initializer)
	}
	hasWriteOrAsyncRead := map[*ast.Symbol]bool{}
	skipClone := map[*ast.Symbol]bool{}

	if initializer != nil && ast.IsVariableDeclarationList(initializer) {
		for _, id := range variables {
			symbol := state.Checker.GetSymbolAtLocation(id)
			if symbol == nil {
				panic(shared.AssertionMessage(""))
			}
			if isIDWriteOrAsyncRead(state, node, id) {
				hasWriteOrAsyncRead[symbol] = true
			}
			if canSkipClone(state, initializer, id) {
				skipClone[symbol] = true
			}
		}
	}

	if initializer != nil {
		if ast.IsVariableDeclarationList(initializer) {
			if isVarDeclaration(initializer) {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoVar.At(shared.AtNode(node)))
			}

			for _, id := range variables {
				symbol := state.Checker.GetSymbolAtLocation(id)
				if symbol == nil {
					panic(shared.AssertionMessage(""))
				}
				if hasWriteOrAsyncRead[symbol] {
					if skipClone[symbol] {
						state.SymbolToIDMap[symbol] = luau.NewTempID(forStatementNodeText(id))
					} else {
						copyID := luau.NewTempID(forStatementNodeText(id) + "Copy")
						state.SymbolToIDMap[symbol] = copyID
					}
				}
			}

			for _, declaration := range initializer.AsVariableDeclarationList().Declarations.Nodes {
				decStatements, decPrereqs := Capture(state, func() *luau.List {
					result := luau.NewList()
					decStatements, decPrereqs := Capture(state, func() *luau.List {
						return transformVariableDeclaration(state, declaration)
					})
					result.PushList(decPrereqs)
					result.PushList(decStatements)
					return result
				})
				result.PushList(decPrereqs)
				result.PushList(decStatements)
			}

			for _, id := range variables {
				symbol := state.Checker.GetSymbolAtLocation(id)
				if symbol == nil {
					panic(shared.AssertionMessage(""))
				}
				if hasWriteOrAsyncRead[symbol] {
					var tempID *luau.TemporaryIdentifier
					if skipClone[symbol] {
						tempID = state.SymbolToIDMap[symbol]
						if tempID == nil {
							panic(shared.AssertionMessage(""))
						}
					} else {
						tempID = luau.NewTempID(forStatementNodeText(id))
						copyID := state.SymbolToIDMap[symbol]
						if copyID == nil {
							panic(shared.AssertionMessage(""))
						}

						// local _i = _iCopy
						result.Push(luau.Create(&luau.VariableDeclaration{
							Left:  tempID,
							Right: copyID,
						}))
					}
					delete(state.SymbolToIDMap, symbol)
					realID := transformIdentifierDefined(state, id)

					// local i = _i
					whileStatements.Push(luau.Create(&luau.VariableDeclaration{
						Left:  realID,
						Right: tempID,
					}))

					// _i = i
					finalizerStatements.Push(luau.Create(&luau.Assignment{
						Left:     tempID,
						Operator: luau.AssignEqual,
						Right:    realID,
					}))
				}
			}
		} else {
			statements, prereqs := Capture(state, func() *luau.List { return transformExpressionStatementInner(state, initializer) })
			result.PushList(prereqs)
			result.PushList(statements)
		}
	}

	if incrementor != nil {
		shouldIncrement := luau.NewTempID("shouldIncrement")

		// local _shouldIncrement = false
		result.Push(luau.Create(&luau.VariableDeclaration{
			Left:  shouldIncrement,
			Right: luau.NewBool(false),
		}))

		incrementorStatements := luau.NewList()
		statements, prereqs := Capture(state, func() *luau.List { return transformExpressionStatementInner(state, incrementor) })
		incrementorStatements.PushList(prereqs)
		incrementorStatements.PushList(statements)

		// if _shouldIncrement then
		// 	[incrementorStatements]
		// else
		// 	_shouldIncrement = true
		// end
		whileStatements.Push(luau.Create(&luau.IfStatement{
			Condition:  shouldIncrement,
			Statements: incrementorStatements,
			ElseBody: luau.MakeList(luau.Create(&luau.Assignment{
				Left:     shouldIncrement,
				Operator: luau.AssignEqual,
				Right:    luau.NewBool(true),
			})),
		}))
	}

	conditionExp, conditionPrereqs := Capture(state, func() luau.Expression {
		if condition != nil {
			return createTruthinessChecks(state, transformExpression(state, condition), condition)
		} else {
			return luau.NewBool(true)
		}
	})

	whileStatements.PushList(conditionPrereqs)

	if !whileStatements.IsEmpty() {
		if condition != nil {
			// if not [conditionExp] then
			//	break
			// end
			whileStatements.Push(luau.Create(&luau.IfStatement{
				Condition:  luau.NewUnary(luau.UnaryNot, conditionExp),
				Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				ElseBody:   luau.NewList(),
			}))
		}
		conditionExp = luau.NewBool(true)
	}

	whileStatements.PushList(transformStatementList(state, statement, getStatements(statement), nil))

	if whileStatements.IsNonEmpty() && finalizerStatements.IsNonEmpty() {
		addFinalizers(whileStatements, whileStatements.Head, finalizerStatements)
	}

	if whileStatements.Tail == nil || !luau.IsFinalStatement(whileStatements.Tail.Value) {
		whileStatements.PushList(finalizerStatements)
	}

	result.Push(luau.Create(&luau.WhileStatement{
		Condition:  conditionExp,
		Statements: whileStatements,
	}))

	if result.Head == result.Tail {
		return result
	}
	return luau.MakeList(luau.Create(&luau.DoStatement{Statements: result}))
}

// getOptimizedIncrementorStepValue returns false where JS returns undefined.
func getOptimizedIncrementorStepValue(state *TransformState, incrementor *ast.Node, idSymbol *ast.Symbol) (float64, bool) {
	if ast.IsBinaryExpression(incrementor) {
		binary := incrementor.AsBinaryExpression()
		if ast.IsIdentifier(binary.Left) &&
			state.Checker.GetSymbolAtLocation(binary.Left) == idSymbol &&
			binary.OperatorToken.Kind == ast.KindPlusEqualsToken &&
			ast.IsNumericLiteral(binary.Right) &&
			isProbablyInteger(state, binary.Right) {
			return float64(jsnum.FromString(forStatementNodeText(binary.Right))), true
		} else if binary.OperatorToken.Kind == ast.KindMinusEqualsToken &&
			ast.IsNumericLiteral(binary.Right) &&
			isProbablyInteger(state, binary.Right) {
			return -float64(jsnum.FromString(forStatementNodeText(binary.Right))), true
		}
	} else if ast.IsPostfixUnaryExpression(incrementor) || ast.IsPrefixUnaryExpression(incrementor) {
		operand, operator := unaryExpressionParts(incrementor)
		if ast.IsIdentifier(operand) &&
			state.Checker.GetSymbolAtLocation(operand) == idSymbol &&
			operator == ast.KindPlusPlusToken {
			return 1, true
		} else if ast.IsIdentifier(operand) &&
			state.Checker.GetSymbolAtLocation(operand) == idSymbol &&
			operator == ast.KindMinusMinusToken {
			return -1, true
		}
	}
	return 0, false
}

func isSizeMacro(state *TransformState, expression *ast.Node) bool {
	if ast.IsCallExpression(expression) {
		expType := state.Checker.GetNonOptionalType(state.GetType(expression.Expression()))
		symbol := getFirstDefinedSymbol(state, expType)
		if symbol != nil {
			macro := getPropertyCallMacro(state, symbol)
			if macro != nil && symbol.Name() == "size" {
				return true
			}
		}
	}
	return false
}

func isMutatedInBody(state *TransformState, identifier *ast.Node, body *ast.Node) bool {
	return forStatementEachSymbolReferenceInFile(
		state,
		identifier,
		ast.GetSourceFileOfNode(identifier),
		func(token *ast.Node) bool {
			parent := SkipUpwards(token).Parent
			if ast.IsAssignmentExpression(parent, false) && SkipDownwards(parent.AsBinaryExpression().Left) == token {
				return true
			} else if isUnaryExpressionWithWrite(parent) {
				if operand, _ := unaryExpressionParts(parent); SkipDownwards(operand) == token {
					return true
				}
			}
			return false
		},
		body,
	)
}

func isProbablyInteger(state *TransformState, expression *ast.Node) bool {
	if ast.IsNumericLiteral(expression) {
		return forStatementNumberIsInteger(jsnum.FromString(forStatementNodeText(expression)))
	} else if ast.IsBinaryExpression(expression) {
		binary := expression.AsBinaryExpression()
		if binary.OperatorToken.Kind == ast.KindPlusToken ||
			binary.OperatorToken.Kind == ast.KindMinusToken ||
			binary.OperatorToken.Kind == ast.KindAsteriskToken ||
			binary.OperatorToken.Kind == ast.KindAsteriskAsteriskToken {
			return isProbablyInteger(state, binary.Left) && isProbablyInteger(state, binary.Right)
		}
	} else if ast.IsPrefixUnaryExpression(expression) {
		prefix := expression.AsPrefixUnaryExpression()
		if prefix.Operator == ast.KindPlusToken || prefix.Operator == ast.KindMinusToken {
			return isProbablyInteger(state, prefix.Operand)
		}
	} else if isSizeMacro(state, expression) {
		return true
	} else if isDefinitelyType(state.GetType(expression), func(t *checker.Type) bool {
		if !t.IsNumberLiteral() {
			return false
		}
		value, ok := t.AsLiteralType().Value().(jsnum.Number)
		return ok && forStatementNumberIsInteger(value)
	}) {
		return true
	}
	return false
}

// transformForStatementOptimized returns nil where JS returns undefined.
func transformForStatementOptimized(state *TransformState, node *ast.Node) *luau.List {
	forStatement := node.AsForStatement()
	initializer, condition, incrementor, statement := forStatement.Initializer, forStatement.Condition, forStatement.Incrementor, forStatement.Statement

	// this function is difficult to break up because it uses several type guard functions..

	// validate initializer exists and is a single identifier `x` with a value that is _probably_ an integer

	if initializer == nil || !ast.IsVariableDeclarationList(initializer) || len(initializer.AsVariableDeclarationList().Declarations.Nodes) != 1 {
		return nil
	}

	declaration := initializer.AsVariableDeclarationList().Declarations.Nodes[0]
	decName, decInit := declaration.Name(), declaration.AsVariableDeclaration().Initializer
	if !ast.IsIdentifier(decName) || decInit == nil {
		return nil
	}

	idSymbol := state.Checker.GetSymbolAtLocation(decName)
	if idSymbol == nil {
		return nil
	}

	if !isProbablyInteger(state, decInit) {
		return nil
	}

	// validate incrementor exists and is _probably_ an integer change in `x`

	if incrementor == nil {
		return nil
	}

	stepValue, hasStepValue := getOptimizedIncrementorStepValue(state, incrementor, idSymbol)
	if !hasStepValue {
		return nil
	}

	// validate condition exists and is a BinaryExpression with an operator that matches the incrementor

	if condition == nil || !ast.IsBinaryExpression(condition) {
		return nil
	}

	conditionBinary := condition.AsBinaryExpression()
	operatorKind := conditionBinary.OperatorToken.Kind
	if operatorKind == ast.KindLessThanToken || operatorKind == ast.KindLessThanEqualsToken {
		// do not optimize for cases which should never run like:
		// for (let i = 10; i < 0; i--)
		if stepValue < 0 {
			return nil
		}
	} else if operatorKind == ast.KindGreaterThanToken || operatorKind == ast.KindGreaterThanEqualsToken {
		// do not optimize for cases which should never run like:
		// for (let i = 0; i > 10; i++)
		if stepValue > 0 {
			return nil
		}
	} else {
		// do not optimize for other comparison operators like !==, ===
		return nil
	}

	if !isProbablyInteger(state, conditionBinary.Right) {
		return nil
	}

	if isMutatedInBody(state, decName, statement) {
		return nil
	}

	// commit to the optimization and start transforming..

	result := luau.NewList()

	id := transformIdentifierDefined(state, decName)

	start, startPrereqs := Capture(state, func() luau.Expression { return transformExpression(state, decInit) })
	result.PushList(startPrereqs)

	end, endPrereqs := Capture(state, func() luau.Expression { return transformExpression(state, conditionBinary.Right) })
	result.PushList(endPrereqs)

	step := luau.NewNumber(stepValue)
	statements := transformStatementList(state, statement, getStatements(statement), nil)

	if operatorKind == ast.KindLessThanToken {
		end = offset(end, -1)
	} else if operatorKind == ast.KindGreaterThanToken {
		end = offset(end, 1)
	}

	result.Push(luau.Create(&luau.NumericForStatement{ID: id, Start: start, End: end, Step: step, Statements: statements}))

	return result
}

func transformForStatement(state *TransformState, node *ast.Node) *luau.List {
	if state.Data.ProjectOptions.OptimizedLoops {
		optimized := transformForStatementOptimized(state, node)
		if optimized != nil {
			return optimized
		}
	}
	return transformForStatementFallback(state, node)
}
