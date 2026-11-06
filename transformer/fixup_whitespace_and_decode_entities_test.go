package transformer

import "testing"

func TestFixupWhitespaceAndDecodeEntities(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"one line é", "é", "é"},
		{"one line keeps both edges", "  日本語  ", "  日本語  "},
		{"lines are trimmed and joined", "\n  é 日本語 😀  \n  😀\n", "é 日本語 😀 😀"},
		{"first line keeps leading whitespace", "  😀\n", "  😀"},
		{"last character of a line is outside the BMP", "😀\n   \n", "😀"},
		{"only whitespace", "\n \n", ""},
		{"non-ASCII whitespace trims", "a \nb", "a b"},
		{"first and last lines keep their outer edges", "　日本語　\n　😀　", "　日本語 😀　"},
		{"entities", "&eacute;&#26085;&#x1F600;&amp;lt;&bogus;", "é日😀&lt;&bogus;"},
		{"entities decode per line", "&#xe9;\n&#x65E5;", "é 日"},
		{"entities make a surrogate pair", "&#xD83D;&#xDE00;", "😀"},
		{"entity makes a lone surrogate", "&#xD83D;", "\xED\xA0\xBD"},
	}
	for _, c := range cases {
		if got := fixupWhitespaceAndDecodeEntities(c.input); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFixupWhitespaceInvalidCodePointPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("String.fromCodePoint throws above U+10FFFF")
		}
	}()
	fixupWhitespaceAndDecodeEntities("&#x110000;")
}

func TestCleanModuleNameCountsUTF16Units(t *testing.T) {
	cases := []struct{ input, want string }{
		{"a-b_c1", "a_b_c1"},
		{"é", "_"},
		{"日本語", "___"},
		{"😀", "__"},
		{"\xED\xA0\xBD", "_"},
	}
	for _, c := range cases {
		if got := cleanModuleName(c.input); got != c.want {
			t.Errorf("cleanModuleName(%q): got %q, want %q", c.input, got, c.want)
		}
	}
}
