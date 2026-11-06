package transformer

// Port of src/TSTransformer/util/types.ts.

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/jsnum"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

type typeCheck = func(*checker.Type) bool

// checkerOfType stands in for the checker that JS type objects carry, which upstream's `type.getConstraint()`,
// `type.getBaseTypes()`, `type.getProperties()` and so on use. The export is in patches/typescript/0001.
func checkerOfType(t *checker.Type) *checker.Checker {
	return t.Checker()
}

func isClassOrInterface(t *checker.Type) bool {
	return t.ObjectFlags()&checker.ObjectFlagsClassOrInterface != 0
}

func getRecursiveBaseTypesInner(result *[]*checker.Type, t *checker.Type) {
	for _, baseType := range checkerOfType(t).GetBaseTypes(t) {
		*result = append(*result, baseType)
		if isClassOrInterface(baseType) {
			getRecursiveBaseTypesInner(result, baseType)
		}
	}
}

func getRecursiveBaseTypes(t *checker.Type) []*checker.Type {
	var result []*checker.Type
	getRecursiveBaseTypesInner(&result, t)
	return result
}

func isDefinitelyTypeInner(t *checker.Type, callbacks []typeCheck) bool {
	if t.IsUnion() {
		return !slices.ContainsFunc(t.Types(), func(u *checker.Type) bool { return !isDefinitelyTypeInner(u, callbacks) })
	} else if t.IsIntersection() {
		return slices.ContainsFunc(t.Types(), func(u *checker.Type) bool { return isDefinitelyTypeInner(u, callbacks) })
	} else {
		if isClassOrInterface(t) && slices.ContainsFunc(getRecursiveBaseTypes(t), func(u *checker.Type) bool { return isDefinitelyTypeInner(u, callbacks) }) {
			return true
		}
		return slices.ContainsFunc(callbacks, func(cb typeCheck) bool { return cb(t) })
	}
}

func isDefinitelyType(t *checker.Type, callbacks ...typeCheck) bool {
	constraint := checkerOfType(t).GetBaseConstraintOfType(t)
	if constraint == nil {
		constraint = t
	}
	return isDefinitelyTypeInner(constraint, callbacks)
}

func isPossiblyTypeInner(t *checker.Type, callbacks []typeCheck) bool {
	if t.Flags()&checker.TypeFlagsUnionOrIntersection != 0 {
		return slices.ContainsFunc(t.Types(), func(u *checker.Type) bool { return isPossiblyTypeInner(u, callbacks) })
	} else {
		if isClassOrInterface(t) && slices.ContainsFunc(getRecursiveBaseTypes(t), func(u *checker.Type) bool { return isPossiblyTypeInner(u, callbacks) }) {
			return true
		}

		// type variable without constraint, any, or unknown
		if t.Flags()&(checker.TypeFlagsTypeVariable|checker.TypeFlagsAnyOrUnknown) != 0 {
			return true
		}

		// defined type
		if isDefinedType(t) {
			// Go funcs aren't comparable, so this is `callbacks[0] === isUndefinedType` by code pointer.
			if len(callbacks) == 1 && reflect.ValueOf(callbacks[0]).Pointer() == reflect.ValueOf(isUndefinedType).Pointer() {
				// if only matching undefined, then defined means not possible
				return false
			}
			return true
		}

		return slices.ContainsFunc(callbacks, func(cb typeCheck) bool { return cb(t) })
	}
}

func isPossiblyType(t *checker.Type, callbacks ...typeCheck) bool {
	constraint := checkerOfType(t).GetBaseConstraintOfType(t)
	if constraint == nil {
		constraint = t
	}
	return isPossiblyTypeInner(constraint, callbacks)
}

func isDefinedType(t *checker.Type) bool {
	if t.Flags() != checker.TypeFlagsObject {
		return false
	}
	c := checkerOfType(t)
	return len(c.GetPropertiesOfType(t)) == 0 &&
		len(c.GetCallSignatures(t)) == 0 &&
		len(c.GetConstructSignatures(t)) == 0 &&
		c.GetIndexTypeOfType(t, c.GetNumberType()) == nil &&
		c.GetIndexTypeOfType(t, c.GetStringType()) == nil
}

func isAnyType(state *TransformState) typeCheck {
	return func(t *checker.Type) bool { return t == state.Checker.GetAnyType() }
}

func isBooleanType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsBoolean|checker.TypeFlagsBooleanLiteral) != 0
}

func isBooleanLiteralType(state *TransformState, value bool) typeCheck {
	return func(t *checker.Type) bool {
		if t.Flags()&checker.TypeFlagsBooleanLiteral != 0 {
			valueType := state.Checker.GetFalseType()
			if value {
				valueType = state.Checker.GetTrueType()
			}
			return t == valueType
		}
		return isBooleanType(t)
	}
}

func isNumberType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsNumber|checker.TypeFlagsNumberLike|checker.TypeFlagsNumberLiteral) != 0
}

func isNumberLiteralType(value float64) typeCheck {
	return func(t *checker.Type) bool {
		if t.IsNumberLiteral() {
			return t.AsLiteralType().Value() == any(jsnum.Number(value))
		}
		return isNumberType(t)
	}
}

func isNaNType(t *checker.Type) bool {
	return isNumberType(t) && !t.IsNumberLiteral()
}

func isStringType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsString|checker.TypeFlagsStringLike|checker.TypeFlagsStringLiteral) != 0
}

func isArrayType(state *TransformState) typeCheck {
	return func(t *checker.Type) bool {
		// typeChecker.isArrayLikeType() will return true for `any`, so rule it out here
		if t.Flags()&checker.TypeFlagsAny != 0 {
			return false
		}
		return checker.IsTupleType(t) ||
			state.Checker.IsArrayLikeType(t) ||
			t.Symbol() == getSymbolOrThrow(state, "ReadonlyArray") ||
			t.Symbol() == getSymbolOrThrow(state, "Array") ||
			t.Symbol() == getSymbolOrThrow(state, "ReadVoxelsArray") ||
			t.Symbol() == getSymbolOrThrow(state, "TemplateStringsArray")
	}
}

func isSetType(state *TransformState) typeCheck {
	return func(t *checker.Type) bool {
		return t.Symbol() == getSymbolOrThrow(state, "Set") ||
			t.Symbol() == getSymbolOrThrow(state, "ReadonlySet") ||
			t.Symbol() == getSymbolOrThrow(state, "WeakSet")
	}
}

func isMapType(state *TransformState) typeCheck {
	return func(t *checker.Type) bool {
		return t.Symbol() == getSymbolOrThrow(state, "Map") ||
			t.Symbol() == getSymbolOrThrow(state, "ReadonlyMap") ||
			t.Symbol() == getSymbolOrThrow(state, "WeakMap")
	}
}

func isGeneratorType(state *TransformState) typeCheck {
	return func(t *checker.Type) bool { return t.Symbol() == getSymbolOrThrow(state, "Generator") }
}

func isIterableFunctionType(state *TransformState) typeCheck {
	return func(t *checker.Type) bool { return t.Symbol() == getSymbolOrThrow(state, "IterableFunction") }
}

func isLuaTupleType(state *TransformState) typeCheck {
	return func(t *checker.Type) bool {
		return state.Checker.GetPropertyOfType(t, tsbridge.NominalLuaTupleName) ==
			getSymbolOrThrow(state, tsbridge.NominalLuaTupleName)
	}
}

func isIterableFunctionLuaTupleType(state *TransformState) typeCheck {
	return func(t *checker.Type) bool {
		if isIterableFunctionType(state)(t) {
			typeArguments := getTypeArguments(state, t)
			return len(typeArguments) > 0 && isLuaTupleType(state)(typeArguments[0])
		}
		return false
	}
}

func isIterableType(state *TransformState) typeCheck {
	return func(t *checker.Type) bool { return t.Symbol() == getSymbolOrThrow(state, "Iterable") }
}

func isObjectType(t *checker.Type) bool {
	return t.Flags()&checker.TypeFlagsObject != 0
}

func isUndefinedType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsUndefined|checker.TypeFlagsVoid) != 0
}

func isEmptyStringType(t *checker.Type) bool {
	if t.IsStringLiteral() {
		return t.AsLiteralType().Value() == any("")
	}
	if isTemplateLiteralType(t) {
		texts := t.AsTemplateLiteralType().Texts()
		return len(texts) == 0 || !slices.ContainsFunc(texts, func(v string) bool { return len(v) != 0 })
	}
	return isStringType(t)
}

func isRobloxType(state *TransformState) typeCheck {
	typesPath := nodepath.Native.Join(state.Data.NodeModulesPath, shared.RbxtsScope, "types")
	return func(t *checker.Type) bool {
		symbol := t.Symbol()
		if symbol == nil {
			return false
		}
		return slices.ContainsFunc(symbol.Declarations(), func(d *ast.Node) bool {
			sourceFile := ast.GetSourceFileOfNode(d)
			return sourceFile != nil && shared.IsPathDescendantOf(sourceFile.FileName().AsString(), typesPath)
		})
	}
}

// type utilities

func walkTypes(t *checker.Type, callback func(*checker.Type)) {
	if t.Flags()&checker.TypeFlagsUnionOrIntersection != 0 {
		for _, u := range t.Types() {
			walkTypes(u, callback)
		}
	} else {
		// in template literal types, constraint === type and this causes infinite recursion
		constraint := checkerOfType(t).GetBaseConstraintOfType(t)
		if constraint != nil && constraint != t {
			walkTypes(constraint, callback)
		} else {
			callback(t)
		}
	}
}

func getFirstConstructSymbol(state *TransformState, expression *ast.Node) *ast.Symbol {
	t := state.GetType(expression)
	if t.Symbol() != nil {
		for _, declaration := range t.Symbol().Declarations() {
			if ast.IsInterfaceDeclaration(declaration) {
				for _, member := range declaration.AsInterfaceDeclaration().Members.Nodes {
					if ast.IsConstructSignatureDeclaration(member) {
						return member.Symbol()
					}
				}
			}
		}
	}
	return nil
}

var (
	traceUnionOnce  sync.Once
	traceUnionPath  string
	traceUnionMutex sync.Mutex
)

// traceUnionMembers is not in upstream. It measures the union order difference between TS 7 and TS 5.5.3 (see "Union
// order" in the guide): with RBXTSGO_TRACE_UNION=path, it appends one line per call on a union or intersection whose
// members carry two or more distinct symbols. It only reads symbol names, and doesn't print types, because printing
// can make the checker create types. Failing to write is ignored, so the trace can't affect a compile.
func traceUnionMembers(state *TransformState, t *checker.Type) {
	traceUnionOnce.Do(func() { traceUnionPath = os.Getenv("RBXTSGO_TRACE_UNION") })
	if traceUnionPath == "" {
		return
	}
	var names []string
	distinct := map[*ast.Symbol]bool{}
	for _, u := range t.Types() {
		if symbol := u.Symbol(); symbol != nil {
			names = append(names, ast.EscapeAllInternalSymbolNames(ast.SymbolName(symbol)))
			distinct[symbol] = true
		}
	}
	if len(distinct) < 2 {
		return
	}
	line := state.SourceFile.FileName().AsString() + ": " + strings.Join(names, " | ") + "\n"

	traceUnionMutex.Lock()
	defer traceUnionMutex.Unlock()
	file, err := os.OpenFile(traceUnionPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(line)
}

func getFirstDefinedSymbol(state *TransformState, t *checker.Type) *ast.Symbol {
	if t.Flags()&checker.TypeFlagsUnionOrIntersection != 0 {
		traceUnionMembers(state, t)
		// See docs/union-order.md. TS 5.5.3 sorts a union by type id, which is creation order, and
		// MacroManager creates the type of every method of a macro class before any file is checked. So in JS those
		// come first in any union, in MacroManager's order. TS 7 sorts by name and position. Call macros come next:
		// that part only matches what JS was measured to do, as docs/union-order.md says.
		var best *ast.Symbol
		bestOrder := -1
		if t.Flags()&checker.TypeFlagsUnion != 0 {
			for _, u := range t.Types() {
				if u.Symbol() == nil {
					continue
				}
				if order, ok := state.Services.MacroManager.MethodCreationOrder(u.Symbol()); ok && (best == nil || order < bestOrder) {
					best = u.Symbol()
					bestOrder = order
				}
			}
			if best != nil {
				return best
			}
			for _, u := range t.Types() {
				if u.Symbol() != nil && state.Services.MacroManager.CallMacro(u.Symbol()) != "" {
					return u.Symbol()
				}
			}
		}
		for _, u := range t.Types() {
			if u.Symbol() != nil && !state.Checker.IsUndefinedSymbol(u.Symbol()) {
				return u.Symbol()
			}
		}
		return nil
	} else {
		return t.Symbol()
	}
}

// getTypeArguments returns nil for a type that isn't a reference. Upstream casts to TypeReference, and JS's
// getTypeArguments gives an empty array for those, where Go's panics.
func getTypeArguments(state *TransformState, t *checker.Type) []*checker.Type {
	if t.ObjectFlags()&checker.ObjectFlagsReference == 0 {
		return nil
	}
	return state.Checker.GetTypeArguments(t)
}
