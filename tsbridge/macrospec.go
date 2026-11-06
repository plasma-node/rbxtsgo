package tsbridge

// DefaultMacroSpec returns the macro names of roblox-ts 3.0.0, read from the keys of IDENTIFIER_MACROS,
// CALL_MACROS, CONSTRUCTOR_MACROS and PROPERTY_CALL_MACROS in the built package, in Object.entries order.
func DefaultMacroSpec() MacroSpec {
	return MacroSpec{
		Identifiers:  []string{"Promise"},
		Calls:        []string{"assert", "typeOf", "typeIs", "classIs", "identity", "$range", "$tuple", "$getModuleTree"},
		Constructors: []string{"ArrayConstructor", "SetConstructor", "MapConstructor", "WeakSetConstructor", "WeakMapConstructor", "ReadonlyMapConstructor", "ReadonlySetConstructor"},
		PropertyCalls: []PropertyMacroClass{
			{Class: "CFrame", Methods: []string{"add", "sub", "mul"}},
			{Class: "UDim", Methods: []string{"add", "sub"}},
			{Class: "UDim2", Methods: []string{"add", "sub"}},
			{Class: "Vector2", Methods: []string{"add", "sub", "mul", "div", "idiv"}},
			{Class: "Vector2int16", Methods: []string{"add", "sub", "mul", "div"}},
			{Class: "Vector3", Methods: []string{"add", "sub", "mul", "div", "idiv"}},
			{Class: "Vector3int16", Methods: []string{"add", "sub", "mul", "div"}},
			{Class: "Number", Methods: []string{"idiv"}},
			{Class: "String", Methods: []string{"size", "byte", "find", "format", "gmatch", "gsub", "lower", "match", "rep", "reverse", "split", "sub", "upper"}},
			{Class: "ArrayLike", Methods: []string{"size"}},
			{Class: "ReadonlyArray", Methods: []string{"isEmpty", "join", "move", "includes", "indexOf", "every", "some", "forEach", "map", "mapFiltered", "filterUndefined", "filter", "reduce", "find", "findIndex"}},
			{Class: "Array", Methods: []string{"push", "pop", "shift", "unshift", "insert", "remove", "unorderedRemove", "sort", "clear"}},
			{Class: "ReadonlySet", Methods: []string{"isEmpty", "size", "has", "forEach"}},
			{Class: "Set", Methods: []string{"delete", "clear", "add"}},
			{Class: "ReadonlyMap", Methods: []string{"isEmpty", "size", "has", "forEach", "get"}},
			{Class: "Map", Methods: []string{"delete", "clear", "set"}},
			{Class: "Promise", Methods: []string{"then"}},
		},
	}
}
