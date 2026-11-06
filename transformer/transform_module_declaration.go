package transformer

// Port of src/TSTransformer/nodes/statements/transformModuleDeclaration.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func isDeclarationOfNamespace(declaration *ast.Node) bool {
	if ast.CanHaveModifiers(declaration) && hasDeclareModifier(declaration) {
		return false
	}

	if ast.IsModuleDeclaration(declaration) && ast.IsInstantiatedModule(declaration, false) {
		return true
	} else if ast.IsFunctionDeclaration(declaration) && declaration.Body() != nil {
		return true
	} else if ast.IsClassDeclaration(declaration) {
		return true
	}
	return false
}

func getValueDeclarationStatement(symbol *ast.Symbol) *ast.Node {
	for _, declaration := range symbol.Declarations() {
		statement := GetAncestor(declaration, ast.IsStatement)
		if statement != nil {
			if ast.IsFunctionDeclaration(statement) && statement.Body() == nil {
				continue
			}
			if ast.IsTypeAliasDeclaration(statement) {
				continue
			}
			if ast.IsInterfaceDeclaration(statement) {
				continue
			}
			if ast.CanHaveModifiers(statement) && hasDeclareModifier(statement) {
				continue
			}
			return statement
		}
	}
	return nil
}

func transformNamespace(state *TransformState, name *ast.Node, body *ast.Node) *luau.List {
	symbol := state.Checker.GetSymbolAtLocation(name)
	if symbol == nil {
		panic(shared.AssertionMessage(""))
	}

	validateIdentifier(state, name)

	nameExp := transformIdentifierDefined(state, name)

	statements := luau.NewList()
	doStatements := luau.NewList()

	containerID := luau.NewTempID("container")
	state.SetModuleIDBySymbol(symbol, containerID)

	if state.IsHoisted[symbol] {
		statements.Push(luau.Create(&luau.Assignment{
			Left:     nameExp,
			Operator: luau.AssignEqual,
			Right:    luau.NewMap(),
		}))
	} else {
		statements.Push(luau.Create(&luau.VariableDeclaration{
			Left:  nameExp,
			Right: luau.NewMap(),
		}))
	}

	moduleExports := state.GetModuleExports(symbol)
	if len(moduleExports) > 0 {
		doStatements.Push(luau.Create(&luau.VariableDeclaration{Left: containerID, Right: nameExp}))
	}

	if ast.IsModuleBlock(body) {
		exportsMap := map[*ast.Node][]string{}
		if len(moduleExports) > 0 {
			for _, exportSymbol := range moduleExports {
				originalSymbol := state.Checker.SkipAlias(exportSymbol)
				if isSymbolOfValue(originalSymbol) && !isSymbolMutable(state, originalSymbol) {
					valueDeclarationStatement := getValueDeclarationStatement(exportSymbol)
					if valueDeclarationStatement != nil {
						exportsMap[valueDeclarationStatement] = append(exportsMap[valueDeclarationStatement], exportSymbol.Name())
					}
				}
			}
		}
		doStatements.PushList(
			transformStatementList(state, body, body.AsModuleBlock().Statements.Nodes, &statementListExportInfo{
				id:      containerID,
				mapping: exportsMap,
			}),
		)
	} else {
		doStatements.PushList(transformNamespace(state, body.Name(), body.Body()))
		doStatements.Push(luau.Create(&luau.Assignment{
			Left:     luau.NewProperty(containerID, body.Name().Text()),
			Operator: luau.AssignEqual,
			Right:    transformIdentifierDefined(state, body.Name()),
		}))
	}

	statements.Push(luau.Create(&luau.DoStatement{Statements: doStatements}))

	return statements
}

func transformModuleDeclaration(state *TransformState, node *ast.Node) *luau.List {
	// type-only namespace
	if !ast.IsInstantiatedModule(node, false) {
		return luau.NewList()
	}

	// disallow merging
	symbol := state.Checker.GetSymbolAtLocation(node.Name())
	if symbol != nil && hasMultipleDefinitions(symbol, func(declaration *ast.Node) bool { return isDeclarationOfNamespace(declaration) }) {
		shared.AddDiagnosticWithCache(
			state.Diagnostic,
			symbol,
			shared.Errors.NoNamespaceMerging.At(shared.AtNode(node)),
			state.Multi.IsReportedByMultipleDefinitionsCache,
		)
		return luau.NewList()
	}

	// ts.StringLiteral is only in the case of `declare module "X" {}`? Should be filtered out above
	if ast.IsStringLiteral(node.Name()) {
		panic(shared.AssertionMessage(""))
	}
	body := node.Body()
	if body == nil || ast.IsIdentifier(body) {
		panic(shared.AssertionMessage(""))
	}
	// unsure how to filter out ts.JSDocNamespaceBody
	return transformNamespace(state, node.Name(), body)
}
