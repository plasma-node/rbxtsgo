package transformer

// With fixEmit, function declarations after a return, break or continue (upstream issues #2847 and #2098, open with no
// fix). In JS a function declaration exists from the start of its block, and can see every binding in the block. Luau
// allows nothing after a final statement, so transformStatementList stops there and the function is gone, which leaves
// its callers with a nil local.

import "github.com/microsoft/TypeScript/tsc/internal/ast"

// hoistFunctionsPastFinalStatement returns the statements to compile for a block: the function declarations after its
// first final statement, then what comes before that statement, and that statement. That's where JS has them, and
// everything else after a final statement stays dropped, because JS never runs it.
//
// It marks the functions in state.HoistedPastFinal, for checkIdentifierHoist. Nothing needs a hoist for such a function,
// because it already exists, and whatever a block's later local such a function refers to does need one.
func hoistFunctionsPastFinalStatement(state *TransformState, statements []*ast.Node) []*ast.Node {
	finalIndex := -1
	for i, statement := range statements {
		if ast.IsReturnStatement(statement) || ast.IsBreakStatement(statement) || ast.IsContinueStatement(statement) {
			finalIndex = i
			break
		}
	}
	if finalIndex < 0 {
		return statements
	}
	var functions []*ast.Node
	for _, statement := range statements[finalIndex+1:] {
		if ast.IsFunctionDeclaration(statement) {
			functions = append(functions, statement)
		}
	}
	if len(functions) == 0 {
		return statements
	}
	if state.hoistedPastFinal == nil {
		state.hoistedPastFinal = map[*ast.Node]bool{}
	}
	for _, function := range functions {
		state.hoistedPastFinal[function] = true
	}
	return append(functions, statements[:finalIndex+1]...)
}
