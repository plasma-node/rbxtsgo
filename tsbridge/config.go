package tsbridge

import (
	"fmt"

	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/core"
)

// NormalizeOptions returns a copy of opts with the options TS 7 removed rewritten, so users don't have
// to edit a tsconfig that works with rbxtsc. It must run after parsing and before compiler.NewProgram,
// whose verifyCompilerOptions reports TS5102 and TS5108 for the old options. Notes describe each change,
// for verbose output. Running it twice changes nothing the second time.
//
// A classic module option is left alone. commonjs stays, see 0003 item 4.
func NormalizeOptions(in *core.CompilerOptions) (*core.CompilerOptions, []string) {
	opts := in.Clone()
	var notes []string

	// A config written for TS 7 compiles as the classic pair, so one set of programs is accepted and the Luau is the same.
	// Left as written, a package's "exports" would change import paths, and top-level await would compile. See 0007.
	if opts.Module == core.ModuleKindPreserve && opts.ModuleResolution == core.ModuleResolutionKindBundler {
		opts.Module = core.ModuleKindCommonJS
		opts.ModuleResolution = core.ModuleResolutionKindNode10
		notes = append(notes, "module preserve and moduleResolution bundler -> the classic commonjs and node10")
	}
	if opts.BaseUrl != "" {
		paths := collections.NewOrderedMapWithSizeHint[string, []string](1)
		if opts.Paths != nil {
			for key, value := range opts.Paths.Entries() {
				paths.Set(key, value)
			}
		}
		if !paths.Has("*") {
			paths.Set("*", []string{"./*"})
		}
		opts.Paths = paths
		// Before TS 7, paths were relative to baseUrl when it was set.
		opts.PathsBasePath = opts.BaseUrl
		notes = append(notes, fmt.Sprintf(`baseUrl %q -> paths {"*": ["./*"]} based there`, opts.BaseUrl))
		opts.BaseUrl = ""
	}
	if opts.ModuleResolution == core.ModuleResolutionKindNode10 {
		opts.ModuleResolution = core.ModuleResolutionKindBundler
		notes = append(notes, "moduleResolution node10 -> bundler")
		// node10 ignores these package.json fields, so deep imports only resolve the same way with them off.
		if opts.ResolvePackageJsonExports.IsUnknown() {
			opts.ResolvePackageJsonExports = core.TSFalse
			notes = append(notes, "resolvePackageJsonExports unset -> false")
		}
		if opts.ResolvePackageJsonImports.IsUnknown() {
			opts.ResolvePackageJsonImports = core.TSFalse
			notes = append(notes, "resolvePackageJsonImports unset -> false")
		}
	}
	if !opts.DownlevelIteration.IsUnknown() {
		opts.DownlevelIteration = core.TSUnknown
		notes = append(notes, "downlevelIteration dropped")
	}
	if opts.Types == nil {
		// TS 5 included every package under typeRoots by default. TS 7 includes none.
		opts.Types = []string{"*"}
		notes = append(notes, `types unset -> ["*"]`)
	}
	// Not removed, but the default changed in TS 6, which adds TS2882 errors that rbxtsc doesn't report.
	if opts.NoUncheckedSideEffectImports.IsUnknown() {
		opts.NoUncheckedSideEffectImports = core.TSFalse
		notes = append(notes, "noUncheckedSideEffectImports unset -> false")
	}
	return opts, notes
}
