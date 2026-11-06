package transformer

// Port of src/TSTransformer/util/createGetService.ts.

import "github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"

// createGetService returns a LuauAST node equivalent for `game:GetService(serviceName)`.
// serviceName is the service to get from game.
func createGetService(serviceName string) *luau.MethodCallExpression {
	return luau.Create(&luau.MethodCallExpression{
		Expression: luau.Globals.Game,
		Name:       "GetService",
		Args:       luau.MakeList[luau.Node](luau.NewString(serviceName)),
	})
}
