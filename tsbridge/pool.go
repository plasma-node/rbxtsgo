package tsbridge

import (
	"context"
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
)

type checkerGroup struct {
	checker *checker.Checker
	files   []*ast.SourceFile
}

// groupByChecker groups files by the checker that owns them, in order of first appearance.
// GetTypeCheckerForFile takes no lock. Files keep their input order inside each group.
func (p *Project) groupByChecker(ctx context.Context, files []*ast.SourceFile) []*checkerGroup {
	var groups []*checkerGroup
	byChecker := map[*checker.Checker]*checkerGroup{}
	for _, file := range files {
		c, done := p.program.GetTypeCheckerForFile(ctx, file)
		done()
		group := byChecker[c]
		if group == nil {
			group = &checkerGroup{checker: c}
			byChecker[c] = group
			groups = append(groups, group)
		}
		group.files = append(group.files, file)
	}
	return groups
}

// Worker is the sole user of one checker. Anything keyed by ts.Type or ts.Symbol (macro tables,
// MultiTransformState, caches) belongs to one Worker, because those values mean nothing on another checker.
type Worker struct {
	checker  *checker.Checker
	files    []*ast.SourceFile
	owned    map[*ast.SourceFile]bool
	resolver *checker.EmitResolver

	macrosOnce sync.Once
	macros     *MacroSymbols
	macrosErr  error
	macroSpec  MacroSpec
}

// Checker returns this worker's checker. Don't pass it, or anything it returned, to another worker.
func (w *Worker) Checker() *checker.Checker { return w.checker }

// Files returns the files this worker owns, in input order.
func (w *Worker) Files() []*ast.SourceFile { return w.files }

// Macros returns this worker's macro symbol table, built on first use from the spec passed to NewWorkers.
func (w *Worker) Macros() (*MacroSymbols, error) {
	w.macrosOnce.Do(func() { w.macros, w.macrosErr = NewMacroSymbols(w.checker, w.macroSpec) })
	return w.macros, w.macrosErr
}

// CheckFileAndGetResolver checks file on this worker's checker, then returns that checker's emit resolver.
// It is the only way to get a resolver in the port, and it replaces ts.TypeChecker.getEmitResolver(file).
//
// Two traps live here. TS's getEmitResolver checks the file implicitly, and tsgo's NewEmitResolver doesn't:
// IsReferencedAliasDeclaration reads links that checking sets, so without the check it silently answers wrong.
// And resolver methods lock the mutex that the pool's exclusive access holds, so resolver calls under that
// lock deadlock. This panics instead, both for that and for a file owned by another checker.
func (w *Worker) CheckFileAndGetResolver(ctx context.Context, file *ast.SourceFile) *checker.EmitResolver {
	if !w.owned[file] {
		panic("tsbridge: " + file.FileName().AsString() + " belongs to another checker's worker")
	}
	if w.checker.IsMutexHeld() {
		panic("tsbridge: checker mutex is held. CheckFileAndGetResolver must not run under GetTypeCheckerForFileExclusive or ForEachCheckerParallel, and diagnostics must not overlap with workers")
	}
	w.checker.GetDiagnostics(ctx, file)
	if w.resolver == nil {
		w.resolver = w.checker.NewEmitResolver(printer.NewEmitContext())
	}
	return w.resolver
}

// NewWorkers makes one worker per checker that owns at least one of files. With 7.1's partitioning,
// some checkers can own no files, so there can be fewer workers than checkers.
func (p *Project) NewWorkers(ctx context.Context, files []*ast.SourceFile, spec MacroSpec) []*Worker {
	groups := p.groupByChecker(ctx, files)
	workers := make([]*Worker, len(groups))
	for i, group := range groups {
		owned := make(map[*ast.SourceFile]bool, len(group.files))
		for _, file := range group.files {
			owned[file] = true
		}
		workers[i] = &Worker{checker: group.checker, files: group.files, owned: owned, macroSpec: spec}
	}
	return workers
}

// RunWorkers runs fn for every worker, one goroutine per worker, and waits. Each worker sees all of its files
// in one fn call, so it can keep per-checker state between them. A panic in fn is re-raised here, after the
// other workers finish. Diagnostics must not run until this returns.
func (p *Project) RunWorkers(workers []*Worker, fn func(w *Worker)) {
	if !p.workersRunning.CompareAndSwap(false, true) {
		panic("tsbridge: RunWorkers is already running")
	}
	defer p.workersRunning.Store(false)
	var wg sync.WaitGroup
	panics := make([]any, len(workers))
	for i, w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { panics[i] = recover() }()
			fn(w)
		}()
	}
	wg.Wait()
	for _, r := range panics {
		if r != nil {
			panic(r)
		}
	}
}
