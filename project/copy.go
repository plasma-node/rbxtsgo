package project

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/pathtranslator"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

// The runtime files from the roblox-ts-3.0.0.tgz tarball, byte for byte. A test compares them to the pinned clone.
//
//go:embed include/*.lua
var includeFiles embed.FS

// copyFilter is the filter option of fs-extra's copySync.
type copyFilter func(src, dest string) (bool, error)

// copySync is fs-extra 11's copySync with { filter, dereference: true }, which is the only form the project uses.
func copySync(src, dest string, filter copyFilter) error {
	destStat, err := checkPaths(src, dest)
	if err != nil {
		return err
	}
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := checkParentPaths(src, srcStat, dest); err != nil {
		return err
	}
	if filter != nil {
		include, err := filter(src, dest)
		if err != nil {
			return err
		}
		if !include {
			return nil
		}
	}
	if destParent := path.Dirname(dest); !exists(destParent) {
		if err := os.MkdirAll(destParent, 0o777); err != nil {
			return err
		}
	}
	return copyStats(destStat, src, dest, filter)
}

// checkPaths is checkPathsSync. It returns dest's stat, or nil if dest doesn't exist.
func checkPaths(src, dest string) (fs.FileInfo, error) {
	srcStat, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	destStat, err := os.Stat(dest)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			destStat = nil
		} else {
			return nil, err
		}
	}
	if destStat != nil {
		if os.SameFile(srcStat, destStat) {
			return nil, errors.New("Source and destination must not be the same.")
		}
		if srcStat.IsDir() && !destStat.IsDir() {
			return nil, fmt.Errorf("Cannot overwrite non-directory '%s' with directory '%s'.", dest, src)
		}
		if !srcStat.IsDir() && destStat.IsDir() {
			return nil, fmt.Errorf("Cannot overwrite directory '%s' with non-directory '%s'.", dest, src)
		}
	}
	if srcStat.IsDir() && isSrcSubdir(src, dest) {
		return nil, fmt.Errorf("Cannot copy '%s' to a subdirectory of itself, '%s'.", src, dest)
	}
	return destStat, nil
}

// isSrcSubdir only compares path strings, like fs-extra.
func isSrcSubdir(src, dest string) bool {
	split := func(p string) []string {
		return slices.DeleteFunc(strings.Split(path.Resolve(p), path.Sep), func(part string) bool { return part == "" })
	}
	srcParts, destParts := split(src), split(dest)
	for i, part := range srcParts {
		if i >= len(destParts) || destParts[i] != part {
			return false
		}
	}
	return true
}

// checkParentPaths is checkParentPathsSync.
func checkParentPaths(src string, srcStat fs.FileInfo, dest string) error {
	srcParent := path.Resolve(path.Dirname(src))
	destParent := path.Resolve(path.Dirname(dest))
	if destParent == srcParent || destParent == path.Dirname(destParent) {
		return nil
	}
	destStat, err := os.Stat(destParent)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return checkParentPaths(src, srcStat, destParent)
		}
		return err
	}
	if os.SameFile(srcStat, destStat) {
		return fmt.Errorf("Cannot copy '%s' to a subdirectory of itself, '%s'.", src, dest)
	}
	return checkParentPaths(src, srcStat, destParent)
}

func copyStats(destStat fs.FileInfo, src, dest string, filter copyFilter) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if srcStat.IsDir() {
		if destStat == nil {
			if err := os.Mkdir(dest, 0o777); err != nil {
				return err
			}
			if err := copyDir(src, dest, filter); err != nil {
				return err
			}
			return os.Chmod(dest, srcStat.Mode().Perm())
		}
		return copyDir(src, dest, filter)
	}
	if destStat != nil {
		if err := os.Remove(dest); err != nil {
			return err
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dest, data, srcStat.Mode().Perm()); err != nil {
		return err
	}
	return os.Chmod(dest, srcStat.Mode().Perm())
}

func copyDir(src, dest string, filter copyFilter) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcItem := path.Join(src, entry.Name())
		destItem := path.Join(dest, entry.Name())
		if filter != nil {
			include, err := filter(srcItem, destItem)
			if err != nil {
				return err
			}
			if !include {
				continue
			}
		}
		destStat, err := checkPaths(srcItem, destItem)
		if err != nil {
			return err
		}
		if err := copyStats(destStat, srcItem, destItem, filter); err != nil {
			return err
		}
	}
	return nil
}

// CopyItem is copyItem.
func CopyItem(data *shared.ProjectData, translator *pathtranslator.Translator, item string) error {
	return copySync(item, translator.GetOutputPath(item), func(src, dest string) (bool, error) {
		if data.ProjectOptions.WriteOnlyChanged && exists(dest) {
			info, err := os.Lstat(src)
			if err != nil {
				return false, err
			}
			if !info.IsDir() {
				// Upstream compares the two decoded as UTF-8. Bytes can only differ from that on invalid UTF-8.
				srcData, err := os.ReadFile(src)
				if err != nil {
					return false, err
				}
				destData, err := os.ReadFile(dest)
				if err != nil {
					return false, err
				}
				if bytes.Equal(srcData, destData) {
					return false, nil
				}
			}
		}
		if strings.HasSuffix(src, shared.DTSExt) {
			return translator.Declaration, nil
		}
		compilable, err := IsCompilableFile(src)
		if err != nil {
			return false, err
		}
		return !compilable, nil
	})
}

// CopyFiles is copyFiles.
func CopyFiles(data *shared.ProjectData, translator *pathtranslator.Translator, sources []string) error {
	for _, source := range sources {
		if err := CopyItem(data, translator, source); err != nil {
			return err
		}
	}
	return nil
}

// CopyInclude is copyInclude: it copies RuntimeLib.lua and Promise.lua to the include path.
func CopyInclude(data *shared.ProjectData) error {
	options := data.ProjectOptions
	if options.NoInclude || options.Type == shared.ProjectTypePackage || (options.Type == "" && data.IsPackage) {
		return nil
	}
	if err := os.MkdirAll(options.IncludePath, 0o777); err != nil {
		return err
	}
	entries, err := includeFiles.ReadDir("include")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		contents, err := includeFiles.ReadFile("include/" + entry.Name())
		if err != nil {
			return err
		}
		dest := path.Join(options.IncludePath, entry.Name())
		if exists(dest) {
			if err := os.Remove(dest); err != nil {
				return err
			}
		}
		if err := os.WriteFile(dest, contents, 0o666); err != nil {
			return err
		}
	}
	return nil
}

func isOutputFileOrphaned(translator *pathtranslator.Translator, filePath string) bool {
	if strings.HasSuffix(filePath, shared.DTSExt) && !translator.Declaration {
		return true
	}
	if slices.ContainsFunc(translator.GetInputPaths(filePath), exists) {
		return false
	}
	return translator.BuildInfoOutputPath != filePath
}

// TryRemoveOutput is tryRemoveOutput.
func TryRemoveOutput(translator *pathtranslator.Translator, outPath string) error {
	if isOutputFileOrphaned(translator, outPath) {
		if err := os.RemoveAll(outPath); err != nil {
			return err
		}
		shared.LogService.WriteLineIfVerbose("remove " + outPath)
	}
	return nil
}

func cleanupDirRecursively(translator *pathtranslator.Translator, dir string) error {
	if !exists(dir) {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		itemPath := path.Join(dir, entry.Name())
		info, err := os.Stat(itemPath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if entry.Name() == ".git" {
				continue
			}
			if err := cleanupDirRecursively(translator, itemPath); err != nil {
				return err
			}
		}
		if err := TryRemoveOutput(translator, itemPath); err != nil {
			return err
		}
	}
	return nil
}

// Cleanup is cleanup.
func Cleanup(translator *pathtranslator.Translator) error {
	return cleanupDirRecursively(translator, translator.OutDir)
}

// WalkDirectory is walkDirectorySync: breadth first, calling back for each descendant.
func WalkDirectory(dir string, callback func(fsPath string) error) error {
	queue := []string{dir}
	for i := 0; i < len(queue); i++ {
		entries, err := os.ReadDir(queue[i])
		if err != nil {
			return err
		}
		for _, entry := range entries {
			fsPath := path.Join(queue[i], entry.Name())
			if err := callback(fsPath); err != nil {
				return err
			}
			info, err := os.Stat(fsPath)
			if err != nil {
				return err
			}
			if info.IsDir() {
				queue = append(queue, fsPath)
			}
		}
	}
	return nil
}
