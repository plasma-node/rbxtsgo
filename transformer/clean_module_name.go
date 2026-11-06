package transformer

// Port of src/TSTransformer/util/cleanModuleName.ts.

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/stringutil"
)

// cleanModuleName is name.replace(/\W/g, "_"). JS replaces per UTF-16 unit, so a character outside the BMP gives two
// underscores. \W is ASCII only without the u flag.
func cleanModuleName(name string) string {
	var result strings.Builder
	for i := 0; i < len(name); {
		r, size := stringutil.DecodeJSStringRune(name[i:])
		switch {
		case r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z':
			result.WriteRune(r)
		case r > 0xFFFF:
			result.WriteString("__")
		default:
			result.WriteByte('_')
		}
		i += size
	}
	return result.String()
}
