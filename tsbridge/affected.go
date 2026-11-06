package tsbridge

import (
	"slices"
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

// Change says what a stage that reads a whole program has to redo, given the program it last ran on.
type Change struct {
	// Full says that anything can have changed. Affected is nil then.
	Full bool
	// Reason says why, for --verbose.
	Reason string
	// Changed counts the files that are different objects in the two programs.
	Changed int
	// Affected holds the changed files and everything that reaches them through imports.
	Affected map[tspath.PathKey]bool
}

// Reaches says whether the file has to be redone.
func (c *Change) Reaches(file *ast.SourceFile) bool { return c.Full || c.Affected[file.PathKey()] }

// AffectedFiles compares two programs on the same options. A stage's answer for a file can only depend on that file and
// on what the checker sees from it. The checker sees what the file imports, transitively, plus the global scope. So:
//
//   - A file is affected if it reaches a changed file through imports, including `import type`, `import("...")` types,
//     module augmentation names and `/// <reference path>`.
//   - Anything that adds to the global scope, or to another module without being imported, reaches every file. That's a
//     declaration file, a file that isn't a module, and any file with `declare global`, `declare module`, or an ambient
//     module. The old version counts too, because removing one of those matters as much as adding it.
//   - So does any change in which files exist, or in where an unchanged file's imports go.
//
// Parsed files are shared between programs while they're unchanged, so comparing pointers finds the changes. A file that
// got parsed again with the same text counts as changed, which only costs time.
func AffectedFiles(previous, next *Project) *Change {
	if previous == nil {
		return &Change{Full: true, Reason: "no earlier program"}
	}
	if previous.program == next.program {
		return &Change{Affected: map[tspath.PathKey]bool{}}
	}
	before, after := previous.program.FilesByPath(), next.program.FilesByPath()
	if len(before) != len(after) {
		return &Change{Full: true, Reason: "files added or removed"}
	}
	var changed []tspath.PathKey
	for path, file := range after {
		old, ok := before[path]
		if !ok {
			return &Change{Full: true, Reason: "files added or removed"}
		}
		if old != file {
			if HasGlobalEffects(old) || HasGlobalEffects(file) {
				return &Change{Full: true, Reason: file.FileName().AsString() + " can affect the global scope"}
			}
			changed = append(changed, path)
		}
	}
	importsBefore, importsAfter := previous.ImportGraph(), next.ImportGraph()
	for path, targets := range importsAfter {
		if before[path] == after[path] && !slices.Equal(targets, importsBefore[path]) {
			return &Change{Full: true, Reason: after[path].FileName().AsString() + " resolves its imports differently"}
		}
	}

	return &Change{Changed: len(changed), Affected: next.Closure(changed)}
}

// Closure gives the files and everything that reaches them through imports.
func (p *Project) Closure(changed []tspath.PathKey) map[tspath.PathKey]bool {
	graph := p.ImportGraph()
	importers := make(map[tspath.PathKey][]tspath.PathKey, len(graph))
	for path, targets := range graph {
		for _, target := range targets {
			importers[target] = append(importers[target], path)
		}
	}
	affected := make(map[tspath.PathKey]bool, len(changed))
	queue := slices.Clone(changed)
	for len(queue) > 0 {
		path := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if affected[path] {
			continue
		}
		affected[path] = true
		queue = append(queue, importers[path]...)
	}
	return affected
}

// HasGlobalEffects says a change to this file can reach files that don't import it.
func HasGlobalEffects(file *ast.SourceFile) bool {
	if ast.IsJsonSourceFile(file) {
		return false
	}
	return file.IsDeclarationFile ||
		!ast.IsExternalOrCommonJSModule(file) ||
		len(file.ModuleAugmentations) > 0 ||
		len(file.AmbientModuleNames) > 0
}

// ImportGraph lists, per file, the program files it reaches directly, sorted. It's worked out once per project.
func (p *Project) ImportGraph() map[tspath.PathKey][]tspath.PathKey {
	p.graphOnce.Do(func() {
		program := p.program
		files := program.FilesByPath()
		graph := make(map[tspath.PathKey][]tspath.PathKey, len(files))
		resolved := program.GetResolvedModules()
		for path, file := range files {
			var targets []tspath.PathKey
			for _, module := range resolved[path] {
				if module == nil || !module.IsResolved() {
					continue
				}
				if target := program.GetSourceFileForResolvedModule(module); target != nil {
					targets = append(targets, target.PathKey())
				}
			}
			for _, reference := range file.ReferencedFiles {
				name := tspath.ToRootedFilePath(reference.FileName, file.FileName().Directory())
				if target := program.GetSourceFileByPath(program.CaseSensitivity().PathKey(name.AsPath())); target != nil {
					targets = append(targets, target.PathKey())
				}
			}
			slices.Sort(targets)
			graph[path] = slices.Compact(targets)
		}
		p.graph = graph
	})
	return p.graph
}

// graphState is embedded in Project.
type graphState struct {
	graphOnce sync.Once
	graph     map[tspath.PathKey][]tspath.PathKey
}
