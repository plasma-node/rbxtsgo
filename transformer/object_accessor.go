package transformer

// Port of src/TSTransformer/util/binding/objectAccessor.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func objectAccessor(state *TransformState, parentID luau.AnyIdentifier, t *checker.Type, name *ast.Node) luau.Expression {
	addIndexDiagnostics(state, name, state.GetType(name))

	if ast.IsIdentifier(name) {
		return luau.NewProperty(parentID, name.Text())
	} else if ast.IsComputedPropertyName(name) {
		return luau.Create(&luau.ComputedIndexExpression{
			Expression: parentID,
			Index:      addOneIfArrayType(state, t, transformExpression(state, name.Expression())),
		})
	} else if ast.IsNumericLiteral(name) || ast.IsStringLiteral(name) || ast.IsNoSubstitutionTemplateLiteral(name) {
		return luau.Create(&luau.ComputedIndexExpression{
			Expression: parentID,
			Index:      transformExpression(state, name),
		})
	} else if ast.IsPrivateIdentifier(name) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoPrivateIdentifier.At(shared.AtNode(name)))
		return luau.NewNone()
	}
	return assertNever(name, "objectAccessor")
}
