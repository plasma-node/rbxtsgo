// Command rbxtsgo is the port of src/CLI from roblox-ts 3.0.0: cli.ts and commands/build.ts. It builds a roblox-ts
// project to Luau. Transformer plugins are a later phase.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/compile"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/plugins"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/watch"
)

var path = nodepath.Native

// tristate tells a flag that was given from one that wasn't, because the tsconfig's "rbxts" section and
// DEFAULT_PROJECT_OPTIONS come in underneath the command line.
type tristate struct{ value, set bool }

func (t *tristate) String() string   { return fmt.Sprint(t.value) }
func (t *tristate) IsBoolFlag() bool { return true }
func (t *tristate) Set(s string) error {
	t.set = true
	t.value = s == "true" || s == "1" || s == ""
	return nil
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	commandLine = args
	flags := flag.NewFlagSet("rbxtsgo", flag.ContinueOnError)
	projectPath := flags.String("project", ".", "project path")
	flags.StringVar(projectPath, "p", ".", "project path")
	var watch, usePolling, verbose, noInclude, logTruthyChanges, writeOnlyChanged, writeTransformedFiles, optimizedLoops, allowCommentDirectives, luauFlag tristate
	flags.Var(&watch, "watch", "enable watch mode")
	flags.Var(&watch, "w", "enable watch mode")
	flags.Var(&usePolling, "usePolling", "use polling for watch mode")
	flags.Var(&verbose, "verbose", "enable verbose logs")
	flags.Var(&noInclude, "noInclude", "do not copy include files")
	flags.Var(&logTruthyChanges, "logTruthyChanges", "logs changes to truthiness evaluation from Lua truthiness rules")
	flags.Var(&writeOnlyChanged, "writeOnlyChanged", "")
	flags.Var(&writeTransformedFiles, "writeTransformedFiles", "writes resulting TypeScript ASTs after transformers to out directory")
	flags.Var(&optimizedLoops, "optimizedLoops", "")
	flags.Var(&allowCommentDirectives, "allowCommentDirectives", "")
	flags.Var(&luauFlag, "luau", "emit files with .luau extension")
	projectType := flags.String("type", "", "override project type: game, model or package")
	includePath := flags.String("includePath", "", "folder to copy runtime files to")
	flags.StringVar(includePath, "i", "", "folder to copy runtime files to")
	rojo := flags.String("rojo", "", "manually select Rojo project file")
	checkers := flags.Int("checkers", 0, "number of type checkers and transform workers (default 4)")
	cpuProfile := flags.String("cpuprofile", "", "write a CPU profile to this file")
	var fixEmit tristate
	flags.Var(&fixEmit, "fix-emit", "turn on fixes for output that is wrong in roblox-ts 3.0.0, which makes the output differ from it where it was wrong. Also tsconfig \"rbxtsgo\": { \"fixEmit\": true }")
	var setget tristate
	flags.Var(&setget, "setget", "turn getters and setters into methods, as rbxts-transformer-setget does")
	setgetPrefix := flags.String("setgetPrefix", "", "prefix of the methods setget makes (default __)")
	if err := flags.Parse(rewriteArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if setget.set {
		pluginOptions.Setget = &setget.value
	}
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "setgetPrefix" {
			pluginOptions.Prefix = setgetPrefix
		}
	})
	if flags.NArg() > 0 && !(flags.NArg() == 1 && flags.Arg(0) == "build") {
		fmt.Fprintln(os.Stderr, "unknown arguments:", strings.Join(flags.Args(), " "))
		return 1
	}
	if *cpuProfile != "" {
		file, err := os.Create(*cpuProfile)
		if err == nil {
			defer file.Close()
			if pprof.StartCPUProfile(file) == nil {
				defer pprof.StopCPUProfile()
			}
		}
	}

	err := build(*projectPath, func(options *shared.ProjectOptions) {
		setBool := func(flag tristate, target *bool) {
			if flag.set {
				*target = flag.value
			}
		}
		setBool(watch, &options.Watch)
		setBool(usePolling, &options.UsePolling)
		setBool(verbose, &options.Verbose)
		setBool(noInclude, &options.NoInclude)
		setBool(logTruthyChanges, &options.LogTruthyChanges)
		setBool(writeOnlyChanged, &options.WriteOnlyChanged)
		setBool(writeTransformedFiles, &options.WriteTransformedFiles)
		setBool(optimizedLoops, &options.OptimizedLoops)
		setBool(allowCommentDirectives, &options.AllowCommentDirectives)
		setBool(luauFlag, &options.Luau)
		setBool(fixEmit, &options.FixEmit)
		if *projectType != "" {
			options.Type = shared.ProjectType(*projectType)
		}
		if *includePath != "" {
			options.IncludePath = *includePath
		}
		if *rojo != "" {
			options.Rojo = *rojo
		}
	}, *checkers)
	if err != nil {
		var diagnosticErr *shared.DiagnosticError
		if errors.As(err, &diagnosticErr) {
			printDiagnostics(diagnosticErr.Diagnostics)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}
	if exitCode != 0 {
		return exitCode
	}
	return 0
}

var exitCode int

// commandLine is what the JS compiler gets when a plugin hands it the build.
var commandLine []string

// pluginOptions is the setget setting from the command line.
var pluginOptions plugins.Options

// rewriteArgs makes yargs' spellings work with the flag package: --no-x means --x=false.
func rewriteArgs(args []string) []string {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		if rest, ok := strings.CutPrefix(arg, "--no-"); ok {
			result = append(result, "--"+rest+"=false")
			continue
		}
		result = append(result, arg)
	}
	return result
}

// findTsConfigPath is findTsConfigPath.
func findTsConfigPath(projectPath string) (string, error) {
	tsConfigPath := path.Resolve(projectPath)
	if info, err := os.Stat(tsConfigPath); err != nil || info.IsDir() {
		// ts.findConfigFile walks up the ancestors.
		found := ""
		for directory := tsConfigPath; ; {
			candidate := path.Join(directory, "tsconfig.json")
			if _, err := os.Stat(candidate); err == nil {
				found = candidate
				break
			}
			parent := path.Dirname(directory)
			if parent == directory {
				break
			}
			directory = parent
		}
		if found == "" {
			return "", errors.New("Unable to find tsconfig.json!")
		}
		tsConfigPath = found
	}
	return path.Resolve(tsConfigPath), nil
}

// applyTsConfigOptions is getTsConfigProjectOptions: the "rbxts" section of tsconfig.json.
func applyTsConfigOptions(tsConfigPath string, options *shared.ProjectOptions) {
	// The "rbxtsgo" block is ours, and is read with its other settings. build and Load report a bad value.
	options.FixEmit, _ = plugins.FixEmitSetting(tsConfigPath)
	text, err := os.ReadFile(tsConfigPath)
	if err != nil {
		return
	}
	fileName := tspath.RootedFilePathFromAbsolute(filepath.ToSlash(tsConfigPath))
	raw, _ := tsoptions.ParseConfigFileTextToJson(fileName, osvfs.FS().CaseSensitivity().PathKey(fileName.AsPath()), string(text))
	section, ok := raw.(interface{ Get(string) (any, bool) })
	if !ok {
		return
	}
	rbxts, ok := section.Get("rbxts")
	if !ok {
		return
	}
	values, ok := rbxts.(interface{ Get(string) (any, bool) })
	if !ok {
		return
	}
	setBool := func(name string, target *bool) {
		if value, ok := values.Get(name); ok {
			if b, ok := value.(bool); ok {
				*target = b
			}
		}
	}
	setString := func(name string, target *string) {
		if value, ok := values.Get(name); ok {
			if s, ok := value.(string); ok {
				*target = s
			}
		}
	}
	setBool("logTruthyChanges", &options.LogTruthyChanges)
	setBool("noInclude", &options.NoInclude)
	setBool("usePolling", &options.UsePolling)
	setBool("verbose", &options.Verbose)
	setBool("watch", &options.Watch)
	setBool("writeOnlyChanged", &options.WriteOnlyChanged)
	setBool("writeTransformedFiles", &options.WriteTransformedFiles)
	setBool("optimizedLoops", &options.OptimizedLoops)
	setBool("allowCommentDirectives", &options.AllowCommentDirectives)
	setBool("luau", &options.Luau)
	setString("includePath", &options.IncludePath)
	setString("rojo", &options.Rojo)
	var projectType string
	setString("type", &projectType)
	if projectType != "" {
		options.Type = shared.ProjectType(projectType)
	}
}

// build is the handler of the build command.
func build(projectPath string, applyFlags func(*shared.ProjectOptions), checkers int) error {
	tsConfigPath, err := findTsConfigPath(projectPath)
	if err != nil {
		return err
	}

	options := shared.DefaultProjectOptions()
	applyTsConfigOptions(tsConfigPath, &options)
	applyFlags(&options)
	shared.LogService.Verbose = options.Verbose
	decision := plugins.Decide(tsConfigPath, pluginOptions)
	if decision.Action == plugins.ActionError {
		return errors.New(decision.Reason)
	}
	if _, err := plugins.FixEmitSetting(tsConfigPath); err != nil {
		return err
	}
	if decision.Action == plugins.ActionFallback {
		// JS 3.0.0 doesn't have these fixes, so handing it the build would silently drop what was asked for.
		if options.FixEmit {
			return errors.New(decision.Reason + ", and the JS compiler doesn't have --fix-emit.\nTo build this project, turn fixEmit off, or remove the plugin from compilerOptions.plugins.")
		}
		code, err := plugins.Delegate(decision.Reason, tsConfigPath, commandLine, os.Stderr)
		exitCode = code
		return err
	}
	if options.Watch {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return watch.Run(ctx, watch.Config{
			Load: func() (*shared.ProjectData, error) {
				// The "rbxts" section can change between loads, so this starts from the defaults again.
				options := shared.DefaultProjectOptions()
				applyTsConfigOptions(tsConfigPath, &options)
				applyFlags(&options)
				if _, err := plugins.FixEmitSetting(tsConfigPath); err != nil {
					return nil, err
				}
				return project.CreateProjectData(tsConfigPath, options)
			},
			Checkers:      checkers,
			UsePolling:    options.UsePolling,
			FullRebuilds:  os.Getenv("RBXTSGO_WATCH_FULL") == "1",
			PluginOptions: pluginOptions,
		})
	}

	data, err := project.CreateProjectData(tsConfigPath, options)
	if err != nil {
		return err
	}
	parsed, err := project.GetParsedCommandLine(data)
	if err != nil {
		return err
	}
	var times compile.Times
	started := time.Now()
	phase := started
	lap := func(target *time.Duration) {
		*target = time.Since(phase)
		phase = time.Now()
	}
	p := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{Checkers: checkers})
	translator := project.CreatePathTranslator(p, data)
	lap(&times.Program)
	defer func() { shared.LogService.WriteLineIfVerbose(formatBuildTimes(&times, time.Since(started))) }()
	if err := project.Cleanup(translator); err != nil {
		return err
	}
	if err := project.CopyInclude(data); err != nil {
		return err
	}
	// upstream makes a Set of these, which keeps first-seen order
	var rootDirs []string
	for _, rootDir := range project.GetRootDirs(p.Options()) {
		if !slices.Contains(rootDirs, rootDir) {
			rootDirs = append(rootDirs, rootDir)
		}
	}
	if err := project.CopyFiles(data, translator, rootDirs); err != nil {
		return err
	}
	lap(&times.Copy)
	// emitBuildInfo and createProgramFactory's hash. With nothing changed, the rest of compileFiles has no work to do.
	var inc *compile.Incremental
	if translator.BuildInfoOutputPath != "" {
		inc = compile.LoadIncremental(data, translator, compile.BuildKey(data, p, fmt.Sprint(decision.Action, decision.Prefix)))
	}
	unchanged := inc.Unchanged(p)
	lap(&times.State)
	if unchanged {
		fmt.Printf("No changes, finished in %s%s", shared.FormatElapsed(time.Since(started)), shared.EOL)
		return nil
	}
	prepared, failure, err := project.PrepareCompile(p, data, translator, p.Program().CaseSensitivity())
	if err != nil {
		return err
	}
	if failure != nil {
		printDiagnostics(failure)
		if shared.HasErrors(failure) {
			exitCode = 1
		}
		return nil
	}
	lap(&times.Prepare)
	applied, err := plugins.Apply(context.Background(), p, decision, p.SourceFilesToCompile())
	lap(&times.Plugin)
	if err != nil {
		return err
	}
	if applied.Exit != nil {
		fmt.Println(strings.Join(applied.Exit, "\n"))
		exitCode = 1
		return nil
	}
	if err := applied.WriteTransformedFiles(data, translator); err != nil {
		return err
	}
	result, err := compile.CompileIncremental(context.Background(), &compile.Context{
		Project:    applied.Project,
		Data:       data,
		Translator: translator,
		Prepared:   prepared,
		Replay:     shared.NewContextReplay(),
		Times:      &times,
	}, inc, p, applied.Files)
	if err != nil {
		return err
	}
	diagnostics := applied.MapDiagnostics(result.Diagnostics)
	printDiagnostics(diagnostics)
	if shared.HasErrors(diagnostics) {
		exitCode = 1
		return nil
	}
	fmt.Printf("Built in %s%s", shared.FormatElapsed(time.Since(started)), shared.EOL)
	return nil
}

// formatBuildTimes is the build's phase timings, for --verbose.
func formatBuildTimes(t *compile.Times, total time.Duration) string {
	ms := func(d time.Duration) string { return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000) }
	return fmt.Sprintf("[timing] build total %s: program %s, copy %s, state %s, prepare %s, plugin %s, check %s, lower %s, write %s, declarations %s; checked %d, lowered %d",
		ms(total), ms(t.Program), ms(t.Copy), ms(t.State), ms(t.Prepare), ms(t.Plugin), ms(t.Check), ms(t.Lower), ms(t.Write), ms(t.Declarations), t.FilesChecked, t.FilesLowered)
}

// printDiagnostics is ts.createDiagnosticReporter(ts.sys, true): pretty output on stdout.
func printDiagnostics(list []*shared.Diagnostic) {
	if len(list) == 0 {
		return
	}
	cwd := shared.ProcessCwd()
	caseSensitivity := osvfs.FS().CaseSensitivity()
	for _, d := range list {
		fmt.Print(shared.FormatDiagnostics([]*shared.Diagnostic{d}, cwd, caseSensitivity, "\n"))
		fmt.Print("\n")
	}
}
