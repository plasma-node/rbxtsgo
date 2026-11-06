package transformer

// Port of src/TSTransformer/nodes/expressions/transformElementAccessExpression.ts.

import (
	"strconv"
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

func transformElementAccessExpressionInner(state *TransformState, node *ast.Node, expression luau.Expression, argumentExpression *ast.Node) luau.Expression {
	// a in a[b]
	validateNotAnyType(state, node.Expression())
	// b in a[b]
	validateNotAnyType(state, node.AsElementAccessExpression().ArgumentExpression)

	expType := state.Checker.GetNonOptionalType(state.GetType(node.Expression()))
	addIndexDiagnostics(state, node, expType)

	index, prereqs := Capture(state, func() luau.Expression { return transformExpression(state, argumentExpression) })

	if !prereqs.IsEmpty() {
		// hack because wrapReturnIfLuaTuple will not wrap this, but now we need to!
		if isLuaTupleType(state)(expType) {
			expression = luau.NewArray(expression)
		}

		expression = state.PushToVar(expression, "exp")
		state.PrereqList(prereqs)
	}

	// LuaTuple<T> checks
	if luau.IsCall(expression) && isLuaTupleType(state)(expType) {
		// wrap in select() if it isn't the first value
		if !luau.IsNumberLiteral(index) || !jsNumberIsZero(index.(*luau.NumberLiteral).Value) {
			expression = luau.NewCall(luau.Globals.Select, offset(index, 1), expression)
		}
		// parentheses to trim off the rest of the values
		return luau.Create(&luau.ParenthesizedExpression{Expression: expression})
	}

	if ast.IsDeleteExpression(SkipUpwards(node).Parent) {
		state.Prereq(
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: convertToIndexableExpression(expression),
					Index:      addOneIfArrayType(state, expType, index),
				}),
				Operator: luau.AssignEqual,
				Right:    luau.NewNil(),
			}),
		)
		return luau.NewNone()
	}

	return luau.Create(&luau.ComputedIndexExpression{
		Expression: convertToIndexableExpression(expression),
		Index:      addOneIfArrayType(state, expType, index),
	})
}

func transformElementAccessExpression(state *TransformState, node *ast.Node) luau.Expression {
	constantValue := getConstantValueLiteral(state, node)
	if constantValue != nil {
		return constantValue
	}

	return transformOptionalChain(state, node)
}

// jsNumberIsZero is `Number(value) === 0` for the text of a luau.NumberLiteral, which is either source text or the
// result of String(number), so it never has surrounding white space. A separator makes Number() give NaN, as in
// `tuple[0_0]`, and a decimal too small for a double gives 0. luau has the full Number() but doesn't export it.
func jsNumberIsZero(value string) bool {
	if value == "" {
		return true
	}
	if len(value) > 2 && value[0] == '0' && strings.ContainsRune("xXoObB", rune(value[1])) {
		return strings.Trim(value[2:], "0") == ""
	}
	rest := strings.TrimLeft(value[:1], "+-") + value[1:]
	i := 0
	skipDigits := func() int {
		start := i
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		return i - start
	}
	amtDigits := skipDigits()
	if i < len(rest) && rest[i] == '.' {
		i++
		amtDigits += skipDigits()
	}
	if amtDigits == 0 {
		return false
	}
	if i < len(rest) && (rest[i] == 'e' || rest[i] == 'E') {
		i++
		if i < len(rest) && (rest[i] == '+' || rest[i] == '-') {
			i++
		}
		if skipDigits() == 0 {
			return false
		}
	}
	if i != len(rest) {
		return false
	}
	number, _ := strconv.ParseFloat(rest, 64)
	return number == 0
}
