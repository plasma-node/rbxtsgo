package transformer

// Port of src/TSTransformer/util/checkVariableHoist.ts.

import (
	"strings"
	"unicode/utf8"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/astnav"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
)

func checkVariableHoist(state *TransformState, node *ast.Node, symbol *ast.Symbol) {
	if _, ok := state.IsHoisted[symbol]; ok {
		return
	}

	statement := GetAncestor(node, ast.IsStatement)
	if statement == nil {
		return
	}

	caseClause := statement.Parent
	if !ast.IsCaseClause(caseClause) {
		return
	}
	caseBlock := caseClause.Parent

	isUsedOutsideOfCaseClause := hoistEachSymbolReferenceInFile(
		node,
		state.Checker,
		ast.GetSourceFileOfNode(node),
		func(token *ast.Node) bool {
			return !isAncestorOf(caseClause, token)
		},
		caseBlock,
	)

	if isUsedOutsideOfCaseClause {
		state.HoistsByStatement[statement.Parent] = append(state.HoistsByStatement[statement.Parent], node)
		state.IsHoisted[symbol] = true
	}
}

// hoistEachSymbolReferenceInFile is ts.FindAllReferences.Core.eachSymbolReferenceInFile from TS 5.5.3, with a required
// searchContainer. The checker has no equivalent: GetReferencesToSymbolInFile always searches the whole file and keeps
// the definition. It leaves out the export specifier test, because an export specifier in a case block is a grammar
// error and never gets here.
func hoistEachSymbolReferenceInFile(definition *ast.Node, c *checker.Checker, sourceFile *ast.SourceFile, cb func(token *ast.Node) bool, searchContainer *ast.Node) bool {
	var symbol *ast.Symbol
	if ast.IsParameterPropertyDeclaration(definition.Parent, definition.Parent.Parent) {
		symbol, _ = c.GetSymbolsOfParameterPropertyDeclaration(definition.Parent, definition.Text())
	} else {
		symbol = c.GetSymbolAtLocation(definition)
	}
	if symbol == nil {
		return false
	}
	for _, position := range hoistGetPossibleSymbolReferencePositions(sourceFile, symbol.Name(), searchContainer) {
		token := astnav.GetTouchingPropertyName(sourceFile, position)
		if token == sourceFile.AsNode() {
			continue
		}
		if !ast.IsIdentifier(token) || token == definition || token.Text() != definition.Text() {
			continue
		}
		referenceSymbol := c.GetSymbolAtLocation(token)
		if referenceSymbol == symbol || c.GetShorthandAssignmentValueSymbol(token.Parent) == symbol {
			if cb(token) {
				return true
			}
		}
	}
	return false
}

// hoistGetPossibleSymbolReferencePositions is getPossibleSymbolReferencePositions from TS 5.5.3, on byte offsets. The
// checker's copy can't take a container other than the file, and tests single bytes as identifier characters, so it
// misses a name followed by a non-ASCII character. A match may lie exactly at container.end, as in JS.
func hoistGetPossibleSymbolReferencePositions(sourceFile *ast.SourceFile, symbolName string, container *ast.Node) []int {
	positions := []int{}
	if symbolName == "" {
		return positions
	}
	text := sourceFile.Text()
	sourceLength := len(text)
	symbolNameLength := len(symbolName)
	position := hoistIndexOf(text, symbolName, container.Pos())
	for position >= 0 {
		if position > container.End() {
			break
		}
		endPosition := position + symbolNameLength
		if position == 0 || !hoistIsIdentifierPartBefore(text, position) {
			if endPosition == sourceLength || !hoistIsIdentifierPartAt(text, endPosition) {
				positions = append(positions, position)
			}
		}
		position = hoistIndexOf(text, symbolName, position+symbolNameLength+1)
	}
	return positions
}

func hoistIndexOf(text string, search string, from int) int {
	if from > len(text) {
		return -1
	}
	index := strings.Index(text[from:], search)
	if index < 0 {
		return -1
	}
	return from + index
}

func hoistIsIdentifierPartBefore(text string, position int) bool {
	r, _ := utf8.DecodeLastRuneInString(text[:position])
	return scanner.IsIdentifierPart(r)
}

func hoistIsIdentifierPartAt(text string, position int) bool {
	r, _ := utf8.DecodeRuneInString(text[position:])
	return scanner.IsIdentifierPart(r)
}
