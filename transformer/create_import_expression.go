package transformer

// Port of src/TSTransformer/util/createImportExpression.ts.

import (
	"slices"
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/rojoresolver"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

func getAbsoluteImport(moduleRbxPath rojoresolver.RbxPath) []luau.Expression {
	pathExpressions := []luau.Expression{}
	if len(moduleRbxPath) == 0 || moduleRbxPath[0] == "" {
		panic(shared.AssertionMessage(""))
	}
	serviceName := moduleRbxPath[0]
	pathExpressions = append(pathExpressions, createGetService(serviceName))
	for i := 1; i < len(moduleRbxPath); i++ {
		pathExpressions = append(pathExpressions, luau.NewString(moduleRbxPath[i]))
	}
	return pathExpressions
}

func getRelativeImport(sourceRbxPath rojoresolver.RbxPath, moduleRbxPath rojoresolver.RbxPath) []luau.Expression {
	relativePath := rojoresolver.Relative(sourceRbxPath, moduleRbxPath)

	// create descending path pieces
	path := []string{}
	i := 0
	for i < len(relativePath) && relativePath[i] == rojoresolver.RbxPathParent {
		path = append(path, shared.ParentField)
		i++
	}

	pathExpressions := []luau.Expression{propertyAccessExpressionChain(luau.Globals.Script, path)}

	// create descending path pieces
	for ; i < len(relativePath); i++ {
		pathPart := relativePath[i]
		if pathPart == rojoresolver.RbxPathParent {
			panic(shared.AssertionMessage(""))
		}
		pathExpressions = append(pathExpressions, luau.NewString(pathPart))
	}

	return pathExpressions
}

func validateModule(state *TransformState, scope string) bool {
	scopedModules := nodepath.Native.Join(state.Data.NodeModulesPath, scope)
	for _, typeRoot := range state.Options.TypeRoots {
		if nodepath.Native.Normalize(scopedModules) == nodepath.Native.Normalize(typeRoot.AsString()) {
			return true
		}
	}
	return false
}

func findRelativeRbxPath(moduleOutPath string, pkgRojoResolvers []*rojoresolver.Resolver) rojoresolver.RbxPath {
	for _, pkgRojoResolver := range pkgRojoResolvers {
		relativeRbxPath := pkgRojoResolver.GetRbxPathFromFilePath(moduleOutPath)
		if relativeRbxPath != nil {
			return relativeRbxPath
		}
	}
	return nil
}

func getNodeModulesImportParts(state *TransformState, sourceFile *ast.SourceFile, moduleSpecifier *ast.Node, moduleOutPath string) []luau.Expression {
	moduleScope := strings.Split(nodepath.Native.Relative(state.Data.NodeModulesPath, moduleOutPath), nodepath.Native.Sep)[0]
	if moduleScope == "" {
		panic(shared.AssertionMessage(""))
	}

	if !strings.HasPrefix(moduleScope, "@") {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoUnscopedModule.At(shared.AtNode(moduleSpecifier)))
		return []luau.Expression{luau.NewNone()}
	}

	if !validateModule(state, moduleScope) {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoInvalidModule.At(shared.AtNode(moduleSpecifier)))
		return []luau.Expression{luau.NewNone()}
	}

	if state.Compile.ProjectType == shared.ProjectTypePackage {
		relativeRbxPath := findRelativeRbxPath(moduleOutPath, state.Compile.PkgRojoResolvers)
		if relativeRbxPath == nil {
			state.Diagnostic.AddDiagnostic(
				shared.Errors.NoRojoData.At(shared.AtNode(moduleSpecifier), nodepath.Native.Relative(state.Data.ProjectPath, moduleOutPath), true),
			)
			return []luau.Expression{luau.NewNone()}
		}

		if len(relativeRbxPath) == 0 || relativeRbxPath[0] == "" {
			panic(shared.AssertionMessage(""))
		}
		moduleName := relativeRbxPath[0]

		return []luau.Expression{
			propertyAccessExpressionChain(
				luau.NewCall(state.TS(moduleSpecifier.Parent, "getModule"),
					luau.Globals.Script,
					luau.NewString(moduleScope),
					luau.NewString(moduleName),
				),
				relativeRbxPath[1:],
			),
		}
	}
	moduleRbxPath := state.RojoResolver().GetRbxPathFromFilePath(moduleOutPath)
	if moduleRbxPath == nil {
		state.Diagnostic.AddDiagnostic(
			shared.Errors.NoRojoData.At(shared.AtNode(moduleSpecifier), nodepath.Native.Relative(state.Data.ProjectPath, moduleOutPath), true),
		)
		return []luau.Expression{luau.NewNone()}
	}

	indexOfScope := slices.Index(moduleRbxPath, moduleScope)
	if indexOfScope <= 0 || moduleRbxPath[indexOfScope-1] != shared.NodeModules {
		state.Diagnostic.AddDiagnostic(
			shared.Errors.NoPackageImportWithoutScope.At(
				shared.AtNode(moduleSpecifier),
				nodepath.Native.Relative(state.Data.ProjectPath, moduleOutPath),
				moduleRbxPath,
			),
		)
		return []luau.Expression{luau.NewNone()}
	}

	return getProjectImportParts(state, sourceFile, moduleSpecifier, moduleOutPath, moduleRbxPath)
}

func getProjectImportParts(state *TransformState, sourceFile *ast.SourceFile, moduleSpecifier *ast.Node, moduleOutPath string, moduleRbxPath rojoresolver.RbxPath) []luau.Expression {
	moduleRbxType := state.RojoResolver().GetRbxTypeFromFilePath(moduleOutPath)
	if moduleRbxType == rojoresolver.RbxTypeScript || moduleRbxType == rojoresolver.RbxTypeLocalScript {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoNonModuleImport.At(shared.AtNode(moduleSpecifier)))
		return []luau.Expression{luau.NewNone()}
	}

	sourceOutPath := state.PathTranslator.GetOutputPath(sourceFile.FileName().AsString())
	sourceRbxPath := state.RojoResolver().GetRbxPathFromFilePath(sourceOutPath)
	if sourceRbxPath == nil {
		state.Diagnostic.AddDiagnostic(
			shared.Errors.NoRojoData.At(shared.AtNode(sourceFile.AsNode()), nodepath.Native.Relative(state.Data.ProjectPath, sourceOutPath), false),
		)
		return []luau.Expression{luau.NewNone()}
	}

	if state.Compile.ProjectType == shared.ProjectTypeGame {
		// in the case of `import("")`, don't do network type check
		// as the call may be guarded by runtime RunService checks
		if !ast.IsImportCall(moduleSpecifier.Parent) &&
			state.RojoResolver().GetNetworkType(moduleRbxPath) == rojoresolver.NetworkTypeServer &&
			state.RojoResolver().GetNetworkType(sourceRbxPath) != rojoresolver.NetworkTypeServer {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoServerImport.At(shared.AtNode(moduleSpecifier)))
			return []luau.Expression{luau.NewNone()}
		}

		fileRelation := state.RojoResolver().GetFileRelation(sourceRbxPath, moduleRbxPath)
		if fileRelation == rojoresolver.FileRelationOutToOut || fileRelation == rojoresolver.FileRelationInToOut {
			return getAbsoluteImport(moduleRbxPath)
		} else if fileRelation == rojoresolver.FileRelationInToIn {
			return getRelativeImport(sourceRbxPath, moduleRbxPath)
		}
		state.Diagnostic.AddDiagnostic(shared.Errors.NoIsolatedImport.At(shared.AtNode(moduleSpecifier)))
		return []luau.Expression{luau.NewNone()}
	}
	return getRelativeImport(sourceRbxPath, moduleRbxPath)
}

// isInsideNodeModules is ts.isInsideNodeModules from TS 5.5.3.
func isInsideNodeModules(fileOrDirectory string) bool {
	return slices.Contains(tspath.GetPathComponents(fileOrDirectory), "node_modules")
}

func getImportParts(state *TransformState, sourceFile *ast.SourceFile, moduleSpecifier *ast.Node) []luau.Expression {
	moduleFile := getSourceFileFromModuleSpecifier(state, moduleSpecifier)
	if moduleFile == nil {
		state.Diagnostic.AddDiagnostic(shared.Errors.NoModuleSpecifierFile.At(shared.AtNode(moduleSpecifier)))
		return []luau.Expression{luau.NewNone()}
	}

	virtualPath := state.GuessVirtualPath(moduleFile.FileName().AsString())
	if virtualPath == "" {
		virtualPath = moduleFile.FileName().AsString()
	}

	if isInsideNodeModules(virtualPath) {
		mappedPath, ok := state.Compile.NodeModulesPathMapping[state.CaseSensitivity.Canonicalize(nodepath.Native.Normalize(virtualPath))]
		if !ok {
			mappedPath = virtualPath
		}
		moduleOutPath := state.PathTranslator.GetImportPath(
			mappedPath,
			/* isNodeModule */ true,
		)
		return getNodeModulesImportParts(state, sourceFile, moduleSpecifier, moduleOutPath)
	}
	moduleOutPath := state.PathTranslator.GetImportPath(virtualPath, false)
	moduleRbxPath := state.RojoResolver().GetRbxPathFromFilePath(moduleOutPath)
	if moduleRbxPath == nil {
		state.Diagnostic.AddDiagnostic(
			shared.Errors.NoRojoData.At(shared.AtNode(moduleSpecifier), nodepath.Native.Relative(state.Data.ProjectPath, moduleOutPath), false),
		)
		return []luau.Expression{luau.NewNone()}
	}
	return getProjectImportParts(state, sourceFile, moduleSpecifier, moduleOutPath, moduleRbxPath)
}

func createImportExpression(state *TransformState, sourceFile *ast.SourceFile, moduleSpecifier *ast.Node) luau.IndexableExpression {
	parts := getImportParts(state, sourceFile, moduleSpecifier)
	parts = slices.Insert(parts, 0, luau.Expression(luau.Globals.Script))
	return luau.NewCall(state.TS(moduleSpecifier.Parent, "import"), parts...)
}
