package plugins

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFixEmitSetting(t *testing.T) {
	for _, test := range []struct {
		name  string
		files map[string]string
		want  bool
		bad   bool
	}{
		{name: "nothing", files: map[string]string{"tsconfig.json": `{}`}},
		{name: "block without it", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "setget": true } }`}},
		{name: "on", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "fixEmit": true } }`}, want: true},
		{name: "off", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "fixEmit": false } }`}},
		{name: "not in compilerOptions", files: map[string]string{"tsconfig.json": `{ "compilerOptions": { "fixEmit": true } }`}},
		{name: "not a boolean", files: map[string]string{"tsconfig.json": `{ "rbxtsgo": { "fixEmit": "yes" } }`}, bad: true},
		{name: "extended", files: map[string]string{
			"tsconfig.json": `{ "extends": "./base.json" }`,
			"base.json":     `{ "rbxtsgo": { "fixEmit": true } }`,
		}, want: true},
		{name: "nearest wins", files: map[string]string{
			"tsconfig.json": `{ "extends": "./base.json", "rbxtsgo": { "fixEmit": false } }`,
			"base.json":     `{ "rbxtsgo": { "fixEmit": true } }`,
		}},
		{name: "last of an extends list wins", files: map[string]string{
			"tsconfig.json": `{ "extends": ["./a.json", "./b.json"] }`,
			"a.json":        `{ "rbxtsgo": { "fixEmit": false } }`,
			"b.json":        `{ "rbxtsgo": { "fixEmit": true } }`,
		}, want: true},
		{name: "extends cycle", files: map[string]string{
			"tsconfig.json": `{ "extends": "./base.json" }`,
			"base.json":     `{ "extends": "./tsconfig.json" }`,
		}},
		{name: "comments", files: map[string]string{"tsconfig.json": `{ /* a */ "rbxtsgo": { "fixEmit": true, }, }`}, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := write(t, test.files)
			got, err := FixEmitSetting(filepath.Join(dir, "tsconfig.json"))
			if (err != nil) != test.bad {
				t.Fatalf("error: %v", err)
			}
			if test.bad && !strings.Contains(err.Error(), "must be true or false") {
				t.Errorf("message: %v", err)
			}
			if got != test.want {
				t.Errorf("got %v", got)
			}
		})
	}
}

func TestPassAlongDropsFixEmit(t *testing.T) {
	got := passAlongArgs([]string{"--fix-emit", "--no-fix-emit", "--fix-emit=false", "--verbose"})
	if want := []string{"--verbose"}; !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}
