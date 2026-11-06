package luau

import "reflect"

// Ported from src/LuauAST/impl/globals.ts and src/LuauAST/impl/strings.ts.
//
// These are shared mutable nodes, exactly like upstream: the first Create
// that receives one adopts it in place, and every later use is cloned (see the
// concurrency note in doc.go). The initialisation order below mirrors the
// upstream file so the same use is the "first".

var (
	coroutineID = NewID("coroutine")
	mathID      = NewID("math")
	stringID    = NewID("string")
	tableID     = NewID("table")
	utf8ID      = NewID("utf8")
)

type globalsType struct {
	G            *Identifier
	TS           *Identifier
	Assert       *Identifier
	Bit32        *Identifier
	Coroutine    struct{ Yield *PropertyAccessExpression }
	Error        *Identifier
	Exports      *Identifier
	Getmetatable *Identifier
	Ipairs       *Identifier
	Next         *Identifier
	Pairs        *Identifier
	Pcall        *Identifier
	Require      *Identifier
	Script       *Identifier
	Select       *Identifier
	Self         *Identifier
	Setmetatable *Identifier
	String       struct {
		Byte, Find, Format, Gmatch, Gsub, Lower, Match, Rep, Reverse, Split, Sub, Upper *PropertyAccessExpression
	}
	Super *Identifier
	Table struct {
		Clear, Clone, Concat, Create, Find, Freeze, Insert, Isfrozen, Maxn, Move, Pack, Remove, Sort,
		Unpack *PropertyAccessExpression
	}
	Utf8     struct{ Charpattern, Codes *PropertyAccessExpression }
	Math     struct{ Min *PropertyAccessExpression }
	Tostring *Identifier
	Type     *Identifier
	Typeof   *Identifier
	Unpack   *Identifier

	// roblox
	Game *Identifier
}

// Globals holds the commonly used Luau global identifiers and members.
var Globals = newGlobals()

func newGlobals() *globalsType {
	g := &globalsType{}
	g.G = NewID("_G")
	g.TS = NewID("TS")
	g.Assert = NewID("assert")
	g.Bit32 = NewID("bit32")
	g.Coroutine.Yield = NewProperty(coroutineID, "yield")
	g.Error = NewID("error")
	g.Exports = NewID("exports")
	g.Getmetatable = NewID("getmetatable")
	g.Ipairs = NewID("ipairs")
	g.Next = NewID("next")
	g.Pairs = NewID("pairs")
	g.Pcall = NewID("pcall")
	g.Require = NewID("require")
	g.Script = NewID("script")
	g.Select = NewID("select")
	g.Self = NewID("self")
	g.Setmetatable = NewID("setmetatable")
	g.String.Byte = NewProperty(stringID, "byte")
	g.String.Find = NewProperty(stringID, "find")
	g.String.Format = NewProperty(stringID, "format")
	g.String.Gmatch = NewProperty(stringID, "gmatch")
	g.String.Gsub = NewProperty(stringID, "gsub")
	g.String.Lower = NewProperty(stringID, "lower")
	g.String.Match = NewProperty(stringID, "match")
	g.String.Rep = NewProperty(stringID, "rep")
	g.String.Reverse = NewProperty(stringID, "reverse")
	g.String.Split = NewProperty(stringID, "split")
	g.String.Sub = NewProperty(stringID, "sub")
	g.String.Upper = NewProperty(stringID, "upper")
	g.Super = NewID("super")
	g.Table.Clear = NewProperty(tableID, "clear")
	g.Table.Clone = NewProperty(tableID, "clone")
	g.Table.Concat = NewProperty(tableID, "concat")
	g.Table.Create = NewProperty(tableID, "create")
	g.Table.Find = NewProperty(tableID, "find")
	g.Table.Freeze = NewProperty(tableID, "freeze")
	g.Table.Insert = NewProperty(tableID, "insert")
	g.Table.Isfrozen = NewProperty(tableID, "isfrozen")
	g.Table.Maxn = NewProperty(tableID, "maxn")
	g.Table.Move = NewProperty(tableID, "move")
	g.Table.Pack = NewProperty(tableID, "pack")
	g.Table.Remove = NewProperty(tableID, "remove")
	g.Table.Sort = NewProperty(tableID, "sort")
	g.Table.Unpack = NewProperty(tableID, "unpack")
	g.Utf8.Charpattern = NewProperty(utf8ID, "charpattern")
	g.Utf8.Codes = NewProperty(utf8ID, "codes")
	g.Math.Min = NewProperty(mathID, "min")
	g.Tostring = NewID("tostring")
	g.Type = NewID("type")
	g.Typeof = NewID("typeof")
	g.Unpack = NewID("unpack")
	g.Game = NewID("game")
	return g
}

type stringsType struct {
	// metamethods
	Index    *StringLiteral // "__index"
	Tostring *StringLiteral // "__tostring"
	Mode     *StringLiteral // "__mode"
	K        *StringLiteral // "k", used for __mode

	// types
	Number *StringLiteral // "number"
	Table  *StringLiteral // "table"

	// opcall
	Success *StringLiteral // "success"
	Value   *StringLiteral // "value"
	Error   *StringLiteral // "error"

	// other
	Comma *StringLiteral // ", ", used for ReadonlyArray.join()
}

// PinShared gives every node in Globals and Strings a parent, so Create always clones them and never writes to them.
// Call it once before starting workers, and those nodes are then safe to use from any goroutine. See doc.go. It changes
// nothing in the output: upstream's first Create adopts the node in place, and that node renders the same as a clone.
func PinShared() {
	pinned := &Comment{}
	var visit func(value reflect.Value)
	visit = func(value reflect.Value) {
		switch value.Kind() {
		case reflect.Pointer:
			if value.IsNil() {
				return
			}
			if node, ok := value.Interface().(Node); ok {
				if node.Parent() == nil {
					node.SetParent(pinned)
				}
				return
			}
			visit(value.Elem())
		case reflect.Struct:
			for i := range value.NumField() {
				visit(value.Field(i))
			}
		}
	}
	visit(reflect.ValueOf(Globals))
	visit(reflect.ValueOf(Strings))
}

// Strings holds commonly used string literal nodes.
var Strings = &stringsType{
	Index:    NewString("__index"),
	Tostring: NewString("__tostring"),
	Mode:     NewString("__mode"),
	K:        NewString("k"),
	Number:   NewString("number"),
	Table:    NewString("table"),
	Success:  NewString("success"),
	Value:    NewString("value"),
	Error:    NewString("error"),
	Comma:    NewString(", "),
}
