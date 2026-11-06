package transformer

// Port of src/TSTransformer/nodes/expressions/transformIdentifier.ts.

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformIdentifierDefined(state *TransformState, node *ast.Node) luau.AnyIdentifier {
	var symbol *ast.Symbol
	if ast.IsShorthandPropertyAssignment(node.Parent) {
		symbol = state.Checker.GetShorthandAssignmentValueSymbol(node.Parent)
	} else {
		symbol = state.Checker.GetSymbolAtLocation(node)
	}
	if symbol == nil {
		panic(shared.AssertionMessage(""))
	}

	replacementId := state.SymbolToIDMap[symbol]
	if replacementId != nil {
		return replacementId
	}

	return luau.Create(&luau.Identifier{
		Name: node.Text(),
	})
}

func getAncestorWhichIsChildOf(parent *ast.Node, node *ast.Node) *ast.Node {
	for node.Parent != nil && node.Parent != parent {
		node = node.Parent
	}
	if node.Parent != nil {
		return node
	}
	return nil
}

// for some reason, symbol.valueDeclaration doesn't point to imports?
func getDeclarationFromImport(symbol *ast.Symbol) *ast.Node {
	for _, declaration := range symbol.Declarations() {
		importDec := GetAncestor(declaration, ast.IsAnyImportSyntax)
		if importDec != nil {
			return declaration
		}
	}
	return nil
}

func checkIdentifierHoist(state *TransformState, node *ast.Node, symbol *ast.Symbol) {
	if _, ok := state.IsHoisted[symbol]; ok {
		return
	}

	declaration := symbol.ValueDeclaration()
	if declaration == nil {
		declaration = getDeclarationFromImport(symbol)
	}

	// parameters cannot be hoisted
	if declaration == nil || GetAncestor(declaration, ast.IsParameterDeclaration) != nil || ast.IsShorthandPropertyAssignment(declaration) {
		return
	}

	// class expressions can self refer
	if ast.IsClassLike(declaration) && isAncestorOf(declaration, node) {
		return
	}

	declarationStatement := GetAncestor(declaration, ast.IsStatement)
	if declarationStatement == nil ||
		ast.IsForStatement(declarationStatement) ||
		ast.IsForOfStatement(declarationStatement) ||
		ast.IsTryStatement(declarationStatement) {
		return
	}

	parent := declarationStatement.Parent
	if parent == nil || !isBlockLike(parent) {
		return
	}

	sibling := getAncestorWhichIsChildOf(parent, node)
	if sibling == nil || !ast.IsStatement(sibling) {
		return
	}

	declarationIdx := slices.Index(parent.Statements(), declarationStatement)
	siblingIdx := slices.Index(parent.Statements(), sibling)

	// With fixEmit, these compile before everything else in the block. See hoistFunctionsPastFinalStatement.
	if state.hoistedPastFinal[declarationStatement] && !state.hoistedPastFinal[sibling] {
		return
	}
	if state.hoistedPastFinal[sibling] && !state.hoistedPastFinal[declarationStatement] {
		siblingIdx = -1
	}

	if siblingIdx > declarationIdx {
		return
	}

	if siblingIdx == declarationIdx {
		// non-async function declarations, class declarations, and variable statements can self refer
		if (ast.IsFunctionDeclaration(declarationStatement) &&
			!ast.HasSyntacticModifier(declarationStatement, ast.ModifierFlagsAsync)) ||
			ast.IsClassDeclaration(declarationStatement) ||
			(ast.IsVariableStatement(declarationStatement) &&
				GetAncestor(node, func(node *ast.Node) bool { return ast.IsStatement(node) || ast.IsFunctionLikeDeclaration(node) }) ==
					declarationStatement) {
			return
		}
	}

	state.HoistsByStatement[sibling] = append(state.HoistsByStatement[sibling], node)
	state.IsHoisted[symbol] = true
}

func transformIdentifier(state *TransformState, node *ast.Node) luau.Expression {
	// synthetic nodes don't have parents or symbols, so skip all the symbol-related logic
	// JSX EntityName functions like `getJsxFactoryEntity()` will return synthetic nodes
	// and transformEntityName will eventually end up here
	if node.Parent == nil || ast.PositionIsSynthesized(node.Pos()) {
		return luau.Create(&luau.Identifier{Name: node.Text()})
	}

	var symbol *ast.Symbol
	if ast.IsShorthandPropertyAssignment(node.Parent) {
		symbol = state.Checker.GetShorthandAssignmentValueSymbol(node.Parent)
	} else {
		symbol = state.Checker.GetSymbolAtLocation(node)
	}
	if symbol == nil {
		panic(shared.AssertionMessage(""))
	}

	if state.Checker.IsUndefinedSymbol(symbol) {
		return luau.NewNil()
	} else if state.Checker.IsArgumentsSymbol(symbol) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoArguments.At(shared.AtNode(node)))
	} else if symbol == getSymbolOrThrow(state, "globalThis") {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoGlobalThis.At(shared.AtNode(node)))
	}

	macro := getIdentifierMacro(state, symbol)
	if macro != nil {
		return macro(state, node)
	}

	constructSymbol := getFirstConstructSymbol(state, node)
	if constructSymbol != nil {
		constructorMacro := getConstructorMacro(state, constructSymbol)
		if constructorMacro != nil {
			// ts.isClassLike accepts undefined and ast.IsClassLike doesn't. It's undefined for `Foo;` at the top of a file.
			classLike := node.Parent.Parent.Parent
			isClassExtendsNode := false
			if classLike != nil && ast.IsClassLike(classLike) {
				extendsNode := getExtendsNode(classLike)
				isClassExtendsNode = extendsNode != nil && extendsNode.Expression() == node
			}
			if isClassExtendsNode {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoMacroExtends.At(shared.AtNode(node)))
			} else {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoConstructorMacroWithoutNew.At(shared.AtNode(node)))
			}
		}
	}

	parent := SkipUpwards(node).Parent
	if (!ast.IsCallExpression(parent) || SkipDownwards(parent.Expression()) != node) &&
		getCallMacro(state, symbol) != nil {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoIndexWithoutCall.At(shared.AtNode(node)))
		return luau.NewNone()
	}

	// exit here for export let so we don't check hoist later
	if symbol.ValueDeclaration() != nil &&
		ast.GetSourceFileOfNode(symbol.ValueDeclaration()) == ast.GetSourceFileOfNode(node) &&
		GetAncestor(symbol.ValueDeclaration(), func(node *ast.Node) bool { return ast.IsModuleDeclaration(node) && !isNamespace(node) }) == nil {
		exportAccess := state.GetModuleIDPropertyAccess(symbol)
		if exportAccess != nil && isSymbolMutable(state, symbol) {
			return exportAccess
		}
	}

	checkIdentifierHoist(state, node, symbol)

	return transformIdentifierDefined(state, node)
}
