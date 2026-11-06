package compile

// Incremental builds: incremental and tsBuildInfoFile. See docs/watch-and-incremental.md for why this has its
// own format, and for what it reuses from watch mode's reasoning.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/pathtranslator"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/rojoresolver"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

// buildInfoVersion names this format. JS's readBuilderProgram finds no "program" in the file and starts over, so sharing
// a tsBuildInfoFile with rbxtsc costs a full build each way and nothing else. That was tried, see 0008.
const buildInfoVersion = "rbxtsgo-1"

type outputRecord struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// fileRecord is one program file as of the last successful build.
type fileRecord struct {
	// Source is the hash of the file on disk, and Compiled is the hash of what got compiled: the printed text with a
	// plugin, otherwise the same.
	Source   string `json:"source"`
	Compiled string `json:"compiled"`
	// Global says this version of the file can reach every file. See tsbridge.AffectedFiles.
	Global bool `json:"global,omitempty"`
	// Imports is where it imports from, in the compiled program.
	Imports []string `json:"imports,omitempty"`
	// Clean says it compiled with no diagnostics. A file with warnings compiles again, so they get shown again.
	Clean   bool           `json:"clean,omitempty"`
	Outputs []outputRecord `json:"outputs,omitempty"`
}

type buildInfoFile struct {
	Version string                 `json:"version"`
	Key     string                 `json:"key"`
	Files   map[string]*fileRecord `json:"files"`
}

// Incremental is the state of one build's tsBuildInfoFile. A nil one means the project isn't incremental.
type Incremental struct {
	path       string
	key        string
	translator *pathtranslator.Translator
	data       *shared.ProjectData
	previous   *buildInfoFile
	hashes     sync.Map // *ast.SourceFile to string
	// sourceHashes is filled from the source program.
	sourceHashes map[string]string
}

// BuildKey is createCompilerHost's contentsToHash, and what TS's builder compares in the compiler options: anything that
// has to force a rebuild when it changes. It takes the raw text of the configs, so a comment forces one too.
func BuildKey(data *shared.ProjectData, p *tsbridge.Project, extra string) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "version=%s,", shared.CompilerVersion)
	if exe, err := os.Executable(); err == nil {
		// Upstream's version only changes with a release. Here a rebuilt compiler can give new output with the same one.
		if info, err := os.Stat(exe); err == nil {
			fmt.Fprintf(hash, "exe=%d,%d,", info.Size(), info.ModTime().UnixNano())
		}
	}
	options := data.ProjectOptions
	// None of these change what gets written, and watch mode always has writeOnlyChanged on.
	options.Watch, options.UsePolling, options.Verbose, options.WriteOnlyChanged = false, false, false, false
	fmt.Fprintf(hash, "isPackage=%v,options=%+v,extra=%s,", data.IsPackage, options, extra)
	configs := append([]string{data.TsConfigPath, data.RojoConfigPath, nodepath.Native.Join(nodepath.Native.Dirname(data.NodeModulesPath), "package.json")}, extendedFiles(p)...)
	// What createNodeModulesPathMapping reads: a package's main or types changes generated requires. A package's
	// declarations are program files, which get hashed anyway.
	for _, typeRoot := range p.Options().TypeRoots {
		entries, err := os.ReadDir(string(typeRoot))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			fmt.Fprintf(hash, "package=%s,", entry.Name())
			configs = append(configs, nodepath.Native.Join(string(typeRoot), entry.Name(), "package.json"))
		}
	}
	for _, file := range configs {
		if file == "" {
			continue
		}
		text, _ := os.ReadFile(file)
		fmt.Fprintf(hash, "%d:", len(text))
		hash.Write(text)
	}
	// A Rojo project can nest others in its $path folders, and those decide generated paths too. Reading the project
	// walks those folders, and what it finds is what the compile would use. An error here is the compile's to report.
	if data.RojoConfigPath != "" {
		if resolver, err := rojoresolver.FromPath(data.RojoConfigPath); err == nil {
			fmt.Fprintf(hash, "rojo=%v,%v,", resolver.IsGame, resolver.Partitions())
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func extendedFiles(p *tsbridge.Project) []string {
	var result []string
	for _, file := range p.Config().ExtendedSourceFiles() {
		result = append(result, file.AsString())
	}
	return result
}

// LoadIncremental reads the tsBuildInfoFile. It returns nil when the project has none configured. A file that's missing,
// damaged or from another compiler is the same as no earlier build.
func LoadIncremental(data *shared.ProjectData, translator *pathtranslator.Translator, key string) *Incremental {
	if translator.BuildInfoOutputPath == "" {
		return nil
	}
	inc := &Incremental{path: translator.BuildInfoOutputPath, key: key, translator: translator, data: data}
	text, err := os.ReadFile(inc.path)
	if err != nil {
		return inc
	}
	var previous buildInfoFile
	if json.Unmarshal(text, &previous) != nil || previous.Version != buildInfoVersion || previous.Key != key || previous.Files == nil {
		return inc
	}
	inc.previous = &previous
	return inc
}

func (inc *Incremental) hashOf(file *ast.SourceFile) string {
	if known, ok := inc.hashes.Load(file); ok {
		return known.(string)
	}
	sum := sha256.Sum256([]byte(file.Text()))
	hash := hex.EncodeToString(sum[:16])
	inc.hashes.Store(file, hash)
	return hash
}

func (inc *Incremental) hashAll(files []*ast.SourceFile) []string {
	result := make([]string, len(files))
	var wg sync.WaitGroup
	gate := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i, file := range files {
		wg.Add(1)
		gate <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-gate }()
			result[i] = inc.hashOf(file)
		}()
	}
	wg.Wait()
	return result
}

// Unchanged says the sources and options are what the last successful build saw, and its outputs are still in place. If
// so there is nothing to compile, or even to print. p is the program before any plugin.
func (inc *Incremental) Unchanged(p *tsbridge.Project) bool {
	if inc == nil {
		return false
	}
	files := p.Program().SourceFiles()
	hashes := inc.hashAll(files)
	inc.sourceHashes = make(map[string]string, len(files))
	for i, file := range files {
		inc.sourceHashes[string(file.PathKey())] = hashes[i]
	}
	if inc.previous == nil || len(inc.previous.Files) != len(files) {
		return false
	}
	for i, file := range files {
		record := inc.previous.Files[string(file.PathKey())]
		if record == nil || record.Source != hashes[i] || (isCompiled(file) && !record.Clean) || !outputsInPlace(record) {
			return false
		}
	}
	return true
}

func isCompiled(file *ast.SourceFile) bool {
	return !file.IsDeclarationFile && !ast.IsJsonSourceFile(file)
}

func outputsInPlace(record *fileRecord) bool {
	for _, output := range record.Outputs {
		info, err := os.Stat(output.Path)
		if err != nil || info.Size() != output.Size || !info.ModTime().Equal(output.ModTime) {
			return false
		}
	}
	return true
}

// Dirty picks the files to compile on project p, the program after any plugin. files are all that a clean build would
// compile, in order. The others' output is what a clean build would write. See tsbridge.AffectedFiles for why.
func (inc *Incremental) Dirty(p *tsbridge.Project, files []*ast.SourceFile) []*ast.SourceFile {
	if inc == nil || inc.previous == nil {
		return files
	}
	programFiles := p.Program().SourceFiles()
	hashes := inc.hashAll(programFiles)
	if len(inc.previous.Files) != len(programFiles) {
		return files
	}
	graph := p.ImportGraph()
	var changed []tspath.PathKey
	for i, file := range programFiles {
		record := inc.previous.Files[string(file.PathKey())]
		if record == nil {
			return files
		}
		if record.Compiled != hashes[i] {
			if record.Global || tsbridge.HasGlobalEffects(file) {
				return files
			}
			changed = append(changed, file.PathKey())
		} else if !slices.Equal(record.Imports, keysToStrings(graph[file.PathKey()])) {
			return files
		}
	}
	affected := p.Closure(changed)
	dirty := make([]*ast.SourceFile, 0, len(files))
	for _, file := range files {
		record := inc.previous.Files[string(file.PathKey())]
		if affected[file.PathKey()] || !record.Clean || !outputsInPlace(record) {
			dirty = append(dirty, file)
		}
	}
	return dirty
}

func keysToStrings(keys []tspath.PathKey) []string {
	if len(keys) == 0 {
		return nil
	}
	result := make([]string, len(keys))
	for i, key := range keys {
		result[i] = string(key)
	}
	return result
}

// Save is program.emitBuildInfo(). dirty is what just compiled, and result is how that went: call this after a compile
// with no errors. Anything else keeps its record. sourceProject is the program before any plugin.
func (inc *Incremental) Save(sourceProject, compiled *tsbridge.Project, dirty []*ast.SourceFile, result *EmitResult) error {
	if inc == nil {
		return nil
	}
	reported := map[*ast.SourceFile]bool{}
	for _, d := range result.Diagnostics {
		reported[d.File()] = true
	}
	redone := make(map[*ast.SourceFile]bool, len(dirty))
	for _, file := range dirty {
		redone[file] = true
	}
	sourceFiles := sourceProject.Program().FilesByPath()
	programFiles := compiled.Program().SourceFiles()
	hashes := inc.hashAll(programFiles)
	graph := compiled.ImportGraph()
	declaration := compiled.Options().Declaration.IsTrue()
	next := &buildInfoFile{Version: buildInfoVersion, Key: inc.key, Files: make(map[string]*fileRecord, len(programFiles))}
	for i, file := range programFiles {
		path := file.PathKey()
		record := &fileRecord{
			Compiled: hashes[i],
			Global:   tsbridge.HasGlobalEffects(file),
			Imports:  keysToStrings(graph[path]),
		}
		if source := sourceFiles[path]; source != nil {
			record.Source = inc.hashOf(source)
		}
		var old *fileRecord
		if inc.previous != nil {
			old = inc.previous.Files[string(path)]
		}
		switch {
		case !isCompiled(file):
		case redone[file]:
			record.Clean = !reported[file]
			fsPath := nodepath.Native.Normalize(file.FileName().AsString())
			candidates := []string{inc.translator.GetOutputPath(fsPath)}
			if declaration {
				candidates = append(candidates, inc.translator.GetOutputDeclarationPath(fsPath))
			}
			for _, candidate := range candidates {
				info, err := os.Stat(candidate)
				if err != nil {
					// Upstream writes nothing for a file with no source, and a clean build would agree.
					record.Clean = false
					continue
				}
				record.Outputs = append(record.Outputs, outputRecord{Path: candidate, Size: info.Size(), ModTime: info.ModTime()})
			}
		case old != nil:
			record.Clean = old.Clean
			record.Outputs = old.Outputs
		}
		next.Files[string(path)] = record
	}
	text, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(nodepath.Native.Dirname(inc.path), 0o777); err != nil {
		return err
	}
	if err := os.WriteFile(inc.path, text, 0o666); err != nil {
		return err
	}
	inc.previous = next
	return nil
}

// CompileIncremental is CompileFiles for a build: only what Dirty picks, then Save. A nil inc compiles all of files.
func CompileIncremental(ctx context.Context, c *Context, inc *Incremental, source *tsbridge.Project, files []*ast.SourceFile) (*EmitResult, error) {
	dirty := inc.Dirty(c.Project, files)
	result, err := CompileFiles(ctx, c, dirty)
	if err != nil {
		return nil, err
	}
	// A diagnostic with no file comes from the whole program, and rbxtsc repeats it per file.
	if len(dirty) < len(files) && slices.ContainsFunc(result.Diagnostics, func(d *shared.Diagnostic) bool { return d.File() == nil }) {
		dirty = files
		if result, err = CompileFiles(ctx, c, files); err != nil {
			return nil, err
		}
	}
	if inc != nil && !result.EmitSkipped && !shared.HasErrors(result.Diagnostics) {
		if err := inc.Save(source, c.Project, dirty, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}
