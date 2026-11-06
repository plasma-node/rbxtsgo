package transformer

// Port of src/TSTransformer/nodes/statements/transformFunctionDeclaration.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformFunctionDeclaration(state *TransformState, node *ast.Node) *luau.List {
	declaration := node.AsFunctionDeclaration()
	if declaration.Body == nil {
		return luau.NewList()
	}

	isExportDefault := ast.HasSyntacticModifier(node, ast.ModifierFlagsExportDefault)

	nodeName := node.Name()
	if !(nodeName != nil || isExportDefault) {
		panic(shared.AssertionMessage(""))
	}

	if nodeName != nil {
		validateIdentifier(state, nodeName)
	}

	var name luau.AnyIdentifier
	if nodeName != nil {
		name = transformIdentifierDefined(state, nodeName)
	} else {
		name = luau.NewID("default")
	}

	transformed := transformParameters(state, node)
	statements, parameters, hasDotDotDot := transformed.statements, transformed.parameters, transformed.hasDotDotDot
	statements.PushList(transformStatementList(state, declaration.Body, declaration.Body.AsBlock().Statements.Nodes, nil))

	localize := isExportDefault
	if nodeName != nil {
		symbol := state.Checker.GetSymbolAtLocation(nodeName)
		if symbol == nil {
			panic(shared.AssertionMessage(""))
		}
		localize = !state.IsHoisted[symbol]
	}

	isAsync := ast.HasSyntacticModifier(node, ast.ModifierFlagsAsync)

	if declaration.AsteriskToken != nil {
		if isAsync {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoAsyncGeneratorFunctions.At(shared.AtNode(node)))
		}
		statements = wrapStatementsAsGenerator(state, node, statements)
	}

	if isAsync {
		right := luau.NewCall(state.TS(node, "async"), luau.Create(&luau.FunctionExpression{
			HasDotDotDot: hasDotDotDot,
			Parameters:   parameters,
			Statements:   statements,
		}))
		if localize {
			return luau.MakeList(luau.Create(&luau.VariableDeclaration{
				Left:  name,
				Right: right,
			}))
		} else {
			return luau.MakeList(luau.Create(&luau.Assignment{
				Left:     name,
				Operator: luau.AssignEqual,
				Right:    right,
			}))
		}
	} else {
		return luau.MakeList(luau.Create(&luau.FunctionDeclaration{
			Localize:     localize,
			Name:         name.(luau.WritableExpression),
			Statements:   statements,
			Parameters:   parameters,
			HasDotDotDot: hasDotDotDot,
		}))
	}
}
