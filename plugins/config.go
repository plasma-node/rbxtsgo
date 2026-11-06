// Package plugins handles compilerOptions.plugins and the built-in setget option. rbxts-transformer-setget runs natively,
// with or without its package. Any other plugin hands the whole build to the JS compiler. See
// docs/setget.md.
package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
)

// SetgetPlugin is the plugin that runs natively, and SetgetVersion is the only version the port matches.
const (
	SetgetPlugin  = "rbxts-transformer-setget"
	SetgetVersion = "2.0.0"
	// BuiltinBlock is the top-level tsconfig key for the built-in option.
	BuiltinBlock = "rbxtsgo"
)

// Action says what a tsconfig's plugins call for.
type Action int

const (
	// ActionNone builds as usual, with no reprint. JS does the same when no entry has a "transform".
	ActionNone Action = iota
	// ActionSetget runs the port.
	ActionSetget
	// ActionFallback hands the build to the JS compiler.
	ActionFallback
	// ActionError stops the build with Reason.
	ActionError
)

// Options is the part of the setting that comes from the command line. Nil means not given.
type Options struct {
	Setget *bool
	Prefix *string
}

// Decision is the result of Decide.
type Decision struct {
	Action Action
	// Prefix is the plugin's customPrefix, or its default.
	Prefix string
	// Reason says why the build needs the JS compiler, naming the plugin, for ActionFallback. For ActionError, it's the
	// error.
	Reason string
}

type pluginConfig struct {
	transform string
	values    interface{ Get(string) (any, bool) }
}

type jsonObject = interface{ Get(string) (any, bool) }

func readConfig(path string) (jsonObject, bool) {
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	fileName := tspath.RootedFilePathFromAbsolute(filepath.ToSlash(path))
	raw, _ := tsoptions.ParseConfigFileTextToJson(fileName, osvfs.FS().CaseSensitivity().PathKey(fileName.AsPath()), string(text))
	object, ok := raw.(jsonObject)
	return object, ok
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// resolveExtends is the part of require.resolve(config.extends, { paths: [dirname] }) that a tsconfig can reach.
func resolveExtends(spec, directory string) (string, bool) {
	try := func(base string) (string, bool) {
		for _, candidate := range []string{base, base + ".json"} {
			if exists(candidate) {
				return candidate, true
			}
		}
		return "", false
	}
	if filepath.IsAbs(spec) {
		return try(spec)
	}
	if strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") {
		return try(filepath.Join(directory, spec))
	}
	for current := directory; ; {
		if found, ok := try(filepath.Join(current, "node_modules", spec)); ok {
			return found, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

// getPluginConfigs is getPluginConfigs: the entries with a string "transform", then those of what the tsconfig extends.
// So unlike every other option, a parent's plugins are added to the child's. active is compilerOptions.plugins as
// TypeScript merges it, where a child's list replaces its parent's. Upstream only looks at the entries if that isn't
// empty.
func getPluginConfigs(path string, seen map[string]bool) (configs []pluginConfig, active *bool, err error) {
	if seen[path] {
		return nil, nil, fmt.Errorf("tsconfig extends itself through %s", path)
	}
	seen[path] = true
	config, ok := readConfig(path)
	if !ok {
		return nil, nil, nil
	}
	if compilerOptions, ok := config.Get("compilerOptions"); ok {
		if object, ok := compilerOptions.(jsonObject); ok {
			if plugins, ok := object.Get("plugins"); ok {
				list, isList := plugins.([]any)
				value := isList && len(list) > 0
				active = &value
				for _, entry := range list {
					if values, ok := entry.(jsonObject); ok {
						if transform, ok := values.Get("transform"); ok {
							if name, ok := transform.(string); ok && name != "" {
								configs = append(configs, pluginConfig{transform: name, values: values})
							}
						}
					}
				}
			}
		}
	}
	extends, ok := config.Get("extends")
	if !ok || extends == nil {
		return configs, active, nil
	}
	spec, ok := extends.(string)
	if !ok {
		// require.resolve throws on a list, if upstream gets this far. TS merges the list, later entries first.
		list, _ := extends.([]any)
		for i := len(list) - 1; i >= 0 && active == nil; i-- {
			if name, ok := list[i].(string); ok {
				if extended, ok := resolveExtends(name, filepath.Dir(path)); ok {
					_, active, _ = getPluginConfigs(extended, seen)
				}
			}
		}
		return nil, active, fmt.Errorf(`"extends" in %s isn't a single path`, path)
	}
	if spec == "" {
		return configs, active, nil
	}
	extended, ok := resolveExtends(spec, filepath.Dir(path))
	if !ok {
		return configs, active, fmt.Errorf(`can't resolve "extends": %q in %s`, spec, path)
	}
	parents, parentActive, err := getPluginConfigs(extended, seen)
	if active == nil {
		active = parentActive
	}
	return append(configs, parents...), active, err
}

// builtinSetting is the top-level "rbxtsgo" block. TS ignores top-level keys it doesn't know, as it does for "rbxts".
// The nearest tsconfig that sets a value wins, and extends is followed as for compilerOptions.
type builtinSetting struct {
	setget *bool
	prefix *string
	err    string
}

func readBuiltinSetting(path string, seen map[string]bool) builtinSetting {
	var setting builtinSetting
	if seen[path] {
		return setting
	}
	seen[path] = true
	config, ok := readConfig(path)
	if !ok {
		return setting
	}
	if block, ok := config.Get(BuiltinBlock); ok {
		if object, ok := block.(jsonObject); ok {
			if value, ok := object.Get("setget"); ok {
				if on, ok := value.(bool); ok {
					setting.setget = &on
				} else {
					setting.err = fmt.Sprintf(`"%s.setget" in %s must be true or false`, BuiltinBlock, path)
				}
			}
			if value, ok := object.Get("setgetPrefix"); ok {
				if prefix, ok := value.(string); ok {
					setting.prefix = &prefix
				} else {
					setting.err = fmt.Sprintf(`"%s.setgetPrefix" in %s must be a string`, BuiltinBlock, path)
				}
			}
		} else {
			setting.err = fmt.Sprintf(`"%s" in %s must be an object`, BuiltinBlock, path)
		}
	}
	extends, _ := config.Get("extends")
	var parents []string
	switch value := extends.(type) {
	case string:
		parents = []string{value}
	case []any:
		for i := len(value) - 1; i >= 0; i-- {
			if name, ok := value[i].(string); ok {
				parents = append(parents, name)
			}
		}
	}
	for _, spec := range parents {
		if extended, ok := resolveExtends(spec, filepath.Dir(path)); ok {
			parent := readBuiltinSetting(extended, seen)
			if setting.setget == nil {
				setting.setget = parent.setget
			}
			if setting.prefix == nil {
				setting.prefix = parent.prefix
			}
			if setting.err == "" {
				setting.err = parent.err
			}
		}
	}
	return setting
}

// Decide works out what a project's setget setting and plugins call for, from the tsconfig at tsConfigPath and what it
// extends, and the command line. Setget is on if the built-in option says so or a plugin entry names it. No package
// needs to be installed for either. JS would warn about a missing plugin and then reject every accessor.
func Decide(tsConfigPath string, options Options) Decision {
	setting := readBuiltinSetting(tsConfigPath, map[string]bool{})
	if setting.err != "" {
		return Decision{Action: ActionError, Reason: setting.err}
	}
	builtin := setting.setget != nil && *setting.setget
	if options.Setget != nil {
		builtin = *options.Setget
	}

	configs, active, err := getPluginConfigs(tsConfigPath, map[string]bool{})
	entriesActive := active != nil && *active
	if err != nil && entriesActive {
		return Decision{Action: ActionFallback, Reason: "rbxtsgo can't read the tsconfig's plugins: " + err.Error()}
	}
	if !entriesActive {
		configs = nil
	}
	if !builtin && len(configs) == 0 {
		return Decision{Action: ActionNone}
	}

	var others []string
	var setget []pluginConfig
	for _, config := range configs {
		if config.transform == SetgetPlugin {
			setget = append(setget, config)
		} else {
			others = append(others, `"`+config.transform+`"`)
		}
	}
	if len(others) > 0 {
		noun := "plugin"
		if len(others) > 1 {
			noun = "plugins"
		}
		if builtin {
			return Decision{Action: ActionError, Reason: fmt.Sprintf("The built-in setget option can't be used with the transformer %s %s, which only the JS compiler can run, because the JS compiler doesn't have the option.\nTo build this project, turn the option off, and add \"%s\" to compilerOptions.plugins with the package installed.", noun, strings.Join(others, ", "), SetgetPlugin)}
		}
		return Decision{Action: ActionFallback, Reason: fmt.Sprintf("rbxtsgo can't run the transformer %s %s", noun, strings.Join(others, ", "))}
	}
	fallback := func(why string) Decision {
		return Decision{Action: ActionFallback, Reason: fmt.Sprintf(`rbxtsgo can't run the transformer plugin "%s" %s`, SetgetPlugin, why)}
	}

	prefix := defaultPrefix
	if len(setget) > 1 && !builtin {
		return fallback("more than once, and it's listed several times here")
	}
	if len(setget) > 0 && !builtin {
		config := setget[0]
		if value, ok := config.values.Get("type"); ok && value != nil && value != "program" {
			return fallback(fmt.Sprintf(`with "type": %v`, value))
		}
		if value, ok := config.values.Get("import"); ok && value != nil && value != "default" {
			return fallback(fmt.Sprintf(`with "import": %v`, value))
		}
		// With a package at another version, what the project asks for is that version, so only JS can give it.
		if manifest, found := findPluginManifest(filepath.Dir(tsConfigPath)); found {
			var pkg struct {
				Version string `json:"version"`
			}
			if text, err := os.ReadFile(manifest); err == nil {
				_ = json.Unmarshal(text, &pkg)
			}
			if pkg.Version != SetgetVersion {
				return fallback(fmt.Sprintf("at version %s, because the port matches %s", pkg.Version, SetgetVersion))
			}
		}
	}
	if len(setget) > 0 {
		if value, ok := setget[0].values.Get("customPrefix"); ok && value != nil {
			text, isString := value.(string)
			if !isString {
				if builtin {
					return Decision{Action: ActionError, Reason: fmt.Sprintf(`"customPrefix" of the plugin "%s" must be a string`, SetgetPlugin)}
				}
				return fallback(fmt.Sprintf(`with "customPrefix": %v`, value))
			}
			prefix = text
		}
	}
	if setting.prefix != nil {
		prefix = *setting.prefix
	}
	if options.Prefix != nil {
		prefix = *options.Prefix
	}
	return Decision{Action: ActionSetget, Prefix: prefix}
}

// findPluginManifest looks where resolve.sync(transform, { basedir }) does.
func findPluginManifest(directory string) (string, bool) {
	for current := directory; ; {
		manifest := filepath.Join(current, "node_modules", SetgetPlugin, "package.json")
		if exists(manifest) {
			return manifest, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}
