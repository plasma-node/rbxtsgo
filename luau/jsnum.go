package luau

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// No upstream file: the TypeScript source relies on the JavaScript runtime for
// Number(string) and String(number) in create.ts (number), renderNumberLiteral.ts
// and renderNumericForStatement.ts. These helpers emulate those two operations
// (ECMAScript StringToNumber and Number::toString(10)).

// jsNumberToString is JavaScript String(v) for a number.
func jsNumberToString(v float64) string {
	switch {
	case v != v:
		return "NaN"
	case v == 0:
		return "0" // also -0
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	}
	if v == math.Trunc(v) && math.Abs(v) < 1<<53 {
		return strconv.FormatInt(int64(v), 10)
	}

	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}

	// Shortest round-trip digits, as "d.ddde+XX".
	e := strconv.FormatFloat(v, 'e', -1, 64)
	ePos := strings.IndexByte(e, 'e')
	exp, _ := strconv.Atoi(e[ePos+1:])
	digits := e[:ePos]
	if len(digits) > 1 {
		digits = digits[:1] + digits[2:]
	}
	k := len(digits)
	n := exp + 1 // value = 0.digits * 10^n

	var b strings.Builder
	b.WriteString(sign)
	switch {
	case k <= n && n <= 21:
		b.WriteString(digits)
		for i := 0; i < n-k; i++ {
			b.WriteByte('0')
		}
	case 0 < n && n <= 21:
		b.WriteString(digits[:n])
		b.WriteByte('.')
		b.WriteString(digits[n:])
	case -6 < n && n <= 0:
		b.WriteString("0.")
		for i := 0; i < -n; i++ {
			b.WriteByte('0')
		}
		b.WriteString(digits)
	default:
		b.WriteByte(digits[0])
		if k > 1 {
			b.WriteByte('.')
			b.WriteString(digits[1:])
		}
		b.WriteByte('e')
		if n-1 >= 0 {
			b.WriteByte('+')
		}
		b.WriteString(strconv.Itoa(n - 1))
	}
	return b.String()
}

// isJSSpace reports whether r is ECMAScript WhiteSpace or LineTerminator.
func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x00a0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

func digitValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 99
}

// parseRadix parses a non-empty run of digits in the given base, rounded to nearest float64.
func parseRadix(digits string, base int) float64 {
	for i := 0; i < len(digits); i++ {
		if digitValue(digits[i]) >= base {
			return math.NaN()
		}
	}
	// 4 bits per hex digit; up to 13 digits is exact in a uint64 and a float64.
	if len(digits) <= 12 {
		u, _ := strconv.ParseUint(digits, base, 64)
		return float64(u)
	}
	i, _ := new(big.Int).SetString(digits, base) // digits were validated above
	f, _ := new(big.Float).SetInt(i).Float64()
	return f
}

// jsToNumber is JavaScript Number(s) for a string.
func jsToNumber(s string) float64 {
	s = strings.TrimFunc(s, isJSSpace)
	if s == "" {
		return 0
	}
	if len(s) > 2 && s[0] == '0' {
		switch s[1] {
		case 'x', 'X':
			return parseRadix(s[2:], 16)
		case 'o', 'O':
			return parseRadix(s[2:], 8)
		case 'b', 'B':
			return parseRadix(s[2:], 2)
		}
	}

	rest := s
	neg := false
	if rest[0] == '+' || rest[0] == '-' {
		neg = rest[0] == '-'
		rest = rest[1:]
	}
	if rest == "Infinity" {
		if neg {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}

	// StrUnsignedDecimalLiteral: digits [. digits?] | . digits, then optional exponent.
	i := 0
	for i < len(rest) && isDigit(rest[i]) {
		i++
	}
	intDigits := i
	fracDigits := 0
	if i < len(rest) && rest[i] == '.' {
		i++
		start := i
		for i < len(rest) && isDigit(rest[i]) {
			i++
		}
		fracDigits = i - start
	}
	if intDigits == 0 && fracDigits == 0 {
		return math.NaN()
	}
	if i < len(rest) && (rest[i] == 'e' || rest[i] == 'E') {
		i++
		if i < len(rest) && (rest[i] == '+' || rest[i] == '-') {
			i++
		}
		start := i
		for i < len(rest) && isDigit(rest[i]) {
			i++
		}
		if i == start {
			return math.NaN()
		}
	}
	if i != len(rest) {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(rest, 64) // out of range gives +-Inf, as in JS
	if neg {
		return -f
	}
	return f
}
