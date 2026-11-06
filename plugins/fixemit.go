package plugins

import (
	"errors"
	"fmt"
	"path/filepath"
)

// FixEmitSetting is "fixEmit" in the tsconfig's top-level "rbxtsgo" block: the output fixes that differ from JS 3.0.0,
// listed in docs/known-differences.md. The nearest tsconfig that sets it wins, and extends is followed, as for the
// block's other settings. An error means the value isn't a boolean, and on is false then.
func FixEmitSetting(tsConfigPath string) (on bool, err error) {
	value, text := readFixEmit(tsConfigPath, map[string]bool{})
	if text != "" {
		return false, errors.New(text)
	}
	return value != nil && *value, nil
}

func readFixEmit(path string, seen map[string]bool) (*bool, string) {
	if seen[path] {
		return nil, ""
	}
	seen[path] = true
	config, ok := readConfig(path)
	if !ok {
		return nil, ""
	}
	if block, ok := config.Get(BuiltinBlock); ok {
		if object, ok := block.(jsonObject); ok {
			if value, ok := object.Get("fixEmit"); ok {
				if on, ok := value.(bool); ok {
					return &on, ""
				}
				return nil, fmt.Sprintf(`"%s.fixEmit" in %s must be true or false`, BuiltinBlock, path)
			}
		}
	}
	extends, _ := config.Get("extends")
	var parents []string
	switch value := extends.(type) {
	case string:
		parents = []string{value}
	case []any:
		// As with compilerOptions, the last entry of an extends list wins.
		for i := len(value) - 1; i >= 0; i-- {
			if name, ok := value[i].(string); ok {
				parents = append(parents, name)
			}
		}
	}
	for _, spec := range parents {
		if extended, ok := resolveExtends(spec, filepath.Dir(path)); ok {
			if value, text := readFixEmit(extended, seen); value != nil || text != "" {
				return value, text
			}
		}
	}
	return nil, ""
}
