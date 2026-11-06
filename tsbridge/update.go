package tsbridge

import (
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/execute/tsc"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/cachedvfs"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
)

type cachedSourceFile struct {
	file    *ast.SourceFile
	modTime time.Time
}

// SourceFileCache keeps parsed files between programs in one process, as execute/watcher.go does. An entry is good
// while the file's mtime stands, and watch mode also evicts on every event, so a save inside the clock's resolution
// can't be missed. Evict everything after a config change, because the parse options can differ.
type SourceFileCache struct {
	files collections.SyncMap[tspath.PathKey, *cachedSourceFile]
}

// NewSourceFileCache makes an empty cache.
func NewSourceFileCache() *SourceFileCache { return &SourceFileCache{} }

// Evict forgets one file.
func (c *SourceFileCache) Evict(path tspath.PathKey) { c.files.Delete(path) }

// Clear forgets every file.
func (c *SourceFileCache) Clear() {
	c.files.Range(func(path tspath.PathKey, _ *cachedSourceFile) bool {
		c.files.Delete(path)
		return true
	})
}

// retain forgets every file that the program doesn't have, so deleted files don't pile up.
func (c *SourceFileCache) retain(program *compiler.Program) {
	files := program.FilesByPath()
	c.files.Range(func(path tspath.PathKey, _ *cachedSourceFile) bool {
		if _, ok := files[path]; !ok {
			c.files.Delete(path)
		}
		return true
	})
}

// cachingHost is watchCompilerHost from execute/watcher.go, which is unexported there.
type cachingHost struct {
	compiler.CompilerHost
	cache *SourceFileCache
}

func (h *cachingHost) GetSourceFile(options ast.SourceFileParseOptions) *ast.SourceFile {
	info := h.CompilerHost.FS().Stat(options.FileName.AsPath())
	if cached, ok := h.cache.files.Load(options.PathKey); ok {
		if info != nil && info.ModTime().Equal(cached.modTime) {
			return cached.file
		}
	}
	file := h.CompilerHost.GetSourceFile(options)
	if file != nil && info != nil {
		h.cache.files.Store(options.PathKey, &cachedSourceFile{file: file, modTime: info.ModTime()})
	} else {
		h.cache.files.Delete(options.PathKey)
	}
	return file
}

// newCycleHost makes a host for one program. The stat and read cache must not outlive the program's construction:
// keeping one across cycles would build from stale contents. So the caller gets a function that turns it off.
func newCycleHost(cache *SourceFileCache) (compiler.CompilerHost, func()) {
	cached := cachedvfs.From(bundled.WrapFS(osvfs.FS()))
	host := compiler.NewCompilerHost(cached, bundled.LibPath(), &tsc.ExtendedConfigCache{}, nil, nil)
	return &cachingHost{CompilerHost: host, cache: cache}, cached.DisableAndClearCache
}

// Update makes the project for the next build, with changedPaths evicted from the cache. It tries
// Program.ReuseProgram, which keeps parsing, binding and module resolution for files whose imports didn't change, and
// otherwise falls back to NewProgram with the cached files. reloadFileNames says that a file was added or removed, so
// the tsconfig's patterns have to be expanded again, and it always means NewProgram. The receiver isn't changed, and
// it must not be used again: drop it so its checkers can be collected.
//
// The project must have been made with ProjectOptions.Cache.
func (p *Project) Update(changedPaths []tspath.PathKey, reloadFileNames bool) (next *Project, reused bool) {
	if p.cache == nil {
		panic("tsbridge: Update needs a project made with ProjectOptions.Cache")
	}
	for _, path := range changedPaths {
		p.cache.Evict(path)
	}
	host, done := newCycleHost(p.cache)
	defer done()

	config := p.config
	if reloadFileNames {
		config = config.ReloadFileNamesOfParsedCommandLine(host.FS())
	}
	next = &Project{config: config, notes: p.notes, cache: p.cache}

	if !reloadFileNames {
		program := p.program
		ok := true
		for _, path := range changedPaths {
			if _, inProgram := program.FilesByPath()[path]; !inProgram {
				// Not part of the program, so no reuse can tell whether it changes module resolution.
				ok = false
				break
			}
			program, _, ok = program.ReuseProgram(path, host, nil, nil)
			if !ok {
				break
			}
		}
		if ok {
			next.program = program
			return next, true
		}
	}
	next.program = compiler.NewProgram(compiler.ProgramOptions{Config: config, Host: host})
	p.cache.retain(next.program)
	return next, false
}
