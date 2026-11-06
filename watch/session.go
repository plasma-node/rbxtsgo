package watch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
	"github.com/microsoft/TypeScript/tsc/internal/execute/watchmanager"
	"github.com/microsoft/TypeScript/tsc/internal/fswatch"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/compile"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/plugins"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/pathtranslator"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

var path = nodepath.Native

// Cycle describes one compile, for tests and for --verbose.
type Cycle struct {
	Result *compile.EmitResult
	// Initial says the cycle ran upstream's initial path: cleanup, include, copying everything.
	Initial bool
	// Reused says Program.ReuseProgram worked.
	Reused bool
	Times  compile.Times
	// Debounce is the time between noticing a change and starting.
	Debounce time.Duration
	Total    time.Duration
	Errors   int
	Written  int
	// Report is free for tests: nothing here sets it.
	Report string
}

// orderedSet is JS's Set<string>, which keeps first-seen order.
type orderedSet struct {
	seen  map[string]bool
	items []string
}

func (o *orderedSet) add(item string) {
	if o.seen == nil {
		o.seen = map[string]bool{}
	}
	if !o.seen[item] {
		o.seen[item] = true
		o.items = append(o.items, item)
	}
}

func (o *orderedSet) clear() { *o = orderedSet{} }

// session is the state that setupProjectWatchProgram keeps in its closure.
type session struct {
	config  Config
	manager *watchmanager.WatchManager

	decision plugins.Decision
	inc      *compile.Incremental
	plugin   *plugins.Session
	prepare  *project.PrepareCache
	// lastCompiled is the program clean was worked out against.
	lastCompiled *tsbridge.Project
	// clean holds the files whose output on disk is what a clean build of the current sources would write, and that
	// gave no diagnostics at all. Those need no checking or lowering.
	clean map[tspath.PathKey]bool

	data       *shared.ProjectData
	parsed     *tsbridge.ParsedConfig
	proj       *tsbridge.Project
	translator *pathtranslator.Translator
	cache      *tsbridge.SourceFileCache
	replay     *shared.ContextReplay

	initialCompileCompleted bool
	// reload says a config file changed, so everything comes from disk again on the next compile.
	reload bool
	// overflow says events were lost, or a package changed: read everything again, and treat the build as a first one.
	overflow bool

	// Events collected but not yet applied. As upstream's filesToAdd, filesToChange and filesToDelete.
	filesToAdd, filesToChange, filesToDelete orderedSet
	// Work that only gets done after a successful compile, so it carries over a failed one, as upstream's sets do.
	filesToCopy, filesToClean orderedSet
	// knownDirs tells a new folder from a folder whose contents changed, which is all that an event on it says.
	knownDirs map[string]bool
	warnings  []*shared.Diagnostic

	last    Cycle
	writes0 int
}

func newSession(config Config, manager *watchmanager.WatchManager) (*session, error) {
	s := &session{config: config, manager: manager, cache: tsbridge.NewSourceFileCache(), replay: shared.NewContextReplay()}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// load reads the project's data and tsconfig.
func (s *session) load() error {
	data, err := s.config.Load()
	if err != nil {
		return err
	}
	// 0003 item 6: a rebuild writes only the bytes that changed.
	data.ProjectOptions.WriteOnlyChanged = true
	parsed, err := project.GetParsedCommandLine(data)
	if err != nil {
		return err
	}
	decision := plugins.Decide(data.TsConfigPath, s.config.PluginOptions)
	if decision.Action == plugins.ActionError {
		return errors.New(decision.Reason)
	}
	if decision.Action == plugins.ActionFallback {
		// Only the JS compiler can do this, and it can't take over a running watcher. At the start, the CLI hands it over.
		return errors.New(decision.Reason + ". Restart to run this project with the JS compiler.")
	}
	s.data = data
	s.parsed = parsed
	s.decision = decision
	s.reset()
	return nil
}

// reset forgets everything that was kept between compiles.
func (s *session) reset() {
	s.proj = nil
	s.cache.Clear()
	s.plugin = plugins.NewSession()
	s.prepare = &project.PrepareCache{}
	s.lastCompiled = nil
	s.clean = map[tspath.PathKey]bool{}
	s.inc = nil
}

// rootDirs is getRootDirs.
func (s *session) rootDirs() []string {
	return project.GetRootDirs(s.parsed.Config.CompilerOptions())
}

// reconcileWatches points the watches at the root folders, as upstream does, and at what upstream doesn't watch: the
// config files and the installed packages. With those watched, what's on disk is what a clean build would see.
func (s *session) reconcileWatches() error {
	desired := map[tspath.RootedDirectoryPath]bool{}
	add := func(dir string, recursive bool) {
		rooted := tspath.RootedDirectoryPathFromAbsolute(tspath.NormalizePath(toSlash(dir)))
		desired[rooted] = desired[rooted] || recursive
	}
	for _, dir := range s.rootDirs() {
		add(dir, true)
	}
	for _, file := range s.configFiles() {
		add(path.Dirname(file), false)
	}
	add(path.Dirname(s.data.TsConfigPath), false)
	for _, typeRoot := range s.parsed.Config.CompilerOptions().TypeRoots {
		add(string(typeRoot), true)
	}
	s.knownDirs = map[string]bool{}
	for _, dir := range s.rootDirs() {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}
		s.knownDirs[toSlash(dir)] = true
		_ = project.WalkDirectory(dir, func(item string) error {
			if info, err := os.Stat(item); err == nil && info.IsDir() {
				s.knownDirs[toSlash(item)] = true
			}
			return nil
		})
	}
	return s.manager.ReconcileWatches(s.manager.ResolveDesiredDirs(desired))
}

func toSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }

// configFiles lists the files that make a change a full reload.
func (s *session) configFiles() []string {
	files := []string{s.data.TsConfigPath}
	for _, extended := range s.parsed.Config.ExtendedSourceFiles() {
		files = append(files, extended.AsString())
	}
	if s.data.RojoConfigPath != "" {
		files = append(files, s.data.RojoConfigPath)
	}
	files = append(files, path.Join(path.Dirname(s.data.NodeModulesPath), "package.json"))
	return files
}

func (s *session) isConfigFile(file string) bool {
	for _, config := range s.configFiles() {
		if strings.EqualFold(toSlash(config), file) {
			return true
		}
	}
	// A Rojo project can appear next to the tsconfig, and one inside a root folder can be nested in a $path.
	return strings.HasSuffix(file, ".project.json") && (strings.EqualFold(toSlash(path.Dirname(s.data.TsConfigPath)), toSlash(path.Dirname(file))) || s.underRootDirs(file))
}

func isUnder(dir, file string) bool {
	dir = strings.TrimSuffix(toSlash(dir), "/")
	return len(file) > len(dir) && file[len(dir)] == '/' && strings.EqualFold(file[:len(dir)], dir)
}

func (s *session) underRootDirs(file string) bool {
	return slices.ContainsFunc(s.rootDirs(), func(dir string) bool { return isUnder(dir, file) })
}

// collect is collectAddEvent, collectChangeEvent and collectDeleteEvent. Events only say "something happened here", so
// a look at the disk decides which of those it was. What's on disk is what the compile will read, so a file that came
// and went between two events is no problem.
func (s *session) collect(changed map[tspath.RootedPath]fswatch.EventKind, overflow bool) {
	if overflow {
		s.overflow = true
	}
	for raw := range changed {
		file := eventPath(raw)
		switch {
		case s.isConfigFile(file):
			s.reload = true
		case slices.ContainsFunc(s.parsed.Config.CompilerOptions().TypeRoots, func(root tspath.RootedDirectoryPath) bool { return isUnder(string(root), file) }):
			s.overflow = true
		case !s.underRootDirs(file) || isUnder(s.parsed.Config.CompilerOptions().OutDir.AsString(), file) || isUnder(s.data.ProjectOptions.IncludePath, file) || isUnder(s.data.NodeModulesPath, file):
			// Only the root folders count, and what the compiler writes itself mustn't trigger it again.
		default:
			info, err := os.Stat(file)
			switch {
			case err != nil:
				delete(s.knownDirs, file)
				for dir := range s.knownDirs {
					if isUnder(file, dir) {
						delete(s.knownDirs, dir)
					}
				}
				s.filesToDelete.add(file)
			case info.IsDir():
				if !s.knownDirs[file] {
					s.knownDirs[file] = true
					s.filesToAdd.add(file)
				}
			default:
				s.filesToChange.add(file)
			}
		}
	}
}

func (s *session) hasPending() bool {
	return s.reload || s.overflow || len(s.filesToAdd.items)+len(s.filesToChange.items)+len(s.filesToDelete.items) > 0
}

// runCompile is runCompile. Upstream rethrows anything but a DiagnosticError, which ends the process. A watcher that
// dies on an internal error loses more than it saves, so here it's reported like any other error.
func (s *session) runCompile() (result *compile.EmitResult) {
	s.last = Cycle{}
	defer func() {
		if r := recover(); r != nil {
			message := fmt.Sprintf("internal error: %v", r)
			if shared.LogService.Verbose {
				message += "\n" + string(debug.Stack())
			}
			// Nothing below can be trusted after this. Start over on the next change.
			s.reset()
			s.initialCompileCompleted = false
			result = failure(message)
		}
	}()
	var err error
	if s.reload {
		if err = s.load(); err == nil {
			s.reload = false
			s.initialCompileCompleted = false
			s.overflow = false
			err = s.reconcileWatches()
		}
	}
	if err == nil {
		if s.overflow {
			s.overflow = false
			s.reset()
			s.initialCompileCompleted = false
			if s.parsed, err = project.GetParsedCommandLine(s.data); err == nil {
				err = s.reconcileWatches()
			}
		}
	}
	if err == nil {
		if !s.initialCompileCompleted {
			result, err = s.runInitialCompile()
		} else {
			result, err = s.runIncrementalCompile()
		}
	}
	if err != nil {
		var diagnosticErr *shared.DiagnosticError
		if errors.As(err, &diagnosticErr) {
			return &compile.EmitResult{EmitSkipped: true, Diagnostics: diagnosticErr.Diagnostics}
		}
		return failure(err.Error())
	}
	return result
}

func failure(message string) *compile.EmitResult {
	return &compile.EmitResult{EmitSkipped: true, Diagnostics: []*shared.Diagnostic{shared.TextDiagnostic(message, diagnostics.CategoryError)}}
}

// refreshProgram is refreshProgram: the program and path translator for this compile.
func (s *session) refreshProgram(changed []tspath.PathKey, structure bool) {
	started := time.Now()
	if s.proj == nil {
		s.proj = tsbridge.NewProject(s.parsed, tsbridge.ProjectOptions{Checkers: s.config.Checkers, Cache: s.cache})
	} else {
		s.proj, s.last.Reused = s.proj.Update(changed, structure)
	}
	s.translator = project.CreatePathTranslator(s.proj, s.data)
	s.last.Times.Program = time.Since(started)
}

func (s *session) pathKey(file string) tspath.PathKey {
	return osvfs.FS().CaseSensitivity().PathKey(tspath.RootedPathFromAbsolute(file))
}

// drainChanges empties the collected events and returns the paths whose parsed files can't be trusted any more. Those
// need evicting even when the program is about to be rebuilt anyway.
func (s *session) drainEvents() (additions, changes, removals []string) {
	additions, changes, removals = s.filesToAdd.items, s.filesToChange.items, s.filesToDelete.items
	s.filesToAdd.clear()
	s.filesToChange.clear()
	s.filesToDelete.clear()
	return additions, changes, removals
}

// runInitialCompile is runInitialCompile.
func (s *session) runInitialCompile() (*compile.EmitResult, error) {
	s.last.Initial = true
	additions, changes, removals := s.drainEvents()
	var keys []tspath.PathKey
	for _, list := range [][]string{additions, changes, removals} {
		for _, file := range list {
			keys = append(keys, s.pathKey(file))
		}
	}
	// Upstream makes a new program here. So does this, keeping only the parsed files nothing has touched.
	s.refreshProgram(keys, true)
	// This did everything those would have done.
	s.filesToCopy.clear()
	s.filesToClean.clear()

	started := time.Now()
	if err := project.Cleanup(s.translator); err != nil {
		return nil, err
	}
	if err := project.CopyInclude(s.data); err != nil {
		return nil, err
	}
	var rootDirs []string
	for _, rootDir := range project.GetRootDirs(s.proj.Options()) {
		if !slices.Contains(rootDirs, rootDir) {
			rootDirs = append(rootDirs, rootDir)
		}
	}
	if err := project.CopyFiles(s.data, s.translator, rootDirs); err != nil {
		return nil, err
	}
	s.last.Times.Copy = time.Since(started)

	result, err := s.compile()
	if err != nil {
		return nil, err
	}
	if !result.EmitSkipped {
		s.initialCompileCompleted = true
	}
	return result, nil
}

// runIncrementalCompile is runIncrementalCompile.
func (s *session) runIncrementalCompile() (*compile.EmitResult, error) {
	additions, changes, removals := s.drainEvents()
	inProgram := s.proj.Program().FilesByPath()
	structure := false
	var keys []tspath.PathKey
	service := &shared.DiagnosticService{}

	for _, fsPath := range additions {
		// Only folders get here: a file's event can't say whether it's new, so those are all in changes.
		// Files in the new folder may not have events of their own.
		structure = true
		s.filesToCopy.add(fsPath)
	}
	for _, fsPath := range changes {
		key := s.pathKey(fsPath)
		_, known := inProgram[key]
		compilable, err := project.IsCompilableFile(fsPath)
		if err != nil {
			// It went away again. The event for that decides what happens next.
			continue
		}
		switch {
		case compilable && known:
			keys = append(keys, key)
		case compilable:
			structure = true
			keys = append(keys, key)
		default:
			// checks for copying `init.*.d.ts`
			if !known {
				project.CheckFileName(service, fsPath)
			}
			// A new .d.ts or .json file can be a root, or can answer an import that was failing.
			if !known && (strings.HasSuffix(fsPath, shared.DTSExt) || strings.HasSuffix(fsPath, ".json")) {
				structure = true
			}
			if known {
				keys = append(keys, key)
			}
			s.filesToCopy.add(fsPath)
		}
	}
	for _, fsPath := range removals {
		key := s.pathKey(fsPath)
		if _, known := inProgram[key]; known || isPossibleProgramPath(fsPath) {
			structure = true
			keys = append(keys, key)
		}
		s.filesToClean.add(fsPath)
	}

	s.refreshProgram(keys, structure)
	result, err := s.compile()
	if err != nil {
		return nil, err
	}
	result.Diagnostics = append(service.Flush(), result.Diagnostics...)
	if result.EmitSkipped || shared.HasErrors(result.Diagnostics) {
		// exit before copying to prevent half-updated out directory
		result.EmitSkipped = true
		return result, nil
	}

	started := time.Now()
	for _, fsPath := range s.filesToClean.items {
		if err := project.TryRemoveOutput(s.translator, s.translator.GetOutputPath(fsPath)); err != nil {
			return nil, err
		}
		if s.proj.Options().Declaration.IsTrue() {
			if err := project.TryRemoveOutput(s.translator, s.translator.GetOutputDeclarationPath(fsPath)); err != nil {
				return nil, err
			}
		}
	}
	for _, fsPath := range s.filesToCopy.items {
		// A folder or file that has gone again has nothing to copy.
		if _, err := os.Stat(fsPath); err != nil {
			continue
		}
		if err := project.CopyItem(s.data, s.translator, fsPath); err != nil {
			return nil, err
		}
	}
	s.last.Times.Copy = time.Since(started)
	s.filesToCopy.clear()
	s.filesToClean.clear()
	return result, nil
}

// isPossibleProgramPath says whether removing this path could change what the program sees. A folder can hold any of
// these, and by the time the event comes, nothing says it was a folder.
func isPossibleProgramPath(fsPath string) bool {
	base := path.Basename(fsPath, "")
	return strings.HasSuffix(base, ".ts") || strings.HasSuffix(base, ".tsx") || strings.HasSuffix(base, ".json") || !strings.Contains(base, ".")
}

// compile is the part of compileFiles this package shares with a build.
func (s *session) compile() (*compile.EmitResult, error) {
	if s.config.BeforeCompile != nil {
		s.config.BeforeCompile()
	}
	started := time.Now()
	prepareCache := s.prepare
	if s.config.FullRebuilds {
		prepareCache = nil
	}
	prepared, failureList, err := project.PrepareCompileCached(s.proj, s.data, s.translator, s.proj.Program().CaseSensitivity(), prepareCache)
	s.last.Times.Prepare = time.Since(started)
	if err != nil {
		return nil, err
	}
	if failureList != nil {
		return &compile.EmitResult{EmitSkipped: shared.HasErrors(failureList), Diagnostics: failureList}, nil
	}

	ctx := context.Background()
	started = time.Now()
	applied, err := s.plugin.Apply(ctx, s.proj, s.decision, s.proj.SourceFilesToCompile(), !s.config.FullRebuilds)
	s.last.Times.Plugin = time.Since(started)
	s.last.Times.FilesTransformed = s.plugin.LastRedone
	if err != nil {
		return nil, err
	}
	if applied.Exit != nil {
		// The plugin ends the process in JS, with these lines on stdout. A watcher reports them and waits for a fix.
		return &compile.EmitResult{EmitSkipped: true, Diagnostics: []*shared.Diagnostic{shared.TextDiagnostic(strings.Join(applied.Exit, "\n"), diagnostics.CategoryError)}}, nil
	}
	if err := applied.WriteTransformedFiles(s.data, s.translator); err != nil {
		return nil, err
	}

	c := &compile.Context{
		Project:    applied.Project,
		Data:       s.data,
		Translator: s.translator,
		Prepared:   prepared,
		Replay:     s.replay,
		Times:      &s.last.Times,
	}
	result, err := s.compileDirty(ctx, c, applied.Files)
	if err != nil {
		return nil, err
	}
	result.Diagnostics = applied.MapDiagnostics(result.Diagnostics)
	if result.EmitSkipped || shared.HasErrors(result.Diagnostics) {
		result.EmitSkipped = true
	}
	return result, nil
}

// compileDirty compiles the files whose output can differ from what's on disk. See 0008 for why the others can be left.
func (s *session) compileDirty(ctx context.Context, c *compile.Context, files []*ast.SourceFile) (*compile.EmitResult, error) {
	change := tsbridge.AffectedFiles(s.lastCompiled, c.Project)
	first := s.lastCompiled == nil
	s.lastCompiled = c.Project
	if s.config.FullRebuilds || change.Full {
		s.clean = map[tspath.PathKey]bool{}
	} else {
		for path := range change.Affected {
			delete(s.clean, path)
		}
	}
	if first && s.translator.BuildInfoOutputPath != "" && !s.config.FullRebuilds {
		// compile.Incremental's records say what's on disk from an earlier run.
		s.inc = compile.LoadIncremental(s.data, s.translator, compile.BuildKey(s.data, s.source(), fmt.Sprint(s.decision.Action, s.decision.Prefix)))
		s.inc.Unchanged(s.source())
		fromDisk := s.inc.Dirty(c.Project, files)
		for _, file := range files {
			s.clean[file.PathKey()] = true
		}
		for _, file := range fromDisk {
			delete(s.clean, file.PathKey())
		}
	}
	dirty := make([]*ast.SourceFile, 0, len(files))
	for _, file := range files {
		if !s.clean[file.PathKey()] {
			dirty = append(dirty, file)
		}
	}
	save := func(result *compile.EmitResult, compiled []*ast.SourceFile) error {
		if s.inc == nil || result.EmitSkipped || shared.HasErrors(result.Diagnostics) {
			return nil
		}
		return s.inc.Save(s.source(), c.Project, compiled, result)
	}
	if len(dirty) == 0 {
		result := &compile.EmitResult{}
		return result, save(result, nil)
	}
	result, err := compile.CompileFiles(ctx, c, dirty)
	if err != nil {
		return nil, err
	}
	// A diagnostic with no file comes from the whole program, and rbxtsc repeats it per file. Those can also grow with
	// what got checked. So a project that has one gets no shortcuts.
	whole := slices.ContainsFunc(result.Diagnostics, func(d *shared.Diagnostic) bool { return d.File() == nil })
	if whole {
		s.clean = map[tspath.PathKey]bool{}
		if len(dirty) < len(files) {
			if result, err = compile.CompileFiles(ctx, c, files); err != nil {
				return nil, err
			}
		}
		return result, save(result, files)
	}
	if result.EmitSkipped {
		return result, nil
	}
	reported := map[*ast.SourceFile]bool{}
	for _, d := range result.Diagnostics {
		reported[d.File()] = true
	}
	for _, file := range dirty {
		if !reported[file] {
			s.clean[file.PathKey()] = true
		}
	}
	return result, save(result, dirty)
}

// source is the program before any plugin.
func (s *session) source() *tsbridge.Project { return s.proj }

// finishCycle reports timings and runs the hook.
func (s *session) finishCycle(started time.Time, debounce time.Duration, result *compile.EmitResult) {
	cycle := s.last
	cycle.Result = result
	cycle.Debounce = debounce
	cycle.Total = time.Since(started)
	for _, d := range result.Diagnostics {
		if d.Category() == diagnostics.CategoryError {
			cycle.Errors++
		}
	}
	cycle.Written = len(result.EmittedFiles)
	if shared.LogService.Verbose {
		shared.LogService.WriteLine(formatTimes(&cycle))
	}
	if s.config.OnCycle != nil {
		s.config.OnCycle(&cycle)
	}
}

func formatTimes(c *Cycle) string {
	kind := "rebuild"
	if c.Initial {
		kind = "initial"
	}
	program := "new program"
	if c.Reused {
		program = "reused program"
	}
	t := &c.Times
	ms := func(d time.Duration) string { return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000) }
	return fmt.Sprintf("[timing] %s total %s: debounce %s, program %s (%s), copy %s, prepare %s, plugin %s, check %s, lower %s, write %s; transformed %d, checked %d, lowered %d, written %d",
		kind, ms(c.Total), ms(c.Debounce), ms(t.Program), program, ms(t.Copy), ms(t.Prepare), ms(t.Plugin), ms(t.Check), ms(t.Lower), ms(t.Write), t.FilesTransformed, t.FilesChecked, t.FilesLowered, c.Written)
}
