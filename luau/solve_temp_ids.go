package luau

import "strconv"

// Ported from src/LuauRenderer/solveTempIds.ts.

func isFullyScopedNode(node Node) bool {
	switch node.Kind() {
	case KindForStatement, KindNumericForStatement, KindFunctionDeclaration, KindFunctionExpression, KindMethodDeclaration:
		return true
	}
	return false
}

// statementsOf returns the statement list of a node that owns one (upstream
// hasStatements guard plus .statements), or nil. A type switch on the
// concrete types is much cheaper than asserting HasStatements.
func statementsOf(node Node) *List {
	switch n := node.(type) {
	case *ForStatement:
		return n.Statements
	case *NumericForStatement:
		return n.Statements
	case *FunctionExpression:
		return n.Statements
	case *DoStatement:
		return n.Statements
	case *FunctionDeclaration:
		return n.Statements
	case *IfStatement:
		return n.Statements
	case *MethodDeclaration:
		return n.Statements
	case *RepeatStatement:
		return n.Statements
	case *WhileStatement:
		return n.Statements
	}
	return nil
}

// isScopeEdge reports whether node is the first (head) or last (tail)
// statement of a block that creates scope.
func isScopeEdge(node Node, head bool) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}

	// is the first statement in a block that creates scope
	if statements := statementsOf(parent); statements != nil {
		if edgeValue(statements, head) == node {
			return true
		}
	}

	// non-list elseBody would have the elseBody itself as a parent,
	// which would be a luau.IfStatement and handled above
	if ifStatement, ok := parent.(*IfStatement); ok {
		if list, ok := ifStatement.ElseBody.(*List); ok && edgeValue(list, head) == node {
			return true
		}
	}

	return false
}

// edgeValue returns list.head?.value or list.tail?.value (nil when empty).
func edgeValue(list *List, head bool) Node {
	edge := list.Tail
	if head {
		edge = list.Head
	}
	if edge == nil {
		return nil
	}
	return edge.Value
}

type scope struct {
	ids     map[string]struct{}
	lastTry map[string]int
	parent  *scope
}

func createScope(parent *scope) *scope {
	return &scope{parent: parent}
}

func (s *scope) hasID(id string) bool {
	for ; s != nil; s = s.parent {
		if _, ok := s.ids[id]; ok {
			return true
		}
	}
	return false
}

func (s *scope) addID(id string) {
	if s.ids == nil {
		s.ids = make(map[string]struct{})
	}
	s.ids[id] = struct{}{}
}

// SolveTempIDs gives every TemporaryIdentifier under ast a unique name in
// state.SeenTempNodes. ast is a *List or a Node. Names that are already in
// SeenTempNodes are kept.
func SolveTempIDs(state *RenderState, ast NodeOrList) {
	if state.SeenTempNodes == nil {
		state.SeenTempNodes = make(map[int]string)
	}

	var tempIDsToProcess []*TemporaryIdentifier
	// Like upstream, a node visited twice keeps the scope of its last visit.
	nodesToScopes := make(map[*TemporaryIdentifier]*scope)

	scopeStack := []*scope{createScope(nil)}

	// Upstream indexes an empty array to get undefined and asserts on it, so an
	// unbalanced pop (possible with the dangling links in List) is an
	// assertion failure here too.
	peekScopeStack := func() *scope {
		Assert(len(scopeStack) > 0, "")
		return scopeStack[len(scopeStack)-1]
	}
	pushScopeStack := func() { scopeStack = append(scopeStack, createScope(peekScopeStack())) }
	popScopeStack := func() {
		if len(scopeStack) > 0 {
			scopeStack = scopeStack[:len(scopeStack)-1]
		}
	}
	registerID := func(name string) { peekScopeStack().addID(name) }
	registerParameters := func(parameters *List) {
		for ln := parameters.Head; ln != nil; ln = ln.Next {
			if id, ok := ln.Value.(*Identifier); ok {
				registerID(id.Name)
			}
		}
	}

	Visit(ast, &Visitor{
		Before: func(node Node) {
			if isFullyScopedNode(node) {
				pushScopeStack()
			}
			if isScopeEdge(node, true) {
				pushScopeStack()
			}

			switch n := node.(type) {
			case *TemporaryIdentifier:
				nodesToScopes[n] = peekScopeStack()
				tempIDsToProcess = append(tempIDsToProcess, n)
			case *VariableDeclaration:
				if list, ok := n.Left.(*List); ok {
					for ln := list.Head; ln != nil; ln = ln.Next {
						if id, ok := ln.Value.(*Identifier); ok {
							registerID(id.Name)
						}
					}
				} else if id, ok := n.Left.(*Identifier); ok {
					registerID(id.Name)
				}
			case *FunctionDeclaration:
				registerParameters(n.Parameters)
			case *FunctionExpression:
				registerParameters(n.Parameters)
			case *MethodDeclaration:
				registerParameters(n.Parameters)
			}
		},
		After: func(node Node) {
			if isFullyScopedNode(node) {
				popScopeStack()
			}
			if isScopeEdge(node, false) {
				popScopeStack()
			}
		},
	})

	for _, tempID := range tempIDsToProcess {
		if _, seen := state.SeenTempNodes[tempID.ID]; seen {
			continue
		}
		sc := nodesToScopes[tempID]
		Assert(sc != nil, "")

		separator := "_"
		if tempID.Name == "" {
			separator = ""
		}

		input := "_" + tempID.Name
		i, ok := sc.lastTry[input]
		if !ok {
			i = 1
		}
		for sc.hasID(input) {
			input = "_" + tempID.Name + separator + strconv.Itoa(i)
			i++
		}
		if sc.lastTry == nil {
			sc.lastTry = make(map[string]int)
		}
		sc.lastTry[input] = i
		sc.addID(input)

		state.SeenTempNodes[tempID.ID] = input
	}
}
