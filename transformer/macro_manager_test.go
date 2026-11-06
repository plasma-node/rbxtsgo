package transformer

import (
	"maps"
	"slices"
	"testing"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

func sortedKeys[V any](table map[string]V) []string {
	return slices.Sorted(maps.Keys(table))
}

func checkMacroKeys(t *testing.T, kind string, got []string, want []string) {
	t.Helper()
	want = slices.Sorted(slices.Values(want))
	if !slices.Equal(got, want) {
		t.Errorf("%s macro names differ from tsbridge.DefaultMacroSpec()\n got: %v\nwant: %v", kind, got, want)
	}
}

// The tables and the spec must agree. A name that's only in the spec is a macro that silently does nothing, and a name
// that's only in a table is a macro that's never looked up.
func TestMacroTablesMatchDefaultMacroSpec(t *testing.T) {
	spec := tsbridge.DefaultMacroSpec()
	tables := getMacroTables()

	checkMacroKeys(t, "identifier", sortedKeys(tables.identifiers), spec.Identifiers)
	checkMacroKeys(t, "call", sortedKeys(tables.calls), spec.Calls)
	checkMacroKeys(t, "constructor", sortedKeys(tables.constructors), spec.Constructors)

	var propertyCalls []string
	for _, class := range spec.PropertyCalls {
		for _, method := range class.Methods {
			propertyCalls = append(propertyCalls, class.Class+"."+method)
		}
	}
	checkMacroKeys(t, "property call", sortedKeys(tables.propertyCalls), propertyCalls)

	for name, macro := range tables.identifiers {
		if macro == nil {
			t.Errorf("identifier macro %s is nil", name)
		}
	}
	for name, macro := range tables.calls {
		if macro == nil {
			t.Errorf("call macro %s is nil", name)
		}
	}
	for name, macro := range tables.constructors {
		if macro == nil {
			t.Errorf("constructor macro %s is nil", name)
		}
	}
	for name, macro := range tables.propertyCalls {
		if macro == nil {
			t.Errorf("property call macro %s is nil", name)
		}
	}
}
