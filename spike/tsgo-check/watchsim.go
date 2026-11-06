package main

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/execute/tsc"
	"github.com/microsoft/TypeScript/tsc/internal/parser"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs"
)

// editHost is a simplified copy of the source file cache in tsgo's watcher, plus an
// in-memory edit, so the spike doesn't have to modify files on disk.
type editHost struct {
	compiler.CompilerHost
	cache  *collections.SyncMap[tspath.PathKey, *ast.SourceFile]
	path   tspath.PathKey
	edited string
	isEdit bool
}

func (h *editHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	if h.isEdit && opts.PathKey == h.path {
		return parser.ParseSourceFile(opts, h.edited, core.GetScriptKindFromFileName(opts.FileName))
	}
	if file, ok := h.cache.Load(opts.PathKey); ok {
		return file
	}
	file := h.CompilerHost.GetSourceFile(opts)
	if file != nil {
		h.cache.Store(opts.PathKey, file)
	}
	return file
}

func median(values []time.Duration) time.Duration {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	return sorted[len(sorted)/2]
}

// runWatchSim times the reusable pieces. Each run edits one file by appending a statement.
func runWatchSim(ctx context.Context, config *tsoptions.ParsedCommandLine, cwd tspath.RootedDirectoryPath, fs vfs.FS, cache *tsc.ExtendedConfigCache, target string, runs int) error {
	abs := tspath.ToRootedFilePath(filepath.ToSlash(target), cwd)
	path := fs.CaseSensitivity().PathKey(abs.AsPath())
	original, ok := fs.ReadFile(abs)
	if !ok {
		return fmt.Errorf("cannot read %s", abs)
	}

	files := &collections.SyncMap[tspath.PathKey, *ast.SourceFile]{}
	base := compiler.NewCompilerHost(fs, bundled.LibPath(), cache, nil, nil)
	host := &editHost{CompilerHost: base, cache: files, path: path}

	cold := time.Now()
	program := compiler.NewProgram(compiler.ProgramOptions{Config: config, Host: host})
	checkAll := func(p *compiler.Program) {
		for _, file := range p.SourceFiles() {
			p.GetSemanticDiagnostics(ctx, file)
		}
	}
	loadTime := time.Since(cold)
	coldCheck := time.Now()
	checkAll(program)
	fmt.Printf("watch-sim cold: load %s, check all %s, %d files\n", loadTime.Round(time.Microsecond), time.Since(coldCheck).Round(time.Microsecond), len(program.SourceFiles()))

	var rebuilds, updates, newChecker, oneFile, userTime, allFiles []time.Duration
	reused := 0
	for i := range runs {
		host.isEdit = true
		host.edited = original + fmt.Sprintf("\nexport const watchSimEdit = %d;\n", i)

		start := time.Now()
		rebuilt := compiler.NewProgram(compiler.ProgramOptions{Config: config, Host: host})
		rebuilds = append(rebuilds, time.Since(start))

		start = time.Now()
		updated, _, wasReused := program.UpdateProgram(path, host, nil, nil)
		updates = append(updates, time.Since(start))
		if wasReused {
			reused++
		}
		updated.BindSourceFiles()

		start = time.Now()
		var edited *ast.SourceFile
		for _, file := range updated.SourceFiles() {
			if file.PathKey() == path {
				edited = file
			}
		}
		c, done := updated.GetTypeCheckerForFile(ctx, edited)
		done()
		newChecker = append(newChecker, time.Since(start))

		start = time.Now()
		c.GetDiagnostics(ctx, edited)
		oneFile = append(oneFile, time.Since(start))

		start = time.Now()
		for _, file := range userFiles(updated) {
			updated.GetSemanticDiagnostics(ctx, file)
		}
		userTime = append(userTime, time.Since(start))

		start = time.Now()
		checkAll(updated)
		allFiles = append(allFiles, time.Since(start))
		_ = rebuilt
	}
	fmt.Printf("watch-sim medians over %d runs (edit of %s):\n", runs, strings.TrimPrefix(abs.AsString(), cwd.AsString()+"/"))
	fmt.Printf("  NewProgram with cached source files:  %s\n", median(rebuilds).Round(time.Microsecond))
	fmt.Printf("  UpdateProgram:                        %s (reused %d of %d)\n", median(updates).Round(time.Microsecond), reused, runs)
	fmt.Printf("  create all checkers in the fresh pool: %s\n", median(newChecker).Round(time.Microsecond))
	fmt.Printf("  check the edited file:                %s\n", median(oneFile).Round(time.Microsecond))
	fmt.Printf("  then check the other user files:      %s\n", median(userTime).Round(time.Microsecond))
	fmt.Printf("  then check everything, including .d.ts: %s\n", median(allFiles).Round(time.Microsecond))
	return nil
}
