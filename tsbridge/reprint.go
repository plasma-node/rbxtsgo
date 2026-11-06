package tsbridge

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/execute/tsc"
	"github.com/microsoft/TypeScript/tsc/internal/parser"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
)

// reprintHost serves printed text for some files, and the previous program's parsed and bound files for the rest.
type reprintHost struct {
	compiler.CompilerHost
	printed  map[tspath.PathKey]string
	previous map[tspath.PathKey]*ast.SourceFile
}

func (h *reprintHost) GetSourceFile(options ast.SourceFileParseOptions) *ast.SourceFile {
	if text, ok := h.printed[options.PathKey]; ok {
		return parser.ParseSourceFile(options, text, core.EnsureScriptKindFromFileName(options.FileName))
	}
	if file := h.previous[options.PathKey]; file != nil {
		return file
	}
	return h.CompilerHost.GetSourceFile(options)
}

// WithPrintedFiles makes a new project on the same config where the given files have the given text. It stands for
// data.transformerWatcher in compileFiles.ts: JS calls updateFile for each file and gets a new program from a language
// service. Every other file, such as all the declaration files, keeps its parse and binding, so those aren't redone.
//
// The result has no file cache, so it can't be used with Update.
func (p *Project) WithPrintedFiles(printed map[tspath.PathKey]string) *Project {
	host := &reprintHost{
		CompilerHost: compiler.NewCachedFSCompilerHost(bundled.WrapFS(osvfs.FS()), bundled.LibPath(), &tsc.ExtendedConfigCache{}, nil, nil),
		printed:      printed,
		previous:     p.program.FilesByPath(),
	}
	program := compiler.NewProgram(compiler.ProgramOptions{Config: p.config, Host: host})
	return &Project{config: p.config, program: program, notes: p.notes}
}

// UpdatePrinted is WithPrintedFiles for a later compile: the receiver is the last printed project, source is the program
// the text came from, changed holds the text that differs from last time, and texts says which files are printed at all.
// Every other printed file keeps its parsed and bound object. full says the source changed in a way that can move imports,
// so reusing the old program's module resolution isn't safe.
func (p *Project) UpdatePrinted(source *Project, changed map[tspath.PathKey]string, texts map[tspath.PathKey]string, full bool) *Project {
	sourceFiles := source.program.FilesByPath()
	previous := p.program.FilesByPath()
	host := &reprintHost{
		CompilerHost: compiler.NewCachedFSCompilerHost(bundled.WrapFS(osvfs.FS()), bundled.LibPath(), &tsc.ExtendedConfigCache{}, nil, nil),
		printed:      changed,
	}
	// Reuse needs the same files, with only printed ones different.
	reusable := !full && len(sourceFiles) == len(previous)
	if reusable {
		for path, file := range sourceFiles {
			old, ok := previous[path]
			if _, printed := texts[path]; !ok || (!printed && old != file) {
				reusable = false
				break
			}
		}
	}
	host.previous = previous
	if reusable {
		if len(changed) == 0 {
			return p
		}
		program := p.program
		ok := true
		for path := range changed {
			program, _, ok = program.ReuseProgram(path, host, nil, nil)
			if !ok {
				break
			}
		}
		if ok {
			return &Project{config: p.config, program: program, notes: p.notes}
		}
	}
	// Unprinted files come from the new source program, and printed ones from the old printed program.
	merged := make(map[tspath.PathKey]*ast.SourceFile, len(sourceFiles))
	for path, file := range sourceFiles {
		if _, printed := texts[path]; printed {
			if old := previous[path]; old != nil {
				merged[path] = old
			}
		} else {
			merged[path] = file
		}
	}
	host.previous = merged
	program := compiler.NewProgram(compiler.ProgramOptions{Config: source.config, Host: host})
	return &Project{config: source.config, program: program, notes: source.notes}
}
