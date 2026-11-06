package transformer

// Port of src/TSTransformer/util/assertNever.ts.

import (
	"fmt"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// assertNever panics where upstream calls LogService.fatal. The `npm ls typescript` hint is left out: Go has no
// TypeScript package to mismatch. The result type only lets callers write `return assertNever(x, "name")` in a
// function that returns an expression. It never returns.
func assertNever(value any, message string) luau.Expression {
	var description string
	if node, isTsNode := value.(*ast.Node); isTsNode && node != nil {
		description = "a TS node of kind " + getKindName(node.Kind)
	} else {
		description = fmt.Sprintf("%v", value)
	}
	panic("Exhaustive assertion failed! " + message + ", value was " + description +
		"\nThis is usually caused by a TypeScript version mismatch.")
}
