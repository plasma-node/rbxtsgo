package luau

// Ported from src/LuauAST/util/assert.ts, isMetamethod.ts,
// isReservedClassField.ts, isReservedIdentifier.ts, isValidIdentifier.ts and
// isValidNumberLiteral.ts.

// AssertionError is the panic value Assert raises. Upstream throws an Error
// with the same message; treat it as a compiler bug.
type AssertionError struct {
	Message string
}

func (e *AssertionError) Error() string {
	return "Assertion Failed! " + e.Message +
		"\nThis is a compiler bug! Please submit a bug report here:" +
		"\nhttps://github.com/roblox-ts/roblox-ts/issues"
}

// Assert panics with *AssertionError when ok is false (upstream assert).
func Assert(ok bool, message string) {
	if !ok {
		panic(&AssertionError{Message: message})
	}
}

// IsMetamethod returns true if the given string is a valid Luau metamethod.
func IsMetamethod(id string) bool {
	switch id {
	case "__index", "__newindex", "__call", "__concat", "__unm", "__add", "__sub", "__mul", "__div",
		"__mod", "__pow", "__tostring", "__metatable", "__eq", "__lt", "__le", "__mode", "__gc", "__len":
		return true
	}
	return false
}

// IsReservedClassField reports whether id is "__index" or "new".
func IsReservedClassField(id string) bool {
	return id == "__index" || id == "new"
}

// IsReservedIdentifier reports whether id is a key of Globals (upstream
// hasOwnProperty on the globals object).
func IsReservedIdentifier(id string) bool {
	switch id {
	case "_G", "TS", "assert", "bit32", "coroutine", "error", "exports", "getmetatable", "ipairs", "next",
		"pairs", "pcall", "require", "script", "select", "self", "setmetatable", "string", "super", "table",
		"utf8", "math", "tostring", "type", "typeof", "unpack", "game":
		return true
	}
	return false
}

func isLuauKeyword(id string) bool {
	switch id {
	case "and", "break", "do", "else", "elseif", "end", "false", "for", "function", "if", "in", "local",
		"nil", "not", "or", "repeat", "return", "then", "true", "until", "while":
		return true
	}
	return false
}

// IsValidIdentifier returns true if the given string is a valid Luau
// identifier, and does not conflict with a keyword. Equivalent to
// /^[A-Za-z_][A-Za-z0-9_]*$/ plus the keyword check.
func IsValidIdentifier(id string) bool {
	if len(id) == 0 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			continue
		}
		if i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return !isLuauKeyword(id)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// skipDigitsUnderscores returns the index after any run of [0-9_] at s[i:].
func skipDigitsUnderscores(s string, i int) int {
	for i < len(s) && (isDigit(s[i]) || s[i] == '_') {
		i++
	}
	return i
}

// IsValidNumberLiteral reports whether text is a Luau number literal.
// Equivalent to the union of the upstream regexes:
//
//	decimal: /^(?:\d[\d_]*(?:\.[\d_]*)?|\.\d[\d_]*)(?:[eE][+-]?_*\d[\d_]*)?$/
//	binary:  /^0_*[bB]_*[01][01_]*$/
//	hex:     /^0_*[xX]_*[\da-fA-F][\da-fA-F_]*$/
func IsValidNumberLiteral(s string) bool {
	return isDecimalLiteral(s) || isBinaryLiteral(s) || isHexLiteral(s)
}

func isDecimalLiteral(s string) bool {
	i := 0
	switch {
	case i < len(s) && isDigit(s[i]):
		i = skipDigitsUnderscores(s, i+1)
		if i < len(s) && s[i] == '.' {
			i = skipDigitsUnderscores(s, i+1)
		}
	case i < len(s) && s[i] == '.' && i+1 < len(s) && isDigit(s[i+1]):
		i = skipDigitsUnderscores(s, i+2)
	default:
		return false
	}
	if i == len(s) {
		return true
	}
	if s[i] != 'e' && s[i] != 'E' {
		return false
	}
	i++
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	for i < len(s) && s[i] == '_' {
		i++
	}
	if i >= len(s) || !isDigit(s[i]) {
		return false
	}
	i = skipDigitsUnderscores(s, i+1)
	return i == len(s)
}

func skipChar(s string, i int, c byte) int {
	for i < len(s) && s[i] == c {
		i++
	}
	return i
}

func isBinaryLiteral(s string) bool {
	if len(s) == 0 || s[0] != '0' {
		return false
	}
	i := skipChar(s, 1, '_')
	if i >= len(s) || (s[i] != 'b' && s[i] != 'B') {
		return false
	}
	i = skipChar(s, i+1, '_')
	if i >= len(s) || (s[i] != '0' && s[i] != '1') {
		return false
	}
	for i++; i < len(s); i++ {
		if s[i] != '0' && s[i] != '1' && s[i] != '_' {
			return false
		}
	}
	return true
}

func isHexLiteral(s string) bool {
	if len(s) == 0 || s[0] != '0' {
		return false
	}
	i := skipChar(s, 1, '_')
	if i >= len(s) || (s[i] != 'x' && s[i] != 'X') {
		return false
	}
	i = skipChar(s, i+1, '_')
	if i >= len(s) || !isHexDigit(s[i]) {
		return false
	}
	for i++; i < len(s); i++ {
		if !isHexDigit(s[i]) && s[i] != '_' {
			return false
		}
	}
	return true
}
