package shared

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
	"github.com/microsoft/TypeScript/tsc/internal/diagnosticwriter"
	"github.com/microsoft/TypeScript/tsc/internal/locale"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

// Diagnostic is what DiagnosticService collects: either a roblox-ts diagnostic or a wrapped TypeScript one.
type Diagnostic struct {
	category diagnostics.Category
	// id is the factory id for a located roblox-ts diagnostic, and -1 otherwise, where JS has undefined.
	id      int
	message string
	file    *ast.SourceFile
	start   int
	length  int
	ts      *ast.Diagnostic
	// context is set on diagnostics from a context factory. ContextReplay uses it to rebuild the text in file order.
	context []string
	factory *contextFactory
}

// FromTS wraps a TypeScript diagnostic.
func FromTS(d *ast.Diagnostic) *Diagnostic {
	return &Diagnostic{category: d.Category(), id: -1, ts: d}
}

// FromTSList wraps several TypeScript diagnostics.
func FromTSList(list []*ast.Diagnostic) []*Diagnostic {
	result := make([]*Diagnostic, len(list))
	for i, d := range list {
		result[i] = FromTS(d)
	}
	return result
}

// Category is diagnostic.category.
func (d *Diagnostic) Category() diagnostics.Category { return d.category }

// ID is getDiagnosticId, or -1 where JS gives undefined.
func (d *Diagnostic) ID() int { return d.id }

// IsRobloxTS reports whether roblox-ts made this diagnostic, rather than TypeScript.
func (d *Diagnostic) IsRobloxTS() bool { return d.ts == nil }

// Code is the TypeScript code, or 0 for a roblox-ts diagnostic.
func (d *Diagnostic) Code() int32 {
	if d.ts != nil {
		return d.ts.Code()
	}
	return 0
}

// Message is the message text. For a TypeScript diagnostic it has only the top line of the chain.
func (d *Diagnostic) Message() string {
	if d.ts != nil {
		return d.ts.String()
	}
	return d.message
}

// File is the source file, or nil for a text diagnostic.
func (d *Diagnostic) File() *ast.SourceFile {
	if d.ts != nil {
		return d.ts.File()
	}
	return d.file
}

// Start is the byte offset of the start. JS has a UTF-16 offset here.
func (d *Diagnostic) Start() int {
	if d.ts != nil {
		return d.ts.Pos()
	}
	return d.start
}

// Length is the length in bytes. See the note on SourceRange about its meaning for those.
func (d *Diagnostic) Length() int {
	if d.ts != nil {
		return d.ts.Len()
	}
	return d.length
}

// Relocated returns the diagnostic with its location, and its related locations, moved by relocate. relocate says false
// to leave one where it is. The message doesn't change.
func (d *Diagnostic) Relocated(relocate func(*ast.SourceFile, core.TextRange) (*ast.SourceFile, core.TextRange, bool)) *Diagnostic {
	moved := *d
	if d.ts != nil {
		moved.ts = relocateTS(d.ts, relocate)
	} else if d.file != nil {
		if file, loc, ok := relocate(d.file, core.NewTextRange(d.start, d.start+d.length)); ok {
			moved.file, moved.start, moved.length = file, loc.Pos(), loc.Len()
		}
	}
	return &moved
}

func relocateTS(d *ast.Diagnostic, relocate func(*ast.SourceFile, core.TextRange) (*ast.SourceFile, core.TextRange, bool)) *ast.Diagnostic {
	moved := d.Clone()
	if d.File() != nil {
		if file, loc, ok := relocate(d.File(), d.Loc()); ok {
			moved.SetFile(file)
			moved.SetLocation(loc)
		}
	}
	if related := d.RelatedInformation(); len(related) > 0 {
		list := make([]*ast.Diagnostic, len(related))
		for i, info := range related {
			list[i] = relocateTS(info, relocate)
		}
		moved.SetRelatedInfo(list)
	}
	return moved
}

// SourceRange is SourceFileWithTextRange.
//
// Upstream's createDiagnosticWithLocation stores range.end as the length here, not end - pos. That's kept, so the
// squiggle in formatted output is too long in the same way.
type SourceRange struct {
	File *ast.SourceFile
	Pos  int
	End  int
}

// Location is ts.Node | SourceFileWithTextRange: where a located diagnostic points.
type Location struct {
	node  *ast.Node
	file  *ast.SourceFile
	start int
	len   int
}

// AtNode points at a node, like node.getStart() and node.getWidth().
func AtNode(node *ast.Node) Location {
	file := ast.GetSourceFileOfNode(node)
	start := scanner.GetTokenPosOfNode(node, file, false)
	return Location{node: node, file: file, start: start, len: node.End() - start}
}

// AtRange points at a text range.
func AtRange(r SourceRange) Location {
	return Location{file: r.File, start: r.Pos, len: r.End}
}

// TextDiagnostic is createTextDiagnostic.
func TextDiagnostic(message string, category diagnostics.Category) *Diagnostic {
	return &Diagnostic{category: category, id: -1, message: message}
}

// HasErrors is hasErrors.
func HasErrors(list []*Diagnostic) bool {
	for _, d := range list {
		if d.category == diagnostics.CategoryError {
			return true
		}
	}
	return false
}

// robloxTSPrefix gets printed as the diagnostic code, because upstream sets code to " roblox-ts" and TypeScript
// prints "TS" + code. The writer always appends a number, so it is written with a marker, then fixed afterwards.
const (
	robloxTSSource  = "TS roblox-ts"
	robloxTSWritten = robloxTSSource + "0: "
	robloxTSWanted  = robloxTSSource + ": "
)

// writerDiagnostic adapts a roblox-ts diagnostic for diagnosticwriter.
type writerDiagnostic struct{ d *Diagnostic }

func (w *writerDiagnostic) File() diagnosticwriter.FileLike {
	if w.d.file == nil {
		return nil
	}
	return w.d.file
}
func (w *writerDiagnostic) Pos() int { return w.d.start }
func (w *writerDiagnostic) End() int { return w.d.start + w.Len() }

// Len clamps to the end of the file. Upstream stores range.end as the length, so start + length can run past the
// end, which TS tolerates and tsgo's writer panics on.
func (w *writerDiagnostic) Len() int {
	if w.d.file != nil {
		return min(w.d.length, len(w.d.file.Text())-w.d.start)
	}
	return w.d.length
}

func (w *writerDiagnostic) Code() int32                                       { return 0 }
func (w *writerDiagnostic) Category() diagnostics.Category                    { return w.d.category }
func (w *writerDiagnostic) Source() string                                    { return robloxTSSource }
func (w *writerDiagnostic) Localize(locale.Locale) string                     { return w.d.message }
func (w *writerDiagnostic) MessageChain() []diagnosticwriter.Diagnostic       { return nil }
func (w *writerDiagnostic) RelatedInformation() []diagnosticwriter.Diagnostic { return nil }

// FormatDiagnostics is formatDiagnostics: ts.formatDiagnosticsWithColorAndContext with the given cwd and newline.
// The TS 7 writer stands in for the TS 5.5.3 one. The tests compare the two on roblox-ts diagnostics.
// Messages from TypeScript itself can differ between those versions.
func FormatDiagnostics(list []*Diagnostic, cwd tspath.RootedDirectoryPath, caseSensitivity tspath.CaseSensitivity, newLine string) string {
	converted := make([]diagnosticwriter.Diagnostic, len(list))
	for i, d := range list {
		if d.ts != nil {
			converted[i] = diagnosticwriter.WrapASTDiagnostic(d.ts)
		} else {
			converted[i] = &writerDiagnostic{d: d}
		}
	}
	options := &diagnosticwriter.FormattingOptions{
		CaseSensitivity:  caseSensitivity,
		CurrentDirectory: cwd,
		NewLine:          newLine,
	}
	var b strings.Builder
	for _, d := range converted {
		diagnosticwriter.FormatDiagnosticWithColorAndContext(&b, d, options)
		// TS 5.5.3 ends every diagnostic with a newline, and the writer only does for those with a code snippet.
		if d.File() == nil {
			b.WriteString(newLine)
		}
	}
	return strings.ReplaceAll(b.String(), robloxTSWritten, robloxTSWanted)
}
