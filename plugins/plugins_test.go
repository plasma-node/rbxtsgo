package plugins

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// write makes files in a new project folder, with the plugin installed at the version the port matches.
func write(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	all := map[string]string{"node_modules/" + SetgetPlugin + "/package.json": `{ "version": "2.0.0" }`}
	for name, text := range files {
		all[name] = text
	}
	for name, text := range all {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDecide(t *testing.T) {
	const setget = `{ "transform": "rbxts-transformer-setget" }`
	yes, no, dollar := true, false, "$"
	cases := []struct {
		name     string
		files    map[string]string
		action   Action
		prefix   string
		inReason string
		options  Options
	}{
		{name: "no plugins", files: map[string]string{"tsconfig.json": `{ "compilerOptions": {} }`}},
		{name: "empty list", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [] } }`}},
		{name: "language service plugin only", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [{ "name": "x" }] } }`}},
		{name: "non-string transform", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [{ "transform": 1 }] } }`}},
		{name: "commented out", files: map[string]string{"tsconfig.json": `{
			"compilerOptions": {
				"plugins": [
					// { "transform": "rbxts-transformer-setget" },
					/* { "transform": "other" } */
				]
			}
		}`}},
		{name: "commented out next to a language service plugin", files: map[string]string{"tsconfig.json": `{
			"compilerOptions": { "plugins": [ /* { "transform": "other" }, */ { "name": "x" } ] }
		}`}},
		{name: "setget", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [` + setget + `] } }`}, action: ActionSetget, prefix: "__"},
		{name: "setget with trailing comma", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [` + setget + `,], }, }`}, action: ActionSetget, prefix: "__"},
		{name: "custom prefix", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [{ "transform": "rbxts-transformer-setget", "customPrefix": "_" }] } }`}, action: ActionSetget, prefix: "_"},
		{name: "null prefix means the default", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [{ "transform": "rbxts-transformer-setget", "customPrefix": null }] } }`}, action: ActionSetget, prefix: "__"},
		{name: "after doesn't matter with one plugin", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [{ "transform": "rbxts-transformer-setget", "after": true }] } }`}, action: ActionSetget, prefix: "__"},
		{name: "another plugin", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [{ "transform": "other" }] } }`}, action: ActionFallback, inReason: `"other"`},
		{name: "setget and another", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [` + setget + `, { "transform": "other" }] } }`}, action: ActionFallback, inReason: `"other"`},
		{name: "setget twice", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [` + setget + `,` + setget + `] } }`}, action: ActionFallback, inReason: SetgetPlugin},
		{name: "other type", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [{ "transform": "rbxts-transformer-setget", "type": "config" }] } }`}, action: ActionFallback, inReason: "config"},
		{name: "other import", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [{ "transform": "rbxts-transformer-setget", "import": "x" }] } }`}, action: ActionFallback, inReason: "import"},
		{name: "number prefix", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [{ "transform": "rbxts-transformer-setget", "customPrefix": 1 }] } }`}, action: ActionFallback, inReason: "customPrefix"},
		{name: "other version", files: map[string]string{
			"tsconfig.json": `{ "compilerOptions": { "plugins": [` + setget + `] } }`,
			"node_modules/" + SetgetPlugin + "/package.json": `{ "version": "1.0.9" }`,
		}, action: ActionFallback, inReason: "1.0.9"},
		{name: "from extends", files: map[string]string{
			"tsconfig.json": `{ "extends": "./base", "compilerOptions": {} }`,
			"base.json":     `{ "compilerOptions": { "plugins": [` + setget + `] } }`,
		}, action: ActionSetget, prefix: "__"},
		{name: "extends adds to the child's", files: map[string]string{
			"tsconfig.json": `{ "extends": "./base.json", "compilerOptions": { "plugins": [` + setget + `] } }`,
			"base.json":     `{ "compilerOptions": { "plugins": [{ "transform": "from-base" }] } }`,
		}, action: ActionFallback, inReason: `"from-base"`},
		{name: "a child's empty list turns it off", files: map[string]string{
			"tsconfig.json": `{ "extends": "./base.json", "compilerOptions": { "plugins": [] } }`,
			"base.json":     `{ "compilerOptions": { "plugins": [{ "transform": "from-base" }] } }`,
		}},
		{name: "extends from a package", files: map[string]string{
			"tsconfig.json": `{ "extends": "shared-config/tsconfig.json" }`,
			"node_modules/shared-config/tsconfig.json": `{ "compilerOptions": { "plugins": [{ "transform": "from-package" }] } }`,
		}, action: ActionFallback, inReason: `"from-package"`},
		{name: "array extends with no plugins", files: map[string]string{
			"tsconfig.json": `{ "extends": ["./a.json"] }`,
			"a.json":        `{ "compilerOptions": {} }`,
		}},
		{name: "array extends with plugins", files: map[string]string{
			"tsconfig.json": `{ "extends": ["./a.json"] }`,
			"a.json":        `{ "compilerOptions": { "plugins": [{ "transform": "x" }] } }`,
		}, action: ActionFallback, inReason: "extends"},
		{name: "block on", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setget": true } }`}, action: ActionSetget, prefix: "__"},
		{name: "block off", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setget": false } }`}},
		{name: "block prefix", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setget": true, "setgetPrefix": "_" } }`}, action: ActionSetget, prefix: "_"},
		{name: "block alone has no effect on a prefix", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setgetPrefix": "_" } }`}},
		{name: "block from extends", files: map[string]string{
			"tsconfig.json": `{ "extends": "./base.json" }`,
			"base.json":     `{ "rbxtsgo": { "setget": true } }`,
		}, action: ActionSetget, prefix: "__"},
		{name: "child block wins", files: map[string]string{
			"tsconfig.json": `{ "extends": "./base.json", "rbxtsgo": { "setget": false } }`,
			"base.json":     `{ "rbxtsgo": { "setget": true } }`,
		}},
		{name: "bad block", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setget": "yes" } }`}, action: ActionError, inReason: "true or false"},
		{name: "flag on", files: map[string]string{"tsconfig.json": `{}`}, options: Options{Setget: &yes}, action: ActionSetget, prefix: "__"},
		{name: "flag off beats block", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setget": true } }`}, options: Options{Setget: &no}},
		{name: "flag prefix beats everything", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setget": true, "setgetPrefix": "_" } }`}, options: Options{Prefix: &dollar}, action: ActionSetget, prefix: "$"},
		{name: "built-in with another plugin", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setget": true }, "compilerOptions": { "plugins": [{ "transform": "other" }] } }`}, action: ActionError, inReason: `"other"`},
		{name: "built-in ignores the installed version", files: map[string]string{
			"tsconfig.json": `{ "rbxtsgo": { "setget": true }, "compilerOptions": { "plugins": [` + setget + `] } }`,
			"node_modules/" + SetgetPlugin + "/package.json": `{ "version": "1.0.9" }`,
		}, action: ActionSetget, prefix: "__"},
		{name: "built-in takes the plugin's prefix", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setget": true }, "compilerOptions": { "plugins": [{ "transform": "rbxts-transformer-setget", "customPrefix": "_" }] } }`}, action: ActionSetget, prefix: "_"},
		{name: "flag off leaves a plugin entry on", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "plugins": [` + setget + `] } }`}, options: Options{Setget: &no}, action: ActionSetget, prefix: "__"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			decision := Decide(filepath.Join(write(t, c.files), "tsconfig.json"), c.options)
			if decision.Action != c.action {
				t.Fatalf("action %d, want %d (%s)", decision.Action, c.action, decision.Reason)
			}
			if decision.Prefix != c.prefix {
				t.Errorf("prefix %q, want %q", decision.Prefix, c.prefix)
			}
			if !strings.Contains(decision.Reason, c.inReason) {
				t.Errorf("reason %q doesn't mention %s", decision.Reason, c.inReason)
			}
		})
	}
}

func TestPluginEntryWorksWithoutThePackage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tsconfig.json")
	if err := os.WriteFile(path, []byte(`{ "compilerOptions": { "plugins": [{ "transform": "rbxts-transformer-setget", "customPrefix": "_" }] } }`), 0o666); err != nil {
		t.Fatal(err)
	}
	if decision := Decide(path, Options{}); decision.Action != ActionSetget || decision.Prefix != "_" {
		t.Fatalf("got %+v", decision)
	}
}

func TestPassAlongArgs(t *testing.T) {
	got := passAlongArgs([]string{"build", "-p", "x", "--checkers", "8", "--verbose", "--cpuprofile=a.prof", "-checkers=2", "--rojo", "y", "--no-noInclude", "--setget", "--no-setget", "--setget=false", "--setgetPrefix", "_"})
	want := []string{"build", "-p", "x", "--verbose", "--rojo", "y", "--no-noInclude"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestDelegate(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("needs node")
	}
	dir := write(t, map[string]string{
		"tsconfig.json":                         `{}`,
		"node_modules/roblox-ts/out/CLI/cli.js": `require("fs").writeFileSync(require("path").join(__dirname, "args.json"), JSON.stringify(process.argv.slice(2))); process.exit(7);`,
	})
	var messages bytes.Buffer
	code, err := Delegate(`rbxtsgo can't run the transformer plugin "other"`, filepath.Join(dir, "tsconfig.json"), []string{"-p", dir, "--checkers", "2", "--verbose"}, &messages)
	if err != nil {
		t.Fatal(err)
	}
	if code != 7 {
		t.Errorf("exit code %d, want 7", code)
	}
	if text := messages.String(); !strings.Contains(text, `"other"`) || !strings.Contains(text, "JS compiler") {
		t.Errorf("message: %s", text)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "node_modules", "roblox-ts", "out", "CLI", "args.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"--verbose"`) || strings.Contains(string(raw), "checkers") {
		t.Errorf("args: %s", raw)
	}
}

func TestDelegateFindsTheCompilerAbove(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("needs node")
	}
	dir := write(t, map[string]string{
		"sub/tsconfig.json":                     `{}`,
		"node_modules/roblox-ts/out/CLI/cli.js": `process.exit(0);`,
	})
	if code, err := Delegate("reason", filepath.Join(dir, "sub", "tsconfig.json"), nil, &bytes.Buffer{}); err != nil || code != 0 {
		t.Errorf("code %d, err %v", code, err)
	}
}

func TestDelegateWithoutTheCompiler(t *testing.T) {
	dir := write(t, map[string]string{"tsconfig.json": `{}`})
	code, err := Delegate(`rbxtsgo can't run the transformer plugin "other"`, filepath.Join(dir, "tsconfig.json"), nil, &bytes.Buffer{})
	if err == nil || code != 1 {
		t.Fatalf("code %d, err %v", code, err)
	}
	for _, part := range []string{`"other"`, "node_modules", "npm install", "roblox-ts"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("error %q doesn't mention %s", err, part)
		}
	}
}
