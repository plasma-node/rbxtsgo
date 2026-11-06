package shared

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
)

// DiagnosticFactory is the factory type from src/Shared/diagnostics.ts, for diagnostics with no context.
type DiagnosticFactory struct {
	// ID is the factory's position in definition order, as in upstream, and is what getDiagnosticId returns.
	ID       int
	category diagnostics.Category
	messages []string
}

// At makes a diagnostic at a location. Use AtNode or AtRange for the location.
func (f *DiagnosticFactory) At(location Location) *Diagnostic {
	return newLocated(f.ID, f.category, f.messages, location)
}

func newLocated(id int, category diagnostics.Category, messages []string, location Location) *Diagnostic {
	return &Diagnostic{
		category: category,
		id:       id,
		message:  strings.Join(messages, "\n"),
		file:     location.file,
		start:    location.start,
		length:   location.len,
	}
}

// contextFactory is a factory with a context formatter. Upstream's result function does
// messages.push(...contextFormatter(...context)) on the array it closed over, so each call adds to what the next
// call prints. That bug is kept, because it's visible in rbxtsc's output: the second "Could not find Rojo data"
// in a build also repeats the first one's text. With parallel workers the order of those calls can differ from
// rbxtsc's file order, so that repeated text can come out in a different order.
type contextFactory struct {
	ID       int
	mu       sync.Mutex
	category diagnostics.Category
	messages []string
}

func (f *contextFactory) at(location Location, context ...string) *Diagnostic {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, context...)
	d := newLocated(f.ID, f.category, f.messages, location)
	d.context = slices.Clone(context)
	d.factory = f
	return d
}

// ContextReplay rebuilds the accumulated text of context diagnostics in a fixed order. contextFactory.at runs in
// whatever order the workers reach it, and some of those diagnostics get dropped because an earlier file stopped the
// compile. A compile merges its diagnostics in file order and passes them through Apply, which gives each one the
// text rbxtsc would have printed: every context its factory received before it. Use one ContextReplay for the life
// of the process, because upstream's accumulation lasts that long too.
type ContextReplay struct {
	added map[*contextFactory][]string
}

// NewContextReplay makes an empty replay.
func NewContextReplay() *ContextReplay {
	return &ContextReplay{added: map[*contextFactory][]string{}}
}

// Apply rewrites the message of every context diagnostic in list, in list order.
func (r *ContextReplay) Apply(list []*Diagnostic) {
	for _, d := range list {
		if d.factory == nil {
			continue
		}
		r.added[d.factory] = append(r.added[d.factory], d.context...)
		d.message = strings.Join(r.added[d.factory], "\n")
	}
}

// NoRojoDataFactory is errors.noRojoData.
type NoRojoDataFactory struct {
	contextFactory
	kleur Kleur
}

// At is errors.noRojoData(node, path, isPackage).
func (f *NoRojoDataFactory) At(location Location, path string, isPackage bool) *Diagnostic {
	context := []string{"Could not find Rojo data. There is no $path in your Rojo config that covers " + path}
	if isPackage {
		context = append(context, suggestion(f.kleur, "Did you forget to add a custom npm scope to your default.project.json?"))
	}
	return f.at(location, context...)
}

// NoPackageImportWithoutScopeFactory is errors.noPackageImportWithoutScope.
type NoPackageImportWithoutScopeFactory struct {
	contextFactory
	kleur Kleur
}

// At is errors.noPackageImportWithoutScope(node, path, rbxPath).
func (f *NoPackageImportWithoutScopeFactory) At(location Location, path string, rbxPath []string) *Diagnostic {
	return f.at(location,
		"Imported package Roblox path is missing an npm scope!",
		"Package path: "+path,
		"Roblox path: "+strings.Join(rbxPath, "."),
		suggestion(f.kleur, "You might need to update your \"node_modules\" in default.project.json to match:\n\"node_modules\": {\n\t\"$className\": \"Folder\",\n\t\"@rbxts\": {\n\t\t\"$path\": \"node_modules/@rbxts\"\n\t}\n}"),
	)
}

func suggestion(k Kleur, text string) string { return "Suggestion: " + k.Yellow(text) }

func issue(k Kleur, id int) string {
	return "More information: " + k.Grey(fmt.Sprintf("https://github.com/roblox-ts/roblox-ts/issues/%d", id))
}

// ErrorDefinitions is `errors` from src/Shared/diagnostics.ts. Fields are in source order.
type ErrorDefinitions struct {
	// reserved identifiers
	NoInvalidIdentifier   *DiagnosticFactory
	NoReservedIdentifier  *DiagnosticFactory
	NoReservedClassFields *DiagnosticFactory
	NoClassMetamethods    *DiagnosticFactory

	// banned statements
	NoForInStatement    *DiagnosticFactory
	NoLabeledStatement  *DiagnosticFactory
	NoDebuggerStatement *DiagnosticFactory

	// banned expressions
	NoNullLiteral       *DiagnosticFactory
	NoPrivateIdentifier *DiagnosticFactory
	NoTypeOfExpression  *DiagnosticFactory
	NoRegex             *DiagnosticFactory
	NoBigInt            *DiagnosticFactory

	// banned features
	NoAny                                     *DiagnosticFactory
	NoVar                                     *DiagnosticFactory
	NoGetterSetter                            *DiagnosticFactory
	NoAutoAccessorModifiers                   *DiagnosticFactory
	NoEqualsEquals                            *DiagnosticFactory
	NoExclamationEquals                       *DiagnosticFactory
	NoEnumMerging                             *DiagnosticFactory
	NoNamespaceMerging                        *DiagnosticFactory
	NoSpreadDestructuring                     *DiagnosticFactory
	NoFunctionExpressionName                  *DiagnosticFactory
	NoPrecedingSpreadElement                  *DiagnosticFactory
	NoLuaTupleDestructureAssignmentExpression *DiagnosticFactory
	NoExportAssignmentLet                     *DiagnosticFactory
	NoGlobalThis                              *DiagnosticFactory
	NoArguments                               *DiagnosticFactory
	NoPrototype                               *DiagnosticFactory
	NoRobloxSymbolInstanceof                  *DiagnosticFactory
	NoNonNumberStringRelationOperator         *DiagnosticFactory
	NoInstanceMethodCollisions                *DiagnosticFactory
	NoStaticMethodCollisions                  *DiagnosticFactory
	NoUnaryPlus                               *DiagnosticFactory
	NoNonNumberUnaryMinus                     *DiagnosticFactory
	NoAwaitForOf                              *DiagnosticFactory
	NoAsyncGeneratorFunctions                 *DiagnosticFactory
	NoNonStringModuleSpecifier                *DiagnosticFactory
	NoIterableIteration                       *DiagnosticFactory
	NoMixedTypeCall                           *DiagnosticFactory
	NoIndexWithoutCall                        *DiagnosticFactory
	NoCommentDirectives                       *DiagnosticFactory

	// macro methods
	NoOptionalMacroCall          *DiagnosticFactory
	NoConstructorMacroWithoutNew *DiagnosticFactory
	NoMacroExtends               *DiagnosticFactory
	NoMacroUnion                 *DiagnosticFactory
	NoMacroObjectSpread          *DiagnosticFactory
	NoVarArgsMacroSpread         *DiagnosticFactory
	NoRangeMacroOutsideForOf     *DiagnosticFactory
	NoTupleMacroOutsideReturn    *DiagnosticFactory

	// import/export
	NoModuleSpecifierFile *DiagnosticFactory
	NoInvalidModule       *DiagnosticFactory
	NoUnscopedModule      *DiagnosticFactory
	NoNonModuleImport     *DiagnosticFactory
	NoIsolatedImport      *DiagnosticFactory
	NoServerImport        *DiagnosticFactory

	// jsx
	NoPrecedingJsxSpreadElement *DiagnosticFactory

	// semantic
	ExpectedMethodGotFunction *DiagnosticFactory
	ExpectedFunctionGotMethod *DiagnosticFactory

	// files
	NoRojoData                  *NoRojoDataFactory
	NoPackageImportWithoutScope *NoPackageImportWithoutScopeFactory

	kleur Kleur
}

// IncorrectFileName is errors.incorrectFileName.
func (e *ErrorDefinitions) IncorrectFileName(originalFileName, suggestedFileName, fullPath string) *Diagnostic {
	return TextDiagnostic(strings.Join([]string{
		"Incorrect file name: `" + originalFileName + "`!",
		"Full path: " + fullPath,
		suggestion(e.kleur, "Change `"+originalFileName+"` to `"+suggestedFileName+"`."),
	}, "\n"), diagnostics.CategoryError)
}

// RojoPathInSrc is errors.rojoPathInSrc.
func (e *ErrorDefinitions) RojoPathInSrc(partitionPath, suggestedPath string) *Diagnostic {
	return TextDiagnostic(strings.Join([]string{
		"Invalid Rojo configuration. $path fields should be relative to out directory.",
		suggestion(e.kleur, `Change the value of $path from "`+partitionPath+`" to "`+suggestedPath+`".`),
	}, "\n"), diagnostics.CategoryError)
}

// WarningDefinitions is `warnings` from src/Shared/diagnostics.ts.
type WarningDefinitions struct {
	RuntimeLibUsedInReplicatedFirst *DiagnosticFactory

	kleur  Kleur
	mu     sync.Mutex
	nextID int
}

func (w *WarningDefinitions) newFactory(messages ...string) *DiagnosticFactory {
	w.mu.Lock()
	defer w.mu.Unlock()
	f := &DiagnosticFactory{ID: w.nextID, category: diagnostics.CategoryWarning, messages: messages}
	w.nextID++
	return f
}

// TruthyChange is warnings.truthyChange. Each call makes a new factory and so a new id, as upstream does.
func (w *WarningDefinitions) TruthyChange(checksStr string) *DiagnosticFactory {
	return w.newFactory("Value will be checked against " + checksStr)
}

// StringOffsetChange is warnings.stringOffsetChange. Each call makes a new factory and so a new id.
func (w *WarningDefinitions) StringOffsetChange(text string) *DiagnosticFactory {
	return w.newFactory("String macros no longer offset inputs: " + text)
}

// TransformerNotFound is warnings.transformerNotFound.
func (w *WarningDefinitions) TransformerNotFound(name, err string) *Diagnostic {
	return TextDiagnostic(strings.Join([]string{
		"Transformer `" + name + "` was not found!",
		"More info: " + err,
		suggestion(w.kleur, "Did you forget to install the package?"),
	}, "\n"), diagnostics.CategoryWarning)
}

// NewDefinitions builds `errors` and `warnings`. Ids count up through both, in source order.
func NewDefinitions(k Kleur) (*ErrorDefinitions, *WarningDefinitions) {
	id := 0
	errorFactory := func(messages ...string) *DiagnosticFactory {
		f := &DiagnosticFactory{ID: id, category: diagnostics.CategoryError, messages: messages}
		id++
		return f
	}
	s := func(text string) string { return suggestion(k, text) }
	e := &ErrorDefinitions{kleur: k}

	// reserved identifiers
	e.NoInvalidIdentifier = errorFactory(
		"Invalid Luau identifier!",
		"Luau identifiers must start with a letter and only contain letters, numbers, and underscores.",
		"Reserved Luau keywords cannot be used as identifiers.",
	)
	e.NoReservedIdentifier = errorFactory("Cannot use identifier reserved for compiler internal usage.")
	e.NoReservedClassFields = errorFactory("Cannot use class field reserved for compiler internal usage.")
	e.NoClassMetamethods = errorFactory("Metamethods cannot be used in class definitions!")

	// banned statements
	e.NoForInStatement = errorFactory("for-in loop statements are not supported!")
	e.NoLabeledStatement = errorFactory("labels are not supported!")
	e.NoDebuggerStatement = errorFactory("`debugger` is not supported!")

	// banned expressions
	e.NoNullLiteral = errorFactory("`null` is not supported!", s("Use `undefined` instead."))
	e.NoPrivateIdentifier = errorFactory("Private identifiers are not supported!")
	e.NoTypeOfExpression = errorFactory("`typeof` operator is not supported!", s("Use `typeIs(value, type)` or `typeOf(value)` instead."))
	e.NoRegex = errorFactory("Regular expressions are not supported!")
	e.NoBigInt = errorFactory("BigInt literals are not supported!")

	// banned features
	e.NoAny = errorFactory("Using values of type `any` is not supported!", s("Use `unknown` instead."))
	e.NoVar = errorFactory("`var` keyword is not supported!", s("Use `let` or `const` instead."))
	e.NoGetterSetter = errorFactory("Getters and Setters are not supported!", issue(k, 457))
	e.NoAutoAccessorModifiers = errorFactory(
		"Getters and Setters are not supported!",
		"The `accessor` keyword requires generating get/set accessors",
		issue(k, 457),
	)
	e.NoEqualsEquals = errorFactory("operator `==` is not supported!", s("Use `===` instead."))
	e.NoExclamationEquals = errorFactory("operator `!=` is not supported!", s("Use `!==` instead."))
	e.NoEnumMerging = errorFactory("Enum merging is not supported!")
	e.NoNamespaceMerging = errorFactory("Namespace merging is not supported!")
	e.NoSpreadDestructuring = errorFactory("Operator `...` is not supported for destructuring!")
	e.NoFunctionExpressionName = errorFactory("Function expression names are not supported!")
	e.NoPrecedingSpreadElement = errorFactory("Spread element must come last in a list of arguments!")
	e.NoLuaTupleDestructureAssignmentExpression = errorFactory("Cannot destructure LuaTuple<T> expression outside of an ExpressionStatement!")
	e.NoExportAssignmentLet = errorFactory("Cannot use `export =` on a `let` variable!", s("Use `const` instead."))
	e.NoGlobalThis = errorFactory("`globalThis` is not supported!")
	e.NoArguments = errorFactory("`arguments` is not supported!")
	e.NoPrototype = errorFactory("`prototype` is not supported!")
	e.NoRobloxSymbolInstanceof = errorFactory(
		"The `instanceof` operator can only be used on roblox-ts classes!",
		s("Use `typeIs(myThing, \"TypeToCheck\") instead"),
	)
	e.NoNonNumberStringRelationOperator = errorFactory("Relation operators can only be used on number or string types!")
	e.NoInstanceMethodCollisions = errorFactory("Static methods cannot use the same name as instance methods!")
	e.NoStaticMethodCollisions = errorFactory("Instance methods cannot use the same name as static methods!")
	e.NoUnaryPlus = errorFactory("Unary `+` is not supported!", s("Use `tonumber(x)` instead."))
	e.NoNonNumberUnaryMinus = errorFactory("Unary `-` is only supported for number types!")
	e.NoAwaitForOf = errorFactory("`await` is not supported in for-of loops!")
	e.NoAsyncGeneratorFunctions = errorFactory("Async generator functions are not supported!")
	e.NoNonStringModuleSpecifier = errorFactory("Module specifiers must be a string literal.")
	e.NoIterableIteration = errorFactory("Iterating on Iterable<T> is not supported! You must use a more specific type.")
	e.NoMixedTypeCall = errorFactory("Attempted to call a function with mixed types! All definitions must either be a method or a callback.")
	e.NoIndexWithoutCall = errorFactory("Cannot index a method without calling it!", s("Use the form `() => a.b()` instead of `a.b`."))
	e.NoCommentDirectives = errorFactory(
		"Usage of `@ts-ignore`, `@ts-expect-error`, and `@ts-nocheck` are not supported!",
		"roblox-ts needs type and symbol info to compile correctly.",
		s("Consider using type assertions or `declare` statements."),
	)

	// macro methods
	e.NoOptionalMacroCall = errorFactory("Macro methods can not be optionally called!", s("Macros always exist. Use a normal call."))
	e.NoConstructorMacroWithoutNew = errorFactory("Cannot index a constructor macro without using the `new` operator!")
	e.NoMacroExtends = errorFactory("Cannot extend from a macro class!", s("Store an instance of the macro class in a property."))
	e.NoMacroUnion = errorFactory("Macro cannot be applied to a union type!")
	e.NoMacroObjectSpread = errorFactory("Macro classes cannot be used in an object spread!", s("Did you mean to use an array spread? `[ ...exp ]`"))
	e.NoVarArgsMacroSpread = errorFactory("Macros which use variadic arguments do not support spread expressions!", issue(k, 1149))
	e.NoRangeMacroOutsideForOf = errorFactory("$range() macro is only valid as an expression of a for-of loop!")
	e.NoTupleMacroOutsideReturn = errorFactory("$tuple() macro is only valid as an expression of a return statement!")

	// import/export
	e.NoModuleSpecifierFile = errorFactory("Could not find file for import. Did you forget to `npm install`?")
	e.NoInvalidModule = errorFactory("You can only use npm scopes that are listed in your typeRoots.")
	e.NoUnscopedModule = errorFactory("You cannot use modules directly under node_modules.")
	e.NoNonModuleImport = errorFactory("Cannot import a non-ModuleScript!")
	e.NoIsolatedImport = errorFactory("Attempted to import a file inside of an isolated container from outside!")
	e.NoServerImport = errorFactory("Cannot import a server file from a shared or client location!", s("Move the file you want to import to a shared location."))

	// jsx
	e.NoPrecedingJsxSpreadElement = errorFactory("JSX spread expression must come last in children!")

	// semantic
	e.ExpectedMethodGotFunction = errorFactory("Attempted to assign non-method where method was expected.")
	e.ExpectedFunctionGotMethod = errorFactory("Attempted to assign method where non-method was expected.")

	// files
	e.NoRojoData = &NoRojoDataFactory{contextFactory: contextFactory{ID: id, category: diagnostics.CategoryError}, kleur: k}
	id++
	e.NoPackageImportWithoutScope = &NoPackageImportWithoutScopeFactory{contextFactory: contextFactory{ID: id, category: diagnostics.CategoryError}, kleur: k}
	id++

	w := &WarningDefinitions{kleur: k, nextID: id}
	w.RuntimeLibUsedInReplicatedFirst = w.newFactory("This statement would generate a call to the runtime library. The runtime library should not be used from ReplicatedFirst.")
	return e, w
}

// Errors and Warnings are the process-wide definitions, like the module-level values upstream.
// Color is detected once, as kleur does.
var Errors, Warnings = NewDefinitions(DetectKleur())

// Color is the process-wide Kleur, for messages built at call sites such as validateCompilerOptions.
var Color = DetectKleur()
