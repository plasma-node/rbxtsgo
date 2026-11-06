package main

import (
	"fmt"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
)

// macroManager mirrors the constructor of roblox-ts's src/TSTransformer/classes/MacroManager.ts.
// Each instance belongs to one checker, because merged global symbols are per-checker clones.
type macroManager struct {
	c            *checker.Checker
	symbols      map[string]*ast.Symbol
	callMacros   map[*ast.Symbol]string
	identifiers  map[*ast.Symbol]string
	constructors map[*ast.Symbol]string
	methods      map[*ast.Symbol]string
	macroOnly    map[*ast.Symbol]bool
	errors       []string
}

var (
	symbolNames = []string{
		"globalThis", "ArrayConstructor", "SetConstructor", "MapConstructor", "WeakSetConstructor", "WeakMapConstructor",
		"ReadonlyMapConstructor", "ReadonlySetConstructor", "Array", "Generator", "IterableFunction", "LuaTuple", "Map",
		"Object", "ReadonlyArray", "ReadonlyMap", "ReadonlySet", "ReadVoxelsArray", "Set", "SharedTable", "String",
		"TemplateStringsArray", "WeakMap", "WeakSet", "Iterable", "$range", "$tuple",
	}
	macroOnlyClasses   = []string{"ReadonlyArray", "Array", "ReadonlyMap", "WeakMap", "Map", "ReadonlySet", "WeakSet", "Set", "String"}
	identifierMacros   = []string{"Promise"}
	callMacros         = []string{"assert", "typeOf", "typeIs", "classIs", "identity", "$range", "$tuple", "$getModuleTree"}
	constructorMacros  = []string{"ArrayConstructor", "SetConstructor", "MapConstructor", "WeakSetConstructor", "WeakMapConstructor", "ReadonlyMapConstructor", "ReadonlySetConstructor"}
	propertyMacroOwner = []string{"ReadonlyArray", "Array", "ReadonlySet", "Set", "WeakSet", "ReadonlyMap", "Map", "WeakMap", "String"}
)

const nominalLuaTuple = "_nominal_LuaTuple"

func newMacroManager(c *checker.Checker) *macroManager {
	m := &macroManager{
		c:            c,
		symbols:      map[string]*ast.Symbol{},
		callMacros:   map[*ast.Symbol]string{},
		identifiers:  map[*ast.Symbol]string{},
		constructors: map[*ast.Symbol]string{},
		methods:      map[*ast.Symbol]string{},
		macroOnly:    map[*ast.Symbol]bool{},
	}
	for _, name := range identifierMacros {
		if s := m.global(name, ast.SymbolFlagsVariable); s != nil {
			m.identifiers[s] = name
		}
	}
	for _, name := range callMacros {
		if s := m.global(name, ast.SymbolFlagsFunction); s != nil {
			m.callMacros[s] = name
		}
	}
	for _, name := range constructorMacros {
		s := m.global(name, ast.SymbolFlagsInterface)
		if s == nil {
			continue
		}
		if construct := firstConstructSymbol(s); construct != nil {
			m.constructors[construct] = name
		} else {
			m.errors = append(m.errors, "no construct signature: "+name)
		}
	}
	// The real macro list names specific methods. Taking every method signature is
	// enough here, since the spike tests symbol identity rather than the macro bodies.
	for _, owner := range propertyMacroOwner {
		s := m.global(owner, ast.SymbolFlagsInterface)
		if s == nil {
			continue
		}
		for _, decl := range s.Declarations() {
			if !ast.IsInterfaceDeclaration(decl) {
				continue
			}
			for _, member := range decl.AsInterfaceDeclaration().Members.Nodes {
				if !ast.IsMethodSignatureDeclaration(member) || !ast.IsIdentifier(member.Name()) {
					continue
				}
				if symbol := c.GetTypeAtLocation(member).Symbol(); symbol != nil {
					m.methods[symbol] = owner + "." + member.Name().Text()
				} else {
					m.errors = append(m.errors, "no type symbol: "+owner+"."+member.Name().Text())
				}
			}
		}
	}
	for _, name := range symbolNames {
		if s := m.global(name, ast.SymbolFlagsAll); s != nil {
			m.symbols[name] = s
		}
	}
	for _, name := range macroOnlyClasses {
		if s := m.symbols[name]; s != nil {
			m.macroOnly[s] = true
		}
	}
	if tuple := m.symbols["LuaTuple"]; tuple != nil {
		for _, decl := range tuple.Declarations() {
			if ast.IsTypeAliasDeclaration(decl) {
				if s := c.GetPropertyOfType(c.GetTypeAtLocation(decl), nominalLuaTuple); s != nil {
					m.symbols[nominalLuaTuple] = s
				}
			}
		}
	}
	if m.symbols[nominalLuaTuple] == nil {
		m.errors = append(m.errors, "could not find "+nominalLuaTuple)
	}
	return m
}

func (m *macroManager) global(name string, meaning ast.SymbolFlags) *ast.Symbol {
	s := m.c.ResolveName(name, nil, meaning, false)
	if s == nil {
		m.errors = append(m.errors, fmt.Sprintf("global not found: %s", name))
		return nil
	}
	if s.Flags()&ast.SymbolFlagsAlias != 0 {
		return m.c.GetAliasedSymbol(s)
	}
	return s
}

func firstConstructSymbol(symbol *ast.Symbol) *ast.Symbol {
	for _, decl := range symbol.Declarations() {
		if !ast.IsInterfaceDeclaration(decl) {
			continue
		}
		for _, member := range decl.AsInterfaceDeclaration().Members.Nodes {
			if ast.IsConstructSignatureDeclaration(member) {
				return member.Symbol()
			}
		}
	}
	return nil
}

// classify mirrors isArrayType, isMapType, isSetType and isLuaTupleType in util/types.ts.
func (m *macroManager) classify(t *checker.Type) string {
	if t == nil {
		return "nil"
	}
	symbol := t.Symbol()
	is := func(names ...string) bool {
		for _, name := range names {
			if symbol != nil && symbol == m.symbols[name] {
				return true
			}
		}
		return false
	}
	result := ""
	add := func(label string, ok bool) {
		if ok {
			if result != "" {
				result += ","
			}
			result += label
		}
	}
	isAny := t.Flags()&checker.TypeFlagsAny != 0
	add("array", !isAny && (checker.IsTupleType(t) || m.c.IsArrayLikeType(t) || is("ReadonlyArray", "Array", "ReadVoxelsArray", "TemplateStringsArray")))
	add("tuple", checker.IsTupleType(t))
	add("map", is("Map", "ReadonlyMap", "WeakMap"))
	add("set", is("Set", "ReadonlySet", "WeakSet"))
	add("luaTuple", m.c.GetPropertyOfType(t, nominalLuaTuple) != nil && m.c.GetPropertyOfType(t, nominalLuaTuple) == m.symbols[nominalLuaTuple])
	if result == "" {
		return "-"
	}
	return result
}
