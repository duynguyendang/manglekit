package declarative

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/mangle/ast"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
)

type mockFlowController struct {
	mu           sync.Mutex
	decls        map[ast.PredicateSym]ast.Decl
	clauses      []ast.Clause
	transformers map[ast.PredicateSym]string
	query        string
	onSolution   func(map[string]any) error
	args         []string
}

func newMockFlowController(decls map[ast.PredicateSym]ast.Decl, clauses []ast.Clause, transformers map[ast.PredicateSym]string) *mockFlowController {
	return &mockFlowController{decls: decls, clauses: clauses, transformers: transformers}
}

func (m *mockFlowController) GetDeclarations(ctx context.Context, desafío string) (map[ast.PredicateSym]ast.Decl, []ast.Clause, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.decls, m.clauses, nil
}

func (m *mockFlowController) GetTransformer(ctx context.Context, name ast.PredicateSym) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.transformers[name]
	return t, ok, nil
}

func (m *mockFlowController) Close() error {
	return nil
}

func (m *mockFlowController) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	if q.Text == "skip" {
		return core.RuleResult{Allowed: true, SkippedStages: map[string]bool{"two": true}}, nil
	}
	return core.RuleResult{Allowed: true}, nil
}

func (m *mockFlowController) Query(ctx context.Context, query string, onSolution func(map[string]any) error) error {
	m.query = query
	m.onSolution = onSolution
	if query == `flow_stage("test", Order, StageName).` {
		onSolution(map[string]any{"Order": "1", "StageName": "one"})
		onSolution(map[string]any{"Order": "2", "StageName": "two"})
	}
	if query == `stage_tool(StageName, ToolName).` {
		onSolution(map[string]any{"StageName": "one", "ToolName": "mockTool"})
		onSolution(map[string]any{"StageName": "two", "ToolName": "bar"})
	}
	return nil
}

func TestDeclarativeOrchestrator(t *testing.T) {
	fc := newMockFlowController(nil, nil, nil)
	tools := map[string]any{
		"mockTool": &mock.Tool{
			Name: "mockTool",
			Fn: func(params mock.Params) (mock.Object, error) {
				fc.args = append(fc.args, "mockTool")
				return map[string]interface{}{
					"a": "foo",
					"b": "y",
				}, nil
			},
		},
		"bar": &mock.Tool{
			Name: "bar",
			Fn: func(p mock.Params) (mock.Object, error) {
				fc.args = append(fc.args, "bar")
				return mock.Object{"a": "bar"}, nil
			},
		},
	}
	o, err := New(fc, tools, "test", core.Observability{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	out, err := o.Run(context.Background(), core.Query{Text: "foo"})
	if err != nil {
		t.Fatal(err)
	}
	want := core.Answer{
		Meta: map[string]any{},
	}
	if diff := cmp.Diff(want, out, cmp.AllowUnexported(core.Answer{})); diff != "" {
		t.Errorf("o.Run() returned diff (-want, +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"mockTool", "bar"}, fc.args); diff != "" {
		t.Errorf("o.Run() returned diff (-want, +got):\n%s", diff)
	}
}

func TestSkip(t *testing.T) {
	fc := newMockFlowController(nil, nil, nil)
	var barCalled bool
	tools := map[string]any{
		"mockTool": &mock.Tool{
			Name: "mockTool",
			Fn: func(params mock.Params) (mock.Object, error) {
				return map[string]interface{}{
					"a": "foo",
					"b": "y",
				}, nil
			},
		},
		"bar": &mock.Tool{
			Name: "bar",
			Fn: func(p mock.Params) (mock.Object, error) {
				barCalled = true
				return mock.Object{"a": "bar"}, nil
			},
		},
	}

	o, err := New(fc, tools, "test", core.Observability{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Run(context.Background(), core.Query{Text: "skip"}); err != nil {
		t.Fatal(err)
	}
	if barCalled {
		t.Errorf("bar was called, but should have been skipped")
	}
}

func TestError(t *testing.T) {
	fc := newMockFlowController(nil, nil, nil)
	tools := map[string]any{
		"mockTool": &mock.Tool{
			Name: "mockTool",
			Fn: func(p mock.Params) (mock.Object, error) {
				return nil, fmt.Errorf("error")
			},
		},
	}
	o, err := New(fc, tools, "test", core.Observability{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Run(context.Background(), core.Query{Text: "foo"}); err == nil {
		t.Errorf("o.Run() succeeded, want error")
	}
}

func TestDependencyInference(t *testing.T) {
	fc := newMockFlowController(nil, nil, nil)
	tools := map[string]any{
		"mockTool": &mock.Tool{
			Name: "mockTool",
			Fn: func(p mock.Params) (mock.Object, error) {
				if got, want := p["a"], "baz"; got != want {
					t.Errorf("p['a'] = %q, want %q", got, want)
				}
				return mock.Object{"a": p["a"]}, nil
			},
		},
		"bar": &mock.Tool{
			Name: "bar",
			Fn: func(p mock.Params) (mock.Object, error) {
				return mock.Object{"a": "bar"}, nil
			},
		},
	}
	o, err := New(fc, tools, "test", core.Observability{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := core.Query{Text: "foo('baz')"}
	if _, err := o.Run(context.Background(), q); err != nil {
		t.Errorf("o.Run() failed: %v", err)
	}
}