package compile

// Declaration emit: the proxyProgram.emit call in compileFiles.ts, and the two afterDeclarations transformers from
// src/Project/transformers/builtin. TS 7 has no hook for custom transformers, so those run on the printed text: parse it,
// find the module specifiers, and replace them. The parser finds exactly the nodes the upstream visitor finds, and text
// outside them can't change.

import (
	"context"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/module"
	"github.com/microsoft/TypeScript/tsc/internal/parser"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
)

// EmitDeclarations writes the declaration files of files and returns the paths it wrote. Like upstream it ignores the
// emit's own diagnostics: pre-emit diagnostics have reported the same problems by the time this runs.
func EmitDeclarations(ctx context.Context, c *Context, files []*ast.SourceFile) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	program := c.Project.Program()
	options := c.Project.Options()
	rewriter := newDeclarationRewriter(program, options)
	type output struct {
		path, text string
	}
	outputs := make([][]output, len(files))
	index := make(map[*ast.SourceFile]int, len(files))
	for i, file := range files {
		index[file] = i
	}
	program.Emit(ctx, compiler.EmitOptions{
		TargetSourceFiles: files,
		EmitOnly:          compiler.EmitOnlyDts,
		WriteFile: func(fileName tspath.RootedFilePath, text string, data *compiler.WriteFileData) error {
			name := fileName.AsString()
			if data != nil && data.SourceFile != nil {
				if i, ok := index[data.SourceFile]; ok {
					if options.RootDir == "" && strings.HasSuffix(name, ".d.ts") {
						// See 0004: TS 5 and TS 7 pick a different default rootDir, and the translator has TS 5's.
						name = c.Translator.GetOutputDeclarationPath(nodepath.Native.Normalize(data.SourceFile.FileName().AsString()))
					}
					if strings.HasSuffix(name, ".d.ts") {
						text = rewriter.rewrite(text, data.SourceFile)
					}
					outputs[i] = append(outputs[i], output{name, text})
				}
			}
			return nil
		},
	})
	var written []string
	for _, list := range outputs {
		for _, item := range list {
			target := nodepath.Native.Normalize(item.path)
			if c.Data.ProjectOptions.WriteOnlyChanged {
				if current, err := os.ReadFile(target); err == nil && string(current) == item.text {
					continue
				}
			}
			if err := os.MkdirAll(nodepath.Native.Dirname(target), 0o777); err != nil {
				return written, err
			}
			if err := os.WriteFile(target, []byte(item.text), 0o666); err != nil {
				return written, err
			}
			written = append(written, target)
		}
	}
	return written, nil
}

type declarationRewriter struct {
	program  *compiler.Program
	options  *core.CompilerOptions
	resolver *module.DefaultResolver
	active   bool
}

func newDeclarationRewriter(program *compiler.Program, options *core.CompilerOptions) *declarationRewriter {
	return &declarationRewriter{
		program: program,
		options: options,
		resolver: module.NewResolver(module.ResolverOptions{
			Host:            &resolutionHost{fs: program.Host().FS(), currentDirectory: program.GetCurrentDirectory()},
			CompilerOptions: options,
		}),
		// `if (!compilerOptions.baseUrl && !compilerOptions.paths) return sourceFile;`. Normalization turns baseUrl into paths.
		active: options.Paths != nil,
	}
}

type resolutionHost struct {
	fs               vfs.FS
	currentDirectory tspath.RootedDirectoryPath
}

func (h *resolutionHost) FS() vfs.FS                                      { return h.fs }
func (h *resolutionHost) GetCurrentDirectory() tspath.RootedDirectoryPath { return h.currentDirectory }

// typeReference finds the reference directives transformTypeReferenceDirectives changes. They only ever sit in the
// comments at the top of a file, which is where the printer puts them.
var typeReference = regexp.MustCompile(`(?m)^(/// <reference types=")types("[^\n]*)$`)

type textEdit struct {
	start, end int
	text       string
}

func (r *declarationRewriter) rewrite(text string, source *ast.SourceFile) string {
	var edits []textEdit
	if r.active {
		parsed := parser.ParseSourceFile(ast.SourceFileParseOptions{
			FileName: source.FileName(),
			PathKey:  source.PathKey() + ".d.ts",
		}, text, core.ScriptKindTS)
		var visit func(node *ast.Node) bool
		replace := func(literal *ast.Node) {
			if literal == nil || !ast.IsStringLiteralLike(literal) {
				return
			}
			if updated, ok := r.update(literal.Text(), source); ok {
				edits = append(edits, textEdit{scanner.GetTokenPosOfNode(literal, parsed, false), literal.End(), `"` + updated + `"`})
			}
		}
		visit = func(node *ast.Node) bool {
			switch {
			case ast.IsExternalModuleReference(node):
				replace(node.Expression())
				return false
			case ast.IsImportDeclaration(node):
				replace(node.ModuleSpecifier())
				return false
			case ast.IsExportDeclaration(node):
				if node.ModuleSpecifier() != nil {
					replace(node.ModuleSpecifier())
					return false
				}
			case ast.IsImportTypeNode(node):
				if argument := node.AsImportTypeNode().Argument; argument != nil && ast.IsLiteralTypeNode(argument) {
					// `return !text ? node : update(...)`. An empty specifier stays.
					literal := argument.AsLiteralTypeNode().Literal
					if ast.IsStringLiteralLike(literal) && literal.Text() != "" {
						replace(literal)
					}
				}
			}
			node.ForEachChild(visit)
			return false
		}
		parsed.AsNode().ForEachChild(visit)
	}
	if len(edits) > 0 {
		sort.Slice(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
		var b strings.Builder
		position := 0
		for _, edit := range edits {
			b.WriteString(text[position:edit.start])
			b.WriteString(edit.text)
			position = edit.end
		}
		b.WriteString(text[position:])
		text = b.String()
	}
	return typeReference.ReplaceAllString(text, "${1}@rbxts/types${2}")
}

// update is update() in transformPaths.ts. Upstream also takes a failed lookup that is a URL, which no file system
// lookup here produces.
func (r *declarationRewriter) update(moduleName string, source *ast.SourceFile) (string, bool) {
	resolved, _, _ := r.resolver.ResolveModuleName(moduleName, source.FileName(), core.ResolutionModeNone, nil)
	if resolved == nil || !resolved.IsResolved() || resolved.IsExternalLibraryImport {
		return "", false
	}
	path := nodepath.Native
	fileDir := tspath.NormalizePath(path.Dirname(source.FileName().AsString()))
	resolvedFileName := resolved.ResolvedFileName.AsString()
	modulePath := path.Dirname(resolvedFileName)
	p := tspath.NormalizeSlashes(tspath.NormalizePath(path.Join(path.Relative(fileDir, modulePath), path.Basename(resolvedFileName, ""))))
	if resolved.Extension != "" && slices.Contains(r.implicitExtensions(), resolved.Extension) {
		p = p[:len(p)-len(resolved.Extension)]
	}
	if p == "" {
		return "", false
	}
	if p[0] != '.' {
		p = "./" + p
	}
	return p, true
}

// implicitExtensions is getImplicitExtensions.
func (r *declarationRewriter) implicitExtensions() []string {
	result := []string{".ts", ".d.ts"}
	if r.options.AllowJs.IsTrue() {
		result = append(result, ".js")
	}
	allowJsx := r.options.Jsx != core.JsxEmitNone
	if allowJsx {
		result = append(result, ".tsx")
	}
	if r.options.AllowJs.IsTrue() && allowJsx {
		result = append(result, ".jsx")
	}
	if r.options.ResolveJsonModule.IsTrue() {
		result = append(result, ".json")
	}
	return result
}
