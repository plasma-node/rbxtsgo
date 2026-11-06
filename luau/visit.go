package luau

// Ported from src/LuauRenderer/util/visit.ts.

// Visitor holds the callbacks Visit calls around every node, in tree order.
// Either may be nil.
type Visitor struct {
	Before func(node Node)
	After  func(node Node)
}

func visitList(list *List, visitor *Visitor) {
	for ln := list.Head; ln != nil; ln = ln.Next {
		visitNode(ln.Value, visitor)
	}
}

func visitNodeOrList(v NodeOrList, visitor *Visitor) {
	switch x := v.(type) {
	case nil:
	case *List:
		visitList(x, visitor)
	case Node:
		visitNode(x, visitor)
	}
}

func visitNode(node Node, visitor *Visitor) {
	if visitor.Before != nil {
		visitor.Before(node)
	}

	switch n := node.(type) {
	// indexable expressions
	case *ComputedIndexExpression:
		visitNode(n.Expression, visitor)
		visitNode(n.Index, visitor)
	case *PropertyAccessExpression:
		visitNode(n.Expression, visitor)
	case *CallExpression:
		visitNode(n.Expression, visitor)
		visitList(n.Args, visitor)
	case *MethodCallExpression:
		visitNode(n.Expression, visitor)
		visitList(n.Args, visitor)
	case *ParenthesizedExpression:
		visitNode(n.Expression, visitor)

	// expressions
	case *FunctionExpression:
		visitList(n.Parameters, visitor)
		visitList(n.Statements, visitor)
	case *BinaryExpression:
		visitNode(n.Left, visitor)
		visitNode(n.Right, visitor)
	case *UnaryExpression:
		visitNode(n.Expression, visitor)
	case *IfExpression:
		visitNode(n.Condition, visitor)
		visitNode(n.Expression, visitor)
		visitNode(n.Alternative, visitor)
	case *InterpolatedString:
		visitList(n.Parts, visitor)
	case *Array:
		visitList(n.Members, visitor)
	case *Map:
		visitList(n.Fields, visitor)
	case *Set:
		visitList(n.Members, visitor)
	case *MixedTable:
		visitList(n.Fields, visitor)

	// statements
	case *Assignment:
		visitNodeOrList(n.Left, visitor)
		visitNodeOrList(n.Right, visitor)
	case *CallStatement:
		visitNode(n.Expression, visitor)
	case *DoStatement:
		visitList(n.Statements, visitor)
	case *WhileStatement:
		visitNode(n.Condition, visitor)
		visitList(n.Statements, visitor)
	case *RepeatStatement:
		visitList(n.Statements, visitor)
		visitNode(n.Condition, visitor)
	case *IfStatement:
		visitNode(n.Condition, visitor)
		visitList(n.Statements, visitor)
		visitNodeOrList(n.ElseBody, visitor)
	case *NumericForStatement:
		visitNode(n.ID, visitor)
		visitNode(n.Start, visitor)
		visitNode(n.End, visitor)
		if n.Step != nil {
			visitNode(n.Step, visitor)
		}
		visitList(n.Statements, visitor)
	case *ForStatement:
		visitList(n.IDs, visitor)
		visitList(n.Statements, visitor)
	case *FunctionDeclaration:
		visitNode(n.Name, visitor)
		visitList(n.Parameters, visitor)
		visitList(n.Statements, visitor)
	case *MethodDeclaration:
		visitNode(n.Expression, visitor)
		visitList(n.Parameters, visitor)
		visitList(n.Statements, visitor)
	case *VariableDeclaration:
		visitNodeOrList(n.Left, visitor)
		visitNodeOrList(n.Right, visitor)
	case *ReturnStatement:
		visitNodeOrList(n.Expression, visitor)

	// fields
	case *MapField:
		visitNode(n.Index, visitor)
		visitNode(n.Value, visitor)

		// All other kinds (identifiers, literals, break, continue, comment,
		// None, InterpolatedStringPart) have no children.
	}

	if visitor.After != nil {
		visitor.After(node)
	}
}

// Visit walks an AST depth first, calling visitor.Before and visitor.After
// around each node. ast is a *List or a Node.
func Visit(ast NodeOrList, visitor *Visitor) {
	if list, ok := ast.(*List); ok {
		visitList(list, visitor)
	} else if node, ok := ast.(Node); ok {
		visitNode(node, visitor)
	}
}
