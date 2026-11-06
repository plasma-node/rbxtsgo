package plugins

import (
	"context"
	"fmt"
	"maps"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

// Session is Apply for a process that compiles many times, as watch mode does. The result must equal what Apply gives on
// the same sources, and watch's edit sequence test checks that against a clean build. It holds because of what each step
// reads:
//
//   - The accessor names come from one scan per file, so a file's scan is good until that file changes.
//   - A file's printed text depends on the file, the names, the prefix and what the checker answers. The checker only
//     sees through imports and the global scope, so tsbridge.AffectedFiles on the source program gives the files whose
//     text can differ. A change in the names redoes everything, as a filter that changed can't be reasoned about.
//   - The printed program depends only on the printed text and the other files. A file whose text came out the same keeps
//     its parsed and bound object, so what comes after can see that nothing changed for it.
//
// A Session belongs to one decision: make a new one after the tsconfig changes.
type Session struct {
	facts      map[*ast.SourceFile]fileAccessorFacts
	names      *accessorNames
	outcomes   map[tspath.PathKey]*fileOutcome
	lastSource *tsbridge.Project
	printed    *tsbridge.Project
	texts      map[tspath.PathKey]string
	// LastRedone is how many files the last Apply transformed again, for --verbose.
	LastRedone int
}

// NewSession makes a Session with nothing to reuse.
func NewSession() *Session { return &Session{} }

// Apply is Apply. When reuse is false it does what a first run does.
func (s *Session) Apply(ctx context.Context, p *tsbridge.Project, decision Decision, files []*ast.SourceFile, reuse bool) (*Applied, error) {
	applied := &Applied{Project: p, Files: files}
	if decision.Action != ActionSetget {
		return applied, nil
	}

	names := s.collectNames(p)
	change := tsbridge.AffectedFiles(s.lastSource, p)
	if !reuse || s.names == nil || !names.same(s.names) {
		change = &tsbridge.Change{Full: true, Reason: "accessor names changed"}
	}
	var redo []*ast.SourceFile
	for _, file := range files {
		if change.Reaches(file) || s.outcomes[file.PathKey()] == nil {
			redo = append(redo, file)
		}
	}
	s.LastRedone = len(redo)

	outcomes := make([]fileOutcome, len(redo))
	index := make(map[*ast.SourceFile]int, len(redo))
	uses := make([]bool, len(redo))
	if !names.empty() {
		forEachParallel(redo, func(i int, file *ast.SourceFile) { uses[i] = fileUsesName(file, names) })
	}
	var needChecker, noChecker []*ast.SourceFile
	for i, file := range redo {
		index[file] = i
		if uses[i] {
			needChecker = append(needChecker, file)
		} else {
			noChecker = append(noChecker, file)
		}
	}
	if len(needChecker) > 0 {
		workers := p.NewWorkers(ctx, needChecker, tsbridge.DefaultMacroSpec())
		p.RunWorkers(workers, func(w *tsbridge.Worker) {
			for _, file := range w.Files() {
				outcome := &outcomes[index[file]]
				transformFile(file, w.Checker(), decision.Prefix, names, outcome)
				if outcome.exit != nil {
					return
				}
			}
		})
	}
	forEachParallel(noChecker, func(_ int, file *ast.SourceFile) {
		transformFile(file, nil, decision.Prefix, names, &outcomes[index[file]])
	})
	// Files with no earlier outcome are in file order here, and the rest had no exit last time, so this finds the same
	// first exit a whole run would.
	for i := range outcomes {
		if outcomes[i].exit != nil {
			applied.Exit = outcomes[i].exit
			return applied, nil
		}
		if outcomes[i].panicText != "" {
			return nil, fmt.Errorf("internal error while transforming %s: %s", redo[i].FileName().AsString(), outcomes[i].panicText)
		}
	}

	// Nothing above changed this state, so an exit leaves it as it was. The next run compares with that.
	nextOutcomes := make(map[tspath.PathKey]*fileOutcome, len(files))
	for _, file := range files {
		nextOutcomes[file.PathKey()] = s.outcomes[file.PathKey()]
	}
	for i, file := range redo {
		nextOutcomes[file.PathKey()] = &outcomes[i]
	}
	s.outcomes = nextOutcomes
	s.names = names
	s.lastSource = p

	changed := map[tspath.PathKey]string{}
	texts := make(map[tspath.PathKey]string, len(files))
	for _, file := range files {
		key := file.PathKey()
		text := s.outcomes[key].text
		texts[key] = text
		if old, ok := s.texts[key]; !ok || old != text {
			changed[key] = text
		}
	}
	s.texts = texts
	switch {
	case s.printed == nil:
		s.printed = p.WithPrintedFiles(changed)
	default:
		s.printed = s.printed.UpdatePrinted(p, changed, texts, change.Full)
	}
	next := s.printed

	applied.Project = next
	applied.Printed = make(map[string]string, len(files))
	applied.maps = make(map[*ast.SourceFile]*reprintMap, len(files))
	applied.Files = make([]*ast.SourceFile, len(files))
	byPath := next.Program().FilesByPath()
	for i, file := range files {
		applied.Printed[file.FileName().AsString()] = texts[file.PathKey()]
		printedFile := byPath[file.PathKey()]
		if printedFile == nil {
			return nil, fmt.Errorf("internal error: %s is missing from the reprinted program", file.FileName().AsString())
		}
		applied.Files[i] = printedFile
		applied.maps[printedFile] = &reprintMap{original: file, generator: s.outcomes[file.PathKey()].generator}
	}
	return applied, nil
}

// collectNames is collectAccessorNames, scanning only files it hasn't seen.
func (s *Session) collectNames(p *tsbridge.Project) *accessorNames {
	files := p.Program().SourceFiles()
	facts := make([]fileAccessorFacts, len(files))
	forEachParallel(files, func(i int, file *ast.SourceFile) {
		if known, ok := s.facts[file]; ok {
			facts[i] = known
		} else {
			facts[i] = scanAccessors(file)
		}
	})
	nextFacts := make(map[*ast.SourceFile]fileAccessorFacts, len(files))
	result := &accessorNames{names: map[string]bool{}}
	mapped, computed := false, false
	for i, fact := range facts {
		nextFacts[files[i]] = fact
		for _, name := range fact.names {
			result.names[name] = true
		}
		mapped = mapped || fact.mappedTypeName
		computed = computed || fact.computed
	}
	s.facts = nextFacts
	result.all = computed || (mapped && len(result.names) > 0)
	return result
}

func (n *accessorNames) same(other *accessorNames) bool {
	return n.all == other.all && maps.Equal(n.names, other.names)
}
