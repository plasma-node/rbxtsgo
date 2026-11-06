package transformer

// Port of src/TSTransformer/nodes/statements/transformExportDeclaration.ts.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func isExportSpecifierValue(state *TransformState, element *ast.Node) bool {
	if element.AsExportSpecifier().IsTypeOnly {
		return false
	}

	if state.Resolver.IsReferencedAliasDeclaration(element) {
		return true
	}

	aliasSymbol := state.Checker.GetSymbolAtLocation(element.Name())
	if aliasSymbol != nil && isSymbolOfValue(state.Checker.SkipAlias(aliasSymbol)) {
		return true
	}

	return false
}

// exportCountImportExpUses is countImportExpUses. transformImportDeclaration.ts has its own function of that name.
func exportCountImportExpUses(state *TransformState, exportClause *ast.Node) int {
	uses := 0
	if exportClause != nil && ast.IsNamedExports(exportClause) {
		for _, element := range exportClause.AsNamedExports().Elements.Nodes {
			if isExportSpecifierValue(state, element) {
				uses++
			}
		}
	} else {
		uses++
	}
	return uses
}

func moduleHasExportEquals(state *TransformState, moduleSpecifier *ast.Node) bool {
	symbol := state.Checker.GetSymbolAtLocation(moduleSpecifier)
	if symbol == nil {
		return false
	}
	_, ok := symbol.Exports()[ast.InternalSymbolNameExportEquals]
	return ok
}

func transformExportFrom(state *TransformState, node *ast.Node) *luau.List {
	declaration := node.AsExportDeclaration()
	moduleSpecifier := declaration.ModuleSpecifier
	if moduleSpecifier == nil || !ast.IsStringLiteral(moduleSpecifier) {
		panic(shared.AssertionMessage(""))
	}

	statements := luau.NewList()
	var importExp luau.IndexableExpression

	exportClause := declaration.ExportClause

	// detect if we need to push to a new var or not
	uses := exportCountImportExpUses(state, exportClause)
	if uses == 1 {
		importExp = createImportExpression(state, ast.GetSourceFileOfNode(node), moduleSpecifier)
	} else if uses > 1 {
		moduleName := strings.Split(moduleSpecifier.Text(), "/")
		importExp = luau.NewTempID(cleanModuleName(moduleName[len(moduleName)-1]))
		statements.Push(luau.Create(&luau.VariableDeclaration{
			Left:  importExp,
			Right: createImportExpression(state, ast.GetSourceFileOfNode(node), moduleSpecifier),
		}))
	}

	if importExp == nil {
		return statements
	}

	moduleID := state.GetModuleIDFromNode(node)
	if exportClause != nil {
		if ast.IsNamedExports(exportClause) {
			// export { a, b, c } from "./module";
			isExportEquals := state.Data.ProjectOptions.FixEmit && moduleHasExportEquals(state, moduleSpecifier)
			for _, element := range exportClause.AsNamedExports().Elements.Nodes {
				if isExportSpecifierValue(state, element) {
					var value luau.Expression
					if importName := element.PropertyNameOrName().Text(); isExportEquals && importName == "default" {
						// the module has no default: the module is what it exports. From upstream's open PR #3115.
						value = importExp
					} else {
						value = luau.NewProperty(importExp, importName)
					}
					statements.Push(luau.Create(&luau.Assignment{
						Left:     luau.NewProperty(moduleID, element.Name().Text()),
						Operator: luau.AssignEqual,
						Right:    value,
					}))
				}
			}
		} else {
			// export * as foo from "./module";
			statements.Push(luau.Create(&luau.Assignment{
				Left:     luau.NewProperty(moduleID, exportClause.Name().Text()),
				Operator: luau.AssignEqual,
				Right:    importExp,
			}))
		}
	} else {
		// export * from "./module";
		keyID := luau.NewTempID("k")
		valueID := luau.NewTempID("v")
		statements.Push(luau.Create(&luau.ForStatement{
			IDs: luau.MakeList(keyID, valueID),
			// importExp may be `nil` in .d.ts files, so default to `{}`
			// boolean `or` is safe, because importExp can only be a table if not `nil`
			Expression: luau.NewBinary(importExp, luau.BinaryOr, luau.NewMap()),
			Statements: luau.MakeList(
				luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: moduleID,
						Index:      keyID,
					}),
					Operator: luau.AssignEqual,
					Right:    valueID,
				}),
			),
		}))
	}

	state.HasExportFrom = true

	return statements
}

func transformExportDeclaration(state *TransformState, node *ast.Node) *luau.List {
	declaration := node.AsExportDeclaration()
	if declaration.IsTypeOnly {
		return luau.NewList()
	}

	if declaration.ModuleSpecifier != nil {
		return transformExportFrom(state, node)
	}

	return luau.NewList()
}
