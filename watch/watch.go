// Package watch is the port of src/Project/functions/setupProjectWatchProgram.ts from roblox-ts 3.0.0: `rbxtsc -w`.
//
// The messages and the order of work match upstream. How a rebuild works doesn't: see
// docs/watch-and-incremental.md.
package watch

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
	"github.com/microsoft/TypeScript/tsc/internal/execute/watchmanager"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/compile"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/plugins"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

// DefaultDebounce is how long a change has to stay quiet before a rebuild starts. fswatch already coalesces for 50ms,
// so this only has to catch what spills over. Upstream's chokidar and its timer wait 150ms between them.
const DefaultDebounce = 20 * time.Millisecond

// maxDebounce bounds the wait while events keep coming, so something rewriting files all the time can't starve a
// rebuild.
const maxDebounce = 500 * time.Millisecond

// pollInterval is chokidar's default interval.
const pollInterval = 100 * time.Millisecond

// Config says what to watch and how.
type Config struct {
	// Load reads the project options and data. It runs at the start and again after a config file changes.
	Load func() (*shared.ProjectData, error)
	// Checkers is the pool size. Zero means up to 8, more than a build uses, because a rebuild is all about latency.
	Checkers int
	// UsePolling walks the watched folders on a timer instead of using native notifications.
	UsePolling bool
	// Debounce replaces DefaultDebounce when it's positive.
	Debounce time.Duration
	// Output receives what upstream writes through ts.sys.write. Nil means stdout.
	Output io.Writer
	// OnCycle runs after each compile and its report. Tests use it. It must not keep Cycle.Result's diagnostics.
	OnCycle func(*Cycle)
	// Backend replaces the file watching backend. Tests use it.
	Backend watchmanager.WatchBackend
	// FullRebuilds turns off everything kept between compiles: each one checks and lowers every file. Tests use it to
	// compare the two, and RBXTSGO_WATCH_FULL=1 sets it, to measure what the shortcuts save.
	FullRebuilds bool
	// PluginOptions are the plugin settings from the command line.
	PluginOptions plugins.Options
	// BeforeCompile runs after a cycle has taken its events, just before it compiles. Tests use it.
	BeforeCompile func()
}

// Run watches until ctx ends. It returns an error only if the first load fails or if watching can't start. Compile
// errors are reported, and the watcher goes on.
func Run(ctx context.Context, config Config) error {
	if config.Output == nil {
		config.Output = os.Stdout
	}
	if config.Debounce <= 0 {
		config.Debounce = DefaultDebounce
	}
	if config.Checkers == 0 {
		// Measured on the generated project: 4 to 8 took a mid-size edit from 227ms to 171ms and a wide one from 406ms to
		// 278ms, for about 40 MB. 16 gained less, and took more memory. A leaf edit stayed the same. See the benchmarks.
		config.Checkers = min(8, runtime.NumCPU())
	}
	fsys := osvfs.FS()
	manager := watchmanager.NewWatchManager(os.Stderr, func(dir tspath.RootedDirectoryPath) bool {
		return fsys.DirectoryExists(dir)
	}, fsys.CaseSensitivity())
	switch {
	case config.Backend != nil:
		manager.SetBackend(config.Backend)
	case config.UsePolling:
		manager.SetBackend(newPollingBackend(pollInterval))
	default:
		manager.EnsureDefaultBackend()
	}
	defer manager.CloseAllWatches()

	s, err := newSession(config, manager)
	if err != nil {
		return err
	}
	// Watching starts before the first compile, as upstream waits for chokidar's "ready", so a save during it counts.
	if err := s.reconcileWatches(); err != nil {
		return err
	}

	report := func(started time.Time, debounce time.Duration) {
		result := s.runCompile()
		s.reportEmitResult(result)
		s.finishCycle(started, debounce, result)
	}
	s.reportText("Starting compilation in watch mode...")
	report(time.Now(), 0)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-manager.DoCycleCh():
		}
		// This is openEventCollection, closeEventCollection and upstream's 100ms timer.
		changed, overflow := manager.DrainEvents()
		s.collect(changed, overflow)
		if !s.hasPending() {
			continue
		}
		started := time.Now()
		s.reportText("File change detected. Starting incremental compilation...")
		s.waitForQuiet(ctx, manager, config.Debounce)
		if ctx.Err() != nil {
			return nil
		}
		// Anything saved from here on raises a new signal, so it can't be lost: it gets the next cycle.
		changed, overflow = manager.DrainEvents()
		s.collect(changed, overflow)
		report(started, time.Since(started))
	}
}

// waitForQuiet returns once no event has come for quiet, or after maxDebounce.
func (s *session) waitForQuiet(ctx context.Context, manager *watchmanager.WatchManager, quiet time.Duration) {
	deadline := time.NewTimer(maxDebounce)
	defer deadline.Stop()
	timer := time.NewTimer(quiet)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			return
		case <-timer.C:
			return
		case <-manager.DoCycleCh():
			changed, overflow := manager.DrainEvents()
			s.collect(changed, overflow)
			timer.Reset(quiet)
		}
	}
}

// reportText is watchReporter: ts.createWatchStatusReporter(ts.sys, true).
func (s *session) reportText(message string) {
	fmt.Fprintf(s.config.Output, "[\x1b[90m%s\x1b[0m] %s%s%s", time.Now().Format("3:04:05 PM"), message, shared.EOL, shared.EOL)
}

// reportEmitResult is reportEmitResult.
func (s *session) reportEmitResult(result *compile.EmitResult) {
	cwd := shared.ProcessCwd()
	caseSensitivity := osvfs.FS().CaseSensitivity()
	amtErrors := 0
	for _, d := range result.Diagnostics {
		fmt.Fprint(s.config.Output, shared.FormatDiagnostics([]*shared.Diagnostic{d}, cwd, caseSensitivity, shared.EOL), shared.EOL)
		if d.Category() == diagnostics.CategoryError {
			amtErrors++
		}
	}
	plural := "s"
	if amtErrors == 1 {
		plural = ""
	}
	s.reportText(fmt.Sprintf("Found %d error%s. Watching for file changes.", amtErrors, plural))
}

// eventPath puts a path from fswatch in the form the project uses: forward slashes, no trailing one.
func eventPath(path tspath.RootedPath) string {
	return strings.TrimSuffix(filepath.ToSlash(path.AsString()), "/")
}
