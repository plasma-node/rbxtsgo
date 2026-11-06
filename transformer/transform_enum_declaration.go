package transformer

// Port of src/TSTransformer/nodes/statements/transformEnumDeclaration.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/jsnum"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func needsInverseEntry(state *TransformState, member *ast.Node) bool {
	_, isString := state.Checker.GetConstantValue(member).(string)
	return !isString
}

func transformEnumDeclaration(state *TransformState, node *ast.Node) *luau.List {
	if ast.HasSyntacticModifier(node, ast.ModifierFlagsConst) && !state.Options.PreserveConstEnums.IsTrue() {
		return luau.NewList()
	}

	symbol := state.Checker.GetSymbolAtLocation(node.Name())
	if symbol != nil &&
		hasMultipleDefinitions(symbol, func(declaration *ast.Node) bool {
			return ast.IsEnumDeclaration(declaration) && !ast.HasSyntacticModifier(declaration, ast.ModifierFlagsConst)
		}) {
		shared.AddDiagnosticWithCache(
			state.Diagnostic,
			symbol,
			shared.Errors.NoEnumMerging.At(shared.AtNode(node)),
			state.Multi.IsReportedByMultipleDefinitionsCache,
		)
		return luau.NewList()
	}

	validateIdentifier(state, node.Name())

	left := transformIdentifierDefined(state, node.Name())
	isHoisted := symbol != nil && state.IsHoisted[symbol]

	members := node.AsEnumDeclaration().Members.Nodes

	noInverseEntries := true
	for _, member := range members {
		if needsInverseEntry(state, member) {
			noInverseEntries = false
			break
		}
	}
	if noInverseEntries {
		fields := make([]luau.KeyValue, 0, len(members))
		for _, member := range members {
			fields = append(fields, luau.KeyValue{
				Index: state.PushToVarIfComplex(transformPropertyName(state, member.Name()), ""),
				Value: luau.NewString(state.Checker.GetConstantValue(member).(string)),
			})
		}
		right := luau.NewMap(fields...)
		if isHoisted {
			return luau.MakeList[luau.Node](luau.Create(&luau.Assignment{Left: left, Operator: luau.AssignEqual, Right: right}))
		}
		return luau.MakeList[luau.Node](luau.Create(&luau.VariableDeclaration{Left: left, Right: right}))
	}

	statements := state.CapturePrereqs(func() {
		inverseID := state.PushToVar(luau.NewMap(), "inverse")
		state.Prereq(luau.Create(&luau.Assignment{
			Left:     left,
			Operator: luau.AssignEqual,
			Right: luau.NewCall(
				luau.Globals.Setmetatable,
				luau.NewMap(),
				luau.NewMap(luau.KeyValue{Index: luau.Strings.Index, Value: inverseID}),
			),
		}))

		for _, member := range members {
			memberName := member.Name()
			name := transformPropertyName(state, memberName)
			mightMutateNode := memberName
			if ast.IsComputedPropertyName(memberName) {
				mightMutateNode = memberName.Expression()
			}
			var index luau.Expression
			if expressionMightMutate(state, name, mightMutateNode) {
				// note: we don't use pushToVarIfComplex here
				// because identifier also needs to be pushed
				// since the value calculation might reassign the variable
				index = state.PushToVar(name, "")
			} else {
				index = name
			}

			var valueExp luau.Expression
			switch value := state.Checker.GetConstantValue(member).(type) {
			case string:
				valueExp = luau.NewString(value)
			case jsnum.Number:
				valueExp = luau.NewNumber(float64(value))
			default:
				// constantValue is always number without initializer, so assert is safe
				if member.Initializer() == nil {
					panic(shared.AssertionMessage(""))
				}
				valueExp = state.PushToVarIfComplex(transformExpression(state, member.Initializer()), "value")
			}

			state.Prereq(luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: left,
					Index:      index,
				}),
				Operator: luau.AssignEqual,
				Right:    valueExp,
			}))

			if needsInverseEntry(state, member) {
				state.Prereq(luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: inverseID,
						Index:      valueExp,
					}),
					Operator: luau.AssignEqual,
					Right:    index,
				}))
			}
		}
	})

	list := luau.MakeList[luau.Node](luau.Create(&luau.DoStatement{Statements: statements}))
	if !isHoisted {
		list.Unshift(luau.Create(&luau.VariableDeclaration{Left: left}))
	}
	return list
}
