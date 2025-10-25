package sandwich_test

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
)

func TestNewSandwich(t *testing.T) {
	t.Parallel()
	deps := core.Resolved{
		Retrievers: map[string]core.Retriever{"mock": mock.NewRetriever(nil)},
		Rerankers:  map[string]core.Reranker{"mock": mock.NewReranker(nil)},
		Rules:      map[string]core.RuleSet{"mock": mock.NewRuleSet()},
		LLMs:       map[string]core.LLMClient{"mock": mock.NewLLM("test-model", "")},
	}

	t.Run("should create a new sandwich orchestrator", func(t *testing.T) {
		t.Parallel()
		opts := &sandwich.SandwichOptions{
			Retriever: "mock",
			Reranker:  "mock",
			LLM:       "mock",
		}
		orch, err := sandwich.NewSandwich(context.Background(), deps, opts)
		if err != nil {
			t.Fatalf("NewSandwich() error = %v", err)
		}
		if orch == nil {
			t.Fatal("NewSandwich() returned a nil orchestrator")
		}
	})

	t.Run("should return an error if retriever is not found", func(t *testing.T) {
		t.Parallel()
		opts := &sandwich.SandwichOptions{
			Retriever: "not-found",
			Reranker:  "mock",
			LLM:       "mock",
		}
		_, err := sandwich.NewSandwich(context.Background(), deps, opts)
		if err == nil {
			t.Fatal("NewSandwich() expected an error, but got nil")
		}
	})
}

func TestSandwich_Execute(t *testing.T) {
	t.Parallel()

	deps := core.Resolved{
		Retrievers: map[string]core.Retriever{"mock": mock.NewRetriever(map[string]string{"test query": "test response"})},
		Rerankers:  map[string]core.Reranker{"mock": mock.NewReranker(nil)},
		Rules:      map[string]core.RuleSet{"mock": mock.NewRuleSet()},
		LLMs:       map[string]core.LLMClient{"mock": mock.NewLLM("test-model", "")},
	}

	opts := &sandwich.SandwichOptions{
		Retriever: "mock",
		Reranker:  "mock",
		LLM:       "mock",
	}

	t.Run("should execute successfully", func(t *testing.T) {
		t.Parallel()
		orch, err := sandwich.NewSandwich(context.Background(), deps, opts)
		if err != nil {
			t.Fatalf("NewSandwich() error = %v", err)
		}

		query := core.Query{Text: "test query"}
		_, err = orch.Execute(context.Background(), "test-session", query)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
	})

	t.Run("should return an error if retriever fails", func(t *testing.T) {
		t.Parallel()
		deps := core.Resolved{
			Retrievers: map[string]core.Retriever{"mock": &mock.Retriever{
				RetrieveFunc: func(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
					return core.RetrieveResult{}, errors.New("retriever failed")
				},
			}},
			Rerankers: map[string]core.Reranker{"mock": mock.NewReranker(nil)},
			Rules:     map[string]core.RuleSet{"mock": mock.NewRuleSet()},
			LLMs:      map[string]core.LLMClient{"mock": mock.NewLLM("test-model", "")},
		}

		orch, err := sandwich.NewSandwich(context.Background(), deps, opts)
		if err != nil {
			t.Fatalf("NewSandwich() error = %v", err)
		}

		query := core.Query{Text: "test query"}
		_, err = orch.Execute(context.Background(), "test-session", query)
		if err == nil {
			t.Error("Execute() expected an error, but got nil")
		}
	})
}

func TestSandwich_Close(t *testing.T) {
	t.Parallel()

	t.Run("should call all closers", func(t *testing.T) {
		t.Parallel()

		var (
			closer1Called bool
			closer2Called bool
		)
		closer1 := func(ctx context.Context) error {
			closer1Called = true
			return nil
		}
		closer2 := func(ctx context.Context) error {
			closer2Called = true
			return nil
		}

		deps := core.Resolved{
			Retrievers: map[string]core.Retriever{"mock": mock.NewRetriever(nil)},
			Rerankers:  map[string]core.Reranker{"mock": mock.NewReranker(nil)},
			Rules:      map[string]core.RuleSet{"mock": mock.NewRuleSet()},
			LLMs:       map[string]core.LLMClient{"mock": mock.NewLLM("test-model", "")},
			Closers:    []core.ResourceCloser{closer1, closer2},
		}

		opts := &sandwich.SandwichOptions{
			Retriever: "mock",
			Reranker:  "mock",
			LLM:       "mock",
		}
		orch, err := sandwich.NewSandwich(context.Background(), deps, opts)
		if err != nil {
			t.Fatalf("NewSandwich() error = %v", err)
		}

		if err := orch.Close(context.Background()); err != nil {
			t.Errorf("Close() error = %v", err)
		}

		if !closer1Called {
			t.Error("expected closer1 to be called, but it was not")
		}
		if !closer2Called {
			t.Error("expected closer2 to be called, but it was not")
		}
	})

	t.Run("should return an error if a closer fails", func(t *testing.T) {
		t.Parallel()

		closer1 := func(ctx context.Context) error {
			return errors.New("closer1 failed")
		}
		closer2 := func(ctx context.Context) error {
			return nil
		}

		deps := core.Resolved{
			Retrievers: map[string]core.Retriever{"mock": mock.NewRetriever(nil)},
			Rerankers:  map[string]core.Reranker{"mock": mock.NewReranker(nil)},
			Rules:      map[string]core.RuleSet{"mock": mock.NewRuleSet()},
			LLMs:       map[string]core.LLMClient{"mock": mock.NewLLM("test-model", "")},
			Closers:    []core.ResourceCloser{closer1, closer2},
		}

		opts := &sandwich.SandwichOptions{
			Retriever: "mock",
			Reranker:  "mock",
			LLM:       "mock",
		}
		orch, err := sandwich.NewSandwich(context.Background(), deps, opts)
		if err != nil {
			t.Fatalf("NewSandwich() error = %v", err)
		}

		if err := orch.Close(context.Background()); err == nil {
			t.Error("Close() expected an error, but got nil")
		}
	})
}
