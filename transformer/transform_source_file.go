package transformer

// Port of src/TSTransformer/nodes/transformSourceFile.ts.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/rojoresolver"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

type exportPair struct {
	name string
	id   luau.AnyIdentifier
}

func getExportPair(state *TransformState, exportSymbol *ast.Symbol) exportPair {
	declarations := exportSymbol.Declarations()
	var declaration *ast.Node
	if len(declarations) > 0 {
		declaration = declarations[0]
	}
	if declaration != nil && ast.IsExportSpecifier(declaration) {
		specifier := declaration.AsExportSpecifier()
		propertyName := specifier.PropertyName
		if propertyName == nil {
			propertyName = declaration.Name()
		}
		return exportPair{declaration.Name().Text(), transformIdentifierDefined(state, propertyName)}
	}
	name := exportSymbol.Name()
	if exportSymbol.Name() == "default" && declaration != nil &&
		(ast.IsFunctionDeclaration(declaration) || ast.IsClassDeclaration(declaration)) && declaration.Name() != nil {
		name = declaration.Name().Text()
	}
	return exportPair{exportSymbol.Name(), luau.NewID(name)}
}

func isExportSymbolFromExportFrom(exportSymbol *ast.Symbol) bool {
	for _, exportSpecifier := range exportSymbol.Declarations() {
		if ast.IsExportSpecifier(exportSpecifier) {
			exportDec := exportSpecifier.Parent.Parent
			if ast.IsExportDeclaration(exportDec) && exportDec.AsExportDeclaration().ModuleSpecifier != nil {
				return true
			}
		}
	}
	return false
}

func getIgnoredExportSymbols(state *TransformState, sourceFile *ast.SourceFile) map[*ast.Symbol]bool {
	ignoredSymbols := map[*ast.Symbol]bool{}
	for _, statement := range sourceFile.Statements.Nodes {
		if !ast.IsExportDeclaration(statement) {
			continue
		}
		declaration := statement.AsExportDeclaration()
		if declaration.ModuleSpecifier == nil {
			continue
		}
		if declaration.ExportClause == nil {
			// export * from "./module";
			moduleSymbol := getOriginalSymbolOfNode(state.Checker, declaration.ModuleSpecifier)
			if moduleSymbol != nil {
				for _, v := range state.GetModuleExports(moduleSymbol) {
					ignoredSymbols[v] = true
				}
			}
		} else if ast.IsNamespaceExport(declaration.ExportClause) {
			// export * as id from "./module";
			idSymbol := state.Checker.GetSymbolAtLocation(declaration.ExportClause.Name())
			if idSymbol != nil {
				ignoredSymbols[idSymbol] = true
			}
		}
	}
	return ignoredSymbols
}

// isExportSymbolOnlyFromDeclare is used to ignore exports in the form of `export declare const x: T;`. However, this
// should still allow exports which are declare + export separately, i.e.
//
//	declare const x: number;
//	export { x };
//
// This mimics TypeScript behavior.
func isExportSymbolOnlyFromDeclare(exportSymbol *ast.Symbol) bool {
	for _, declaration := range exportSymbol.Declarations() {
		statement := GetAncestor(declaration, ast.IsStatement)
		if statement == nil || !ast.CanHaveModifiers(statement) || !hasDeclareModifier(statement) {
			return false
		}
	}
	return true
}

func hasDeclareModifier(node *ast.Node) bool {
	for _, modifier := range node.ModifierNodes() {
		if modifier.Kind == ast.KindDeclareKeyword {
			return true
		}
	}
	return false
}

// handleExports adds export information to the end of the tree.
func handleExports(state *TransformState, sourceFile *ast.SourceFile, symbol *ast.Symbol, statements *luau.List) {
	ignoredExportSymbols := getIgnoredExportSymbols(state, sourceFile)

	mustPushExports := state.HasExportFrom
	var exportPairs []exportPair
	if !state.HasExportEquals {
		for _, exportSymbol := range state.GetModuleExports(symbol) {
			if ignoredExportSymbols[exportSymbol] {
				continue
			}

			// ignore prototype exports
			if exportSymbol.Flags()&ast.SymbolFlagsPrototype != 0 {
				continue
			}

			// export { default as x } from "./module";
			if isExportSymbolFromExportFrom(exportSymbol) {
				continue
			}

			originalSymbol := state.Checker.SkipAlias(exportSymbol)

			// only export values
			if !isSymbolOfValue(originalSymbol) {
				continue
			}

			// handle this in transformIdentifier
			if isSymbolMutable(state, originalSymbol) {
				mustPushExports = true
				continue
			}

			// ignore exports in the form of `export declare const x: T;`
			if isExportSymbolOnlyFromDeclare(exportSymbol) {
				continue
			}

			exportPairs = append(exportPairs, getExportPair(state, exportSymbol))
		}
	}

	if state.HasExportEquals {
		// local exports variable is created in transformExportAssignment
		nodes := sourceFile.Statements.Nodes
		finalStatement := nodes[len(nodes)-1]
		if !(ast.IsExportAssignment(finalStatement) && finalStatement.AsExportAssignment().IsExportEquals) {
			statements.Push(luau.Create(&luau.ReturnStatement{Expression: luau.Globals.Exports}))
		}
	} else if mustPushExports {
		// if there's an export let/from, we need to put `local exports = {}` at the top of the file
		statements.Unshift(luau.Create(&luau.VariableDeclaration{Left: luau.Globals.Exports, Right: luau.NewMap()}))
		for _, pair := range exportPairs {
			statements.Push(luau.Create(&luau.Assignment{
				Left:     luau.NewProperty(luau.Globals.Exports, pair.name),
				Operator: luau.AssignEqual,
				Right:    pair.id,
			}))
		}
		statements.Push(luau.Create(&luau.ReturnStatement{Expression: luau.Globals.Exports}))
	} else if len(exportPairs) > 0 {
		// only regular exports, we can do this as just returning an object at the bottom of the file
		fields := luau.NewList()
		for _, pair := range exportPairs {
			fields.Push(luau.Create(&luau.MapField{Index: luau.NewString(pair.name), Value: pair.id}))
		}
		statements.Push(luau.Create(&luau.ReturnStatement{Expression: luau.Create(&luau.Map{Fields: fields})}))
	}
}

func getLastNonCommentStatement(listNode *luau.ListNode) *luau.ListNode {
	for listNode != nil && luau.IsComment(listNode.Value) {
		listNode = listNode.Prev
	}
	return listNode
}

// TransformSourceFile is transformSourceFile. It creates and returns a list of Luau statements.
func TransformSourceFile(state *TransformState, node *ast.SourceFile) *luau.List {
	symbol := state.Checker.GetSymbolAtLocation(node.AsNode())
	if symbol == nil {
		panic(shared.AssertionMessage(""))
	}
	state.SetModuleIDBySymbol(symbol, luau.Globals.Exports)

	// transform the `ts.Statements` of the source file into a `list.list<...>`
	statements := transformStatementList(state, node.AsNode(), node.Statements.Nodes, nil)

	handleExports(state, node, symbol, statements)

	// moduleScripts must `return nil` if they do not export any values
	lastStatement := getLastNonCommentStatement(statements.Tail)
	if lastStatement == nil || !luau.IsReturnStatement(lastStatement.Value) {
		outputPath := state.PathTranslator.GetOutputPath(nodepath.Native.Normalize(node.FileName().AsString()))
		if state.RojoResolver().GetRbxTypeFromFilePath(outputPath) == rojoresolver.RbxTypeModuleScript {
			statements.Push(luau.Create(&luau.ReturnStatement{Expression: luau.NewNil()}))
		}
	}

	headerStatements := luau.NewList()

	// add build information to the tree
	headerStatements.Push(luau.NewComment(" Compiled with roblox-ts v" + shared.CompilerVersion))

	// add the Runtime library to the tree if it is used
	if state.UsesRuntimeLib {
		headerStatements.Push(state.CreateRuntimeLibImport(node))
	}

	// extract Luau directive comments like --!strict so we can put them before headerStatements
	directiveComments := luau.NewList()
	for statements.Head != nil && luau.IsComment(statements.Head.Value) && strings.HasPrefix(statements.Head.Value.(*luau.Comment).Text, "!") {
		directiveComments.Push(statements.Shift())
	}

	statements.UnshiftList(headerStatements)
	statements.UnshiftList(directiveComments)

	return statements
}
