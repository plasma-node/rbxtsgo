package tsbridge

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/execute/tsc"
	"github.com/microsoft/TypeScript/tsc/internal/outputpaths"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
)

// DefaultCheckers matches tsgo's own default pool size.
const DefaultCheckers = 4

// ParsedConfig is a tsconfig as written, before normalization. Validate the original options on this:
// the validation in src/Project requires "moduleResolution": "Node", which normalization rewrites.
type ParsedConfig struct {
	Config      *tsoptions.ParsedCommandLine
	Diagnostics []*ast.Diagnostic
	Cwd         tspath.RootedDirectoryPath
	Path        tspath.RootedFilePath
}

// ParseConfig reads a tsconfig. The diagnostics are the parser's, as in ts.getParsedCommandLineOfConfigFile.
func ParseConfig(tsconfigPath string) (*ParsedConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	rootedCwd := tspath.RootedDirectoryPathFromAbsolute(tspath.NormalizePath(filepath.ToSlash(cwd)))
	path := tspath.ToRootedFilePath(filepath.ToSlash(tsconfigPath), rootedCwd)
	config, diags := tsoptions.GetParsedCommandLineOfConfigFile(path, nil, nil, bundled.WrapFS(osvfs.FS()), &tsc.ExtendedConfigCache{})
	if config == nil {
		return nil, fmt.Errorf("could not parse %s", path)
	}
	return &ParsedConfig{Config: config, Diagnostics: diags, Cwd: rootedCwd, Path: path}, nil
}

// ProjectOptions configures NewProject.
type ProjectOptions struct {
	// Checkers is the pool size. Zero means DefaultCheckers. tsgo caps it at the number of program files.
	Checkers int
	// Cache keeps parsed files between programs. Set it to use Project.Update.
	Cache *SourceFileCache
}

// Project is a program built from a normalized tsconfig.
type Project struct {
	config  *tsoptions.ParsedCommandLine
	program *compiler.Program
	notes   []string
	cache   *SourceFileCache
	graphState

	// workersRunning catches diagnostics overlapping with the workers, which hold no lock.
	workersRunning atomic.Bool
}

// NewProject normalizes the config in place and creates the program.
func NewProject(parsed *ParsedConfig, options ProjectOptions) *Project {
	config := parsed.Config
	opts, notes := NormalizeOptions(config.CompilerOptions())
	checkers := options.Checkers
	if checkers == 0 {
		checkers = DefaultCheckers
	}
	opts.Checkers = &checkers
	config.SetCompilerOptions(opts)
	// createProgramFactory passes rootNames, options, host and oldProgram, so JS's program never sees references. An import
	// across projects gets the source file and TS6059, not a project reference's output.
	config.ParsedConfig.ProjectReferences = nil

	if options.Cache != nil {
		host, done := newCycleHost(options.Cache)
		defer done()
		program := compiler.NewProgram(compiler.ProgramOptions{Config: config, Host: host})
		options.Cache.retain(program)
		return &Project{config: config, program: program, notes: notes, cache: options.Cache}
	}
	host := compiler.NewCachedFSCompilerHost(bundled.WrapFS(osvfs.FS()), bundled.LibPath(), &tsc.ExtendedConfigCache{}, nil, nil)
	program := compiler.NewProgram(compiler.ProgramOptions{Config: config, Host: host})
	return &Project{config: config, program: program, notes: notes}
}

// Program returns the underlying program. See the package comment for the calls that aren't allowed on it.
func (p *Project) Program() *compiler.Program { return p.program }

// Options returns the normalized compiler options.
func (p *Project) Options() *core.CompilerOptions { return p.config.CompilerOptions() }

// Config returns the normalized parsed config.
func (p *Project) Config() *tsoptions.ParsedCommandLine { return p.config }

// NormalizationNotes describes what NormalizeOptions changed.
func (p *Project) NormalizationNotes() []string { return p.notes }

// SourceFilesToCompile mirrors getChangedSourceFiles on a fresh build, where every file counts as changed:
// all program files except declaration files and JSON, in program order. This includes a non-declaration
// .ts file inside node_modules, because rbxtsc doesn't exclude those either.
func (p *Project) SourceFilesToCompile() []*ast.SourceFile {
	var files []*ast.SourceFile
	for _, file := range p.program.SourceFiles() {
		if !file.IsDeclarationFile && !ast.IsJsonSourceFile(file) {
			files = append(files, file)
		}
	}
	return files
}

// CommonSourceDirectory is program.getCommonSourceDirectory() as TS 5.5.3 computes it. Don't use
// Program.CommonSourceDirectory for output paths: since TS 6 it falls back to the tsconfig's folder when rootDir is
// unset, where TS 5 uses the common folder of the emitted files. A project with only rootDirs would get different
// output paths.
func (p *Project) CommonSourceDirectory() tspath.RootedDirectoryPath {
	options := p.Options()
	if options.RootDir != "" || options.Composite.IsTrue() {
		return p.program.CommonSourceDirectory()
	}
	var emitted []tspath.RootedFilePath
	for _, file := range p.program.SourceFiles() {
		if !file.IsDeclarationFile && !p.program.IsSourceFileFromExternalLibrary(file) {
			emitted = append(emitted, file.FileName())
		}
	}
	return outputpaths.GetComputedCommonSourceDirectory(emitted, p.program.BaseDirectory(), p.program.CaseSensitivity())
}
