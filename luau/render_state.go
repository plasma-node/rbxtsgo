package luau

import "strconv"

// Ported from src/LuauRenderer/RenderState.ts.
//
// Unlike upstream, where every renderX returns a string, the Go renderer
// appends to one buffer held here. Every wrapping decision upstream makes
// (parentheses, bracket spacing, semicolons) is taken from the AST, never from
// rendered text, so streaming produces the same bytes. The public methods that
// return strings (Newline, Indented, Line) are kept for parity and are not used
// on the hot path.

const indentCharacter = "\t"

// RenderState stores the state of one render process.
type RenderState struct {
	indent int // number of indent characters

	// SeenTempNodes maps TemporaryIdentifier ids to their solved names.
	SeenTempNodes map[int]string

	listNodesStack []*ListNode
	tempIDFallback int

	buf     []byte // output being built
	scratch []byte // holds one already-rendered value while a MapField renders its key first
}

// NewRenderState creates an empty render state (upstream `new RenderState()`).
func NewRenderState() *RenderState {
	return &RenderState{SeenTempNodes: make(map[int]string)}
}

// GetTempName returns the name for a temporary identifier. SeenTempNodes should
// already be fully populated by SolveTempIDs. This is the fallback for when it
// does not catch something properly: unseen ids get "_0", "_1", ... in render
// order.
func (s *RenderState) GetTempName(node *TemporaryIdentifier) string {
	if s.SeenTempNodes == nil {
		s.SeenTempNodes = make(map[int]string)
	}
	name, ok := s.SeenTempNodes[node.ID]
	if !ok {
		name = "_" + strconv.Itoa(s.tempIDFallback)
		s.tempIDFallback++
		s.SeenTempNodes[node.ID] = name
	}
	Assert(name != "", "")
	return name
}

// PushListNode pushes a list node onto the list node stack.
func (s *RenderState) PushListNode(listNode *ListNode) {
	s.listNodesStack = append(s.listNodesStack, listNode)
}

// PeekListNode returns the top of the list node stack, or nil.
func (s *RenderState) PeekListNode() *ListNode {
	if len(s.listNodesStack) == 0 {
		return nil
	}
	return s.listNodesStack[len(s.listNodesStack)-1]
}

// PopListNode pops the top of the list node stack.
func (s *RenderState) PopListNode() *ListNode {
	n := len(s.listNodesStack)
	if n == 0 {
		return nil
	}
	top := s.listNodesStack[n-1]
	s.listNodesStack[n-1] = nil
	s.listNodesStack = s.listNodesStack[:n-1]
	return top
}

// Newline adds a newline to the end of the text.
func (s *RenderState) Newline(text string) string { return text + "\n" }

// Indented prefixes the text with the current indent.
func (s *RenderState) Indented(text string) string {
	b := make([]byte, 0, s.indent+len(text))
	for i := 0; i < s.indent; i++ {
		b = append(b, indentCharacter...)
	}
	return string(append(b, text...))
}

// Line renders a line: the current indent, text, a semicolon if endNode needs
// one, and "\n". A nil endNode means no semicolon is added.
func (s *RenderState) Line(text string, endNode Statement) string {
	result := s.Indented(text)
	if endNode != nil && needsSemicolon(s, endNode) {
		result += ";"
	}
	return result + "\n"
}

// Block runs callback one indent level deeper.
func (s *RenderState) Block(callback func()) {
	s.indent++
	callback()
	s.indent--
}

// ---------------------------------------------------------------------------
// Streaming writers used by the renderer.

func (s *RenderState) writeString(text string) { s.buf = append(s.buf, text...) }

func (s *RenderState) writeByte(c byte) { s.buf = append(s.buf, c) }

func (s *RenderState) writeIndent() {
	for i := 0; i < s.indent; i++ {
		s.buf = append(s.buf, '\t')
	}
}

// writeLineEnd finishes a line started with writeIndent: an optional ";" when
// endNode needs one (endNode may be nil), then "\n".
func (s *RenderState) writeLineEnd(endNode Statement) {
	if endNode != nil && needsSemicolon(s, endNode) {
		s.buf = append(s.buf, ';')
	}
	s.buf = append(s.buf, '\n')
}

// writeLine writes indent, text and "\n".
func (s *RenderState) writeLine(text string) {
	s.writeIndent()
	s.buf = append(s.buf, text...)
	s.buf = append(s.buf, '\n')
}
