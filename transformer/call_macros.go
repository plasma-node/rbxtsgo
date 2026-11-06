package transformer

// Port of src/TSTransformer/macros/callMacros.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

var primitiveLuauTypes = map[string]bool{
	"nil":      true,
	"boolean":  true,
	"string":   true,
	"number":   true,
	"table":    true,
	"userdata": true,
	"function": true,
	"thread":   true,
	"vector":   true,
	"buffer":   true,
}

// callMacros is CALL_MACROS.
func callMacros() map[string]callMacro {
	return map[string]callMacro{
		"assert": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			args[0] = createTruthinessChecks(state, args[0], node.AsCallExpression().Arguments.Nodes[0])
			return luau.NewCall(luau.Globals.Assert, args...)
		},

		"typeOf": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.NewCall(luau.Globals.Typeof, args...)
		},

		"typeIs": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			value, typeStr := args[0], args[1]
			typeFunc := luau.Globals.Typeof
			if luau.IsStringLiteral(typeStr) && primitiveLuauTypes[typeStr.(*luau.StringLiteral).Value] {
				typeFunc = luau.Globals.Type
			}
			return luau.NewBinary(luau.NewCall(typeFunc, value), luau.BinaryEqual, typeStr)
		},

		"classIs": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			value, typeStr := args[0], args[1]
			return luau.NewBinary(luau.NewProperty(convertToIndexableExpression(value), "ClassName"), luau.BinaryEqual, typeStr)
		},

		"identity": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return args[0]
		},

		"$range": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoRangeMacroOutsideForOf.At(shared.AtNode(node.Expression())))
			return luau.NewNone()
		},

		"$tuple": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoTupleMacroOutsideReturn.At(shared.AtNode(node)))
			return luau.NewNone()
		},

		"$getModuleTree": func(state *TransformState, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			parts := getImportParts(state, ast.GetSourceFileOfNode(node), node.AsCallExpression().Arguments.Nodes[0])
			// converts the flat array into { root, { "rest", "of", "path" } }
			return luau.NewArray(parts[0], luau.NewArray(parts[1:]...))
		},
	}
}
