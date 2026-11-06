package compile

import "testing"

func TestWriteableText(t *testing.T) {
	high := "\xED\xA0\xBD" // U+D83D
	low := "\xED\xB8\x80"  // U+DE00
	cases := []struct{ name, in, want string }{
		{"plain", "abc é 日本語 😀", "abc é 日本語 😀"},
		{"lone high", "a" + high + "b", "a�b"},
		{"lone low", low, "�"},
		{"two lone highs", high + high, "��"},
		{"low then high", low + high, "��"},
		{"pair", high + low, "😀"},
		{"high, then a pair", high + high + low, "�😀"},
		{"cut off at the end", "a\xED\xA0", "a\xED\xA0"},
	}
	for _, c := range cases {
		if got := WriteableText(c.in); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
