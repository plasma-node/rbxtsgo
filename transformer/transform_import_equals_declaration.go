package transformer

// Port of src/TSTransformer/nodes/statements/transformImportEqualsDeclaration.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformImportEqualsDeclaration(state *TransformState, node *ast.Node) *luau.List {
	moduleReference := node.AsImportEqualsDeclaration().ModuleReference
	if ast.IsExternalModuleReference(moduleReference) {
		expression := moduleReference.AsExternalModuleReference().Expression
		if !ast.IsStringLiteral(expression) {
			panic(shared.AssertionMessage(""))
		}
		importExp := createImportExpression(state, ast.GetSourceFileOfNode(node), expression)

		statements := luau.NewList()

		aliasSymbol := state.Checker.GetSymbolAtLocation(node.Name())
		if aliasSymbol == nil {
			panic(shared.AssertionMessage(""))
		}
		if isSymbolOfValue(state.Checker.SkipAlias(aliasSymbol)) {
			statements.PushList(state.CapturePrereqs(func() {
				transformVariable(state, node.Name(), importExp)
			}))
		}

		// ensure we emit something
		if state.Options.VerbatimModuleSyntax.IsTrue() && statements.IsEmpty() {
			if call, ok := importExp.(*luau.CallExpression); ok {
				statements.Push(luau.Create(&luau.CallStatement{Expression: call}))
			}
		}

		return statements
	}
	// Identifier | QualifiedName
	// see: https://github.com/roblox-ts/roblox-ts/issues/1895
	return state.CapturePrereqs(func() {
		transformVariable(state, node.Name(), transformEntityName(state, moduleReference))
	})
}
