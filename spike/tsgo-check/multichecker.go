package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
)

func allCheckers(program *compiler.Program) []*checker.Checker {
	var mu sync.Mutex
	byIndex := map[int]*checker.Checker{}
	program.ForEachCheckerParallel(func(idx int, c *checker.Checker) {
		mu.Lock()
		byIndex[idx] = c
		mu.Unlock()
	})
	result := make([]*checker.Checker, len(byIndex))
	for idx, c := range byIndex {
		result[idx] = c
	}
	return result
}

// runMultiCheckerReport measures what is and isn't shared between checkers. It runs
// after the parallel walk, on one goroutine, so it can safely touch every checker.
func runMultiCheckerReport(ctx context.Context, program *compiler.Program) {
	checkers := allCheckers(program)
	fmt.Printf("\nmulti-checker identity across %d checkers\n", len(checkers))
	if len(checkers) < 2 {
		fmt.Println("  needs at least two checkers")
		return
	}

	// This goes first: the type queries below mark aliases on every checker they touch.
	aliasReferencedByOwnership(ctx, program, checkers)

	managers := make([]*macroManager, len(checkers))
	for i, c := range checkers {
		managers[i] = newMacroManager(c)
	}

	fmt.Printf("  %-26s %8s %6s %10s\n", "global symbol", "distinct", "decls", "transient")
	sharedCount, clonedCount := 0, 0
	for _, name := range symbolNames {
		seen := map[*ast.Symbol]bool{}
		var first *ast.Symbol
		for _, m := range managers {
			if s := m.symbols[name]; s != nil {
				seen[s] = true
				if first == nil {
					first = s
				}
			}
		}
		if first == nil {
			fmt.Printf("  %-26s missing\n", name)
			continue
		}
		if len(seen) == 1 {
			sharedCount++
		} else {
			clonedCount++
		}
		fmt.Printf("  %-26s %8d %6d %10v\n", name, len(seen), len(first.Declarations()), first.Flags()&ast.SymbolFlagsTransient != 0)
	}
	fmt.Printf("  symbols shared by all checkers: %d, cloned per checker: %d\n", sharedCount, clonedCount)

	for _, table := range []struct {
		name string
		get  func(*macroManager) map[*ast.Symbol]string
	}{
		{"call macros", func(m *macroManager) map[*ast.Symbol]string { return m.callMacros }},
		{"identifier macros", func(m *macroManager) map[*ast.Symbol]string { return m.identifiers }},
		{"constructor macros", func(m *macroManager) map[*ast.Symbol]string { return m.constructors }},
		{"method symbols", func(m *macroManager) map[*ast.Symbol]string { return m.methods }},
	} {
		a, b := table.get(managers[0]), table.get(managers[1])
		same := 0
		for key := range a {
			if _, ok := b[key]; ok {
				same++
			}
		}
		fmt.Printf("  %-20s %d entries, %d keys identical between checker 0 and 1\n", table.name, len(a), same)
	}

	// The same node through two checkers, and through a mismatched macro manager.
	typesCompared, sameType, sameID, stringsDiffer := 0, 0, 0, 0
	classifications, wrongManager := 0, 0
	symbolsCompared, sameSymbol := 0, 0
	identityHits, identityWrong := map[string]int{}, map[string]int{}
	for _, file := range userFiles(program) {
		var visit func(node *ast.Node)
		visit = func(node *ast.Node) {
			if ast.IsExpressionNode(node) {
				a := checkers[0].GetTypeAtLocation(node)
				b := checkers[1].GetTypeAtLocation(node)
				typesCompared++
				if a == b {
					sameType++
				}
				if a.Id() == b.Id() {
					sameID++
				}
				if checkers[0].TypeToString(a) != checkers[1].TypeToString(b) {
					stringsDiffer++
				}
				classifications++
				if managers[0].classify(a) != classifyWith(checkers[0], managers[1], a) {
					wrongManager++
				}
				// classify hides the Array case, because IsArrayLikeType also accepts arrays.
				// This is the bare `type.symbol === macroManager.getSymbolOrThrow(name)` check.
				if symbol := a.Symbol(); symbol != nil {
					for _, name := range symbolNames {
						own, other := symbol == managers[0].symbols[name], symbol == managers[1].symbols[name]
						if own {
							identityHits[name]++
						}
						if own != other {
							identityWrong[name]++
						}
					}
				}
			}
			if ast.IsIdentifier(node) {
				a := checkers[0].GetSymbolAtLocation(node)
				b := checkers[1].GetSymbolAtLocation(node)
				if a != nil || b != nil {
					symbolsCompared++
					if a == b {
						sameSymbol++
					}
				}
			}
			node.ForEachChild(func(child *ast.Node) bool {
				visit(child)
				return false
			})
		}
		visit(file.AsNode())
	}
	fmt.Printf("  types at the same node: %d compared, %d same pointer, %d same TypeId, %d print differently\n", typesCompared, sameType, sameID, stringsDiffer)
	fmt.Printf("  symbols at the same identifier: %d compared, %d same pointer\n", symbolsCompared, sameSymbol)
	fmt.Printf("  classify with another checker's macro manager: %d classified, %d wrong\n", classifications, wrongManager)
	for _, name := range symbolNames {
		if identityHits[name] > 0 {
			fmt.Printf("  type.symbol == %s: %d hits with own manager, %d wrong with another's\n", name, identityHits[name], identityWrong[name])
		}
	}
}

// classifyWith classifies a type from c using a macro manager built on another checker.
func classifyWith(c *checker.Checker, other *macroManager, t *checker.Type) string {
	mixed := *other
	mixed.c = c
	return mixed.classify(t)
}

// aliasReferencedByOwnership shows that alias data exists only on a checker that checked the file.
func aliasReferencedByOwnership(ctx context.Context, program *compiler.Program, checkers []*checker.Checker) {
	fmt.Println("  IsReferencedAliasDeclaration on owner checker vs one that never checked the file:")
	for _, file := range userFiles(program) {
		owner, done := program.GetTypeCheckerForFile(ctx, file)
		done()
		var other *checker.Checker
		for _, c := range checkers {
			if c != owner {
				other = c
				break
			}
		}
		ownerResolver := owner.NewEmitResolver(printer.NewEmitContext())
		otherResolver := other.NewEmitResolver(printer.NewEmitContext())
		agree, differ := 0, 0
		for _, statement := range file.Statements.Nodes {
			if !ast.IsImportDeclaration(statement) || statement.AsImportDeclaration().ImportClause == nil {
				continue
			}
			bindings := statement.AsImportDeclaration().ImportClause.AsImportClause().NamedBindings
			if bindings == nil || !ast.IsNamedImports(bindings) {
				continue
			}
			for _, element := range bindings.AsNamedImports().Elements.Nodes {
				if ownerResolver.IsReferencedAliasDeclaration(element) == otherResolver.IsReferencedAliasDeclaration(element) {
					agree++
				} else {
					differ++
				}
			}
		}
		fmt.Printf("    %s: %d agree, %d differ\n", file.FileName().BaseName(), agree, differ)
	}
}

// runLockTest calls an emit resolver method in each way of holding a checker.
func runLockTest(program *compiler.Program) {
	ctx := context.Background()
	var file *ast.SourceFile
	for _, f := range userFiles(program) {
		if f.LanguageVariant == 1 { // JSX
			file = f
		}
	}
	if file == nil {
		fmt.Println("lock test: needs a .tsx file")
		return
	}
	attempt := func(name string, call func()) {
		finished := make(chan struct{})
		go func() {
			call()
			close(finished)
		}()
		select {
		case <-finished:
			fmt.Printf("lock test: %-40s ok\n", name)
		case <-time.After(3 * time.Second):
			fmt.Printf("lock test: %-40s DEADLOCK (no return after 3s)\n", name)
		}
	}

	// The non-exclusive attempt runs first, because a deadlocked attempt leaks its lock.
	attempt("GetTypeCheckerForFile (non-exclusive)", func() {
		c, done := program.GetTypeCheckerForFile(ctx, file)
		defer done()
		c.NewEmitResolver(printer.NewEmitContext()).GetJsxFactoryEntity(file.AsNode())
	})
	attempt("checker method under exclusive lock", func() {
		c, done := program.GetTypeCheckerForFileExclusive(ctx, file)
		defer done()
		c.GetTypeAtLocation(file.Statements.Nodes[0])
	})
	attempt("emit resolver under exclusive lock", func() {
		c, done := program.GetTypeCheckerForFileExclusive(ctx, file)
		defer done()
		c.NewEmitResolver(printer.NewEmitContext()).GetJsxFactoryEntity(file.AsNode())
	})
}
