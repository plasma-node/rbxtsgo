package transformer

// Port of src/TSTransformer/util/createTruthinessChecks.ts.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func willCreateTruthinessChecks(t *checker.Type) bool {
	return isPossiblyType(t, isNumberLiteralType(0)) ||
		isPossiblyType(t, isNaNType) ||
		isPossiblyType(t, isEmptyStringType)
}

func createTruthinessChecks(state *TransformState, exp luau.Expression, node *ast.Node) luau.Expression {
	t := state.GetType(node)
	isAssignableToZero := isPossiblyType(t, isNumberLiteralType(0))
	isAssignableToNaN := isPossiblyType(t, isNaNType)
	isAssignableToEmptyString := isPossiblyType(t, isEmptyStringType)

	if isAssignableToZero || isAssignableToNaN || isAssignableToEmptyString {
		exp = state.PushToVarIfComplex(exp, "value")
	}

	checks := []luau.Expression{}

	if isAssignableToZero {
		checks = append(checks, luau.NewBinary(exp, luau.BinaryNotEqual, luau.NewNumber(0)))
	}

	// workaround for https://github.com/microsoft/TypeScript/issues/32778
	if isAssignableToZero || isAssignableToNaN {
		checks = append(checks, luau.NewBinary(exp, luau.BinaryEqual, exp))
	}

	if isAssignableToEmptyString {
		checks = append(checks, luau.NewBinary(exp, luau.BinaryNotEqual, luau.NewString("")))
	}

	checks = append(checks, exp)

	if state.Data.ProjectOptions.LogTruthyChanges &&
		(isAssignableToZero || isAssignableToNaN || isAssignableToEmptyString) {
		checkStrs := []string{}
		if isAssignableToZero {
			checkStrs = append(checkStrs, "0")
		}
		if isAssignableToZero || isAssignableToNaN {
			checkStrs = append(checkStrs, "NaN")
		}
		if isAssignableToEmptyString {
			checkStrs = append(checkStrs, `""`)
		}
		state.Diagnostic.AddDiagnostic(shared.Warnings.TruthyChange(strings.Join(checkStrs, ", ")).At(shared.AtNode(node)))
	}

	return binaryExpressionChain(checks, luau.BinaryAnd)
}
