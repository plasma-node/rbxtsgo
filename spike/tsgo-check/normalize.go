package main

import (
	"fmt"

	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
)

// normalize rewrites options that TS 7 removed, so users don't have to edit their tsconfig.
// It must run before compiler.NewProgram, whose verifyCompilerOptions reports them.
func normalize(config *tsoptions.ParsedCommandLine) []string {
	var notes []string
	opts := *config.CompilerOptions()

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
		// node10 ignores these package.json fields, so turn them off to resolve the same way.
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
		opts.Types = []string{"*"}
		notes = append(notes, `types unset -> ["*"]`)
	}

	// Not removed, but its default changed in TS 6, which adds errors that rbxtsc doesn't report.
	if opts.NoUncheckedSideEffectImports.IsUnknown() {
		opts.NoUncheckedSideEffectImports = core.TSFalse
		notes = append(notes, "noUncheckedSideEffectImports unset -> false")
	}

	config.SetCompilerOptions(&opts)
	return notes
}
