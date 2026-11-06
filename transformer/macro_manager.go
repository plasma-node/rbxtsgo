package transformer

// Port of src/TSTransformer/classes/MacroManager.ts, the lookup side, and of src/TSTransformer/macros/types.ts, which
// can't have a file of its own because types.go is util/types.ts. The symbol side is in tsbridge/macros.go: it maps a
// symbol to a macro's name, and the tables here map that name to the macro.

import (
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

type identifierMacro func(state *TransformState, node *ast.Node) luau.Expression

type constructorMacro func(state *TransformState, node *ast.Node) luau.Expression

type callMacro func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression

// propertyCallMacro is an alias, so that runCallMacro can take either kind of macro, as upstream's does.
type propertyCallMacro = callMacro

// macroTables holds the macros by name. It's shared between workers and is read-only once built.
type macroTables struct {
	identifiers   map[string]identifierMacro
	calls         map[string]callMacro
	constructors  map[string]constructorMacro
	propertyCalls map[string]propertyCallMacro
}

// These have no initializer on purpose: the macros reach transformExpression, which reaches the lookups below, and
// that would be an initialization cycle.
var (
	macroTablesOnce  sync.Once
	macroTablesValue *macroTables
)

func getMacroTables() *macroTables {
	macroTablesOnce.Do(func() {
		macroTablesValue = &macroTables{
			identifiers:   identifierMacros(),
			calls:         callMacros(),
			constructors:  constructorMacros(),
			propertyCalls: propertyCallMacros(),
		}
	})
	return macroTablesValue
}

func getSymbolOrThrow(state *TransformState, name string) *ast.Symbol {
	symbol := state.Services.MacroManager.Symbol(name)
	if symbol == nil {
		panic(shared.AssertionMessage(""))
	}
	return symbol
}

func getIdentifierMacro(state *TransformState, symbol *ast.Symbol) identifierMacro {
	name := state.Services.MacroManager.IdentifierMacro(symbol)
	if name == "" {
		return nil
	}
	return getMacroTables().identifiers[name]
}

func getCallMacro(state *TransformState, symbol *ast.Symbol) callMacro {
	name := state.Services.MacroManager.CallMacro(symbol)
	if name == "" {
		return nil
	}
	return getMacroTables().calls[name]
}

func getConstructorMacro(state *TransformState, symbol *ast.Symbol) constructorMacro {
	name := state.Services.MacroManager.ConstructorMacro(symbol)
	if name == "" {
		return nil
	}
	return getMacroTables().constructors[name]
}

func getPropertyCallMacro(state *TransformState, symbol *ast.Symbol) propertyCallMacro {
	name := state.Services.MacroManager.PropertyCallMacro(symbol)
	if name == "" {
		if state.Services.MacroManager.IsUnimplementedMacroMethod(symbol) {
			panic(shared.AssertionMessage("Macro " + symbol.Parent().Name() + "." + symbol.Name() + "() is not implemented!"))
		}
		return nil
	}
	return getMacroTables().propertyCalls[name]
}
