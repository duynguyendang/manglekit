package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
)

// mockRetriever is a simple mock implementation of the retrieve.Retriever interface.
type mockRetriever struct {
	retrieve.Retriever
	err error
}

func (m *mockRetriever) Retrieve(ctx context.Context, req retrieve.Request) (retrieve.Result, error) {
	if m.err != nil {
		return retrieve.Result{}, m.err
	}
	return retrieve.Result{Docs: []core.Doc{{Text: "mock"}}}, nil
}

// mockReranker is a simple mock implementation of the rerank.Reranker interface.
type mockReranker struct {
	rerank.Reranker
	err error
}

func (m *mockReranker) Rerank(ctx context.Context, req rerank.Request) ([]rerank.ScoredDoc, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []rerank.ScoredDoc{{Doc: core.Doc{Text: "mock"}}}, nil
}

// mockLLM is a simple mock implementation of the llm.Client interface.
type mockLLM struct {
	llm.Client
	err error
}

func (m *mockLLM) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	if m.err != nil {
		return llm.Response{}, m.err
	}
	return llm.Response{Text: "mock"}, nil
}

// mockRules is a simple mock implementation of the core.RuleSet interface.
type mockRules struct {
	core.RuleSet
	err    error
	result core.RuleResult
}

func (m *mockRules) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	if m.err != nil {
		return core.RuleResult{}, m.err
	}
	return m.result, nil
}

func TestNewSandwich(t *testing.T) {
	opts := core.Options{
		Retriever: &mockRetriever{},
		LLM:       &mockLLM{},
		Reranker:  &mockReranker{},
	}
	o, err := NewSandwich(opts)
	if err != nil {
		t.Fatalf("expected NewSandwich to succeed, got %v", err)
	}
	if o == nil {
		t.Fatal("expected orchestrator to be non-nil")
	}
}

func TestRun(t *testing.T) {
	opts := core.Options{
		Retriever: &mockRetriever{},
		LLM:       &mockLLM{},
		Reranker:  &mockReranker{},
		Rules:     &mockRules{result: core.RuleResult{Allowed: true}},
	}
	o, err := NewSandwich(opts)
	if err != nil {
		t.Fatalf("expected NewSandwich to succeed, got %v", err)
	}

	q := core.Query{Text: "test"}
	a, err := o.Run(context.Background(), q)
	if err != nil {
		t.Fatalf("expected Run to succeed, got %v", err)
	}
	if a.Text != "mock" {
		t.Errorf("expected answer text to be mock, got %s", a.Text)
	}
}

func TestRun_PreRulesDeny(t *testing.T) {
	opts := core.Options{
		Retriever: &mockRetriever{},
		LLM:       &mockLLM{},
		Reranker:  &mockReranker{},
		Rules:     &mockRules{result: core.RuleResult{Allowed: false, Reason: "denied"}},
	}
	o, err := NewSandwich(opts)
	if err != nil {
		t.Fatalf("expected NewSandwich to succeed, got %v", err)
	}

	q := core.Query{Text: "test"}
	_, err = o.Run(context.Background(), q)
	if err == nil {
		t.Fatal("expected Run to fail")
	}
	if !errors.Is(err, core.ErrDenied) {
		t.Errorf("expected error to be of type ErrDenied, got %v", err)
	}
}

func TestRun_PostRulesDeny(t *testing.T) {
	rules := &mockRules{}
	opts := core.Options{
		Retriever: &mockRetriever{},
		LLM:       &mockLLM{},
		Reranker:  &mockReranker{},
		Rules:     rules,
	}
	o, err := NewSandwich(opts)
	if err != nil {
		t.Fatalf("expected NewSandwich to succeed, got %v", err)
	}

	rules.result = core.RuleResult{Allowed: true}
	q := core.Query{Text: "test"}
	_, err = o.Run(context.Background(), q)
	if err != nil {
		t.Fatalf("expected Run to succeed, got %v", err)
	}

	rules.result = core.RuleResult{Allowed: false, Reason: "denied"}
	_, err = o.Run(context.Background(), q)
	if err == nil {
		t.Fatal("expected Run to fail")
	}
	if !errors.Is(err, core.ErrDenied) {
		t.Errorf("expected error to be of type ErrDenied, got %v", err)
	}
}
