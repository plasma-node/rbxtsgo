package rojoresolver

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
)

// object is a parsed JSON object that keeps JavaScript's property order, because Object.keys decides the order
// the tree is walked in, which decides partition order and which mapping wins. encoding/json maps can't do that.
type object struct {
	keys   []string
	values map[string]any
}

func (o *object) get(key string) (any, bool) {
	value, ok := o.values[key]
	return value, ok
}

func (o *object) set(key string, value any) {
	if _, exists := o.values[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

// isArrayIndex reports whether key is a canonical array index, which JS objects list first, ascending.
func isArrayIndex(key string) (uint32, bool) {
	value, err := strconv.ParseUint(key, 10, 32)
	if err != nil || value == math.MaxUint32 || strconv.FormatUint(value, 10) != key {
		return 0, false
	}
	return uint32(value), true
}

// orderKeys puts array index keys first, ascending, as Object.keys does.
func (o *object) orderKeys() {
	var indexes []string
	var names []string
	for _, key := range o.keys {
		if _, ok := isArrayIndex(key); ok {
			indexes = append(indexes, key)
		} else {
			names = append(names, key)
		}
	}
	slices.SortFunc(indexes, func(a, b string) int {
		x, _ := isArrayIndex(a)
		y, _ := isArrayIndex(b)
		return int(int64(x) - int64(y))
	})
	o.keys = append(indexes, names...)
}

// parseJSON parses like JSON.parse: objects become *object, arrays []any, numbers json.Number.
func parseJSON(text string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	value, err := parseValue(decoder)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected data after JSON value")
	}
	return value, nil
}

func parseValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return token, nil
	}
	switch delim {
	case '{':
		result := &object{values: map[string]any{}}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			value, err := parseValue(decoder)
			if err != nil {
				return nil, err
			}
			result.set(keyToken.(string), value)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		result.orderKeys()
		return result, nil
	case '[':
		result := []any{}
		for decoder.More() {
			value, err := parseValue(decoder)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, errors.New("unexpected JSON delimiter")
}

// validationError is one entry of ajv's errors array, as errorsText prints it.
type validationError struct {
	instancePath string
	message      string
}

// errorsText is ajv.errorsText with default options.
func errorsText(errs []validationError) string {
	parts := make([]string, len(errs))
	for i, err := range errs {
		parts[i] = "data" + err.instancePath + " " + err.message
	}
	return strings.Join(parts, ", ")
}

func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

func isBool(value any) bool {
	_, ok := value.(bool)
	return ok
}

func isObject(value any) bool {
	_, ok := value.(*object)
	return ok
}

// isInteger follows ajv's integer check: a finite number with no fractional part.
func isInteger(value any) bool {
	number, ok := value.(json.Number)
	if !ok {
		return false
	}
	parsed, err := strconv.ParseFloat(number.String(), 64)
	if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return false
	}
	return math.Mod(parsed, 1) == 0
}

// validateRojoConfig is a hand port of what ajv generates for rojo-schema.json. Ajv stops at the first failing
// keyword, except inside anyOf, where it keeps the errors of each branch. The order of checks here decides which
// error is reported, so it follows ajv's: type, required, then properties in schema order, then patternProperties.
// The tests compare it with real ajv.
func validateRojoConfig(data any) []validationError {
	root, ok := data.(*object)
	if !ok {
		return []validationError{{"", "must be object"}}
	}
	for _, name := range []string{"name", "tree"} {
		if _, exists := root.get(name); !exists {
			return []validationError{{"", "must have required property '" + name + "'"}}
		}
	}
	if !isString(root.values["name"]) {
		return []validationError{{"/name", "must be string"}}
	}
	if port, exists := root.get("servePort"); exists && !isInteger(port) {
		return []validationError{{"/servePort", "must be integer"}}
	}
	return validateTree(root.values["tree"], "/tree")
}

func validateTree(data any, path string) []validationError {
	tree, ok := data.(*object)
	if !ok {
		return []validationError{{path, "must be object"}}
	}
	if value, exists := tree.get("$className"); exists && !isString(value) {
		return []validationError{{path + "/$className", "must be string"}}
	}
	if value, exists := tree.get("$ignoreUnknownInstances"); exists && !isBool(value) {
		return []validationError{{path + "/$ignoreUnknownInstances", "must be boolean"}}
	}
	if value, exists := tree.get("$path"); exists {
		if errs := validatePath(value, path+"/$path"); errs != nil {
			return errs
		}
	}
	if value, exists := tree.get("$properties"); exists && !isObject(value) {
		return []validationError{{path + "/$properties", "must be object"}}
	}
	for _, key := range tree.keys {
		if matchesChildPattern(key) {
			if errs := validateTree(tree.values[key], path+"/"+escapeJSONPointer(key)); errs != nil {
				return errs
			}
		}
	}
	return nil
}

func validatePath(value any, path string) []validationError {
	if isString(value) {
		return nil
	}
	errs := []validationError{{path, "must be string"}}
	if object, ok := value.(*object); ok {
		optional, exists := object.get("optional")
		if !exists || isString(optional) {
			return nil
		}
		errs = append(errs, validationError{path + "/optional", "must be string"})
	} else {
		errs = append(errs, validationError{path, "must be object"})
	}
	return append(errs, validationError{path, "must match a schema in anyOf"})
}

// matchesChildPattern is the regex ^[^\$].*$ with the u flag: a first character that isn't $, then no line
// terminators after it, because . doesn't match those.
func matchesChildPattern(key string) bool {
	runes := []rune(key)
	if len(runes) == 0 || runes[0] == '$' {
		return false
	}
	for _, r := range runes[1:] {
		if r == '\n' || r == '\r' || r == ' ' || r == ' ' {
			return false
		}
	}
	return true
}

func escapeJSONPointer(key string) string {
	return strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}
