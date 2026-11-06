// Package project is the port of src/Project from roblox-ts 3.0.0: options, tsconfig validation, file-name checks,
// and the file copying and cleanup around compilation. Compiling itself comes after the transformer port.
//
// Not ported, on purpose: createProgramFactory and getChangedFilePaths (no incremental builds yet, see 0003 item 6),
// setupProjectWatchProgram (watch mode is a later phase), createTransformerList and friends (plugins are a later
// phase), and VirtualProject (the playground).
package project

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/pathtranslator"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/rojoresolver"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

var path = nodepath.Native

var packageRegex = regexp.MustCompile(`^@[a-z0-9-]*/`)

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// findPackageJSON is ts.findPackageJson: the nearest package.json in this directory or an ancestor.
func findPackageJSON(directory string) string {
	for {
		candidate := path.Join(directory, "package.json")
		if exists(candidate) {
			return candidate
		}
		parent := path.Dirname(directory)
		if parent == directory {
			return ""
		}
		directory = parent
	}
}

// CreateProjectData is createProjectData. Rojo's warnings go to the log, as upstream.
func CreateProjectData(tsConfigPath string, options shared.ProjectOptions) (*shared.ProjectData, error) {
	projectPath := path.Dirname(tsConfigPath)

	pkgJSONPath := findPackageJSON(projectPath)
	if pkgJSONPath == "" {
		return nil, shared.NewProjectError("Unable to find package.json")
	}

	isPackage := false
	if data, err := os.ReadFile(pkgJSONPath); err == nil {
		var pkgJSON struct {
			Name any `json:"name"`
		}
		if json.Unmarshal(data, &pkgJSON) == nil {
			if name, ok := pkgJSON.Name.(string); ok {
				isPackage = packageRegex.MatchString(name)
			}
		}
	}

	// intentionally uses the empty string case too
	if options.IncludePath == "" {
		options.IncludePath = path.Resolve(path.Join(projectPath, "include"))
	} else {
		options.IncludePath = path.Resolve(options.IncludePath)
	}

	nodeModulesPath := path.Join(path.Dirname(pkgJSONPath), shared.NodeModules)

	rojoConfigPath := ""
	if options.Rojo != "" {
		rojoConfigPath = path.Resolve(options.Rojo)
	} else {
		found, warnings, err := rojoresolver.FindRojoConfigFilePath(projectPath)
		if err != nil {
			return nil, err
		}
		rojoConfigPath = found
		for _, warning := range warnings {
			shared.LogService.Warn(warning)
		}
	}

	return &shared.ProjectData{
		TsConfigPath:    tsConfigPath,
		IsPackage:       isPackage,
		NodeModulesPath: nodeModulesPath,
		ProjectOptions:  options,
		ProjectPath:     projectPath,
		RojoConfigPath:  rojoConfigPath,
	}, nil
}

func validateTypeRoots(nodeModulesPath string, typeRoots []string) bool {
	typesPath := path.Resolve(nodeModulesPath)
	return slices.ContainsFunc(typeRoots, func(typeRoot string) bool {
		return path.Resolve(typeRoot) == typesPath
	})
}

// removedOption reads an option from the raw tsconfig, for options TS 7 no longer parses.
func rawCompilerOption(parsed *tsbridge.ParsedConfig, name string) (any, bool) {
	raw, ok := parsed.Config.Raw.(interface {
		Get(string) (any, bool)
	})
	if !ok {
		return nil, false
	}
	compilerOptions, ok := raw.Get("compilerOptions")
	if !ok {
		return nil, false
	}
	options, ok := compilerOptions.(interface {
		Get(string) (any, bool)
	})
	if !ok {
		return nil, false
	}
	return options.Get(name)
}

// ValidateCompilerOptions is validateCompilerOptions. It must run on the options as written, before normalization,
// because it requires "moduleResolution": "Node". The check on "target" is commented out upstream too.
func ValidateCompilerOptions(parsed *tsbridge.ParsedConfig, projectPath string) error {
	y := shared.Color.Yellow
	opts := parsed.Config.CompilerOptions()
	var errs []string

	// required compiler options
	if !opts.NoLib.IsTrue() {
		errs = append(errs, y(`"noLib"`)+" must be "+y("true"))
	}
	if !opts.Strict.IsTrue() {
		errs = append(errs, y(`"strict"`)+" must be "+y("true"))
	}
	// TS 7 removed "moduleResolution": "node", so a config written for it has this pair instead. See 0007.
	ts7Style := opts.Module == core.ModuleKindPreserve && opts.ModuleResolution == core.ModuleResolutionKindBundler
	if opts.Module != core.ModuleKindCommonJS && !ts7Style {
		errs = append(errs, y(`"module"`)+" must be "+y("commonjs"))
	}
	if opts.ModuleDetection != core.ModuleDetectionKindForce {
		errs = append(errs, y(`"moduleDetection"`)+" must be "+y(`"force"`))
	}
	if opts.ModuleResolution != core.ModuleResolutionKindNode10 && !ts7Style {
		errs = append(errs, y(`"moduleResolution"`)+" must be "+y(`"Node"`))
	}
	if !opts.AllowSyntheticDefaultImports.IsTrue() {
		errs = append(errs, y(`"allowSyntheticDefaultImports"`)+" must be "+y("true"))
	}

	rbxtsModules := path.Join(projectPath, shared.NodeModules, shared.RbxtsScope)
	if opts.TypeRoots == nil || !validateTypeRoots(rbxtsModules, typeRootStrings(opts)) {
		errs = append(errs, y(`"typeRoots"`)+" must contain "+y(rbxtsModules))
	}

	for _, typesLocation := range opts.Types {
		// Technically checked to exist above. But if that's an error, we still want this error too,
		// to avoid "fix one error, get a new one on the next compile".
		typeRoots := typeRootStrings(opts)
		if typeRoots == nil {
			typeRoots = []string{"node_modules/@rbxts"}
		}
		found := slices.ContainsFunc(typeRoots, func(typeRoot string) bool {
			typesPath := path.Resolve(projectPath, typeRoot, typesLocation)
			return exists(typesPath) || exists(typesPath+shared.DTSExt)
		})
		if !found {
			errs = append(errs, y(`"types"`)+" "+y(typesLocation)+" were not found. Make sure the path is relative to `typeRoots`")
		}
	}

	// configurable compiler options
	if opts.RootDir == "" && opts.RootDirs == nil {
		errs = append(errs, y(`"rootDir"`)+" or "+y(`"rootDirs"`)+" must be defined")
	}
	if opts.OutDir == "" {
		errs = append(errs, y(`"outDir"`)+" must be defined")
	}
	if value, ok := rawCompilerOption(parsed, "importsNotUsedAsValues"); ok {
		suggested := "false"
		if text, isString := value.(string); isString && strings.EqualFold(text, "preserve") {
			suggested = "true"
		}
		errs = append(errs, y(`"importsNotUsedAsValues"`)+" is no longer supported, use "+y(`"verbatimModuleSyntax": `+suggested)+" instead")
	}

	if len(errs) > 0 {
		var list strings.Builder
		for _, err := range errs {
			list.WriteString("- " + err + "\n")
		}
		return shared.NewProjectError(strings.Join([]string{
			`Invalid "tsconfig.json" configuration!`,
			"https://roblox-ts.com/docs/quick-start#project-folder-setup",
			list.String(),
		}, "\n"))
	}
	return nil
}

// GetParsedCommandLine is getParsedCommandLine: parse, fail on parser errors, validate.
func GetParsedCommandLine(data *shared.ProjectData) (*tsbridge.ParsedConfig, error) {
	parsed, err := tsbridge.ParseConfig(data.TsConfigPath)
	if err != nil {
		return nil, shared.NewProjectError("Unable to load TS program!")
	}
	// An option that TS 5.5.3 still parses but TS 7 doesn't must get the validation message, not a parser error.
	if _, removed := rawCompilerOption(parsed, "importsNotUsedAsValues"); removed {
		if err := ValidateCompilerOptions(parsed, data.ProjectPath); err != nil {
			return nil, err
		}
	}
	if len(parsed.Diagnostics) > 0 {
		return nil, &shared.DiagnosticError{Diagnostics: shared.FromTSList(parsed.Diagnostics)}
	}
	if err := ValidateCompilerOptions(parsed, data.ProjectPath); err != nil {
		return nil, err
	}
	return parsed, nil
}

// GetRootDirs is getRootDirs.
func GetRootDirs(options *core.CompilerOptions) []string {
	if options.RootDir != "" {
		return []string{options.RootDir.AsString()}
	}
	if options.RootDirs == nil {
		panic(shared.AssertionMessage(""))
	}
	result := make([]string, len(options.RootDirs))
	for i, dir := range options.RootDirs {
		result[i] = string(dir)
	}
	return result
}

// CreatePathTranslator is createPathTranslator.
func CreatePathTranslator(p *tsbridge.Project, data *shared.ProjectData) *pathtranslator.Translator {
	options := p.Options()
	rootDir := shared.FindAncestorDir(append([]string{p.CommonSourceDirectory().AsString()}, GetRootDirs(options)...))
	buildInfoPath := p.Config().GetBuildInfoFileName().AsString()
	if buildInfoPath != "" {
		buildInfoPath = path.Normalize(buildInfoPath)
	}
	return pathtranslator.New(rootDir, options.OutDir.AsString(), buildInfoPath, options.Declaration.IsTrue(), data.ProjectOptions.Luau)
}

// CheckFileName is checkFileName.
func CheckFileName(service *shared.DiagnosticService, filePath string) {
	baseName := path.Basename(filePath, "")
	if warning, ok := shared.FilenameWarnings[baseName]; ok {
		service.AddDiagnostic(shared.Errors.IncorrectFileName(baseName, warning, filePath))
	}
}

// CheckRojoConfig is checkRojoConfig.
func CheckRojoConfig(service *shared.DiagnosticService, data *shared.ProjectData, resolver *rojoresolver.Resolver, rootDirs []string, translator *pathtranslator.Translator) {
	if data.RojoConfigPath == "" {
		return
	}
	for _, partition := range resolver.Partitions() {
		for _, rootDir := range rootDirs {
			if shared.IsPathDescendantOf(partition.FsPath, rootDir) {
				rojoConfigDir := path.Dirname(data.RojoConfigPath)
				outPath := translator.GetOutputPath(partition.FsPath)
				inputPath := path.Relative(rojoConfigDir, partition.FsPath)
				suggestedPath := path.Relative(rojoConfigDir, outPath)
				service.AddDiagnostic(shared.Errors.RojoPathInSrc(inputPath, suggestedPath))
			}
		}
	}
}

// FileUsesCommentDirectives is fileUsesCommentDirectives, the one custom pre-emit check.
func FileUsesCommentDirectives(data *shared.ProjectData, file *ast.SourceFile) []*shared.Diagnostic {
	if data.ProjectOptions.AllowCommentDirectives {
		return nil
	}
	var diagnostics []*shared.Diagnostic
	for _, directive := range file.CommentDirectives {
		diagnostics = append(diagnostics, shared.Errors.NoCommentDirectives.At(shared.AtRange(shared.SourceRange{File: file, Pos: directive.Loc.Pos(), End: directive.Loc.End()})))
	}
	for _, pragma := range file.Pragmas {
		if pragma.Name == "ts-nocheck" {
			diagnostics = append(diagnostics, shared.Errors.NoCommentDirectives.At(shared.AtRange(shared.SourceRange{File: file, Pos: pragma.Pos(), End: pragma.End()})))
		}
	}
	return diagnostics
}

// IsCompilableFile is isCompilableFile.
func IsCompilableFile(fsPath string) (bool, error) {
	info, err := os.Stat(fsPath)
	if err != nil {
		return false, err
	}
	if info.IsDir() || strings.HasSuffix(fsPath, shared.DTSExt) {
		return false, nil
	}
	return strings.HasSuffix(fsPath, shared.TSExt) || strings.HasSuffix(fsPath, shared.TSXExt), nil
}

// CreateNodeModulesPathMapping is createNodeModulesPathMapping. Keys are canonical file names.
func CreateNodeModulesPathMapping(typeRoots []string, caseSensitivity tspath.CaseSensitivity) (map[string]string, error) {
	mapping := map[string]string{}
	for _, scopePath := range typeRoots {
		if !exists(scopePath) {
			continue
		}
		entries, err := os.ReadDir(scopePath)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			pkgPath := path.Join(scopePath, entry.Name())
			pkgJSONPath, ok := shared.RealPathExists(path.Join(pkgPath, "package.json"))
			if !ok {
				continue
			}
			data, err := os.ReadFile(pkgJSONPath)
			if err != nil {
				return nil, err
			}
			var pkgJSON struct {
				Main    *string `json:"main"`
				Typings *string `json:"typings"`
				Types   *string `json:"types"`
			}
			if err := json.Unmarshal(data, &pkgJSON); err != nil {
				return nil, err
			}
			// both "types" and "typings" are valid
			typesPath := "index.d.ts"
			if pkgJSON.Types != nil {
				typesPath = *pkgJSON.Types
			} else if pkgJSON.Typings != nil {
				typesPath = *pkgJSON.Typings
			}
			if pkgJSON.Main != nil && *pkgJSON.Main != "" {
				mapping[caseSensitivity.Canonicalize(path.Resolve(pkgPath, typesPath))] = path.Resolve(pkgPath, *pkgJSON.Main)
			}
		}
	}
	return mapping, nil
}

// inferProjectType is inferProjectType in compileFiles.ts.
func inferProjectType(data *shared.ProjectData, resolver *rojoresolver.Resolver) shared.ProjectType {
	if data.IsPackage {
		return shared.ProjectTypePackage
	}
	if resolver.IsGame {
		return shared.ProjectTypeGame
	}
	return shared.ProjectTypeModel
}

// CompileContext is what compileFiles.ts works out before transforming, and what TransformState needs.
type CompileContext struct {
	RojoResolver           *rojoresolver.Resolver
	PkgRojoResolvers       []*rojoresolver.Resolver
	NodeModulesPathMapping map[string]string
	// RuntimeLibRbxPath is nil where JS has undefined, which it is for packages.
	RuntimeLibRbxPath rojoresolver.RbxPath
	ProjectType       shared.ProjectType
}

// PrepareCompile is the part of compileFiles before the plugin and transform steps. Failure text matches
// emitResultFailure. A non-nil failure means stop with those diagnostics. Plain errors are what the JS throws.
func PrepareCompile(p *tsbridge.Project, data *shared.ProjectData, translator *pathtranslator.Translator, caseSensitivity tspath.CaseSensitivity) (*CompileContext, []*shared.Diagnostic, error) {
	return PrepareCompileCached(p, data, translator, caseSensitivity, nil)
}

// PrepareCache keeps what PrepareCompile reads from the Rojo project and the packages, so that a long-running process
// doesn't walk those folders for every compile. Whoever holds one must drop it when one of those changes.
type PrepareCache struct {
	resolver     *rojoresolver.Resolver
	pkgResolvers []*rojoresolver.Resolver
	mapping      map[string]string
}

// PrepareCompileCached is PrepareCompile. The cache can be nil.
func PrepareCompileCached(p *tsbridge.Project, data *shared.ProjectData, translator *pathtranslator.Translator, caseSensitivity tspath.CaseSensitivity, cache *PrepareCache) (*CompileContext, []*shared.Diagnostic, error) {
	options := p.Options()
	outDir := options.OutDir.AsString()

	var resolver *rojoresolver.Resolver
	var err error
	switch {
	case cache != nil && cache.resolver != nil:
		resolver = cache.resolver
	case data.RojoConfigPath != "":
		resolver, err = rojoresolver.FromPath(data.RojoConfigPath)
	default:
		resolver, err = rojoresolver.Synthetic(outDir)
	}
	if err != nil {
		return nil, nil, err
	}
	for _, warning := range resolver.Warnings() {
		shared.LogService.Warn(warning)
	}

	service := &shared.DiagnosticService{}
	CheckRojoConfig(service, data, resolver, GetRootDirs(options), translator)
	for _, file := range p.Program().SourceFiles() {
		if !strings.HasPrefix(path.Normalize(file.FileName().AsString()), data.NodeModulesPath) {
			CheckFileName(service, file.FileName().AsString())
		}
	}

	if options.TypeRoots == nil {
		panic("typeRoots must be set: ValidateCompilerOptions checks this")
	}
	var pkgResolvers []*rojoresolver.Resolver
	var mapping map[string]string
	if cache != nil && cache.resolver != nil {
		pkgResolvers, mapping = cache.pkgResolvers, cache.mapping
	} else {
		pkgResolvers = make([]*rojoresolver.Resolver, len(options.TypeRoots))
		for i, typeRoot := range options.TypeRoots {
			pkgResolvers[i], err = rojoresolver.Synthetic(string(typeRoot))
			if err != nil {
				return nil, nil, err
			}
		}
		mapping, err = CreateNodeModulesPathMapping(typeRootStrings(options), caseSensitivity)
		if err != nil {
			return nil, nil, err
		}
		if cache != nil {
			cache.resolver, cache.pkgResolvers, cache.mapping = resolver, pkgResolvers, mapping
		}
	}

	projectType := data.ProjectOptions.Type
	if projectType == "" {
		projectType = inferProjectType(data, resolver)
	}

	failure := func(message string) []*shared.Diagnostic {
		return []*shared.Diagnostic{shared.TextDiagnostic(message, shared.CategoryError)}
	}
	if projectType != shared.ProjectTypePackage && data.RojoConfigPath == "" {
		return nil, failure("Non-package projects must have a Rojo project file!"), nil
	}

	var runtimeLibRbxPath rojoresolver.RbxPath
	if projectType != shared.ProjectTypePackage {
		runtimeLibRbxPath = resolver.GetRbxPathFromFilePath(path.Join(data.ProjectOptions.IncludePath, "RuntimeLib.lua"))
		if runtimeLibRbxPath == nil {
			return nil, failure("Rojo project contained no data for include folder!"), nil
		}
		if resolver.GetNetworkType(runtimeLibRbxPath) != rojoresolver.NetworkTypeUnknown {
			return nil, failure("Runtime library cannot be in a server-only or client-only container!"), nil
		}
		if resolver.IsIsolated(runtimeLibRbxPath) {
			return nil, failure("Runtime library cannot be in an isolated container!"), nil
		}
	}

	if service.HasErrors() {
		return nil, service.Flush(), nil
	}

	return &CompileContext{
		RojoResolver:           resolver,
		PkgRojoResolvers:       pkgResolvers,
		NodeModulesPathMapping: mapping,
		RuntimeLibRbxPath:      runtimeLibRbxPath,
		ProjectType:            projectType,
	}, nil, nil
}

func typeRootStrings(options *core.CompilerOptions) []string {
	if options.TypeRoots == nil {
		return nil
	}
	result := make([]string, len(options.TypeRoots))
	for i, typeRoot := range options.TypeRoots {
		result[i] = string(typeRoot)
	}
	return result
}
