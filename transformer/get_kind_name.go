package transformer

// Port of src/TSTransformer/util/getKindName.ts.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
)

// getKindName is only used in assertion messages. Upstream's chain of tests is there because ts.SyntaxKind[kind] gives
// the alias for a kind that has one, like FirstAssignment. Kind.String() has no aliases. It leaves one name to fix up.
func getKindName(kind ast.Kind) string {
	if kind == ast.KindEndOfFile {
		return "EndOfFileToken"
	}
	return strings.TrimPrefix(kind.String(), "Kind")
}
