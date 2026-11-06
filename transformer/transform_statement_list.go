package transformer

// Port of src/TSTransformer/nodes/transformStatementList.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/ls/lsutil"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func getLastToken(state *TransformState, parent *ast.Node, statements []*ast.Node) *ast.Node {
	if len(statements) > 0 {
		lastStatement := statements[len(statements)-1]
		lastToken := lsutil.GetLastToken(lastStatement.Parent, state.SourceFile)
		if lastToken != nil && !ast.IsNodeDescendantOf(lastToken, lastStatement) {
			return lastToken
		}
	} else if parent != nil {
		// if statements is empty, there still might be an `EOF` or `}` token to look for
		return lsutil.GetLastToken(parent, state.SourceFile)
	}
	return nil
}

// statementListExportInfo is the exportInfo argument of transformStatementList: information about exporting.
type statementListExportInfo struct {
	id      luau.AnyIdentifier
	mapping map[*ast.Node][]string
}

// transformStatementList converts a ts.Statement array into a list of Luau statements. exportInfo can be nil.
func transformStatementList(state *TransformState, parent *ast.Node, statements []*ast.Node, exportInfo *statementListExportInfo) *luau.List {
	// make a new Luau tree
	result := luau.NewList()

	removeComments := state.Options.RemoveComments.IsTrue()

	// iterate through each statement in the `statements` array
	for _, statement := range statements {
		// capture prerequisite statements for the `ts.Statement`
		// transform the statement into a luau.List<...>
		transformedStatements, prereqStatements := Capture(state, func() *luau.List { return transformStatement(state, statement) })

		// iterate through each of the leading comments of the statement
		if !removeComments {
			result.PushList(state.GetLeadingComments(statement))
		}

		// check statement for hoisting
		// hoisting is the use of a variable before it was declared
		if hoistDeclaration := createHoistDeclaration(state, statement); hoistDeclaration != nil {
			result.Push(hoistDeclaration)
		}

		result.PushList(prereqStatements)
		result.PushList(transformedStatements)

		if transformedStatements.Tail != nil && luau.IsFinalStatement(transformedStatements.Tail.Value) {
			break
		}

		// namespace export handling
		if exportInfo != nil {
			containerID := exportInfo.id
			if exportMapping, ok := exportInfo.mapping[statement]; ok {
				for _, exportName := range exportMapping {
					result.Push(luau.Create(&luau.Assignment{
						Left:     luau.NewProperty(containerID, exportName),
						Operator: luau.AssignEqual,
						Right:    luau.NewID(exportName),
					}))
				}
			}
		}
	}

	if !removeComments {
		if lastToken := getLastToken(state, parent, statements); lastToken != nil {
			result.PushList(state.GetLeadingComments(lastToken))
		}
	}

	return result
}
