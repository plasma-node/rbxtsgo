package shared

// From src/Shared/constants.ts.
const (
	// CompilerVersion is the roblox-ts version this port matches.
	CompilerVersion = "3.0.0"

	NodeModules = "node_modules"
	RbxtsScope  = "@rbxts"

	TSExt  = ".ts"
	TSXExt = ".tsx"
	DExt   = ".d"
	DTSExt = DExt + TSExt

	IndexName = "index"
	InitName  = "init"

	ServerSubext = ".server"
	ClientSubext = ".client"
	ModuleSubext = ""

	ParentField = "Parent"
)

// FilenameWarnings maps a file name that won't work to the name to suggest.
var FilenameWarnings = func() map[string]string {
	warnings := map[string]string{}
	for _, scriptType := range []string{ServerSubext, ClientSubext, ModuleSubext} {
		for _, fileType := range []string{TSExt, TSXExt, DTSExt} {
			warnings[InitName+scriptType+fileType] = IndexName + scriptType + fileType
		}
	}
	return warnings
}()

// ProjectType is the enum of the same name. "" stands for undefined in ProjectOptions.Type.
type ProjectType string

const (
	ProjectTypeGame    ProjectType = "game"
	ProjectTypeModel   ProjectType = "model"
	ProjectTypePackage ProjectType = "package"
)

// ProjectOptions is the interface of the same name in src/Shared/types.ts.
type ProjectOptions struct {
	IncludePath string
	// Rojo is "" where JS has undefined. Upstream treats those the same way.
	Rojo string
	// Type is "" where JS has undefined.
	Type                   ProjectType
	LogTruthyChanges       bool
	NoInclude              bool
	UsePolling             bool
	Verbose                bool
	Watch                  bool
	WriteOnlyChanged       bool
	WriteTransformedFiles  bool
	OptimizedLoops         bool
	AllowCommentDirectives bool
	Luau                   bool
	// FixEmit is rbxtsgo's own. It turns on the output fixes that differ from JS 3.0.0 for code that compiles there. See
	// docs/known-differences.md. BuildKey hashes these options, so changing it forces a rebuild.
	FixEmit bool
}

// DefaultProjectOptions is DEFAULT_PROJECT_OPTIONS.
func DefaultProjectOptions() ProjectOptions {
	return ProjectOptions{OptimizedLoops: true, Luau: true}
}

// ProjectData is the interface of the same name in src/Shared/types.ts. transformerWatcher is left out, because
// transformer plugins are a later phase.
type ProjectData struct {
	IsPackage       bool
	NodeModulesPath string
	ProjectOptions  ProjectOptions
	ProjectPath     string
	// RojoConfigPath is "" where JS has undefined.
	RojoConfigPath string
	TsConfigPath   string
}
