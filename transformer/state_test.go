package transformer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

func newFixtureState(t *testing.T, ctx context.Context) (*tsbridge.Project, []*tsbridge.Worker, func(*tsbridge.Worker, *ast.SourceFile) *TransformState) {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "spike", "fixture"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "@rbxts", "compiler-types")); err != nil {
		t.Skip("run npm ci in spike/fixture to enable this test")
	}
	options := shared.DefaultProjectOptions()
	options.Type = shared.ProjectTypePackage
	options.Rojo = ""
	data, err := project.CreateProjectData(filepath.Join(dir, "tsconfig.json"), options)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := project.GetParsedCommandLine(data)
	if err != nil {
		t.Fatal(err)
	}
	p := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{Checkers: 2})
	translator := project.CreatePathTranslator(p, data)
	sensitivity := tspath.CaseSensitive
	compile, failure, err := project.PrepareCompile(p, data, translator, sensitivity)
	if err != nil || failure != nil {
		t.Fatalf("PrepareCompile: %v %v", err, failure)
	}
	workers := p.NewWorkers(ctx, p.SourceFilesToCompile(), tsbridge.DefaultMacroSpec())
	stateOptions := StateOptions{Program: p.Program(), Data: data, PathTranslator: translator, Compile: compile, Options: p.Options()}
	return p, workers, func(w *tsbridge.Worker, file *ast.SourceFile) *TransformState {
		macros, err := w.Macros()
		if err != nil {
			t.Fatal(err)
		}
		return NewTransformState(ctx, w, stateOptions, &TransformServices{MacroManager: macros}, NewMultiTransformState(), &shared.DiagnosticService{}, file)
	}
}

func TestTransformStateOnEveryWorker(t *testing.T) {
	ctx := context.Background()
	p, workers, newState := newFixtureState(t, ctx)
	p.RunWorkers(workers, func(w *tsbridge.Worker) {
		for _, file := range w.Files() {
			state := newState(w, file)
			if state.Resolver == nil || state.Checker != w.Checker() {
				t.Errorf("state not bound to its worker")
			}
			if state.Services.MacroManager.Checker() != w.Checker() {
				t.Errorf("macro table from another checker")
			}

			// A prereq and a captured expression render through the Luau port.
			statements := state.CapturePrereqs(func() {
				id := state.PushToVar(luau.NewCall(luau.NewProperty(luau.NewID("Instance"), "new")), "")
				if !strings.HasPrefix(id.Name, "instance") && id.Name != "" {
					t.Errorf("temp name = %q", id.Name)
				}
			})
			if statements.IsEmpty() {
				t.Errorf("no prereq was captured")
			}
			if got := luau.RenderAST(statements); !strings.Contains(got, "Instance.new()") {
				t.Errorf("rendered %q", got)
			}
			if len(state.PrereqStatementsStack) != 0 {
				t.Errorf("stack not balanced")
			}
			state.NoPrereqs(func() luau.Expression { return luau.NewNil() })
		}
	})
}

func TestTransformStateRejectsAnotherWorkersFile(t *testing.T) {
	ctx := context.Background()
	_, workers, newState := newFixtureState(t, ctx)
	if len(workers) < 2 {
		t.Skip("the partitioner put every file on one checker")
	}
	defer func() {
		if recover() == nil {
			t.Errorf("a file owned by another worker must panic")
		}
	}()
	newState(workers[0], workers[1].Files()[0])
}

func TestTraversal(t *testing.T) {
	ctx := context.Background()
	p, _, _ := newFixtureState(t, ctx)
	for _, file := range p.SourceFilesToCompile() {
		var visit func(node *ast.Node)
		visit = func(node *ast.Node) {
			if GetModuleAncestor(node) == nil {
				t.Errorf("no module ancestor")
			}
			if ast.IsParenthesizedExpression(node) {
				if SkipDownwards(node) == node {
					t.Errorf("SkipDownwards stayed on a parenthesized expression")
				}
				if SkipUpwards(node.Expression()) != SkipUpwards(node) {
					t.Errorf("SkipUpwards disagrees from inside and outside")
				}
			}
			node.ForEachChild(func(child *ast.Node) bool {
				visit(child)
				return false
			})
		}
		visit(file.AsNode())
	}
}
