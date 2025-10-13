package declarative_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	declarative "github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/openai/openai-go"
)

type stageDef struct {
	name  string
	order int
	tool  string
}

type mockFlowController struct {
	stages    []stageDef
	preResult core.RuleResult
	preErr    error
}

func (m *mockFlowController) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	if stage == core.Pre {
		if m.preErr != nil {
			return core.RuleResult{}, m.preErr
		}
		if m.preResult.SkippedStages == nil {
			m.preResult.SkippedStages = map[string]bool{}
		}
		return m.preResult, nil
	}
	return core.RuleResult{Allowed: true}, nil
}

func (m *mockFlowController) Query(ctx context.Context, query string, onSolution func(map[string]any) error) error {
	switch {
	case strings.HasPrefix(query, "flow_stage"):
		for _, st := range m.stages {
			sol := map[string]any{"Order": fmt.Sprintf("%d", st.order), "StageName": st.name}
			if err := onSolution(sol); err != nil {
				return err
			}
		}
	case strings.HasPrefix(query, "stage_tool"):
		for _, st := range m.stages {
			sol := map[string]any{"StageName": st.name, "ToolName": st.tool}
			if err := onSolution(sol); err != nil {
				return err
			}
		}
	}
	return nil
}

type mockRetriever struct {
	calls  []retrieve.Request
	result retrieve.Result
	err    error
	onCall func()
}

func (m *mockRetriever) Retrieve(ctx context.Context, req retrieve.Request) (retrieve.Result, error) {
	m.calls = append(m.calls, req)
	if m.onCall != nil {
		m.onCall()
	}
	if m.err != nil {
		return retrieve.Result{}, m.err
	}
	return m.result, nil
}

type mockLLM struct {
	calls    []llm.Request
	response llm.Response
	err      error
	onCall   func(llm.Request)
}

func (m *mockLLM) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	m.calls = append(m.calls, req)
	if m.onCall != nil {
		m.onCall(req)
	}
	if m.err != nil {
		return llm.Response{}, m.err
	}
	return m.response, nil
}

type mockPostRules struct {
	calls  int
	result core.PostRuleResult
	err    error
}

func (m *mockPostRules) Post(ctx context.Context, q core.Query, docs []core.Doc, meta map[string]any) (core.PostRuleResult, error) {
	m.calls++
	if m.err != nil {
		return core.PostRuleResult{}, m.err
	}
	return m.result, nil
}

func TestDeclarativeOrchestratorHappyPath(t *testing.T) {
	var order []string
	retriever := &mockRetriever{
		result: retrieve.Result{Docs: []core.Doc{{ID: "doc1", Text: "alpha"}}},
		onCall: func() { order = append(order, "retriever") },
	}
	generator := &mockLLM{
		response: llm.Response{Text: "final answer"},
		onCall:   func(req llm.Request) { order = append(order, "llm") },
	}

	fc := &mockFlowController{stages: []stageDef{
		{name: "retrieve", order: 1, tool: "retriever_tool"},
		{name: "generate", order: 2, tool: "llm_tool"},
	}, preResult: core.RuleResult{Allowed: true}}

	tools := map[string]any{
		"retriever_tool": retriever,
		"llm_tool":       generator,
	}

	orch, err := declarative.New(fc, tools, "flow", core.Observability{}, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	answer, err := orch.Run(context.Background(), core.Query{Text: "hello"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if answer.Text != "final answer" {
		t.Fatalf("unexpected answer text: %q", answer.Text)
	}
	if want := []string{"retriever", "llm"}; len(order) != len(want) || order[0] != want[0] || order[1] != want[1] {
		t.Fatalf("unexpected call order: %v", order)
	}
	if len(generator.calls) != 1 || len(generator.calls[0].Context) != 1 || generator.calls[0].Context[0] != "alpha" {
		t.Fatalf("llm context mismatch: %#v", generator.calls)
	}
}

func TestDeclarativeOrchestratorSkipsStages(t *testing.T) {
	retriever := &mockRetriever{result: retrieve.Result{Docs: []core.Doc{{ID: "doc1"}}}}
	generator := &mockLLM{response: llm.Response{Text: "should not run"}}

	fc := &mockFlowController{stages: []stageDef{
		{name: "retrieve", order: 1, tool: "retriever_tool"},
		{name: "generate", order: 2, tool: "llm_tool"},
	}, preResult: core.RuleResult{Allowed: true, SkippedStages: map[string]bool{"generate": true}}}

	orch, err := declarative.New(fc, map[string]any{
		"retriever_tool": retriever,
		"llm_tool":       generator,
	}, "flow", core.Observability{}, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if _, err := orch.Run(context.Background(), core.Query{Text: "hello"}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(generator.calls) != 0 {
		t.Fatalf("llm should not have been called")
	}
}

func TestDeclarativeOrchestratorErrorAndDenial(t *testing.T) {
	t.Run("tool error", func(t *testing.T) {
		retriever := &mockRetriever{err: errors.New("boom")}
		fc := &mockFlowController{stages: []stageDef{{name: "retrieve", order: 1, tool: "retriever_tool"}}, preResult: core.RuleResult{Allowed: true}}

		orch, err := declarative.New(fc, map[string]any{"retriever_tool": retriever}, "flow", core.Observability{}, nil)
		if err != nil {
			t.Fatalf("New returned error: %v", err)
		}

		_, err = orch.Run(context.Background(), core.Query{Text: "hello"})
		if err == nil || !strings.Contains(err.Error(), "retriever_tool") {
			t.Fatalf("expected retriever error, got %v", err)
		}
	})

	t.Run("denial propagation", func(t *testing.T) {
		retriever := &mockRetriever{result: retrieve.Result{Docs: []core.Doc{{ID: "doc1"}}}}
		post := &mockPostRules{result: core.PostRuleResult{Denied: true, Reason: "blocked"}}
		generator := &mockLLM{response: llm.Response{Text: "should be skipped"}}

		fc := &mockFlowController{stages: []stageDef{
			{name: "retrieve", order: 1, tool: "retriever_tool"},
			{name: "post", order: 2, tool: "post_rules"},
			{name: "generate", order: 3, tool: "llm_tool"},
		}, preResult: core.RuleResult{Allowed: true}}

		tools := map[string]any{
			"retriever_tool": retriever,
			"post_rules":     post,
			"llm_tool":       generator,
		}

		orch, err := declarative.New(fc, tools, "flow", core.Observability{}, nil)
		if err != nil {
			t.Fatalf("New returned error: %v", err)
		}

		answer, err := orch.Run(context.Background(), core.Query{Text: "hello"})
		if !errors.Is(err, core.ErrDenied) {
			t.Fatalf("expected ErrDenied, got %v", err)
		}
		if answer.Meta["denial_reason"] != "blocked" {
			t.Fatalf("expected denial reason to propagate, got %#v", answer.Meta)
		}
		if len(generator.calls) != 0 {
			t.Fatalf("llm should not have been called after denial")
		}
	})
}

// --- Regression harness for dependency inference ---

type stubDeclarativePlan struct {
	stages []stageDef
	pre    core.RuleResult
}

var nextPlan stubDeclarativePlan

type stubRuleEngine struct{}

func (s *stubRuleEngine) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	if stage == core.Pre {
		if nextPlan.pre.SkippedStages == nil {
			nextPlan.pre.SkippedStages = map[string]bool{}
		}
		return nextPlan.pre, nil
	}
	return core.RuleResult{Allowed: true}, nil
}

func (s *stubRuleEngine) Query(ctx context.Context, query string, onSolution func(map[string]any) error) error {
	switch {
	case strings.HasPrefix(query, "flow_stage"):
		for _, st := range nextPlan.stages {
			sol := map[string]any{"Order": fmt.Sprintf("%d", st.order), "StageName": st.name}
			if err := onSolution(sol); err != nil {
				return err
			}
		}
	case strings.HasPrefix(query, "stage_tool"):
		for _, st := range nextPlan.stages {
			sol := map[string]any{"StageName": st.name, "ToolName": st.tool}
			if err := onSolution(sol); err != nil {
				return err
			}
		}
	}
	return nil
}

type stubLLMClient struct {
	template string
	calls    []llm.Request
}

func (s *stubLLMClient) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	s.calls = append(s.calls, req)
	return llm.Response{Text: fmt.Sprintf("tmpl:%s|prompt:%s|ctx:%d", s.template, req.Prompt, len(req.Context))}, nil
}

func installDeclarativeStubs(t *testing.T) {
	origRetriever := manglekit.Registry.Retriever["bm25"]
	origLLM := manglekit.Registry.LLM["openai"]
	origRules := manglekit.Registry.Rules["mangle"]

	manglekit.Registry.Retriever["bm25"] = func(opts retrieve.BM25Options) (retrieve.Retriever, error) {
		return &mockRetriever{result: retrieve.Result{Docs: []core.Doc{{ID: "doc1", Text: "note", Source: "bm25"}}}}, nil
	}
	manglekit.Registry.LLM["openai"] = func(opts llm.OpenAIOptions, _ *openai.Client) (llm.Client, error) {
		return &stubLLMClient{template: opts.PromptTemplate}, nil
	}
	manglekit.Registry.Rules["mangle"] = func(ctx context.Context, opts core.MangleOptions) (core.RuleSet, error) {
		return &stubRuleEngine{}, nil
	}

	t.Cleanup(func() {
		manglekit.Registry.Retriever["bm25"] = origRetriever
		manglekit.Registry.LLM["openai"] = origLLM
		manglekit.Registry.Rules["mangle"] = origRules
	})
}

func TestDeclarativeOrchestratorDependencyInferenceLiteralString(t *testing.T) {
	installDeclarativeStubs(t)
	t.Setenv("OPENAI_API_KEY", "fake")

	nextPlan = stubDeclarativePlan{
		stages: []stageDef{
			{name: "retrieve_stage", order: 1, tool: "retriever_tool"},
			{name: "generate_stage", order: 2, tool: "writer"},
		},
		pre: core.RuleResult{Allowed: true},
	}

	cfg := manglekit.Config{
		Orchestrator: manglekit.OrchestratorConfig{Type: "declarative", FlowName: "literal_flow"},
		Tools: map[string]manglekit.ToolConfig{
			"retriever_tool": {
				Provider: "bm25",
				Params:   map[string]any{"path": "ignored"},
			},
			"writer": {
				Provider: "openai",
				Params: map[string]any{
					"model":          "stub-model",
					"promptTemplate": "retriever_tool",
				},
			},
		},
	}

	builder := manglekit.NewBuilder().WithConfig(&cfg).WithRules(&core.MangleOptions{Path: []string{"unused.dlog"}})

	orch, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("builder failed: %v", err)
	}
	defer orch.Close(context.Background())

	answer, err := orch.Run(context.Background(), core.Query{Text: "prompt"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(answer.Text, "tmpl:retriever_tool") {
		t.Fatalf("unexpected answer text: %q", answer.Text)
	}
}
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
