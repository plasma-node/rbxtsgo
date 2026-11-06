package transformer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

var typeFlagNames = []struct {
	name string
	flag checker.TypeFlags
}{
	{"Any", checker.TypeFlagsAny}, {"Unknown", checker.TypeFlagsUnknown}, {"String", checker.TypeFlagsString},
	{"Number", checker.TypeFlagsNumber}, {"Boolean", checker.TypeFlagsBoolean}, {"Enum", checker.TypeFlagsEnum},
	{"BigInt", checker.TypeFlagsBigInt}, {"StringLiteral", checker.TypeFlagsStringLiteral},
	{"NumberLiteral", checker.TypeFlagsNumberLiteral}, {"BooleanLiteral", checker.TypeFlagsBooleanLiteral},
	{"EnumLiteral", checker.TypeFlagsEnumLiteral}, {"Undefined", checker.TypeFlagsUndefined}, {"Null", checker.TypeFlagsNull},
	{"Void", checker.TypeFlagsVoid}, {"Never", checker.TypeFlagsNever}, {"TypeParameter", checker.TypeFlagsTypeParameter},
	{"Object", checker.TypeFlagsObject}, {"Union", checker.TypeFlagsUnion}, {"Intersection", checker.TypeFlagsIntersection},
	{"Index", checker.TypeFlagsIndex}, {"IndexedAccess", checker.TypeFlagsIndexedAccess}, {"Conditional", checker.TypeFlagsConditional},
	{"TemplateLiteral", checker.TypeFlagsTemplateLiteral},
}

var symbolFlagNames = []struct {
	name string
	flag ast.SymbolFlags
}{
	{"Variable", ast.SymbolFlagsVariable}, {"Property", ast.SymbolFlagsProperty}, {"Function", ast.SymbolFlagsFunction},
	{"Class", ast.SymbolFlagsClass}, {"Interface", ast.SymbolFlagsInterface}, {"Enum", ast.SymbolFlagsEnum},
	{"EnumMember", ast.SymbolFlagsEnumMember}, {"ValueModule", ast.SymbolFlagsValueModule}, {"NamespaceModule", ast.SymbolFlagsNamespaceModule},
	{"TypeLiteral", ast.SymbolFlagsTypeLiteral}, {"ObjectLiteral", ast.SymbolFlagsObjectLiteral}, {"Method", ast.SymbolFlagsMethod},
	{"Accessor", ast.SymbolFlagsAccessor}, {"Signature", ast.SymbolFlagsSignature}, {"TypeParameter", ast.SymbolFlagsTypeParameter},
	{"TypeAlias", ast.SymbolFlagsTypeAlias}, {"Alias", ast.SymbolFlagsAlias}, {"Optional", ast.SymbolFlagsOptional},
}

func q(value any) string {
	var b strings.Builder
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		panic(err)
	}
	return strings.TrimRight(b.String(), "\n")
}

type prober struct {
	c     *checker.Checker
	lines []string
}

func (p *prober) add(format string, args ...any) {
	p.lines = append(p.lines, fmt.Sprintf(format, args...))
}

func (p *prober) describeType(t *checker.Type) string {
	if t == nil {
		return "none"
	}
	var flags []string
	for _, f := range typeFlagNames {
		if t.Flags()&f.flag != 0 {
			flags = append(flags, f.name)
		}
	}
	symbol := "-"
	if s := t.Symbol(); s != nil {
		symbol = ast.EscapeAllInternalSymbolNames(s.Name())
	}
	return fmt.Sprintf("%s flags=%s symbol=%s", q(p.c.TypeToString(t)), strings.Join(flags, "|"), symbol)
}

func describeSymbol(s *ast.Symbol) string {
	if s == nil {
		return "none"
	}
	var flags []string
	for _, f := range symbolFlagNames {
		if s.Flags()&f.flag != 0 {
			flags = append(flags, f.name)
		}
	}
	parent := "-"
	if s.Parent() != nil {
		parent = ast.EscapeAllInternalSymbolNames(s.Parent().Name())
	}
	return fmt.Sprintf("%s flags=%s decls=%d value=%v parent=%s", ast.EscapeAllInternalSymbolNames(s.Name()), strings.Join(flags, "|"), len(s.Declarations()), s.ValueDeclaration() != nil, parent)
}

func (p *prober) visit(file *ast.SourceFile, node *ast.Node) {
	c := p.c
	here := fmt.Sprintf("%d-%d", scanner.GetTokenPosOfNode(node, file, false), node.End())
	if ast.IsIdentifier(node) {
		p.add("%s id %s", here, q(node.Text()))
		s := c.GetSymbolAtLocation(node)
		p.add("  symbol %s", describeSymbol(s))
		if s != nil {
			p.add("  undefined=%v arguments=%v merged=%v", c.IsUndefinedSymbol(s), c.IsArgumentsSymbol(s), c.GetMergedSymbol(s) == s)
			if s.Flags()&ast.SymbolFlagsAlias != 0 {
				p.add("  skipAlias %s", describeSymbol(c.SkipAlias(s)))
			}
		}
	}
	if ast.IsStringLiteral(node) || ast.IsNumericLiteral(node) || ast.IsNoSubstitutionTemplateLiteral(node) {
		p.add("%s literal %s", here, q(node.Text()))
	}
	if ast.IsTemplateHead(node) || ast.IsTemplateMiddle(node) || ast.IsTemplateTail(node) {
		p.add("%s template %s raw %s", here, q(node.Text()), q(node.TemplateLiteralLikeData().RawText))
	}
	if ast.IsJsxText(node) {
		text := node.AsJsxText()
		p.add("%s jsxtext %s ws=%v", here, q(text.Text), text.ContainsOnlyTriviaWhiteSpaces)
	}
	if ast.IsStatement(node) {
		var comments []string
		var factory ast.NodeFactory
		for r := range scanner.GetLeadingCommentRanges(&factory, file.Text(), node.Pos()) {
			comments = append(comments, file.Text()[r.Pos():r.End()])
		}
		if len(comments) > 0 {
			p.add("%s comments %s", here, q(comments))
		}
	}
	if ast.IsExpressionNode(node) && !(ast.IsFunctionExpression(node.Parent) && node.Parent.Name() == node) {
		t := c.GetTypeAtLocation(node)
		text := scanner.GetSourceTextOfNodeFromSourceFile(file, node, false)
		if len(text) > 40 {
			text = text[:40]
		}
		p.add("%s expr %s %s", here, q(text), p.describeType(t))
		p.add("  tuple=%v arrayLike=%v calls=%d constructs=%d props=%d", checker.IsTupleType(t), c.IsArrayLikeType(t), len(c.GetCallSignatures(t)), len(c.GetConstructSignatures(t)), len(c.GetPropertiesOfType(t)))
		p.add("  nonOptional %s", q(c.TypeToString(c.GetNonOptionalType(t))))
		if index := c.GetIndexTypeOfType(t, c.GetNumberType()); index != nil {
			p.add("  numberIndex %s", q(c.TypeToString(index)))
		} else {
			p.add("  numberIndex none")
		}
		contextual := "none"
		if ast.IsExpression(node) {
			if ct := c.GetContextualType(node, checker.ContextFlagsNone); ct != nil {
				contextual = q(c.TypeToString(ct))
			}
		}
		p.add("  contextual %s", contextual)
		if ast.IsPropertyAccessExpression(node) || ast.IsElementAccessExpression(node) {
			p.add("  constant %s", q(c.GetConstantValue(node)))
		}
		if ast.IsCallExpression(node) || ast.IsNewExpression(node) {
			if signature := c.GetResolvedSignature(node); signature != nil {
				p.add("  signature params=%d returns %s", len(signature.Parameters()), q(c.TypeToString(c.GetReturnTypeOfSignature(signature))))
			} else {
				p.add("  signature params=-1 returns none")
			}
		}
	}
	node.ForEachChild(func(child *ast.Node) bool {
		p.visit(file, child)
		return false
	})
}

// TestProbeAgainstTypeScript compares what the checker and AST answer with TS 5.5.3, node by node. The port's output
// depends on these answers. See docs/bridge.md and docs/union-order.md for the differences.
func TestProbeAgainstTypeScript(t *testing.T) {
	spikeFixture, err := filepath.Abs(filepath.Join("..", "spike", "fixture"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(spikeFixture, "node_modules", "@rbxts", "compiler-types")); err != nil {
		t.Skip("run npm ci in spike/fixture to enable this test")
	}
	syntaxFixture, err := filepath.Abs(filepath.Join("testdata", "syntax"))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("rbxts packages", func(t *testing.T) { runProbe(t, spikeFixture, "probe.txt", 17, 0) })
	t.Run("syntax", func(t *testing.T) {
		runProbe(t, syntaxFixture, "probe-syntax.txt", syntaxTypeTextDifferences, syntaxLiteralDifferences)
	})
}

func runProbe(t *testing.T, dir, goldenName string, expectedTypeText, expectedLiteral int) {
	parsed, err := tsbridge.ParseConfig(filepath.Join(dir, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	project := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{Checkers: 1})
	ctx := context.Background()
	var lines []string
	workers := project.NewWorkers(ctx, project.SourceFilesToCompile(), tsbridge.MacroSpec{})
	base := tspath.NormalizePath(filepath.ToSlash(dir)) + "/"
	for _, file := range project.Program().SourceFiles() {
		if file.IsDeclarationFile {
			continue
		}
		worker := workers[0]
		worker.CheckFileAndGetResolver(ctx, file)
		p := &prober{c: worker.Checker()}
		p.add("== %s", strings.TrimPrefix(file.FileName().AsString(), base))
		p.visit(file, file.AsNode())
		for _, line := range p.lines {
			// The fixture folder is written as <ROOT>, so the golden holds no machine path and the test runs from any folder.
			lines = append(lines, strings.ReplaceAll(line, strings.TrimSuffix(base, "/"), "<ROOT>"))
		}
	}

	data, err := os.ReadFile(filepath.Join("testdata", goldenName))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(strings.TrimRight(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"), "\n")
	if dump := os.Getenv("PROBE_DUMP"); dump != "" {
		if err := os.WriteFile(dump+goldenName, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if len(lines) != len(want) {
		t.Fatalf("%d lines here, %d from TS 5.5.3", len(lines), len(want))
	}
	textDifferences, literalDifferences := 0, 0
	inUnicode := false
	for i := range want {
		if strings.HasPrefix(want[i], "== ") {
			inUnicode = strings.HasSuffix(want[i], "unicode.ts")
		}
		got, wanted := lines[i], want[i]
		if inUnicode {
			got, wanted = offsets.ReplaceAllString(got, ""), offsets.ReplaceAllString(wanted, "")
		}
		if got == wanted {
			continue
		}
		if isLiteralLine(wanted) {
			literalDifferences++
			t.Logf("literal text differs, line %d\n got %s\nwant %s", i+1, got, wanted)
			continue
		}
		if structure(got) != structure(wanted) {
			t.Errorf("line %d\n got %s\nwant %s", i+1, got, wanted)
			continue
		}
		textDifferences++
	}
	t.Logf("%d lines compared, %d differ only in how a type prints, %d in literal text", len(want), textDifferences, literalDifferences)
	if textDifferences != expectedTypeText {
		t.Errorf("%d lines differ in how a type prints, expected %d", textDifferences, expectedTypeText)
	}
	if literalDifferences != expectedLiteral {
		t.Errorf("%d lines differ in literal text, expected %d", literalDifferences, expectedLiteral)
	}
}

const (
	syntaxTypeTextDifferences = 12
	// The lone surrogate in unicode.ts. See "Lone surrogates" in the guide.
	syntaxLiteralDifferences = 1
)

var offsets = regexp.MustCompile(`^\d+-\d+ `)

func isLiteralLine(line string) bool {
	line = offsets.ReplaceAllString(line, "")
	return strings.HasPrefix(line, "literal ") || strings.HasPrefix(line, "template ") || strings.HasPrefix(line, "jsxtext ")
}

var (
	exprLine      = regexp.MustCompile(`^((?:\d+-\d+ )?expr "(?:[^"\\]|\\.)*") .* (flags=\S* symbol=\S+)$`)
	signatureLine = regexp.MustCompile(`^(  signature params=-?\d+ returns) .*$`)
)

// structure removes printed type text from a line, keeping only whether the type exists. TS 7 prints some unions in a
// different member order, and shortens long types differently.
func structure(line string) string {
	switch {
	case strings.HasPrefix(line, "  contextual "), strings.HasPrefix(line, "  nonOptional "), strings.HasPrefix(line, "  numberIndex "):
		fields := strings.SplitN(strings.TrimLeft(line, " "), " ", 2)
		if fields[1] == "none" {
			return fields[0] + " none"
		}
		return fields[0] + " some"
	}
	if match := exprLine.FindStringSubmatch(line); match != nil {
		return match[1] + " " + match[2]
	}
	if match := signatureLine.FindStringSubmatch(line); match != nil {
		return match[1]
	}
	return line
}
