package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
)

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
			Retrievers: map[string]core.Retriever{"mock": &mock.Retriever{}},
			Rerankers:  map[string]core.Reranker{"mock": &mock.Reranker{}},
			Rules:      map[string]core.RuleSet{"mock": mock.NewRuleSet()},
			LLMs:       map[string]core.LLMClient{"mock": &mock.LLM{}},
			Closers:    []core.ResourceCloser{closer1, closer2},
		}

		opts := SandwichOptions{
			Retriever: "mock",
			Reranker:  "mock",
			LLM:       "mock",
		}
		orch, err := NewSandwich(context.Background(), deps, opts)
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
			Retrievers: map[string]core.Retriever{"mock": &mock.Retriever{}},
			Rerankers:  map[string]core.Reranker{"mock": &mock.Reranker{}},
			Rules:      map[string]core.RuleSet{"mock": mock.NewRuleSet()},
			LLMs:       map[string]core.LLMClient{"mock": &mock.LLM{}},
			Closers:    []core.ResourceCloser{closer1, closer2},
		}

		opts := SandwichOptions{
			Retriever: "mock",
			Reranker:  "mock",
			LLM:       "mock",
		}
		orch, err := NewSandwich(context.Background(), deps, opts)
		if err != nil {
			t.Fatalf("NewSandwich() error = %v", err)
		}

		if err := orch.Close(context.Background()); err == nil {
			t.Error("Close() expected an error, but got nil")
		}
	})
}
