package declarative

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/retrieve"
)

// mockFlowController is a simple mock implementation of the core.FlowController interface.
type mockFlowController struct {
	core.FlowController
	err    error
	result core.RuleResult
	flow   []flowStage
}

func (m *mockFlowController) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	if m.err != nil {
		return core.RuleResult{}, m.err
	}
	return m.result, nil
}

func (m *mockFlowController) Query(ctx context.Context, query string, onSolution func(map[string]any) error) error {
	if m.err != nil {
		return m.err
	}
	for i, f := range m.flow {
		sol := map[string]any{
			"StageName": f.Name,
			"Order":     strconv.Itoa(i),
		}
		if f.Tool != "" {
			sol["ToolName"] = f.Tool
		}
		if err := onSolution(sol); err != nil {
			return err
		}
	}
	return nil
}

// mockRetrieverTool is a simple mock implementation of a tool.
type mockRetrieverTool struct {
	err error
}

func (m *mockRetrieverTool) Retrieve(ctx context.Context, req retrieve.Request) (retrieve.Result, error) {
	if m.err != nil {
		return retrieve.Result{}, m.err
	}
	return retrieve.Result{Docs: []core.Doc{{Text: "mock"}}}, nil
}

// mockLLMTool is a simple mock implementation of a tool.
type mockLLMTool struct {
	err error
}

func (m *mockLLMTool) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	if m.err != nil {
		return llm.Response{}, m.err
	}
	return llm.Response{Text: "mock"}, nil
}

func TestNewDeclarativeOrchestrator(t *testing.T) {
	fc := &mockFlowController{}
	tools := map[string]any{
		"mock-tool": &mockRetrieverTool{},
	}
	o, err := New(fc, tools, "test-flow", nil, core.Observability{}, nil)
	if err != nil {
		t.Fatalf("expected New to succeed, got %v", err)
	}
	if o == nil {
		t.Fatal("expected orchestrator to be non-nil")
	}
}

func TestRun_Declarative(t *testing.T) {
	fc := &mockFlowController{
		result: core.RuleResult{Allowed: true},
		flow: []flowStage{
			{Name: "retrieve", Tool: "mock-retriever"},
			{Name: "llm", Tool: "mock-llm"},
		},
	}
	tools := map[string]any{
		"mock-retriever": &mockRetrieverTool{},
		"mock-llm":       &mockLLMTool{},
	}
	o, err := New(fc, tools, "test-flow", nil, core.Observability{}, nil)
	if err != nil {
		t.Fatalf("expected New to succeed, got %v", err)
	}

	q := core.Query{Text: "test"}
	a, err := o.Execute(context.Background(), "test-session", q)
	if err != nil {
		t.Fatalf("expected Execute to succeed, got %v", err)
	}
	if a.Text != "mock" {
		t.Errorf("expected answer text to be mock, got %s", a.Text)
	}
}

func TestRun_PreRulesDeny_Declarative(t *testing.T) {
	fc := &mockFlowController{
		result: core.RuleResult{Allowed: false, Reason: "denied"},
		flow: []flowStage{
			{Name: "retrieve", Tool: "mock-retriever"},
		},
	}
	tools := map[string]any{
		"mock-retriever": &mockRetrieverTool{},
	}
	o, err := New(fc, tools, "test-flow", nil, core.Observability{}, nil)
	if err != nil {
		t.Fatalf("expected New to succeed, got %v", err)
	}

	q := core.Query{Text: "test"}
	_, err = o.Execute(context.Background(), "test-session", q)
	if err == nil {
		t.Fatal("expected Execute to fail")
	}
	if !errors.Is(err, core.ErrDenied) {
		t.Errorf("expected error to be of type ErrDenied, got %v", err)
	}
}
