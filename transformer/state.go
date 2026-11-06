// Package transformer is the port of src/TSTransformer from roblox-ts 3.0.0. This file is
// classes/TransformState.ts, classes/MultiTransformState.ts, util/traversal.ts and util/valueToIdStr.ts.
// See docs/bridge.md for the rules on checker access.
package transformer

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/pathtranslator"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/rojoresolver"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

func init() {
	// Globals and Strings are shared by every worker.
	luau.PinShared()
}

// MultiTransformState is state that lives for one compile step, for one worker. Upstream has one per compile.
// It's keyed by symbols, so here each worker has its own, because symbols can't cross checkers.
//
// isReportedByNoAnyCache and isReportedByMultipleDefinitionsCache make a diagnostic report once per symbol. With
// several workers, a symbol is reported once per checker that reaches it, so the same error can appear more than
// once. Sort and deduplicate the combined diagnostics at the end of a compile.
type MultiTransformState struct {
	IsMethodCache                        map[*ast.Symbol]bool
	IsDefinedAsLetCache                  map[*ast.Symbol]bool
	IsReportedByNoAnyCache               map[*ast.Symbol]bool
	IsReportedByMultipleDefinitionsCache map[*ast.Symbol]bool
	GetModuleExportsCache                map[*ast.Symbol][]*ast.Symbol
	GetModuleExportsAliasMapCache        map[*ast.Symbol]map[*ast.Symbol]string
}

// NewMultiTransformState makes the state for one worker.
func NewMultiTransformState() *MultiTransformState {
	return &MultiTransformState{
		IsMethodCache:                        map[*ast.Symbol]bool{},
		IsDefinedAsLetCache:                  map[*ast.Symbol]bool{},
		IsReportedByNoAnyCache:               map[*ast.Symbol]bool{},
		IsReportedByMultipleDefinitionsCache: map[*ast.Symbol]bool{},
		GetModuleExportsCache:                map[*ast.Symbol][]*ast.Symbol{},
		GetModuleExportsAliasMapCache:        map[*ast.Symbol]map[*ast.Symbol]string{},
	}
}

// TransformServices is the interface of the same name. The macro table is this worker's own.
type TransformServices struct {
	MacroManager *tsbridge.MacroSymbols
}

// TryUses is the interface of the same name.
type TryUses struct {
	UsesReturn   bool
	UsesBreak    bool
	UsesContinue bool
}

// StateOptions is the part of TransformState's constructor arguments that one compile shares across workers.
// None of it is keyed by types or symbols. CompileContext comes from project.PrepareCompile.
type StateOptions struct {
	Program        *compiler.Program
	Data           *shared.ProjectData
	PathTranslator *pathtranslator.Translator
	Compile        *project.CompileContext
	Options        *core.CompilerOptions
}

// TransformState represents the state of the transformation between TS and the Luau AST, for one source file.
//
// Upstream's constructor takes the type checker and calls typeChecker.getEmitResolver(sourceFile). Here it takes a
// worker, and the resolver can only come from Worker.CheckFileAndGetResolver.
type TransformState struct {
	StateOptions

	Worker     *tsbridge.Worker
	Checker    *checker.Checker
	Resolver   *checker.EmitResolver
	Services   *TransformServices
	Multi      *MultiTransformState
	Diagnostic *shared.DiagnosticService
	SourceFile *ast.SourceFile

	sourceFileText      string
	isInReplicatedFirst bool

	HasExportEquals bool
	HasExportFrom   bool
	UsesRuntimeLib  bool

	ClassIdentifierMap map[*ast.Node]luau.AnyIdentifier

	TryUsesStack          []*TryUses
	PrereqStatementsStack []*luau.List

	// With fixEmit. See createContinueStatements and hoistFunctionsPastFinalStatement.
	switchContinueFlags map[*ast.Node]*luau.TemporaryIdentifier
	hoistedPastFinal    map[*ast.Node]bool

	HoistsByStatement map[*ast.Node][]*ast.Node
	IsHoisted         map[*ast.Symbol]bool
	SymbolToIDMap     map[*ast.Symbol]*luau.TemporaryIdentifier

	// CaseSensitivity is what upstream's getCanonicalFileName decides.
	CaseSensitivity tspath.CaseSensitivity

	getTypeCache               map[*ast.Node]*checker.Type
	moduleIDBySymbol           map[*ast.Symbol]luau.AnyIdentifier
	classElementToObjectKeyMap map[*ast.Node]luau.Expression
}

// NewTransformState is the constructor. The worker must own file. Multi, services and the diagnostic service
// belong to the worker. Output paths go through this platform's path rules, as in upstream.
func NewTransformState(ctx context.Context, worker *tsbridge.Worker, options StateOptions, services *TransformServices, multi *MultiTransformState, diagnostic *shared.DiagnosticService, file *ast.SourceFile) *TransformState {
	state := &TransformState{
		StateOptions:               options,
		Worker:                     worker,
		Checker:                    worker.Checker(),
		Resolver:                   worker.CheckFileAndGetResolver(ctx, file),
		Services:                   services,
		Multi:                      multi,
		Diagnostic:                 diagnostic,
		SourceFile:                 file,
		sourceFileText:             file.Text(),
		CaseSensitivity:            options.Program.CaseSensitivity(),
		ClassIdentifierMap:         map[*ast.Node]luau.AnyIdentifier{},
		HoistsByStatement:          map[*ast.Node][]*ast.Node{},
		IsHoisted:                  map[*ast.Symbol]bool{},
		SymbolToIDMap:              map[*ast.Symbol]*luau.TemporaryIdentifier{},
		getTypeCache:               map[*ast.Node]*checker.Type{},
		moduleIDBySymbol:           map[*ast.Symbol]luau.AnyIdentifier{},
		classElementToObjectKeyMap: map[*ast.Node]luau.Expression{},
	}
	sourceOutPath := options.PathTranslator.GetOutputPath(nodepath.Native.Normalize(file.FileName().AsString()))
	rbxPath := options.Compile.RojoResolver.GetRbxPathFromFilePath(sourceOutPath)
	state.isInReplicatedFirst = len(rbxPath) > 0 && rbxPath[0] == "ReplicatedFirst"
	return state
}

// RojoResolver is state.rojoResolver.
func (s *TransformState) RojoResolver() *rojoresolver.Resolver { return s.Compile.RojoResolver }

// PushTryUsesStack pushes tryUses information onto the stack and returns it.
func (s *TransformState) PushTryUsesStack() *TryUses {
	uses := &TryUses{}
	s.TryUsesStack = append(s.TryUsesStack, uses)
	return uses
}

// MarkTryUses marks the current try statement as exiting with return, break or continue. Pass a function that sets
// the field: mark(func(u *TryUses) { u.UsesReturn = true }).
func (s *TransformState) MarkTryUses(mark func(*TryUses)) {
	if len(s.TryUsesStack) != 0 {
		mark(s.TryUsesStack[len(s.TryUsesStack)-1])
	}
}

// PopTryUsesStack pops tryUses information from the stack.
func (s *TransformState) PopTryUsesStack() {
	s.TryUsesStack = s.TryUsesStack[:len(s.TryUsesStack)-1]
}

// Prereq pushes a prerequisite statement onto the current list.
func (s *TransformState) Prereq(statement luau.Statement) {
	s.PrereqStatementsStack[len(s.PrereqStatementsStack)-1].Push(statement)
}

// PrereqList pushes a list of prerequisite statements onto the current list.
func (s *TransformState) PrereqList(statements *luau.List) {
	s.PrereqStatementsStack[len(s.PrereqStatementsStack)-1].PushList(statements)
}

// PushPrereqStatementsStack makes and pushes a new prerequisite list.
func (s *TransformState) PushPrereqStatementsStack() *luau.List {
	statements := luau.NewList()
	s.PrereqStatementsStack = append(s.PrereqStatementsStack, statements)
	return statements
}

// PopPrereqStatementsStack pops and returns the top prerequisite list.
func (s *TransformState) PopPrereqStatementsStack() *luau.List {
	last := len(s.PrereqStatementsStack) - 1
	if last < 0 {
		panic(shared.AssertionMessage(""))
	}
	popped := s.PrereqStatementsStack[last]
	s.PrereqStatementsStack = s.PrereqStatementsStack[:last]
	return popped
}

// GetLeadingComments returns a node's leading comments as Luau comments. Offsets are bytes into the UTF-8 text.
func (s *TransformState) GetLeadingComments(node *ast.Node) *luau.List {
	comments := luau.NewList()
	var factory ast.NodeFactory
	for commentRange := range scanner.GetLeadingCommentRanges(&factory, s.sourceFileText, node.Pos()) {
		end := commentRange.End()
		if commentRange.Kind != ast.KindSingleLineCommentTrivia {
			end -= 2
		}
		comments.Push(luau.NewComment(s.sourceFileText[commentRange.Pos()+2 : end]))
	}
	return comments
}

// CapturePrereqs returns the prerequisite statements that callback makes.
func (s *TransformState) CapturePrereqs(callback func()) *luau.List {
	s.PushPrereqStatementsStack()
	callback()
	return s.PopPrereqStatementsStack()
}

// Capture returns the result and the prerequisite statements that callback makes.
func Capture[T any](s *TransformState, callback func() T) (T, *luau.List) {
	var value T
	prereqs := s.CapturePrereqs(func() { value = callback() })
	return value, prereqs
}

// NoPrereqs runs callback and asserts that it made no prerequisite statements.
func (s *TransformState) NoPrereqs(callback func() luau.Expression) luau.Expression {
	expression, statements := Capture(s, callback)
	if !statements.IsEmpty() {
		panic(shared.AssertionMessage(""))
	}
	return expression
}

// GetType is state.getType: the type at a node, skipping up through wrappers such as parentheses, with a cache.
func (s *TransformState) GetType(node *ast.Node) *checker.Type {
	if t, ok := s.getTypeCache[node]; ok {
		return t
	}
	t := s.Checker.GetTypeAtLocation(SkipUpwards(node))
	s.getTypeCache[node] = t
	return t
}

// TS is state.TS: a property of the runtime library, which it also marks as used.
func (s *TransformState) TS(node *ast.Node, name string) *luau.PropertyAccessExpression {
	s.UsesRuntimeLib = true
	if s.Compile.ProjectType == shared.ProjectTypeGame && s.isInReplicatedFirst {
		s.Diagnostic.AddDiagnostic(shared.Warnings.RuntimeLibUsedInReplicatedFirst.At(shared.AtNode(node)))
	}
	return luau.NewProperty(Global(luau.Globals.TS), name)
}

// Global returns a private copy of a luau.Globals node for use from a worker. Globals are shared and mutable: the
// first Create that takes one adopts it, which races between workers. See luau/doc.go. A copy renders the same.
func Global[T luau.Node](node T) T {
	return luau.ShallowClone(node).(T)
}

// PushToVar declares a new Luau variable and pushes expression to it. expression can be nil, to declare without a
// value. name can be "", to get one from the expression.
func (s *TransformState) PushToVar(expression luau.Expression, name string) *luau.TemporaryIdentifier {
	if name == "" && expression != nil {
		name = valueToIDStr(expression)
	}
	temp := luau.NewTempID(name)
	declaration := &luau.VariableDeclaration{Left: temp}
	if expression != nil {
		declaration.Right = expression
	}
	s.Prereq(luau.Create(declaration))
	return temp
}

// PushToVarIfComplex uses PushToVar unless the expression is simple.
func (s *TransformState) PushToVarIfComplex(expression luau.Expression, name string) luau.Expression {
	if luau.IsSimple(expression) {
		return expression
	}
	return s.PushToVar(expression, name)
}

// PushToVarIfNonID uses PushToVar unless the expression is an identifier.
func (s *TransformState) PushToVarIfNonID(expression luau.Expression, name string) luau.AnyIdentifier {
	if luau.IsAnyIdentifier(expression) {
		return expression.(luau.AnyIdentifier)
	}
	return s.PushToVar(expression, name)
}

// GetModuleExports is state.getModuleExports.
func (s *TransformState) GetModuleExports(moduleSymbol *ast.Symbol) []*ast.Symbol {
	if exports, ok := s.Multi.GetModuleExportsCache[moduleSymbol]; ok {
		return exports
	}
	exports := sortExportsInBindOrder(moduleSymbol, s.Checker.GetExportsOfModule(moduleSymbol))
	s.Multi.GetModuleExportsCache[moduleSymbol] = exports
	return exports
}

// GetModuleExportsAliasMap is state.getModuleExportsAliasMap.
func (s *TransformState) GetModuleExportsAliasMap(moduleSymbol *ast.Symbol) map[*ast.Symbol]string {
	if aliasMap, ok := s.Multi.GetModuleExportsAliasMapCache[moduleSymbol]; ok {
		return aliasMap
	}
	aliasMap := map[*ast.Symbol]string{}
	for _, exportSymbol := range s.GetModuleExports(moduleSymbol) {
		originalSymbol := s.Checker.SkipAlias(exportSymbol)
		declarations := exportSymbol.Declarations()
		if len(declarations) > 0 && ast.IsExportSpecifier(declarations[0]) {
			aliasMap[originalSymbol] = declarations[0].Name().Text()
		} else {
			aliasMap[originalSymbol] = exportSymbol.Name()
		}
	}
	s.Multi.GetModuleExportsAliasMapCache[moduleSymbol] = aliasMap
	return aliasMap
}

func (s *TransformState) getModuleSymbolFromNode(node *ast.Node) *ast.Symbol {
	moduleAncestor := GetModuleAncestor(node)
	location := moduleAncestor
	if !ast.IsSourceFile(moduleAncestor) {
		location = moduleAncestor.Name()
	}
	exportSymbol := s.Checker.GetSymbolAtLocation(location)
	if exportSymbol == nil {
		panic(shared.AssertionMessage(""))
	}
	return exportSymbol
}

func (s *TransformState) getModuleIDFromSymbol(moduleSymbol *ast.Symbol) luau.AnyIdentifier {
	moduleID, ok := s.moduleIDBySymbol[moduleSymbol]
	if !ok {
		panic(shared.AssertionMessage(""))
	}
	return moduleID
}

// SetModuleIDBySymbol is state.setModuleIdBySymbol.
func (s *TransformState) SetModuleIDBySymbol(moduleSymbol *ast.Symbol, moduleID luau.AnyIdentifier) {
	s.moduleIDBySymbol[moduleSymbol] = moduleID
}

// GetModuleIDFromNode is state.getModuleIdFromNode.
func (s *TransformState) GetModuleIDFromNode(node *ast.Node) luau.AnyIdentifier {
	return s.getModuleIDFromSymbol(s.getModuleSymbolFromNode(node))
}

// GetModuleIDPropertyAccess is state.getModuleIdPropertyAccess. It returns nil where JS returns undefined.
func (s *TransformState) GetModuleIDPropertyAccess(idSymbol *ast.Symbol) *luau.PropertyAccessExpression {
	if declaration := idSymbol.ValueDeclaration(); declaration != nil {
		moduleSymbol := s.getModuleSymbolFromNode(declaration)
		if alias := s.GetModuleExportsAliasMap(moduleSymbol)[idSymbol]; alias != "" {
			return luau.NewProperty(s.getModuleIDFromSymbol(moduleSymbol), alias)
		}
	}
	return nil
}

// SetClassElementObjectKey stores the key in `obj[key] = value` for a class element, so it can be used later.
func (s *TransformState) SetClassElementObjectKey(classElement *ast.Node, key luau.Expression) {
	if _, exists := s.classElementToObjectKeyMap[classElement]; exists {
		panic(shared.AssertionMessage(""))
	}
	s.classElementToObjectKeyMap[classElement] = key
}

// GetClassElementObjectKey is state.getClassElementObjectKey. It returns nil where JS returns undefined.
func (s *TransformState) GetClassElementObjectKey(classElement *ast.Node) luau.Expression {
	return s.classElementToObjectKeyMap[classElement]
}

func isWrapperExpression(node *ast.Node) bool {
	return ast.IsNonNullExpression(node) || ast.IsParenthesizedExpression(node) || ast.IsAsExpression(node) ||
		ast.IsTypeAssertion(node) || ast.IsSatisfiesExpression(node)
}

// isAncestorOf is isAncestorOf in util/traversal.ts.
func isAncestorOf(ancestor *ast.Node, node *ast.Node) bool {
	for node != nil {
		if ancestor == node {
			return true
		}
		node = node.Parent
	}
	return false
}

// SkipDownwards is skipDownwards in util/traversal.ts.
func SkipDownwards(node *ast.Node) *ast.Node {
	for isWrapperExpression(node) {
		node = node.Expression()
	}
	return node
}

// SkipUpwards is skipUpwards in util/traversal.ts.
func SkipUpwards(node *ast.Node) *ast.Node {
	parent := node.Parent
	for parent != nil && isWrapperExpression(parent) {
		node = parent
		parent = node.Parent
	}
	return node
}

// GetAncestor is getAncestor in util/traversal.ts. It returns nil where JS returns undefined.
func GetAncestor(node *ast.Node, check func(*ast.Node) bool) *ast.Node {
	current := node
	for current != nil && !check(current) {
		current = current.Parent
	}
	return current
}

// GetModuleAncestor is getModuleAncestor in util/traversal.ts.
func GetModuleAncestor(node *ast.Node) *ast.Node {
	return GetAncestor(node, func(n *ast.Node) bool { return ast.IsSourceFile(n) || ast.IsModuleDeclaration(n) })
}

// valueToIDStr is valueToIdStr in util/valueToIdStr.ts.
func valueToIDStr(value luau.Expression) string {
	valueStr, ok := expressionToStr(value)
	if ok && luau.IsValidIdentifier(valueStr) {
		return uncapitalizeFirstLetter(valueStr)
	}
	return ""
}

func expressionToStr(expression luau.Node) (string, bool) {
	switch v := expression.(type) {
	case *luau.Identifier:
		return v.Name, true // X -> "X"
	case *luau.PropertyAccessExpression:
		return v.Name, true // A.B -> "B"
	case *luau.CallExpression:
		// X.new() -> "X"
		if property, ok := v.Expression.(*luau.PropertyAccessExpression); ok && property.Name == "new" {
			return expressionToStr(property.Expression)
		}
	}
	return "", false
}

func uncapitalizeFirstLetter(str string) string {
	if str == "" {
		return str
	}
	return strings.ToLower(str[:1]) + str[1:]
}

// sortExportsInBindOrder puts a module's exports in the order JS has them. TS 5.5.3 keeps symbol tables in insertion
// order, which is the order the binder declares them: function declarations of a statement list first, then the rest
// in source order, and exports that came through `export *` after the module's own. TS 7's symbol tables are Go maps,
// so GetExportsOfModule returns a different order on every call. The order reaches output in handleExports, and it
// decides which alias wins in getModuleExportsAliasMap. moduleSymbol's own declarations tell its own exports from the
// others. The order among `export *` symbols is only made stable, not matched to JS, because handleExports ignores them.
func sortExportsInBindOrder(moduleSymbol *ast.Symbol, exports []*ast.Symbol) []*ast.Symbol {
	type key struct {
		star        bool
		file        string
		notFunction bool
		pos         int
		name        string
	}
	moduleFiles := map[string]bool{}
	for _, declaration := range moduleSymbol.Declarations() {
		moduleFiles[ast.GetSourceFileOfNode(declaration).FileName().AsString()] = true
	}
	keys := make(map[*ast.Symbol]key, len(exports))
	for _, symbol := range exports {
		k := key{notFunction: true, pos: int(^uint(0) >> 1), name: symbol.Name()}
		found := false
		for _, declaration := range symbol.Declarations() {
			isFunction := ast.IsFunctionDeclaration(declaration) && declaration.Parent != nil &&
				(ast.IsSourceFile(declaration.Parent) || ast.IsModuleBlock(declaration.Parent))
			if !found || (isFunction && k.notFunction) || (isFunction == !k.notFunction && declaration.Pos() < k.pos) {
				found = true
				k.file = ast.GetSourceFileOfNode(declaration).FileName().AsString()
				k.notFunction = !isFunction
				k.pos = declaration.Pos()
			}
		}
		k.star = found && !moduleFiles[k.file]
		keys[symbol] = k
	}
	sorted := slices.Clone(exports)
	slices.SortStableFunc(sorted, func(a, b *ast.Symbol) int {
		ka, kb := keys[a], keys[b]
		if ka.star != kb.star {
			if ka.star {
				return 1
			}
			return -1
		}
		return cmp.Or(
			cmp.Compare(ka.file, kb.file),
			cmp.Compare(boolInt(ka.notFunction), boolInt(kb.notFunction)),
			cmp.Compare(ka.pos, kb.pos),
			cmp.Compare(ka.name, kb.name),
		)
	})
	return sorted
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// CreateRuntimeLibImport is createRuntimeLibImport. It returns `local TS = ...`.
func (s *TransformState) CreateRuntimeLibImport(sourceFile *ast.SourceFile) luau.Statement {
	runtimeLibRbxPath := s.Compile.RuntimeLibRbxPath
	if runtimeLibRbxPath != nil {
		if s.Compile.ProjectType == shared.ProjectTypeGame {
			// create an expression to obtain the service where RuntimeLib is stored
			if len(runtimeLibRbxPath) == 0 {
				panic(shared.AssertionMessage(""))
			}
			var expression luau.IndexableExpression = createGetService(runtimeLibRbxPath[0])
			// iterate through the rest of the path
			// for each instance in the path, create a new WaitForChild call to be added on to the end of the final expression
			for _, name := range runtimeLibRbxPath[1:] {
				expression = luau.Create(&luau.MethodCallExpression{
					Expression: expression,
					Name:       "WaitForChild",
					Args:       luau.MakeList[luau.Node](luau.NewString(name)),
				})
			}
			// nest the chain of `WaitForChild`s inside a require call
			expression = luau.NewCall(luau.Globals.Require, expression)
			// create a variable declaration for this call
			return luau.Create(&luau.VariableDeclaration{Left: luau.Globals.TS, Right: expression})
		}
		sourceOutPath := s.PathTranslator.GetOutputPath(nodepath.Native.Normalize(sourceFile.FileName().AsString()))
		rbxPath := s.RojoResolver().GetRbxPathFromFilePath(sourceOutPath)
		if rbxPath == nil {
			s.Diagnostic.AddDiagnostic(shared.Errors.NoRojoData.At(shared.AtNode(sourceFile.AsNode()), nodepath.Native.Relative(s.Data.ProjectPath, sourceOutPath), false))
			return luau.Create(&luau.VariableDeclaration{Left: luau.Globals.TS, Right: luau.NewNone()})
		}
		path := rojoresolver.Relative(rbxPath, runtimeLibRbxPath)
		names := make([]string, len(path))
		for i, name := range path {
			if name == rojoresolver.RbxPathParent {
				name = shared.ParentField
			}
			names[i] = name
		}
		return luau.Create(&luau.VariableDeclaration{
			Left:  luau.Globals.TS,
			Right: luau.NewCall(luau.Globals.Require, propertyAccessExpressionChain(luau.Globals.Script, names)),
		})
	}
	// we pass RuntimeLib access to packages via `_G[script] = TS`
	// access it here via `local TS = _G[script]`
	return luau.Create(&luau.VariableDeclaration{
		Left: luau.Globals.TS,
		Right: luau.Create(&luau.ComputedIndexExpression{
			Expression: luau.Globals.G,
			Index:      luau.Globals.Script,
		}),
	})
}

// GuessVirtualPath is guessVirtualPath: it attempts to reverse a symlink lookup. It returns "" where JS returns
// undefined. The program's symlink cache holds the same data as TS 5.5.3's getSymlinkCache.
func (s *TransformState) GuessVirtualPath(fsPath string) string {
	reverseSymlinkMap := s.Program.GetSymlinkCache().DirectoriesByRealpath()
	original := fsPath
	for {
		// reverseSymlinkMap always has trailing slashes, as it is constructed from `SymlinkedDirectory.real`
		parent := tspath.EnsureTrailingDirectorySeparator(nodepath.Native.Dirname(fsPath))
		if fsPath == parent {
			return ""
		}
		fsPath = parent
		key := s.CaseSensitivity.PathKey(tspath.ToRootedPath(fsPath, s.Program.GetCurrentDirectory()))
		if symlinks, ok := reverseSymlinkMap.Load(key); ok {
			// TS keeps these in an array and takes the first. Here they're a set, so take the smallest for a fixed answer.
			first := ""
			symlinks.Range(func(symlink tspath.RootedDirectoryPath) bool {
				if text := symlink.AsString(); first == "" || text < first {
					first = text
				}
				return true
			})
			if first != "" {
				return nodepath.Native.Join(first, nodepath.Native.Relative(fsPath, original))
			}
		}
	}
}
