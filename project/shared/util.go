package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/nodepath"
)

// DiagnosticError is DiagnosticError from src/Shared/errors. ProjectError is this with one text diagnostic.
type DiagnosticError struct {
	Diagnostics []*Diagnostic
}

func (e *DiagnosticError) Error() string {
	return FormatDiagnostics(e.Diagnostics, ProcessCwd(), tspath.CaseSensitive, EOL)
}

// NewProjectError is `new ProjectError(message)`.
func NewProjectError(message string) *DiagnosticError {
	return &DiagnosticError{Diagnostics: []*Diagnostic{TextDiagnostic(message, diagnostics.CategoryError)}}
}

// ProcessCwd is process.cwd() in the form diagnosticwriter takes.
func ProcessCwd() tspath.RootedDirectoryPath {
	dir, err := os.Getwd()
	if err != nil {
		dir = "/"
	}
	return tspath.RootedDirectoryPathFromAbsolute(tspath.NormalizePath(filepath.ToSlash(dir)))
}

// IsPathDescendantOf is isPathDescendantOf.
func IsPathDescendantOf(filePath, dirPath string) bool {
	return dirPath == filePath || !strings.HasPrefix(nodepath.Native.Relative(dirPath, filePath), "..")
}

// FindAncestorDir is findAncestorDir.
func FindAncestorDir(dirs []string) string {
	path := nodepath.Native
	normalized := make([]string, len(dirs))
	for i, dir := range dirs {
		dir = path.Normalize(dir)
		if !strings.HasSuffix(dir, path.Sep) {
			dir += path.Sep
		}
		normalized[i] = dir
	}
	current := normalized[0]
	for {
		all := true
		for _, dir := range normalized {
			if !strings.HasPrefix(dir, current) {
				all = false
				break
			}
		}
		if all {
			return current
		}
		current = path.Join(current, "..")
	}
}

// RealPathExists is realPathExistsSync. ok is false where JS returns undefined.
func RealPathExists(fsPath string) (string, bool) {
	if _, err := os.Stat(fsPath); err != nil {
		return "", false
	}
	real, err := filepath.EvalSymlinks(fsPath)
	if err != nil {
		return "", false
	}
	return real, true
}

// DiagnosticService is TSTransformer/classes/DiagnosticService.ts as an instance, so each worker can have its own.
// Upstream's is a static class. The compile step collects each worker's list in file order at the end.
type DiagnosticService struct {
	diagnostics []*Diagnostic
	single      bool
}

// AddSingleDiagnostic is addSingleDiagnostic. Upstream keys this by diagnostic.code, and every roblox-ts diagnostic
// has the same code, so only the first one between flushes gets through. That holds per service here.
func (s *DiagnosticService) AddSingleDiagnostic(d *Diagnostic) {
	if !s.single {
		s.single = true
		s.AddDiagnostic(d)
	}
}

// AddDiagnostic is addDiagnostic.
func (s *DiagnosticService) AddDiagnostic(d *Diagnostic) { s.diagnostics = append(s.diagnostics, d) }

// AddDiagnostics is addDiagnostics.
func (s *DiagnosticService) AddDiagnostics(list []*Diagnostic) {
	s.diagnostics = append(s.diagnostics, list...)
}

// Flush is flush.
func (s *DiagnosticService) Flush() []*Diagnostic {
	current := s.diagnostics
	s.diagnostics = nil
	s.single = false
	return current
}

// HasErrors is hasErrors.
func (s *DiagnosticService) HasErrors() bool { return HasErrors(s.diagnostics) }

// AddDiagnosticWithCache is DiagnosticService.addDiagnosticWithCache. The cache lives in the worker's state.
func AddDiagnosticWithCache[T comparable](s *DiagnosticService, cacheBy T, d *Diagnostic, cache map[T]bool) {
	if !cache[cacheBy] {
		cache[cacheBy] = true
		s.AddDiagnostic(d)
	}
}

// LogServiceType is LogService. Writes are locked, because workers can log.
type LogServiceType struct {
	Verbose bool
	mu      sync.Mutex
	partial bool
}

// LogService is the process-wide logger, like the static class upstream.
var LogService = &LogServiceType{}

func (l *LogServiceType) write(message string) {
	l.partial = !strings.HasSuffix(message, "\n")
	fmt.Fprint(os.Stdout, message)
}

// Write is LogService.write.
func (l *LogServiceType) Write(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.write(message)
}

// WriteLine is LogService.writeLine.
func (l *LogServiceType) WriteLine(messages ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.partial {
		l.write("\n")
	}
	for _, message := range messages {
		l.write(message + "\n")
	}
}

// WriteLineIfVerbose is LogService.writeLineIfVerbose.
func (l *LogServiceType) WriteLineIfVerbose(messages ...string) {
	if l.Verbose {
		l.WriteLine(messages...)
	}
}

// Warn is LogService.warn.
func (l *LogServiceType) Warn(message string) {
	l.WriteLine(Color.Yellow("Compiler Warning:") + " " + message)
}

// CategoryError is the error category, so callers don't need the internal diagnostics import for it.
const CategoryError = diagnostics.CategoryError

// AssertionMessage is the text of assert(false, message).
func AssertionMessage(message string) string {
	return "Assertion Failed! " + message + "\nThis is a compiler bug! Please submit a bug report here:\nhttps://github.com/roblox-ts/roblox-ts/issues"
}
