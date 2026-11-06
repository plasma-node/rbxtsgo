package transformer

// Port of src/TSTransformer/nodes/expressions/transformBinaryExpression.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func transformOptimizedArrayAssignmentPattern(state *TransformState, assignmentPattern *ast.Node, rhs luau.NodeOrList) {
	variables := luau.NewList()
	writes := luau.NewList()
	writesPrereqs := luau.NewList()
	statements := state.CapturePrereqs(func() {
		for _, element := range assignmentPattern.AsArrayLiteralExpression().Elements.Nodes {
			if ast.IsOmittedExpression(element) {
				writes.Push(luau.NewTempID(""))
			} else if ast.IsSpreadElement(element) {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoSpreadDestructuring.At(shared.AtNode(element)))
			} else {
				var initializer *ast.Node
				if ast.IsBinaryExpression(element) {
					binary := element.AsBinaryExpression()
					initializer = SkipDownwards(binary.Right)
					element = SkipDownwards(binary.Left)
				}

				if ast.IsIdentifier(element) ||
					ast.IsElementAccessExpression(element) ||
					ast.IsPropertyAccessExpression(element) {
					id, idPrereqs := Capture(state, func() luau.WritableExpression { return transformWritableExpression(state, element, true) })
					writesPrereqs.PushList(idPrereqs)
					writes.Push(id)
					if initializer != nil {
						state.Prereq(transformInitializer(state, id, initializer))
					}
				} else if ast.IsArrayLiteralExpression(element) {
					id := luau.NewTempID("binding")
					variables.Push(id)
					writes.Push(id)
					if initializer != nil {
						state.Prereq(transformInitializer(state, id, initializer))
					}
					transformArrayAssignmentPattern(state, element, id)
				} else if ast.IsObjectLiteralExpression(element) {
					id := luau.NewTempID("binding")
					variables.Push(id)
					writes.Push(id)
					if initializer != nil {
						state.Prereq(transformInitializer(state, id, initializer))
					}
					transformObjectAssignmentPattern(state, element, id)
				} else {
					panic(shared.AssertionMessage("transformOptimizedArrayAssignmentPattern invalid element: " + getKindName(element.Kind)))
				}
			}
		}
	})
	if !variables.IsEmpty() {
		state.Prereq(
			luau.Create(&luau.VariableDeclaration{
				Left:  variables,
				Right: nil,
			}),
		)
	}
	state.PrereqList(writesPrereqs)
	if writes.IsEmpty() {
		panic(shared.AssertionMessage(""))
	}
	state.Prereq(
		luau.Create(&luau.Assignment{
			Left:     writes,
			Operator: luau.AssignEqual,
			Right:    rhs,
		}),
	)
	state.PrereqList(statements)
}

func transformBinaryExpression(state *TransformState, node *ast.Node) luau.Expression {
	binary := node.AsBinaryExpression()
	operatorKind := binary.OperatorToken.Kind

	validateNotAnyType(state, binary.Left)
	validateNotAnyType(state, binary.Right)

	// banned
	if operatorKind == ast.KindEqualsEqualsToken {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoEqualsEquals.At(shared.AtNode(node)))
		return luau.NewNone()
	} else if operatorKind == ast.KindExclamationEqualsToken {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoExclamationEquals.At(shared.AtNode(node)))
		return luau.NewNone()
	}

	// logical
	if operatorKind == ast.KindAmpersandAmpersandToken ||
		operatorKind == ast.KindBarBarToken ||
		operatorKind == ast.KindQuestionQuestionToken {
		return transformLogical(state, node)
	}

	if ast.IsLogicalOrCoalescingAssignmentExpression(node) {
		return transformLogicalOrCoalescingAssignmentExpression(state, node)
	}

	if ast.IsAssignmentOperator(operatorKind) {
		// in destructuring, rhs must be executed first
		if ast.IsArrayLiteralExpression(binary.Left) {
			rightExp := transformExpression(state, binary.Right)

			// optimize empty array destructure
			if len(binary.Left.AsArrayLiteralExpression().Elements.Nodes) == 0 {
				if isUsedAsStatement(node) && luau.IsArray(rightExp) && rightExp.(*luau.Array).Members.IsEmpty() {
					return luau.NewNone()
				}
				return rightExp
			}

			if luau.IsCall(rightExp) && isLuaTupleType(state)(state.GetType(binary.Right)) {
				transformOptimizedArrayAssignmentPattern(state, binary.Left, rightExp)
				if !isUsedAsStatement(node) {
					state.Diagnostic.AddDiagnostic(shared.Errors.NoLuaTupleDestructureAssignmentExpression.At(shared.AtNode(node)))
				}
				return luau.NewNone()
			}

			if luau.IsArray(rightExp) && !rightExp.(*luau.Array).Members.IsEmpty() && isUsedAsStatement(node) {
				transformOptimizedArrayAssignmentPattern(state, binary.Left, rightExp.(*luau.Array).Members)
				return luau.NewNone()
			}

			parentId := state.PushToVar(rightExp, "binding")
			transformArrayAssignmentPattern(state, binary.Left, parentId)
			return parentId
		} else if ast.IsObjectLiteralExpression(binary.Left) {
			rightExp := transformExpression(state, binary.Right)

			// optimize empty object destructure
			if len(binary.Left.AsObjectLiteralExpression().Properties.Nodes) == 0 {
				if isUsedAsStatement(node) && luau.IsMap(rightExp) && rightExp.(*luau.Map).Fields.IsEmpty() {
					return luau.NewNone()
				}
				return rightExp
			}

			parentId := state.PushToVar(rightExp, "binding")
			transformObjectAssignmentPattern(state, binary.Left, parentId)
			return parentId
		}

		writableType := state.GetType(binary.Left)
		valueType := state.GetType(binary.Right)
		operator := getSimpleAssignmentOperator(writableType, operatorKind, valueType)
		assignment := transformWritableAssignment(
			state,
			binary.Left,
			binary.Right,
			true,
			operator == "",
		)
		writable, readable, value := assignment.writable, assignment.readable, assignment.value
		if operator != "" {
			return createAssignmentExpression(
				state,
				writable,
				operator,
				getAssignableValue(operator, value, valueType),
			)
		} else {
			return createCompoundAssignmentExpression(
				state,
				node,
				writable,
				writableType,
				readable,
				operatorKind,
				value,
				valueType,
			)
		}
	}

	ordered := ensureTransformOrder(state, []*ast.Node{binary.Left, binary.Right}, nil)
	left, right := ordered[0], ordered[1]

	if operatorKind == ast.KindInKeyword {
		return luau.NewBinary(
			luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(right),
				Index:      left,
			}),
			luau.BinaryNotEqual,
			luau.NewNil(),
		)
	} else if operatorKind == ast.KindInstanceOfKeyword {
		if isPossiblyType(state.GetType(binary.Right), isRobloxType(state)) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoRobloxSymbolInstanceof.At(shared.AtNode(binary.Right)))
		}
		return luau.NewCall(state.TS(node, "instanceof"), left, right)
	}

	leftType := state.GetType(binary.Left)
	rightType := state.GetType(binary.Right)

	if operatorKind == ast.KindLessThanToken ||
		operatorKind == ast.KindLessThanEqualsToken ||
		operatorKind == ast.KindGreaterThanToken ||
		operatorKind == ast.KindGreaterThanEqualsToken {
		if (!isDefinitelyType(leftType, isStringType) && !isDefinitelyType(leftType, isNumberType)) ||
			(!isDefinitelyType(rightType, isStringType) && !isDefinitelyType(leftType, isNumberType)) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoNonNumberStringRelationOperator.At(shared.AtNode(node)))
		}
	}

	return createBinaryFromOperator(state, node, left, leftType, operatorKind, right, rightType)
}
