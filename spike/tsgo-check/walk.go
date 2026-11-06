package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

// stat counts calls to one checker query. A query that was never called, or that never
// returned anything, shows up in the summary as a coverage gap.
type stat struct{ calls, useful, panics int }

type walkReport struct {
	evidence string
	stats    map[string]*stat
	panics   []string
	errors   []string
	elapsed  time.Duration
	groups   int
}

func (r *walkReport) summary() string {
	names := make([]string, 0, len(r.stats))
	for name := range r.stats {
		names = append(names, name)
	}
	slices.Sort(names)
	var b strings.Builder
	fmt.Fprintf(&b, "\nparallel walk: %d checker groups, %s\n", r.groups, r.elapsed.Round(time.Millisecond))
	fmt.Fprintf(&b, "%-44s %7s %7s %7s\n", "query", "calls", "useful", "panics")
	for _, name := range names {
		s := r.stats[name]
		fmt.Fprintf(&b, "%-44s %7d %7d %7d\n", name, s.calls, s.useful, s.panics)
	}
	for _, message := range r.errors {
		fmt.Fprintln(&b, "macro manager error:", message)
	}
	for _, message := range r.panics {
		fmt.Fprintln(&b, "panic:", message)
	}
	return b.String()
}

type walker struct {
	c        *checker.Checker
	resolver *checker.EmitResolver
	mm       *macroManager
	file     *ast.SourceFile
	out      strings.Builder
	stats    map[string]*stat
	panics   []string
	seenJsx  bool
}

// try runs one query, counts it, and turns a panic into a recorded result.
func (w *walker) try(name string, query func() bool) {
	s := w.stats[name]
	if s == nil {
		s = &stat{}
		w.stats[name] = s
	}
	s.calls++
	defer func() {
		if r := recover(); r != nil {
			s.panics++
			w.panics = append(w.panics, fmt.Sprintf("%s at %s: %v", name, w.where(nil), r))
		}
	}()
	if query() {
		s.useful++
	}
}

func (w *walker) where(node *ast.Node) string {
	if node == nil {
		return w.file.FileName().AsString()
	}
	line, _ := scanner.GetECMALineAndUTF16CharacterOfPosition(w.file, scanner.SkipTrivia(w.file.Text(), node.Pos()))
	return fmt.Sprintf("L%d", line+1)
}

func (w *walker) text(node *ast.Node) string {
	text := strings.Join(strings.Fields(scanner.GetTextOfNode(node)), " ")
	if len(text) > 48 {
		text = text[:45] + "..."
	}
	return text
}

func (w *walker) line(node *ast.Node, format string, args ...any) {
	fmt.Fprintf(&w.out, "  %s %s\n", w.where(node), fmt.Sprintf(format, args...))
}

func (w *walker) typeString(t *checker.Type) string {
	if t == nil {
		return "<nil>"
	}
	return w.c.TypeToString(t)
}

func (w *walker) visit(node *ast.Node) {
	w.queriesFor(node)
	node.ForEachChild(func(child *ast.Node) bool {
		w.visit(child)
		return false
	})
}

func (w *walker) queriesFor(node *ast.Node) {
	c := w.c
	switch {
	case ast.IsImportDeclaration(node):
		w.importDeclaration(node)
	case ast.IsVariableDeclaration(node) && ast.IsIdentifier(node.Name()):
		t := c.GetTypeAtLocation(node.Name())
		w.line(node, "var %s: %s [%s]", node.Name().Text(), w.typeString(t), w.mm.classify(t))
	case ast.IsCallExpression(node):
		w.callLike(node, node.Expression())
	case ast.IsNewExpression(node):
		w.callLike(node, node.Expression())
		w.newExpression(node)
	case ast.IsJsxOpeningElement(node) || ast.IsJsxSelfClosingElement(node):
		w.callLike(node, nil)
		w.jsxFactory(node)
	case ast.IsJsxFragment(node):
		w.jsxFactory(node)
	case ast.IsShorthandPropertyAssignment(node):
		w.try("GetShorthandAssignmentValueSymbol", func() bool {
			s := c.GetShorthandAssignmentValueSymbol(node)
			w.line(node, "shorthand %s -> symbol %s", node.Name().Text(), symbolName(s))
			return s != nil
		})
	case ast.IsEnumMember(node) || ast.IsPropertyAccessExpression(node) || ast.IsElementAccessExpression(node):
		w.try("GetConstantValue", func() bool {
			value := c.GetConstantValue(node)
			if value != nil {
				w.line(node, "constant %s = %v", w.text(node), value)
			}
			return value != nil
		})
	case ast.IsClassDeclaration(node) || ast.IsInterfaceDeclaration(node) || ast.IsTypeAliasDeclaration(node) || ast.IsEnumDeclaration(node):
		w.declaredType(node)
	case ast.IsTypeParameterDeclaration(node):
		t := c.GetTypeAtLocation(node)
		w.try("GetConstraintOfTypeParameter", func() bool { return c.GetConstraintOfTypeParameter(t) != nil })
		w.try("GetBaseConstraintOfType (type.getConstraint)", func() bool {
			constraint := c.GetBaseConstraintOfType(t)
			w.line(node, "type parameter %s constraint %s", w.typeString(t), w.typeString(constraint))
			return constraint != nil
		})
	}

	if ast.IsBinaryExpression(node) && ast.IsAssignmentExpression(node, true) && ast.IsAssignmentPattern(node.AsBinaryExpression().Left) {
		left := node.AsBinaryExpression().Left
		w.try("GetTypeOfAssignmentPattern", func() bool {
			t := c.GetTypeOfAssignmentPattern(left)
			w.line(node, "assignment pattern %s: %s [%s]", w.text(left), w.typeString(t), w.mm.classify(t))
			return t != nil
		})
	}
	if parent := node.Parent; parent != nil && ast.IsObjectLiteralExpression(parent) && (ast.IsPropertyAssignment(node) || ast.IsMethodDeclaration(node) || ast.IsShorthandPropertyAssignment(node)) {
		w.try("GetContextualTypeForObjectLiteralElement", func() bool {
			t := c.GetContextualTypeForObjectLiteralElement(node, checker.ContextFlagsNone)
			if t != nil {
				w.line(node, "object member %s contextual type %s", w.text(node.Name()), w.typeString(t))
			}
			return t != nil
		})
	}
	if ast.IsPartOfTypeNode(node) && ast.IsTypeNode(node) {
		w.try("GetTypeFromTypeNode", func() bool { return c.GetTypeFromTypeNode(node) != nil })
	}
	if node.Kind == ast.KindTrueKeyword || node.Kind == ast.KindFalseKeyword {
		t := c.GetTypeAtLocation(node)
		w.line(node, "literal %s: identical to GetTrueType=%v GetFalseType=%v", w.text(node), t == c.GetTrueType(), t == c.GetFalseType())
	}
	if ast.IsVariableDeclaration(node) && node.Type() != nil && (node.Type().Kind == ast.KindLiteralType) {
		t := c.GetTypeAtLocation(node.Name())
		w.line(node, "declared literal %s: identical to GetTrueType=%v GetFalseType=%v", w.text(node.Name()), t == c.GetTrueType(), t == c.GetFalseType())
	}
	if ast.IsIdentifier(node) {
		w.identifier(node)
	}
	if ast.IsExpressionNode(node) {
		w.expression(node)
	}
}

func (w *walker) identifier(node *ast.Node) {
	c := w.c
	var symbol *ast.Symbol
	w.try("GetSymbolAtLocation", func() bool {
		symbol = c.GetSymbolAtLocation(node)
		return symbol != nil
	})
	if symbol == nil {
		return
	}
	if name, ok := w.mm.identifiers[symbol]; ok {
		w.line(node, "identifier macro %s", name)
	}
	w.try("GetTypeOfSymbolAtLocation", func() bool { return c.GetTypeOfSymbolAtLocation(symbol, node) != nil })
	w.try("GetMergedSymbol", func() bool { return c.GetMergedSymbol(symbol) != nil })
	w.try("IsUndefinedSymbol", func() bool {
		ok := c.IsUndefinedSymbol(symbol)
		if ok {
			w.line(node, "undefined symbol")
		}
		return ok
	})
	w.try("IsArgumentsSymbol", func() bool {
		ok := c.IsArgumentsSymbol(symbol)
		if ok {
			w.line(node, "arguments symbol")
		}
		return ok
	})
	if symbol.Flags()&ast.SymbolFlagsAlias != 0 {
		w.try("GetTypeOnlyAliasDeclaration", func() bool {
			decl := c.GetTypeOnlyAliasDeclaration(symbol)
			if decl != nil {
				w.line(node, "type-only alias %s", node.Text())
			}
			return decl != nil
		})
	}
}

func (w *walker) expression(node *ast.Node) {
	c := w.c
	var t *checker.Type
	w.try("GetTypeAtLocation", func() bool {
		t = c.GetTypeAtLocation(node)
		return t != nil
	})
	if t == nil {
		return
	}
	w.try("IsArrayLikeType", func() bool { return c.IsArrayLikeType(t) })
	w.try("IsTupleType", func() bool { return checker.IsTupleType(t) })
	w.try("GetNonOptionalType", func() bool { return c.GetNonOptionalType(t) != nil })
	w.try("GetNonNullableType", func() bool { return c.GetNonNullableType(t) != nil })
	w.try("GetBaseConstraintOfType (type.getConstraint)", func() bool { return c.GetBaseConstraintOfType(t) != nil })
	w.try("GetIndexTypeOfType", func() bool { return c.GetIndexTypeOfType(t, c.GetNumberType()) != nil })
	w.try("GetPropertiesOfType (type.getProperties)", func() bool { return len(c.GetPropertiesOfType(t)) > 0 })
	w.try("GetPropertyOfType (type.getProperty)", func() bool { return c.GetPropertyOfType(t, "size") != nil })
	w.try("GetTypeOfPropertyOfType", func() bool { return c.GetTypeOfPropertyOfType(t, "size") != nil })
	w.try("GetCallSignatures (type.getCallSignatures)", func() bool { return len(c.GetCallSignatures(t)) > 0 })
	if t.ObjectFlags()&checker.ObjectFlagsReference != 0 {
		w.try("GetTypeArguments", func() bool { return len(c.GetTypeArguments(t)) > 0 })
	}
	if ast.IsExpression(node) {
		w.try("GetContextualType", func() bool { return c.GetContextualType(node, checker.ContextFlagsNone) != nil })
	}
}

func (w *walker) callLike(node *ast.Node, callee *ast.Node) {
	c := w.c
	var declaration string
	w.try("GetResolvedSignature", func() bool {
		signature := c.GetResolvedSignature(node)
		if signature == nil {
			return false
		}
		if d := signature.Declaration(); d != nil {
			declaration = d.Kind.String()
		}
		return true
	})
	macro := "-"
	if callee != nil {
		// Same lookup as upstream's transformCallExpression: the symbol of the callee's type.
		if symbol := w.firstDefinedSymbol(c.GetNonOptionalType(c.GetTypeAtLocation(callee))); symbol != nil {
			if name, ok := w.mm.callMacros[symbol]; ok {
				macro = "call:" + name
			} else if name, ok := w.mm.methods[symbol]; ok {
				macro = "property:" + name
			} else if symbol.Parent() != nil && w.mm.macroOnly[c.GetMergedSymbol(symbol.Parent())] {
				macro = "macro-only class, no method entry: " + symbol.Name()
			}
		}
	}
	w.line(node, "call %s -> %s, macro=%s", w.text(node), declaration, macro)
}

// firstDefinedSymbol mirrors getFirstDefinedSymbol in util/types.ts.
func (w *walker) firstDefinedSymbol(t *checker.Type) *ast.Symbol {
	if t.IsUnion() || t.IsIntersection() {
		for _, member := range t.Types() {
			if symbol := member.Symbol(); symbol != nil && !w.c.IsUndefinedSymbol(symbol) {
				return symbol
			}
		}
		return nil
	}
	return t.Symbol()
}

func (w *walker) newExpression(node *ast.Node) {
	t := w.c.GetTypeAtLocation(node.Expression())
	macro := "-"
	if symbol := t.Symbol(); symbol != nil {
		if construct := firstConstructSymbol(symbol); construct != nil {
			if name, ok := w.mm.constructors[construct]; ok {
				macro = name
			}
		}
	}
	result := w.c.GetTypeAtLocation(node)
	w.line(node, "new %s: %s [%s], constructor macro=%s", w.text(node.Expression()), w.typeString(result), w.mm.classify(result), macro)
}

func (w *walker) jsxFactory(node *ast.Node) {
	w.try("resolver.GetJsxFactoryEntity", func() bool {
		entity := w.resolver.GetJsxFactoryEntity(node)
		if !w.seenJsx && entity != nil {
			w.seenJsx = true
			// The entity is synthetic. Upstream's transformIdentifier skips symbol lookups
			// for such nodes, because they have no parent, so the spike doesn't query them.
			w.line(node, "jsx factory entity %s, synthetic=%v", entityText(entity), entity.Parent == nil)
		}
		return entity != nil
	})
	if ast.IsJsxFragment(node) {
		w.try("resolver.GetJsxFragmentFactoryEntity", func() bool {
			entity := w.resolver.GetJsxFragmentFactoryEntity(node)
			if entity != nil {
				w.line(node, "jsx fragment entity %s", entityText(entity))
			}
			return entity != nil
		})
	}
}

func entityText(node *ast.Node) string {
	if ast.IsQualifiedName(node) {
		q := node.AsQualifiedName()
		return entityText(q.Left) + "." + q.Right.Text()
	}
	return node.Text()
}

func (w *walker) declaredType(node *ast.Node) {
	c := w.c
	symbol := node.Symbol()
	if symbol == nil {
		return
	}
	var t *checker.Type
	w.try("GetDeclaredTypeOfSymbol", func() bool {
		t = c.GetDeclaredTypeOfSymbol(c.GetMergedSymbol(symbol))
		return t != nil
	})
	if t == nil {
		return
	}
	extra := ""
	if t.ObjectFlags()&checker.ObjectFlagsClassOrInterface != 0 {
		w.try("GetBaseTypes (type.getBaseTypes)", func() bool {
			base := c.GetBaseTypes(t)
			if len(base) > 0 {
				extra = " extends " + w.typeString(base[0])
			}
			return len(base) > 0
		})
	}
	w.line(node, "declared %s: %s%s", symbol.Name(), w.typeString(t), extra)
}

func (w *walker) importDeclaration(node *ast.Node) {
	c := w.c
	decl := node.AsImportDeclaration()
	var module *ast.Symbol
	w.try("ResolveExternalModuleName", func() bool {
		module = c.ResolveExternalModuleName(decl.ModuleSpecifier, nil)
		return module != nil
	})
	exports := 0
	if module != nil {
		w.try("GetExportsOfModule", func() bool {
			exports = len(c.GetExportsOfModule(module))
			return exports > 0
		})
	}
	w.line(node, "import %s: module symbol %v, %d exports", w.text(decl.ModuleSpecifier), module != nil, exports)
	if decl.ImportClause == nil {
		return
	}
	referenced := func(n *ast.Node, label string) {
		w.try("resolver.IsReferencedAliasDeclaration", func() bool {
			ok := w.resolver.IsReferencedAliasDeclaration(n)
			w.line(n, "  alias %s referenced=%v", label, ok)
			return ok
		})
	}
	clause := decl.ImportClause.AsImportClause()
	if clause.Name() != nil {
		referenced(decl.ImportClause, clause.Name().Text())
	}
	if bindings := clause.NamedBindings; bindings != nil && ast.IsNamedImports(bindings) {
		for _, element := range bindings.AsNamedImports().Elements.Nodes {
			referenced(element, element.Name().Text())
		}
	}
}

func symbolName(symbol *ast.Symbol) string {
	if symbol == nil {
		return "<nil>"
	}
	return symbol.Name()
}

// globalQueries exercises the queries that aren't tied to a node.
func (w *walker) globalQueries() {
	c := w.c
	w.try("GetVoidType", func() bool { return c.GetVoidType() != nil })
	w.try("GetAnyType", func() bool { return c.GetAnyType() != nil })
	w.try("GetTrueType", func() bool { return c.GetTrueType() != nil })
	w.try("GetFalseType", func() bool { return c.GetFalseType() != nil })
	w.try("GetStringLiteralType", func() bool { return c.GetStringLiteralType("a") == c.GetStringLiteralType("a") })
	w.try("CreateArrayType", func() bool {
		t := c.CreateArrayType(c.GetStringType())
		return c.TypeToString(t) == "string[]"
	})
	w.try("GetUnionType", func() bool {
		t := c.GetUnionType([]*checker.Type{c.GetStringType(), c.GetNumberType()})
		return t != nil && t.IsUnion()
	})
	w.try("IsTypeAssignableTo", func() bool {
		return c.IsTypeAssignableTo(c.GetStringLiteralType("a"), c.GetStringType()) && !c.IsTypeAssignableTo(c.GetStringType(), c.GetNumberType())
	})
	w.try("ResolveName", func() bool { return c.ResolveName("Array", nil, ast.SymbolFlagsInterface, false) != nil })
	w.try("NewEmitResolver", func() bool { return c.NewEmitResolver(printer.NewEmitContext()) != nil })
}

// checkerGroups groups user files by the checker that owns them.
func checkerGroups(ctx context.Context, program *compiler.Program, files []*ast.SourceFile) (map[*checker.Checker][]*ast.SourceFile, []*checker.Checker) {
	groups := map[*checker.Checker][]*ast.SourceFile{}
	var order []*checker.Checker
	for _, file := range files {
		// Non-exclusive access. Emit resolver methods lock the same mutex that the
		// exclusive variant holds, so holding it here would deadlock (see -lock-test).
		c, done := program.GetTypeCheckerForFile(ctx, file)
		done()
		if _, ok := groups[c]; !ok {
			order = append(order, c)
		}
		groups[c] = append(groups[c], file)
	}
	return groups, order
}

// runWalk transforms-in-spirit: one goroutine per checker, each with its own macro manager.
func runWalk(ctx context.Context, program *compiler.Program, projectDir tspath.RootedDirectoryPath) *walkReport {
	start := time.Now()
	files := userFiles(program)
	groups, order := checkerGroups(ctx, program, files)

	evidence := make(map[*ast.SourceFile]string, len(files))
	report := &walkReport{stats: map[string]*stat{}, groups: len(order)}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, c := range order {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mm := newMacroManager(c)
			stats := map[string]*stat{}
			var panics []string
			for _, file := range groups[c] {
				// Marks alias references, as ts.TypeChecker.getEmitResolver does implicitly.
				c.GetDiagnostics(ctx, file)
				w := &walker{c: c, resolver: c.NewEmitResolver(printer.NewEmitContext()), mm: mm, file: file, stats: stats}
				w.globalQueries()
				w.visit(file.AsNode())
				panics = append(panics, w.panics...)
				mu.Lock()
				evidence[file] = w.out.String()
				mu.Unlock()
			}
			mu.Lock()
			defer mu.Unlock()
			report.panics = append(report.panics, panics...)
			report.errors = append(report.errors, mm.errors...)
			for name, s := range stats {
				total := report.stats[name]
				if total == nil {
					total = &stat{}
					report.stats[name] = total
				}
				total.calls += s.calls
				total.useful += s.useful
				total.panics += s.panics
			}
		}()
	}
	wg.Wait()
	report.elapsed = time.Since(start)
	slices.Sort(report.errors)
	report.errors = slices.Compact(report.errors)

	var b strings.Builder
	for _, file := range files {
		fmt.Fprintf(&b, "== %s\n%s", relative(projectDir, file.FileName().AsString()), evidence[file])
	}
	report.evidence = b.String()
	return report
}
