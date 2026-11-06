package tsbridge

import (
	"errors"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
)

const typesNotice = "\nYou may need to update your @rbxts/compiler-types!"

// NominalLuaTupleName is NOMINAL_LUA_TUPLE_NAME in MacroManager.ts.
const NominalLuaTupleName = "_nominal_LuaTuple"

// SymbolNames is the values of SYMBOL_NAMES in MacroManager.ts at roblox-ts 3.0.0, in source order.
var SymbolNames = []string{
	"globalThis",
	"ArrayConstructor", "SetConstructor", "MapConstructor", "WeakSetConstructor", "WeakMapConstructor",
	"ReadonlyMapConstructor", "ReadonlySetConstructor",
	"Array", "Generator", "IterableFunction", "LuaTuple", "Map", "Object", "ReadonlyArray", "ReadonlyMap",
	"ReadonlySet", "ReadVoxelsArray", "Set", "String", "TemplateStringsArray", "WeakMap", "WeakSet",
	"Iterable",
	"$range", "$tuple",
}

var macroOnlyClasses = map[string]bool{
	"ReadonlyArray": true, "Array": true, "ReadonlyMap": true, "WeakMap": true, "Map": true,
	"ReadonlySet": true, "WeakSet": true, "Set": true, "String": true,
}

// PropertyMacroClass is one entry of PROPERTY_CALL_MACROS: a class and its macro method names.
type PropertyMacroClass struct {
	Class   string
	Methods []string
}

// MacroSpec names the macros to look up. The macro bodies belong to the transformer, so this only has names,
// and lookups below return the name. The order is the upstream Object.entries order, which decides which
// error is reported first when compiler-types is out of date.
type MacroSpec struct {
	Identifiers   []string
	Calls         []string
	Constructors  []string
	PropertyCalls []PropertyMacroClass
}

// MacroSymbols is the symbol side of MacroManager.ts for one checker. It maps symbols to macro names.
//
// Never share one across checkers. Globals declared in several files are cloned per checker, so a table from
// another checker gives wrong answers with no error, and only once a project adds an augmentation such as
// `declare global { interface Array<T> }`. See 0002.
type MacroSymbols struct {
	checker     *checker.Checker
	symbols     map[string]*ast.Symbol
	identifiers map[*ast.Symbol]string
	calls       map[*ast.Symbol]string
	constructor map[*ast.Symbol]string
	property    map[*ast.Symbol]string
	// methodOrder is the order in which MacroManager's constructor first asks for the type of each method of a macro
	// class. In TS 5.5.3 that's the order of those types' ids, because nothing else has run yet.
	methodOrder map[*ast.Symbol]int
}

// NewMacroSymbols ports the MacroManager constructor. Errors have the upstream ProjectError text.
func NewMacroSymbols(c *checker.Checker, spec MacroSpec) (*MacroSymbols, error) {
	m := &MacroSymbols{
		checker:     c,
		symbols:     map[string]*ast.Symbol{},
		identifiers: map[*ast.Symbol]string{},
		calls:       map[*ast.Symbol]string{},
		constructor: map[*ast.Symbol]string{},
		property:    map[*ast.Symbol]string{},
		methodOrder: map[*ast.Symbol]int{},
	}
	global := func(name string, meaning ast.SymbolFlags) (*ast.Symbol, error) {
		if symbol := c.ResolveName(name, nil, meaning, false); symbol != nil {
			return symbol, nil
		}
		return nil, errors.New("MacroManager could not find symbol for " + name + typesNotice)
	}
	for _, name := range spec.Identifiers {
		symbol, err := global(name, ast.SymbolFlagsVariable)
		if err != nil {
			return nil, err
		}
		m.identifiers[symbol] = name
	}
	for _, name := range spec.Calls {
		symbol, err := global(name, ast.SymbolFlagsFunction)
		if err != nil {
			return nil, err
		}
		m.calls[symbol] = name
	}
	for _, name := range spec.Constructors {
		symbol, err := global(name, ast.SymbolFlagsInterface)
		if err != nil {
			return nil, err
		}
		construct, err := firstConstructSymbol(symbol, name)
		if err != nil {
			return nil, err
		}
		m.constructor[construct] = name
	}
	for _, class := range spec.PropertyCalls {
		symbol, err := global(class.Class, ast.SymbolFlagsInterface)
		if err != nil {
			return nil, err
		}
		methods := map[string]*ast.Symbol{}
		for _, decl := range symbol.Declarations() {
			if !ast.IsInterfaceDeclaration(decl) {
				continue
			}
			for _, member := range decl.AsInterfaceDeclaration().Members.Nodes {
				if ast.IsMethodSignatureDeclaration(member) && ast.IsIdentifier(member.Name()) {
					methodSymbol := c.GetTypeAtLocation(member).Symbol()
					if methodSymbol == nil {
						panic("tsbridge: method signature " + member.Name().Text() + " has no type symbol")
					}
					methods[member.Name().Text()] = methodSymbol
					if _, seen := m.methodOrder[methodSymbol]; !seen {
						m.methodOrder[methodSymbol] = len(m.methodOrder)
					}
				}
			}
		}
		for _, method := range class.Methods {
			methodSymbol := methods[method]
			if methodSymbol == nil {
				return nil, errors.New("MacroManager could not find method for " + class.Class + "." + method + typesNotice)
			}
			m.property[methodSymbol] = class.Class + "." + method
		}
	}
	for _, name := range SymbolNames {
		symbol, err := global(name, ast.SymbolFlagsAll)
		if err != nil {
			return nil, err
		}
		m.symbols[name] = symbol
	}
	for _, decl := range m.symbols["LuaTuple"].Declarations() {
		if ast.IsTypeAliasDeclaration(decl) {
			if symbol := c.GetPropertyOfType(c.GetTypeAtLocation(decl), NominalLuaTupleName); symbol != nil {
				m.symbols[NominalLuaTupleName] = symbol
			}
			break
		}
	}
	return m, nil
}

func firstConstructSymbol(symbol *ast.Symbol, name string) (*ast.Symbol, error) {
	for _, decl := range symbol.Declarations() {
		if !ast.IsInterfaceDeclaration(decl) {
			continue
		}
		// getFirstDeclarationOrThrow takes the first interface declaration only.
		for _, member := range decl.AsInterfaceDeclaration().Members.Nodes {
			if ast.IsConstructSignatureDeclaration(member) {
				return member.Symbol(), nil
			}
		}
		return nil, errors.New("MacroManager could not find constructor for " + name + typesNotice)
	}
	return nil, errors.New("")
}

// Checker returns the checker this table belongs to.
func (m *MacroSymbols) Checker() *checker.Checker { return m.checker }

// Symbol returns the named global, or nil. Upstream's getSymbolOrThrow asserts on nil.
func (m *MacroSymbols) Symbol(name string) *ast.Symbol { return m.symbols[name] }

// MethodCreationOrder returns where a method of a macro class falls in TS 5.5.3's type id order, and false for any
// other symbol.
func (m *MacroSymbols) MethodCreationOrder(symbol *ast.Symbol) (int, bool) {
	order, ok := m.methodOrder[symbol]
	return order, ok
}

// IsMacroOnlyClass is MacroManager.isMacroOnlyClass.
func (m *MacroSymbols) IsMacroOnlyClass(symbol *ast.Symbol) bool {
	return m.symbols[symbol.Name()] == symbol && macroOnlyClasses[symbol.Name()]
}

// IdentifierMacro returns the macro name for symbol, or "" if it has none.
func (m *MacroSymbols) IdentifierMacro(symbol *ast.Symbol) string { return m.identifiers[symbol] }

// CallMacro returns the macro name for symbol, or "" if it has none.
func (m *MacroSymbols) CallMacro(symbol *ast.Symbol) string { return m.calls[symbol] }

// ConstructorMacro returns the macro name for symbol, or "" if it has none.
func (m *MacroSymbols) ConstructorMacro(symbol *ast.Symbol) string { return m.constructor[symbol] }

// PropertyCallMacro returns the macro name as "Class.method" for symbol, or "" if it has none.
// Upstream's assert on an unimplemented macro-only method is left to the caller, which has the macro bodies.
func (m *MacroSymbols) PropertyCallMacro(symbol *ast.Symbol) string { return m.property[symbol] }

// IsUnimplementedMacroMethod is the condition upstream's getPropertyCallMacro asserts on: no macro for this
// method, but its class is macro-only.
func (m *MacroSymbols) IsUnimplementedMacroMethod(symbol *ast.Symbol) bool {
	parent := symbol.Parent()
	return m.property[symbol] == "" && parent != nil && m.symbols[parent.Name()] == parent && m.IsMacroOnlyClass(parent)
}
