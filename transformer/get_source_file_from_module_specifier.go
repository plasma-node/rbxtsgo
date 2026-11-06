package transformer

// Port of src/TSTransformer/util/getSourceFileFromModuleSpecifier.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/module"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs"
)

// getSourceFileFromModuleSpecifier returns nil where JS returns undefined. A resolved file goes through
// GetSourceFileForResolvedModule, because the program leaves out a package that duplicates another one, where JS keeps a
// redirect file under its own path. There the Go answer is the redirect's target.
func getSourceFileFromModuleSpecifier(state *TransformState, moduleSpecifier *ast.Node) *ast.SourceFile {
	symbol := state.Checker.GetSymbolAtLocation(moduleSpecifier)
	if symbol == nil {
		symbol = state.Checker.ResolveExternalModuleName(moduleSpecifier, nil)
	}
	if symbol != nil {
		declaration := symbol.ValueDeclaration()

		if declaration != nil && ast.IsModuleDeclaration(declaration) && ast.IsStringLiteralLike(declaration.Name()) {
			sourceFile := ast.GetSourceFileOfNode(moduleSpecifier)
			mode := state.Program.GetModeForUsageLocation(sourceFile, declaration.Name())
			resolvedModuleInfo := state.Program.GetResolvedModule(sourceFile, declaration.Name().Text(), mode)
			if resolvedModuleInfo != nil && resolvedModuleInfo.IsResolved() {
				return state.Program.GetSourceFileForResolvedModule(resolvedModuleInfo)
			}
		}

		if declaration != nil && ast.IsSourceFile(declaration) {
			return declaration.AsSourceFile()
		}
	}

	// Fallback for $getModuleTree when module is not referenced by any regular import
	if ast.IsStringLiteralLike(moduleSpecifier) {
		sourceFile := ast.GetSourceFileOfNode(moduleSpecifier)
		// This has to really resolve, as ts.resolveModuleName does. Program.GetResolvedModuleFromModuleSpecifier only
		// looks up imports the program saw, and this specifier isn't one, so it would always miss here.
		resolver := module.NewResolver(module.ResolverOptions{
			Host: &moduleSpecifierResolutionHost{
				fs:               state.Program.Host().FS(),
				currentDirectory: state.Program.GetCurrentDirectory(),
			},
			CompilerOptions: state.Options,
		})
		result, _, _ := resolver.ResolveModuleName(moduleSpecifier.Text(), sourceFile.FileName(), core.ResolutionModeNone, nil)
		if result != nil && result.IsResolved() {
			return state.Program.GetSourceFileForResolvedModule(result)
		}
	}
	return nil
}

// moduleSpecifierResolutionHost stands in for ts.sys.
type moduleSpecifierResolutionHost struct {
	fs               vfs.FS
	currentDirectory tspath.RootedDirectoryPath
}

func (h *moduleSpecifierResolutionHost) FS() vfs.FS { return h.fs }

func (h *moduleSpecifierResolutionHost) GetCurrentDirectory() tspath.RootedDirectoryPath {
	return h.currentDirectory
}
