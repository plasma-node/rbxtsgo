package transformer

// Port of src/TSTransformer/nodes/statements/transformImportDeclaration.ts.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

// importCountImportExpUses is countImportExpUses. transformExportDeclaration.ts has its own function of that name.
func importCountImportExpUses(state *TransformState, importClause *ast.Node) int {
	uses := 0

	if importClause.Name() != nil {
		symbol := getOriginalSymbolOfNode(state.Checker, importClause.Name())
		if state.Resolver.IsReferencedAliasDeclaration(importClause) && (symbol == nil || isSymbolOfValue(symbol)) {
			uses++
		}
	}

	if namedBindings := importClause.AsImportClause().NamedBindings; namedBindings != nil {
		if ast.IsNamespaceImport(namedBindings) {
			uses++
		} else {
			for _, element := range namedBindings.AsNamedImports().Elements.Nodes {
				symbol := getOriginalSymbolOfNode(state.Checker, element.Name())
				if state.Resolver.IsReferencedAliasDeclaration(element) && (symbol == nil || isSymbolOfValue(symbol)) {
					uses++
				}
			}
		}
	}

	return uses
}

func transformImportDeclaration(state *TransformState, node *ast.Node) *luau.List {
	declaration := node.AsImportDeclaration()

	// no emit for type only
	importClause := declaration.ImportClause
	if importClause != nil && importClause.IsTypeOnly() {
		return luau.NewList()
	}

	statements := luau.NewList()

	moduleSpecifier := declaration.ModuleSpecifier
	if !ast.IsStringLiteral(moduleSpecifier) {
		panic(shared.AssertionMessage(""))
	}
	// upstream's Lazy
	var importExpValue luau.IndexableExpression
	importExpIsInitialized := false
	getImportExp := func() luau.IndexableExpression {
		if !importExpIsInitialized {
			importExpIsInitialized = true
			importExpValue = createImportExpression(state, ast.GetSourceFileOfNode(node), moduleSpecifier)
		}
		return importExpValue
	}

	if importClause != nil {
		// detect if we need to push to a new var or not
		uses := importCountImportExpUses(state, importClause)
		if uses > 1 {
			moduleName := strings.Split(moduleSpecifier.Text(), "/")
			id := luau.NewTempID(cleanModuleName(moduleName[len(moduleName)-1]))
			statements.Push(luau.Create(&luau.VariableDeclaration{
				Left:  id,
				Right: getImportExp(),
			}))
			importExpIsInitialized = true
			importExpValue = id
		}

		// default import logic
		importClauseName := importClause.Name()
		if importClauseName != nil {
			symbol := getOriginalSymbolOfNode(state.Checker, importClauseName)
			if state.Resolver.IsReferencedAliasDeclaration(importClause) && (symbol == nil || isSymbolOfValue(symbol)) {
				moduleFile := getSourceFileFromModuleSpecifier(state, moduleSpecifier)
				var moduleSymbol *ast.Symbol
				if moduleFile != nil {
					moduleSymbol = state.Checker.GetSymbolAtLocation(moduleFile.AsNode())
				}
				hasDefaultExport := false
				if moduleSymbol != nil {
					for _, v := range state.GetModuleExports(moduleSymbol) {
						if v.Name() == "default" {
							hasDefaultExport = true
							break
						}
					}
				}
				if hasDefaultExport {
					statements.PushList(state.CapturePrereqs(func() {
						transformVariable(state, importClauseName, luau.NewProperty(getImportExp(), "default"))
					}))
				} else {
					statements.PushList(state.CapturePrereqs(func() {
						transformVariable(state, importClauseName, getImportExp())
					}))
				}
			}
		}

		importClauseNamedBindings := importClause.AsImportClause().NamedBindings
		if importClauseNamedBindings != nil {
			// namespace import logic
			if ast.IsNamespaceImport(importClauseNamedBindings) {
				statements.PushList(state.CapturePrereqs(func() {
					transformVariable(state, importClauseNamedBindings.Name(), getImportExp())
				}))
			} else {
				// named elements import logic
				for _, element := range importClauseNamedBindings.AsNamedImports().Elements.Nodes {
					symbol := getOriginalSymbolOfNode(state.Checker, element.Name())
					// check that import is referenced and has a value at runtime
					if state.Resolver.IsReferencedAliasDeclaration(element) && (symbol == nil || isSymbolOfValue(symbol)) {
						statements.PushList(state.CapturePrereqs(func() {
							transformVariable(
								state,
								element.Name(),
								luau.NewProperty(getImportExp(), element.PropertyNameOrName().Text()),
							)
						}))
					}
				}
			}
		}
	}

	// ensure we emit something
	if importClause == nil || (state.Options.VerbatimModuleSyntax.IsTrue() && statements.IsEmpty()) {
		expression := getImportExp()
		if call, ok := expression.(*luau.CallExpression); ok {
			statements.Push(luau.Create(&luau.CallStatement{Expression: call}))
		}
	}

	return statements
}
