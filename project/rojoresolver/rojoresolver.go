// Package rojoresolver is a port of @roblox-ts/rojo-resolver 1.1.0, which maps file paths to Roblox instance
// paths using a Rojo project file. Output must match the JS version byte for byte, so the tests compare it with
// the real npm package.
//
// Where the JS throws, this returns an error. The error's text is not meant to match, since in rbxtsc those are
// uncaught exceptions rather than diagnostics.
package rojoresolver

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
)

const (
	luaExt  = ".lua"
	luauExt = ".luau"
	jsonExt = ".json"
	tomlExt = ".toml"

	initName = "init"

	serverSubext = ".server"
	clientSubext = ".client"

	rojoDefaultName = "default.project.json"
	rojoOldName     = "roblox-project.json"
)

var rojoFileRegex = regexp.MustCompile(`^.+\.project\.json$`)

func isModuleExt(ext string) bool { return ext == luauExt || ext == jsonExt || ext == tomlExt }
func isScriptExt(ext string) bool { return ext == luauExt }

// RbxType is the enum of the same name, with the same values.
type RbxType int

const (
	RbxTypeModuleScript RbxType = iota
	RbxTypeScript
	RbxTypeLocalScript
	RbxTypeUnknown
)

// FileRelation is the enum of the same name, with the same values.
type FileRelation int

const (
	FileRelationOutToOut FileRelation = iota // absolute
	FileRelationOutToIn                      // error
	FileRelationInToOut                      // absolute
	FileRelationInToIn                       // relative
)

// NetworkType is the enum of the same name, with the same values.
type NetworkType int

const (
	NetworkTypeUnknown NetworkType = iota
	NetworkTypeClient
	NetworkTypeServer
)

// RbxPath is a Roblox tree path. A nil RbxPath stands for undefined.
type RbxPath = []string

// RbxPathParent marks a step up in a relative path, where JS has a Symbol. It can't be an instance name,
// because it has a NUL byte.
const RbxPathParent = "\x00Parent"

var (
	defaultIsolatedContainers = []RbxPath{
		{"StarterPack"},
		{"StarterGui"},
		{"StarterPlayer", "StarterPlayerScripts"},
		{"StarterPlayer", "StarterCharacterScripts"},
		{"StarterPlayer", "StarterCharacter"},
		{"PluginDebugService"},
	}
	clientContainers = []RbxPath{{"StarterPack"}, {"StarterGui"}, {"StarterPlayer"}}
	serverContainers = []RbxPath{{"ServerStorage"}, {"ServerScriptService"}}
)

// PartitionInfo is one $path that maps a folder.
type PartitionInfo struct {
	RbxPath RbxPath
	FsPath  string
}

var path = nodepath.Native

func stripRojoExts(filePath string) string {
	ext := path.Extname(filePath)
	if isModuleExt(ext) {
		filePath = filePath[:len(filePath)-len(ext)]
		if isScriptExt(ext) {
			subext := path.Extname(filePath)
			if subext == serverSubext || subext == clientSubext {
				filePath = filePath[:len(filePath)-len(subext)]
			}
		}
	}
	return filePath
}

func arrayStartsWith(a, b []string) bool {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isPathDescendantOf(filePath, dirPath string) bool {
	return dirPath == filePath || !strings.HasPrefix(path.Relative(dirPath, filePath), "..")
}

func convertToLuau(filePath string) string {
	ext := path.Extname(filePath)
	if ext == luaExt {
		return filePath[:len(filePath)-len(ext)] + luauExt
	}
	return filePath
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// readDir returns directory entries in the order Node's readdirSync does. libuv sorts by byte value on Unix.
// On Windows it keeps the OS order, which on NTFS ignores case, so os.ReadDir's byte sort would differ.
func readDirEntries(dir string) ([]os.DirEntry, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	if runtime.GOOS != "windows" {
		slices.SortFunc(entries, func(a, b os.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	}
	return entries, nil
}

func readDir(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	if runtime.GOOS != "windows" {
		slices.Sort(names)
	}
	return names, nil
}

// FindRojoConfigFilePath is RojoResolver.findRojoConfigFilePath. The path is "" where JS has undefined.
func FindRojoConfigFilePath(projectPath string) (string, []string, error) {
	var warnings []string
	defaultPath := path.Join(projectPath, rojoDefaultName)
	if pathExists(defaultPath) {
		return defaultPath, warnings, nil
	}
	names, err := readDir(projectPath)
	if err != nil {
		return "", nil, err
	}
	var candidates []string
	for _, fileName := range names {
		if fileName != rojoDefaultName && (fileName == rojoOldName || rojoFileRegex.MatchString(fileName)) {
			candidates = append(candidates, path.Join(projectPath, fileName))
		}
	}
	if len(candidates) > 1 {
		warnings = append(warnings, "Multiple *.project.json files found, using "+candidates[0])
	}
	if len(candidates) == 0 {
		return "", warnings, nil
	}
	return candidates[0], warnings, nil
}

// Resolver is RojoResolver.
type Resolver struct {
	warnings          []string
	rbxPath           []string
	partitions        []PartitionInfo
	filePathToRbxPath map[string]RbxPath
	// IsGame is true once a tree has "$className": "DataModel".
	IsGame bool
}

func newResolver() *Resolver {
	return &Resolver{filePathToRbxPath: map[string]RbxPath{}}
}

// Warnings is getWarnings.
func (r *Resolver) Warnings() []string { return r.warnings }

// Partitions is getPartitions: last seen first, because the JS uses unshift.
func (r *Resolver) Partitions() []PartitionInfo { return r.partitions }

// FromPath is RojoResolver.fromPath.
func FromPath(rojoConfigFilePath string) (*Resolver, error) {
	r := newResolver()
	if err := r.parseConfig(path.Resolve(rojoConfigFilePath), true); err != nil {
		return nil, err
	}
	return r, nil
}

// Synthetic is RojoResolver.synthetic: a resolver for ProjectType.Package, which forces all imports to be relative.
func Synthetic(basePath string) (*Resolver, error) {
	r := newResolver()
	tree := &object{keys: []string{"$path"}, values: map[string]any{"$path": basePath}}
	if err := r.parseTree(basePath, "", tree, true); err != nil {
		return nil, err
	}
	return r, nil
}

// FromTreeJSON is RojoResolver.fromTree, with the tree given as JSON text to keep its property order.
func FromTreeJSON(basePath, treeJSON string) (*Resolver, error) {
	value, err := parseJSON(treeJSON)
	if err != nil {
		return nil, err
	}
	r := newResolver()
	if err := r.parseTree(basePath, "", value, true); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Resolver) parseConfig(rojoConfigFilePath string, doNotPush bool) error {
	realPath, err := filepath.EvalSymlinks(rojoConfigFilePath)
	if err != nil {
		return err
	}
	if !pathExists(realPath) {
		r.warnings = append(r.warnings, fmt.Sprintf(`RojoResolver: Path does not exist "%s"`, rojoConfigFilePath))
		return nil
	}
	data, err := os.ReadFile(realPath)
	if err != nil {
		return err
	}
	configJSON, parseErr := parseJSON(string(data))
	if parseErr != nil {
		// The JS has try/finally with no catch: the finally block warns, then the SyntaxError propagates.
		r.warnings = append(r.warnings, "RojoResolver: Invalid configuration! "+errorsText(validateRojoConfig(nil)))
		return fmt.Errorf("%s: %w", realPath, parseErr)
	}
	if errs := validateRojoConfig(configJSON); errs != nil {
		r.warnings = append(r.warnings, "RojoResolver: Invalid configuration! "+errorsText(errs))
		return nil
	}
	config := configJSON.(*object)
	return r.parseTree(path.Dirname(rojoConfigFilePath), config.values["name"].(string), config.values["tree"], doNotPush)
}

func (r *Resolver) parseTree(basePath, name string, treeValue any, doNotPush bool) error {
	if !doNotPush {
		r.rbxPath = append(r.rbxPath, name)
	}
	if tree, ok := treeValue.(*object); ok {
		if value, exists := tree.get("$path"); exists {
			itemPath, err := pathValue(value)
			if err != nil {
				return err
			}
			if err := r.parsePath(path.Resolve(basePath, itemPath)); err != nil {
				return err
			}
		}
		if className, _ := tree.values["$className"].(string); className == "DataModel" {
			r.IsGame = true
		}
		for _, childName := range tree.keys {
			if !strings.HasPrefix(childName, "$") {
				if err := r.parseTree(basePath, childName, tree.values[childName], false); err != nil {
					return err
				}
			}
		}
	}
	if !doNotPush {
		r.rbxPath = r.rbxPath[:len(r.rbxPath)-1]
	}
	return nil
}

func pathValue(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case *object:
		if optional, ok := v.values["optional"].(string); ok {
			return optional, nil
		}
	}
	// path.resolve throws here, on a $path object with no "optional".
	return "", errors.New(`$path must be a string or have an "optional" string`)
}

func (r *Resolver) parsePath(itemPath string) error {
	itemPath = convertToLuau(itemPath)
	realPath := itemPath
	exists := pathExists(itemPath)
	if exists {
		resolved, err := filepath.EvalSymlinks(itemPath)
		if err != nil {
			return err
		}
		realPath = resolved
	}
	ext := path.Extname(itemPath)
	if isModuleExt(ext) {
		r.filePathToRbxPath[itemPath] = append(RbxPath{}, r.rbxPath...)
		return nil
	}
	isDirectory := false
	if pathExists(realPath) {
		info, err := os.Stat(realPath)
		if err != nil {
			return err
		}
		isDirectory = info.IsDir()
	}
	if isDirectory {
		names, err := readDir(realPath)
		if err != nil {
			return err
		}
		if slices.Contains(names, rojoDefaultName) {
			return r.parseConfig(path.Join(itemPath, rojoDefaultName), true)
		}
	}
	r.partitions = slices.Insert(r.partitions, 0, PartitionInfo{FsPath: itemPath, RbxPath: append(RbxPath{}, r.rbxPath...)})
	if isDirectory {
		return r.searchDirectory(itemPath, "")
	}
	return nil
}

func (r *Resolver) searchDirectory(directory, item string) error {
	realPath, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return err
	}
	return r.searchRealDirectory(directory, realPath, item)
}

// searchRealDirectory is the walk of searchDirectory. The JS calls realpathSync and statSync on every child, and doing
// that here took most of a rebuild on Windows (see 0006). The entry's own type gives the same answer for anything that
// isn't a link, so only links pay for it.
func (r *Resolver) searchRealDirectory(directory, realPath, item string) error {
	children, err := readDirEntries(realPath)
	if err != nil {
		return err
	}

	if slices.ContainsFunc(children, func(child os.DirEntry) bool { return child.Name() == rojoDefaultName }) {
		return r.parseConfig(path.Join(directory, rojoDefaultName), false)
	}

	if item != "" {
		r.rbxPath = append(r.rbxPath, item)
	}

	kinds := make([]childKind, len(children))
	for i, child := range children {
		kind, err := kindOfChild(path.Join(directory, child.Name()), path.Join(realPath, child.Name()), child)
		if err != nil {
			return err
		}
		kinds[i] = kind
	}

	// *.project.json
	for i, child := range children {
		if kinds[i].regular && child.Name() != rojoDefaultName && rojoFileRegex.MatchString(child.Name()) {
			if err := r.parseConfig(path.Join(directory, child.Name()), false); err != nil {
				return err
			}
		}
	}

	// folders
	for i, child := range children {
		if kinds[i].directory {
			if err := r.searchRealDirectory(path.Join(directory, child.Name()), kinds[i].realPath, child.Name()); err != nil {
				return err
			}
		}
	}

	if item != "" {
		r.rbxPath = r.rbxPath[:len(r.rbxPath)-1]
	}
	return nil
}

type childKind struct {
	regular, directory bool
	realPath           string
}

func kindOfChild(childPath, realChild string, child os.DirEntry) (childKind, error) {
	if entryType := child.Type(); entryType&(fs.ModeSymlink|fs.ModeIrregular) == 0 {
		return childKind{regular: entryType.IsRegular(), directory: entryType.IsDir(), realPath: realChild}, nil
	}
	resolved, err := filepath.EvalSymlinks(childPath)
	if err != nil {
		return childKind{}, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return childKind{}, err
	}
	return childKind{regular: info.Mode().IsRegular(), directory: info.IsDir(), realPath: resolved}, nil
}

// GetRbxPathFromFilePath returns nil where the JS returns undefined. A partition root gives a non-nil path.
func (r *Resolver) GetRbxPathFromFilePath(filePath string) RbxPath {
	filePath = path.Resolve(filePath)
	filePath = convertToLuau(filePath)

	if rbxPath, ok := r.filePathToRbxPath[filePath]; ok {
		return rbxPath
	}

	ext := path.Extname(filePath)
	for _, partition := range r.partitions {
		if isPathDescendantOf(filePath, partition.FsPath) {
			stripped := stripRojoExts(filePath)
			relativePath := path.Relative(partition.FsPath, stripped)
			relativeParts := []string{}
			if relativePath != "" {
				relativeParts = strings.Split(relativePath, path.Sep)
			}
			if isScriptExt(ext) && len(relativeParts) > 0 && relativeParts[len(relativeParts)-1] == initName {
				relativeParts = relativeParts[:len(relativeParts)-1]
			}
			return append(append(RbxPath{}, partition.RbxPath...), relativeParts...)
		}
	}
	return nil
}

// GetRbxTypeFromFilePath is getRbxTypeFromFilePath.
func (r *Resolver) GetRbxTypeFromFilePath(filePath string) RbxType {
	filePath = convertToLuau(filePath)
	ext := path.Extname(filePath)
	subext := path.Extname(path.Basename(filePath, ext))
	if !isScriptExt(ext) {
		// non-script exts cannot use .server, .client, etc.
		return RbxTypeModuleScript
	}
	switch subext {
	case "":
		return RbxTypeModuleScript
	case serverSubext:
		return RbxTypeScript
	case clientSubext:
		return RbxTypeLocalScript
	}
	return RbxTypeUnknown
}

// getContainer returns the index of the matching container, or -1. Callers in the JS compare containers by
// identity, and an index in the same list says the same thing.
func (r *Resolver) getContainer(from []RbxPath, rbxPath RbxPath) int {
	if r.IsGame && rbxPath != nil {
		for i, container := range from {
			if arrayStartsWith(rbxPath, container) {
				return i
			}
		}
	}
	return -1
}

// GetFileRelation is getFileRelation.
func (r *Resolver) GetFileRelation(fileRbxPath, moduleRbxPath RbxPath) FileRelation {
	fileContainer := r.getContainer(defaultIsolatedContainers, fileRbxPath)
	moduleContainer := r.getContainer(defaultIsolatedContainers, moduleRbxPath)
	switch {
	case fileContainer != -1 && moduleContainer != -1:
		if fileContainer == moduleContainer {
			return FileRelationInToIn
		}
		return FileRelationOutToIn
	case fileContainer != -1:
		return FileRelationInToOut
	case moduleContainer != -1:
		return FileRelationOutToIn
	}
	return FileRelationOutToOut
}

// IsIsolated is isIsolated.
func (r *Resolver) IsIsolated(rbxPath RbxPath) bool {
	return r.getContainer(defaultIsolatedContainers, rbxPath) != -1
}

// GetNetworkType is getNetworkType.
func (r *Resolver) GetNetworkType(rbxPath RbxPath) NetworkType {
	if r.getContainer(serverContainers, rbxPath) != -1 {
		return NetworkTypeServer
	}
	if r.getContainer(clientContainers, rbxPath) != -1 {
		return NetworkTypeClient
	}
	return NetworkTypeUnknown
}

// Relative is RojoResolver.relative. Steps up are RbxPathParent.
func Relative(rbxFrom, rbxTo RbxPath) []string {
	maxLength := max(len(rbxFrom), len(rbxTo))
	diffIndex := maxLength
	for i := range maxLength {
		if i >= len(rbxFrom) || i >= len(rbxTo) || rbxFrom[i] != rbxTo[i] {
			diffIndex = i
			break
		}
	}
	result := []string{}
	for i := diffIndex; i < len(rbxFrom); i++ {
		result = append(result, RbxPathParent)
	}
	if diffIndex < len(rbxTo) {
		result = append(result, rbxTo[diffIndex:]...)
	}
	return result
}
