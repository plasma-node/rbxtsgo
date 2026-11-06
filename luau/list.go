package luau

// Ported from src/LuauAST/impl/List.ts (namespace luau.list).
//
// A List is a doubly linked list of nodes. Like upstream, the helpers mutate
// and share list nodes, and several of them leave dangling links behind on
// purpose; see the comments on PushList, Shift and UnshiftList. Do not tidy
// these up: the transformer relies on the exact behavior.

// ListNode is one link of a List.
type ListNode struct {
	Prev  *ListNode
	Next  *ListNode
	Value Node
}

// List is a doubly linked list of nodes (upstream luau.List<T>). The element
// type documented on each node field is not enforced.
type List struct {
	Head     *ListNode
	Tail     *ListNode
	ReadOnly bool
}

func (*List) isNodeOrList() {}

// AsList reports whether v is a list (upstream luau.list.isList).
func AsList(v NodeOrList) (*List, bool) {
	l, ok := v.(*List)
	return l, ok
}

// IsList reports whether v is a list.
func IsList(v NodeOrList) bool {
	_, ok := v.(*List)
	return ok
}

// MakeListNode makes a detached list node (upstream list.makeNode).
func MakeListNode(value Node) *ListNode {
	Assert(value != nil, "")
	return &ListNode{Value: value}
}

// NewList makes an empty list.
func NewList() *List { return &List{} }

// MakeList makes a list holding values in order (upstream list.make).
func MakeList[T Node](values ...T) *List {
	list := &List{}
	if len(values) == 0 {
		return list
	}
	for _, v := range values {
		Assert(Node(v) != nil, "Not all List values were a luau.Node")
	}
	head := &ListNode{Value: values[0]}
	tail := head
	for i := 1; i < len(values); i++ {
		node := &ListNode{Value: values[i]}
		tail.Next = node
		node.Prev = tail
		tail = node
	}
	list.Head = head
	list.Tail = tail
	return list
}

// Clone returns a new list holding shallow clones of each element
// (upstream list.clone). The clones keep their old parent.
func (l *List) Clone() *List {
	newList := NewList()
	for node := l.Head; node != nil; node = node.Next {
		newList.Push(node.Value.shallowClone())
	}
	return newList
}

// Push appends value.
func (l *List) Push(value Node) {
	Assert(value != nil, "")
	Assert(!l.ReadOnly, "")
	node := &ListNode{Value: value}
	if l.Tail != nil {
		l.Tail.Next = node
		node.Prev = l.Tail
	} else {
		l.Head = node
	}
	l.Tail = node
}

// PushList moves the nodes of other onto the end of l and marks other
// read-only. other keeps its Head and Tail, which now point into l's chain.
func (l *List) PushList(other *List) {
	Assert(!l.ReadOnly, "")
	Assert(!other.ReadOnly, "")
	other.ReadOnly = true

	if other.Head != nil && other.Tail != nil {
		if l.Head != nil && l.Tail != nil {
			l.Tail.Next = other.Head
			other.Head.Prev = l.Tail
			l.Tail = other.Tail
		} else {
			l.Head = other.Head
			l.Tail = other.Tail
		}
	}
}

// Shift removes and returns the first value, or nil when empty. The removed
// list node keeps its Next pointer.
func (l *List) Shift() Node {
	Assert(!l.ReadOnly, "")
	if l.Head != nil {
		head := l.Head
		if head.Next != nil {
			l.Head = head.Next
			head.Next.Prev = nil
		} else {
			l.Tail = nil
			l.Head = nil
		}
		return head.Value
	}
	return nil
}

// Unshift prepends value.
func (l *List) Unshift(value Node) {
	Assert(value != nil, "")
	Assert(!l.ReadOnly, "")
	node := &ListNode{Value: value}
	if l.Head != nil {
		l.Head.Prev = node
		node.Next = l.Head
	} else {
		l.Tail = node
	}
	l.Head = node
}

// UnshiftList moves the nodes of other onto the front of l and marks other
// read-only. other keeps its Head and Tail.
func (l *List) UnshiftList(other *List) {
	Assert(!l.ReadOnly, "")
	Assert(!other.ReadOnly, "")
	other.ReadOnly = true

	if other.Head != nil && other.Tail != nil {
		if l.Head != nil && l.Tail != nil {
			l.Head.Prev = other.Tail
			other.Tail.Next = l.Head
			l.Head = other.Head
		} else {
			l.Head = other.Head
			l.Tail = other.Tail
		}
	}
}

func (l *List) IsEmpty() bool    { return l.Head == nil }
func (l *List) IsNonEmpty() bool { return l.Head != nil }

// ForEach calls callback for each value, reading Next after each call so
// callbacks may append.
func (l *List) ForEach(callback func(value Node)) {
	for node := l.Head; node != nil; node = node.Next {
		callback(node.Value)
	}
}

// ForEachListNode calls callback for each list node.
func (l *List) ForEachListNode(callback func(node *ListNode)) {
	for node := l.Head; node != nil; node = node.Next {
		callback(node)
	}
}

// MapToArray maps each value through callback (upstream list.mapToArray).
func MapToArray[U any](l *List, callback func(value Node) U) []U {
	result := make([]U, 0, l.Size())
	for node := l.Head; node != nil; node = node.Next {
		result = append(result, callback(node.Value))
	}
	return result
}

// ToArray returns the values as a slice.
func (l *List) ToArray() []Node {
	result := make([]Node, 0, l.Size())
	for node := l.Head; node != nil; node = node.Next {
		result = append(result, node.Value)
	}
	return result
}

// Every reports whether callback holds for every value.
func (l *List) Every(callback func(value Node) bool) bool {
	for node := l.Head; node != nil; node = node.Next {
		if !callback(node.Value) {
			return false
		}
	}
	return true
}

// Some reports whether callback holds for any value.
func (l *List) Some(callback func(value Node) bool) bool {
	for node := l.Head; node != nil; node = node.Next {
		if callback(node.Value) {
			return true
		}
	}
	return false
}

// Size counts the nodes reachable from Head through Next.
func (l *List) Size() int {
	size := 0
	for node := l.Head; node != nil; node = node.Next {
		size++
	}
	return size
}
