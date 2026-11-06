package plugins

// Port of out/transformer.js and out/util.js from rbxts-transformer-setget@2.0.0, which ships compiled, on the Go AST.
// Copyright (c) 2020, wAD4444, ISC license. The notice is in LICENSE-rbxts-transformer-setget next to this file.
//
// Keep the plugin's quirks: its output is what JS rbxtsc compiles. Most of them look like bugs, and they're on purpose
// here: `y = o.x` becomes `o.__setx(o.__getx())` when x has a setter, because visitBinaryExpression takes the first
// property access anywhere under the expression. A postfix operator reuses the untransformed object expression. Each
// compound assignment evaluates the object twice. See docs/setget.md.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

const (
	defaultPrefix = "__"
	setterPrefix  = "set"
	getterPrefix  = "get"
)

// transformContext is TransformContext, for one file. Checker is nil when no property access in the file can be an
// accessor, because then nothing asks it.
type transformContext struct {
	file    *ast.SourceFile
	emit    *printer.EmitContext
	factory *printer.NodeFactory
	visitor *ast.NodeVisitor
	checker *checker.Checker
	prefix  string
	names   *accessorNames
}

// pluginExit is process.exit(1) after writeLine: it unwinds out of the transform of the first file that hits it.
type pluginExit struct{ lines []string }

func newTransformContext(file *ast.SourceFile, c *checker.Checker, prefix string, names *accessorNames) *transformContext {
	emit := printer.NewEmitContext()
	context := &transformContext{file: file, emit: emit, factory: emit.Factory, checker: c, prefix: prefix, names: names}
	context.visitor = emit.NewNodeVisitor(context.visitNode)
	return context
}

func (c *transformContext) transformFile() *ast.SourceFile { return c.visitor.VisitSourceFile(c.file) }

func (c *transformContext) transform(node *ast.Node) *ast.Node { return c.visitor.VisitEachChild(node) }

func (c *transformContext) getText(node *ast.Node) string {
	return scanner.GetSourceTextOfNodeFromSourceFile(c.file, node, false)
}

// name makes the identifier that replaces original's name. The source map range does nothing to the printed text. It
// lets a diagnostic on the new name point at the old one.
func (c *transformContext) name(kind string, original *ast.Node) *ast.Node {
	identifier := c.factory.NewIdentifier(c.prefix + kind + c.getText(original))
	c.emit.SetSourceMapRange(identifier, original.Loc)
	return identifier
}

func (c *transformContext) mapped(node *ast.Node, original *ast.Node) *ast.Node {
	c.emit.SetSourceMapRange(node, original.Loc)
	return node
}

func (c *transformContext) visitPropertyAccessExpression(node *ast.Node) *ast.Node {
	getterDeclaration, setterDeclaration := c.getGetterSetterDeclarations(node)
	if getterDeclaration == nil && setterDeclaration == nil {
		return c.transform(node)
	}
	access := node.AsPropertyAccessExpression()
	assignmentExpression := getAncestorOfType(node, isAssignmentExpression)
	isSetter := setterDeclaration != nil && assignmentExpression != nil
	if isSetter {
		left := assignmentExpression.AsBinaryExpression().Left
		isSetter = node == left || isChildOfNode(left, node)
	}
	if !isSetter {
		updated := c.factory.UpdatePropertyAccessExpression(access, c.transform(node).Expression(), access.QuestionDotToken, c.name(getterPrefix, access.Name()), node.Flags)
		// createCallChain marks the call as a chain even without a ?. token. That's why JS writes (a.__getb()).c, with
		// parentheses that survive into the reparse: an access on a chain that isn't part of it needs them.
		return c.mapped(c.factory.NewCallExpression(updated, access.QuestionDotToken, c.factory.NewNodeList(nil), c.factory.NewNodeList(nil), ast.NodeFlagsOptionalChain), node)
	}
	return c.factory.UpdatePropertyAccessExpression(access, c.transform(node).Expression(), access.QuestionDotToken, c.name(setterPrefix, access.Name()), node.Flags)
}

var assignmentTokenLookup = map[ast.Kind]ast.Kind{
	ast.KindPlusEqualsToken:                              ast.KindPlusToken,
	ast.KindMinusEqualsToken:                             ast.KindMinusToken,
	ast.KindAsteriskEqualsToken:                          ast.KindAsteriskToken,
	ast.KindAsteriskAsteriskEqualsToken:                  ast.KindAsteriskAsteriskToken,
	ast.KindSlashEqualsToken:                             ast.KindSlashToken,
	ast.KindPercentEqualsToken:                           ast.KindPercentToken,
	ast.KindLessThanLessThanEqualsToken:                  ast.KindLessThanLessThanToken,
	ast.KindGreaterThanGreaterThanEqualsToken:            ast.KindGreaterThanGreaterThanToken,
	ast.KindGreaterThanGreaterThanGreaterThanEqualsToken: ast.KindGreaterThanGreaterThanGreaterThanToken,
	ast.KindAmpersandEqualsToken:                         ast.KindAmpersandToken,
	ast.KindBarEqualsToken:                               ast.KindBarToken,
	ast.KindBarBarEqualsToken:                            ast.KindBarBarToken,
	ast.KindAmpersandAmpersandEqualsToken:                ast.KindAmpersandAmpersandToken,
	ast.KindQuestionQuestionEqualsToken:                  ast.KindQuestionQuestionToken,
	ast.KindCaretEqualsToken:                             ast.KindCaretToken,
}

func (c *transformContext) newCall(expression, name *ast.Node, arguments []*ast.Node, original *ast.Node) *ast.Node {
	access := c.factory.NewPropertyAccessExpression(expression, nil, name, ast.NodeFlagsNone)
	var list *ast.NodeList
	list = c.factory.NewNodeList(arguments)
	return c.mapped(c.factory.NewCallExpression(c.mapped(access, original), nil, nil, list, ast.NodeFlagsNone), original)
}

func (c *transformContext) visitBinaryExpression(node *ast.Node) *ast.Node {
	propertyAccessExpression := getChildOfType(node, ast.IsPropertyAccessExpression)
	if propertyAccessExpression == nil {
		return c.transform(node)
	}
	getterDeclaration, setterDeclaration := c.getGetterSetterDeclarations(propertyAccessExpression)
	if setterDeclaration == nil {
		return c.transform(node)
	}
	original := propertyAccessExpression
	name := original.AsPropertyAccessExpression().Name()
	binary := node.AsBinaryExpression()
	operator := binary.OperatorToken.Kind

	if operator == ast.KindEqualsToken {
		expression := c.transform(original).Expression()
		right := c.transform(node).AsBinaryExpression().Right
		return c.newCall(expression, c.name(setterPrefix, name), []*ast.Node{right}, node)
	}
	if ast.IsAssignmentExpression(node, false) && ast.IsCompoundAssignment(operator) {
		if getterDeclaration == nil {
			c.exit(`Required getter declaration of "` + shared.Color.Yellow(c.getText(propertyAccessExpression)) + `" for compound assignment: ` + shared.Color.Yellow(c.getText(node)))
		}
		if len(getDescendantsOfType(original, ast.IsNewExpression)) > 0 {
			c.exit("Cannot compile a compound assignment with a new expression inside: "+shared.Color.Yellow(c.getText(node)), shared.Color.Yellow("Suggestion:")+" extract new expression into a variable.")
		}
		setExpression := c.transform(original).Expression()
		getExpression := c.transform(original).Expression()
		getter := c.newCall(getExpression, c.name(getterPrefix, name), nil, original)
		token := assignmentTokenLookup[operator]
		right := parenthesizeRightSideOfBinary(c.factory, token, c.transform(node).AsBinaryExpression().Right)
		sum := c.mapped(c.factory.NewBinaryExpression(nil, getter, nil, c.factory.NewToken(token), right), node)
		return c.newCall(setExpression, c.name(setterPrefix, name), []*ast.Node{sum}, node)
	}
	return c.transform(node)
}

var postfixUnaryOperatorLookup = map[ast.Kind]ast.Kind{
	ast.KindPlusPlusToken:   ast.KindPlusToken,
	ast.KindMinusMinusToken: ast.KindMinusToken,
}

func (c *transformContext) visitPostfixUnaryExpression(node *ast.Node) *ast.Node {
	propertyAccessExpression := getChildOfType(node, ast.IsPropertyAccessExpression)
	if propertyAccessExpression == nil {
		return c.transform(node)
	}
	getterDeclaration, setterDeclaration := c.getGetterSetterDeclarations(propertyAccessExpression)
	if setterDeclaration == nil {
		return c.transform(node)
	}
	original := propertyAccessExpression.AsPropertyAccessExpression()
	if getterDeclaration == nil {
		c.exit(`Required getter declaration of "` + shared.Color.Yellow(c.getText(propertyAccessExpression)) + `" for postfix unary expression: ` + shared.Color.Yellow(c.getText(node)))
	}
	if len(getDescendantsOfType(propertyAccessExpression, ast.IsNewExpression)) > 0 {
		c.exit("Cannot compile a postfix unary expression with a new expression inside: "+shared.Color.Yellow(c.getText(node)), shared.Color.Yellow("Suggestion:")+" extract new expression into a variable.")
	}
	getter := c.newCall(original.Expression, c.name(getterPrefix, original.Name()), nil, propertyAccessExpression)
	token := postfixUnaryOperatorLookup[node.AsPostfixUnaryExpression().Operator]
	one := c.factory.NewNumericLiteral("1", ast.TokenFlagsNone)
	sum := c.mapped(c.factory.NewBinaryExpression(nil, getter, nil, c.factory.NewToken(token), one), node)
	return c.newCall(original.Expression, c.name(setterPrefix, original.Name()), []*ast.Node{sum}, node)
}

func (c *transformContext) visitAccessor(node *ast.Node, kind string) *ast.Node {
	var data *ast.FunctionLikeBase
	var bodyData *ast.BodyBase
	var modifiers *ast.ModifierList
	if kind == setterPrefix {
		accessor := node.AsSetAccessorDeclaration()
		data, bodyData, modifiers = &accessor.FunctionLikeBase, &accessor.BodyBase, accessor.Modifiers()
	} else {
		accessor := node.AsGetAccessorDeclaration()
		data, bodyData, modifiers = &accessor.FunctionLikeBase, &accessor.BodyBase, accessor.Modifiers()
	}
	var method *ast.Node
	if node.Parent != nil && (ast.IsInterfaceDeclaration(node.Parent) || ast.IsTypeLiteralNode(node.Parent)) {
		// JS makes a MethodDeclaration here too, and prints it the same way as this. The Go printer only takes signatures
		// in an interface or a type literal.
		method = c.factory.NewMethodSignatureDeclaration(modifiers, c.name(kind, node.Name()), node.PostfixToken(), data.TypeParameters, data.Parameters, data.Type)
	} else {
		method = c.factory.NewMethodDeclaration(modifiers, nil, c.name(kind, node.Name()), node.PostfixToken(), data.TypeParameters, data.Parameters, data.Type, nil, bodyData.Body)
	}
	return c.transform(c.mapped(method, node))
}

func (c *transformContext) visitNode(node *ast.Node) *ast.Node {
	switch node.Kind {
	case ast.KindGetAccessor:
		return c.visitAccessor(node, getterPrefix)
	case ast.KindSetAccessor:
		return c.visitAccessor(node, setterPrefix)
	case ast.KindPropertyAccessExpression:
		return c.visitPropertyAccessExpression(node)
	case ast.KindBinaryExpression:
		return c.visitBinaryExpression(node)
	case ast.KindPostfixUnaryExpression:
		return c.visitPostfixUnaryExpression(node)
	}
	return c.transform(node)
}

func isAssignmentExpression(node *ast.Node) bool { return ast.IsAssignmentExpression(node, false) }

func getAncestorOfType(node *ast.Node, check func(*ast.Node) bool) *ast.Node {
	currentNode := node
	for currentNode.Parent != nil {
		parent := currentNode.Parent
		if check(parent) {
			return parent
		}
		if (ast.IsAsExpression(parent) || ast.IsParenthesizedExpression(parent)) && parent.Expression() == currentNode {
			currentNode = parent
			continue
		}
		break
	}
	return nil
}

func getChildOfType(node *ast.Node, check func(*ast.Node) bool) *ast.Node {
	var foundChild *ast.Node
	node.ForEachChild(func(child *ast.Node) bool {
		if check(child) {
			foundChild = child
		} else {
			foundChild = getChildOfType(child, check)
		}
		return foundChild != nil
	})
	return foundChild
}

// exit is writeLine, then process.exit(1).
func (c *transformContext) exit(messages ...string) {
	prefix := "[" + shared.Color.Grey("rbxts-transformer-setget") + "]: "
	lines := make([]string, len(messages))
	for i, message := range messages {
		lines[i] = prefix + strings.ReplaceAll(message, "\n", "\n"+prefix)
	}
	panic(&pluginExit{lines: lines})
}

func getDescendantsOfType(node *ast.Node, check func(*ast.Node) bool) []*ast.Node {
	var children []*ast.Node
	node.ForEachChild(func(child *ast.Node) bool {
		if check(child) {
			children = append(children, child)
		}
		children = append(children, getDescendantsOfType(child, check)...)
		return false
	})
	return children
}

func isChildOfNode(parent *ast.Node, node *ast.Node) bool {
	return parent.ForEachChild(func(child *ast.Node) bool {
		return child == node || isChildOfNode(child, node)
	})
}

// getGetterSetterDeclarations asks the checker only if some accessor in the program could go by this name. See
// accessorNames. That's what keeps the first program from being checked almost in full.
func (c *transformContext) getGetterSetterDeclarations(node *ast.Node) (getterDeclaration, setterDeclaration *ast.Node) {
	if !c.names.mayBeAccessor(node.AsPropertyAccessExpression().Name()) {
		return nil, nil
	}
	nodeSymbol := c.checker.GetSymbolAtLocation(node)
	if nodeSymbol == nil {
		return nil, nil
	}
	for _, declaration := range nodeSymbol.Declarations() {
		if ast.IsGetAccessorDeclaration(declaration) {
			getterDeclaration = declaration
		}
		if ast.IsSetAccessorDeclaration(declaration) {
			setterDeclaration = declaration
		}
		if getterDeclaration != nil && setterDeclaration != nil {
			break
		}
	}
	return getterDeclaration, setterDeclaration
}

// parenthesizeRightSideOfBinary is what ts.factory.createBinaryExpression does to its right operand. The patched
// printer leaves a parse tree node alone, so it has to happen here. The left operand is always a call. That rules out
// the case where `+` on two literals needs none.
func parenthesizeRightSideOfBinary(factory *printer.NodeFactory, operator ast.Kind, operand *ast.Node) *ast.Node {
	emitted := ast.SkipPartiallyEmittedExpressions(operand)
	if emitted.Kind == ast.KindParenthesizedExpression || !binaryOperandNeedsParentheses(operator, operand, emitted) {
		return operand
	}
	return factory.NewParenthesizedExpression(operand)
}

func binaryOperandNeedsParentheses(operator ast.Kind, operand, emitted *ast.Node) bool {
	precedence := binaryOperatorPrecedence(operator)
	isRightAssociative := operator == ast.KindAsteriskAsteriskToken
	if operand.Kind == ast.KindArrowFunction && precedence > 3 {
		return true
	}
	operandPrecedence := expressionPrecedence(emitted)
	switch {
	case operandPrecedence < precedence:
		return !(isRightAssociative && operand.Kind == ast.KindYieldExpression)
	case operandPrecedence > precedence:
		return false
	}
	if ast.IsBinaryExpression(emitted) && emitted.AsBinaryExpression().OperatorToken.Kind == operator {
		switch operator {
		case ast.KindAsteriskToken, ast.KindBarToken, ast.KindAmpersandToken, ast.KindCaretToken, ast.KindCommaToken:
			return false
		}
	}
	return !isExpressionRightAssociative(emitted)
}

// expressionPrecedence is getExpressionPrecedence from TS 5.5.3, in its numbers. ast.GetExpressionPrecedence can't stand
// in for it: there, ?? ranks above a conditional and a call ranks as a member, and here those change the result.
func expressionPrecedence(node *ast.Node) int {
	switch node.Kind {
	case ast.KindSpreadElement:
		return 1
	case ast.KindYieldExpression:
		return 2
	case ast.KindConditionalExpression:
		return 4
	case ast.KindBinaryExpression:
		return binaryOperatorPrecedence(node.AsBinaryExpression().OperatorToken.Kind)
	case ast.KindTypeAssertionExpression, ast.KindNonNullExpression, ast.KindPrefixUnaryExpression, ast.KindTypeOfExpression, ast.KindVoidExpression, ast.KindDeleteExpression, ast.KindAwaitExpression:
		return 16
	case ast.KindPostfixUnaryExpression:
		return 17
	case ast.KindCallExpression:
		return 18
	case ast.KindNewExpression:
		if node.ArgumentList() == nil {
			return 18
		}
		return 19
	case ast.KindTaggedTemplateExpression, ast.KindPropertyAccessExpression, ast.KindElementAccessExpression, ast.KindMetaProperty:
		return 19
	case ast.KindAsExpression, ast.KindSatisfiesExpression:
		return 11
	case ast.KindThisKeyword, ast.KindSuperKeyword, ast.KindIdentifier, ast.KindPrivateIdentifier, ast.KindNullKeyword, ast.KindTrueKeyword, ast.KindFalseKeyword, ast.KindNumericLiteral, ast.KindBigIntLiteral, ast.KindStringLiteral, ast.KindArrayLiteralExpression, ast.KindObjectLiteralExpression, ast.KindFunctionExpression, ast.KindArrowFunction, ast.KindClassExpression, ast.KindRegularExpressionLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindTemplateExpression, ast.KindParenthesizedExpression, ast.KindOmittedExpression, ast.KindJsxElement, ast.KindJsxSelfClosingElement, ast.KindJsxFragment:
		return 20
	}
	return -1
}

func binaryOperatorPrecedence(operator ast.Kind) int {
	switch operator {
	case ast.KindCommaToken:
		return 0
	case ast.KindQuestionQuestionToken:
		return 4
	case ast.KindBarBarToken:
		return 5
	case ast.KindAmpersandAmpersandToken:
		return 6
	case ast.KindBarToken:
		return 7
	case ast.KindCaretToken:
		return 8
	case ast.KindAmpersandToken:
		return 9
	case ast.KindEqualsEqualsToken, ast.KindExclamationEqualsToken, ast.KindEqualsEqualsEqualsToken, ast.KindExclamationEqualsEqualsToken:
		return 10
	case ast.KindLessThanToken, ast.KindGreaterThanToken, ast.KindLessThanEqualsToken, ast.KindGreaterThanEqualsToken, ast.KindInstanceOfKeyword, ast.KindInKeyword, ast.KindAsKeyword, ast.KindSatisfiesKeyword:
		return 11
	case ast.KindLessThanLessThanToken, ast.KindGreaterThanGreaterThanToken, ast.KindGreaterThanGreaterThanGreaterThanToken:
		return 12
	case ast.KindPlusToken, ast.KindMinusToken:
		return 13
	case ast.KindAsteriskToken, ast.KindSlashToken, ast.KindPercentToken:
		return 14
	case ast.KindAsteriskAsteriskToken:
		return 15
	}
	if ast.IsAssignmentOperator(operator) {
		return 3
	}
	return -1
}

// isExpressionRightAssociative is getExpressionAssociativity(node) === Right.
func isExpressionRightAssociative(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindNewExpression:
		return node.ArgumentList() == nil
	case ast.KindPrefixUnaryExpression, ast.KindTypeOfExpression, ast.KindVoidExpression, ast.KindDeleteExpression, ast.KindAwaitExpression, ast.KindConditionalExpression, ast.KindYieldExpression:
		return true
	case ast.KindBinaryExpression:
		operator := node.AsBinaryExpression().OperatorToken.Kind
		return operator == ast.KindAsteriskAsteriskToken || ast.IsAssignmentOperator(operator)
	}
	return false
}

// accessorNames is the set of names any accessor in the program goes by. A property access can resolve to an accessor
// only if its name is in here, so every other one skips the checker. Two things can break that link, and either one
// turns the filter off: a computed accessor name, whose value only the checker knows, and a mapped type with an `as`
// clause, where a property keeps the declarations of a member with another name.
type accessorNames struct {
	names map[string]bool
	all   bool
}

func (n *accessorNames) mayBeAccessor(name *ast.Node) bool { return n.all || n.names[name.Text()] }

// empty reports that nothing in the program can be an accessor, so no checker is needed at all.
func (n *accessorNames) empty() bool { return !n.all && len(n.names) == 0 }

// fileAccessorFacts is what one syntactic pass over a file finds.
type fileAccessorFacts struct {
	names          []string
	computed       bool
	mappedTypeName bool
}

func scanAccessors(file *ast.SourceFile) fileAccessorFacts {
	var facts fileAccessorFacts
	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		switch node.Kind {
		case ast.KindGetAccessor, ast.KindSetAccessor:
			name := node.Name()
			if name != nil {
				if ast.IsComputedPropertyName(name) {
					facts.computed = true
				} else {
					facts.names = append(facts.names, name.Text())
				}
			}
		case ast.KindMappedType:
			if node.AsMappedTypeNode().NameType != nil {
				facts.mappedTypeName = true
			}
		}
		return node.ForEachChild(walk)
	}
	file.AsNode().ForEachChild(walk)
	return facts
}

// fileUsesName reports whether a property access in file might be an accessor.
func fileUsesName(file *ast.SourceFile, names *accessorNames) bool {
	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		if node.Kind == ast.KindPropertyAccessExpression && names.mayBeAccessor(node.AsPropertyAccessExpression().Name()) {
			return true
		}
		return node.ForEachChild(walk)
	}
	return file.AsNode().ForEachChild(walk)
}
