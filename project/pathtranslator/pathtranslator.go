// Package pathtranslator is a port of @roblox-ts/path-translator 1.1.0, which maps between source and output
// paths. Quirks are kept on purpose, including what PathInfo does to a path with no directory part. Output
// must match the JS version byte for byte, so the tests compare it with the real npm package.
package pathtranslator

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
)

const (
	tsExt          = ".ts"
	tsxExt         = ".tsx"
	dExt           = ".d"
	dtsExt         = dExt + tsExt
	transformedExt = ".transformed"
	luaExt         = ".lua"
	luauExt        = ".luau"
	indexName      = "index"
	initName       = "init"
)

type pathInfo struct {
	path     *nodepath.Path
	dirName  string
	fileName string
	exts     []string
}

func newPathInfo(path *nodepath.Path, filePath string) *pathInfo {
	dirName := path.Dirname(filePath)
	// The slice start can run past the end. JS's slice gives "" there.
	rest := ""
	if start := len(dirName) + len(path.Sep); start < len(filePath) {
		rest = filePath[start:]
	}
	parts := strings.Split(rest, ".")
	exts := make([]string, 0, len(parts)-1)
	for _, part := range parts[1:] {
		exts = append(exts, "."+part)
	}
	return &pathInfo{path: path, dirName: dirName, fileName: parts[0], exts: exts}
}

// peek returns "" where JS returns undefined. No extension is ever "", since each starts with a dot.
func (p *pathInfo) peek(depth int) string {
	i := len(p.exts) - (depth + 1)
	if i < 0 {
		return ""
	}
	return p.exts[i]
}

func (p *pathInfo) pop() string {
	if len(p.exts) == 0 {
		return ""
	}
	last := p.exts[len(p.exts)-1]
	p.exts = p.exts[:len(p.exts)-1]
	return last
}

func (p *pathInfo) push(ext string) { p.exts = append(p.exts, ext) }

func (p *pathInfo) join() string {
	return p.path.Join(p.dirName, p.fileName+strings.Join(p.exts, ""))
}

// Translator is PathTranslator.
type Translator struct {
	RootDir string
	OutDir  string
	// BuildInfoOutputPath is "" where JS has undefined.
	BuildInfoOutputPath string
	Declaration         bool
	UseLuauExtension    bool

	path *nodepath.Path
}

// New makes a Translator using this platform's path rules.
func New(rootDir, outDir, buildInfoOutputPath string, declaration, useLuauExtension bool) *Translator {
	return NewWithPath(nodepath.Native, rootDir, outDir, buildInfoOutputPath, declaration, useLuauExtension)
}

// NewWithPath makes a Translator with the given path rules. Tests use it to cover both platforms on one machine.
func NewWithPath(path *nodepath.Path, rootDir, outDir, buildInfoOutputPath string, declaration, useLuauExtension bool) *Translator {
	return &Translator{
		RootDir:             rootDir,
		OutDir:              outDir,
		BuildInfoOutputPath: buildInfoOutputPath,
		Declaration:         declaration,
		UseLuauExtension:    useLuauExtension,
		path:                path,
	}
}

func (t *Translator) luauExtension() string {
	if t.UseLuauExtension {
		return luauExt
	}
	return luaExt
}

func (t *Translator) toOut(info *pathInfo) string {
	return t.path.Join(t.OutDir, t.path.Relative(t.RootDir, info.join()))
}

func (t *Translator) toRoot(info *pathInfo) string {
	return t.path.Join(t.RootDir, t.path.Relative(t.OutDir, info.join()))
}

// GetOutputPath maps an input path to an output path. .ts(x) but not .d.ts(x) becomes .lua(u),
// index becomes init, and src/* becomes out/*.
func (t *Translator) GetOutputPath(filePath string) string {
	info := newPathInfo(t.path, filePath)
	if (info.peek(0) == tsExt || info.peek(0) == tsxExt) && info.peek(1) != dExt {
		info.pop()
		if info.fileName == indexName {
			info.fileName = initName
		}
		info.push(t.luauExtension())
	}
	return t.toOut(info)
}

// GetOutputDeclarationPath maps an input path to an output .d.ts path.
func (t *Translator) GetOutputDeclarationPath(filePath string) string {
	info := newPathInfo(t.path, filePath)
	if (info.peek(0) == tsExt || info.peek(0) == tsxExt) && info.peek(1) != dExt {
		info.pop()
		info.push(dtsExt)
	}
	return t.toOut(info)
}

// insert is Array.prototype.splice(index, 0, value), which clamps a negative index to 0.
func (p *pathInfo) insert(index int, value string) {
	if index < 0 {
		index = 0
	}
	p.exts = append(p.exts[:index:index], append([]string{value}, p.exts[index:]...)...)
}

// GetOutputTransformedPath maps an input path to an output .transformed.ts(x) path.
func (t *Translator) GetOutputTransformedPath(filePath string) string {
	info := newPathInfo(t.path, filePath)
	if info.peek(1) == dExt {
		info.insert(len(info.exts)-2, transformedExt)
	} else {
		info.insert(len(info.exts)-1, transformedExt)
	}
	return t.toOut(info)
}

// GetInputPaths maps an output path to the input paths it could have come from.
func (t *Translator) GetInputPaths(filePath string) []string {
	var possible []string
	info := newPathInfo(t.path, filePath)
	tryWith := func(ext string) {
		info.push(ext)
		possible = append(possible, t.toRoot(info))
		info.pop()
	}

	// index.*.lua(u) cannot come from a .ts file
	if info.peek(0) == t.luauExtension() && info.fileName != indexName {
		info.pop()
		tryWith(tsExt)
		tryWith(tsxExt)
		if info.fileName == initName {
			original := info.fileName
			info.fileName = indexName
			tryWith(tsExt)
			tryWith(tsxExt)
			info.fileName = original
		}
		info.push(t.luauExtension())
	}

	if t.Declaration {
		if (info.peek(0) == tsExt || info.peek(0) == tsxExt) && info.peek(1) == dExt {
			ext := info.pop()
			info.pop()
			tryWith(tsExt)
			tryWith(tsxExt)
			info.push(dExt)
			info.push(ext)
		}
	}

	return append(possible, t.toRoot(info))
}

// GetImportPath maps a source path to an import path, which is passed to the Rojo resolver and used virtually,
// so it might not exist. Inside node_modules the compiled file is assumed to be a sibling of filePath.
func (t *Translator) GetImportPath(filePath string, isNodeModule bool) string {
	info := newPathInfo(t.path, filePath)
	if info.peek(0) == tsExt || info.peek(0) == tsxExt {
		info.pop()
		if info.peek(0) == dExt {
			info.pop()
		}
		if info.fileName == indexName {
			info.fileName = initName
		}
		info.push(t.luauExtension())
	}
	if isNodeModule {
		return info.join()
	}
	return t.toOut(info)
}
