package tsbridge

import (
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
)

// newUncheckedResolverForTest makes a resolver the way the bridge forbids, to show why it forbids it.
func newUncheckedResolverForTest(w *Worker) *checker.EmitResolver {
	return w.checker.NewEmitResolver(printer.NewEmitContext())
}
