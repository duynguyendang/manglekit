package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mocks for testing
type mockRetriever struct {
	Result retrieve.Result
	Err    error
}

func (m *mockRetriever) Retrieve(req retrieve.Request) (retrieve.Result, error) {
	return m.Result, m.Err
}

type mockReranker struct {
	Result []rerank.ScoredDoc
	Err    error
}

func (m *mockReranker) Rerank(req rerank.Request) ([]rerank.ScoredDoc, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	// Simple pass-through if no specific result is set
	if m.Result != nil {
		return m.Result, nil
	}
	res := make([]rerank.ScoredDoc, len(req.Docs))
	for i, d := range req.Docs {
		res[i] = rerank.ScoredDoc{Doc: d, Score: 1.0 - float64(i)*0.1}
	}
	return res, nil
}

type mockLLM struct {
	Result llm.Response
	Err    error
}

func (m *mockLLM) Complete(req llm.Request) (llm.Response, error) {
	return m.Result, m.Err
}

type mockRuleSet struct {
	PreResult  core.RuleResult
	PreErr     error
	PostResult core.RuleResult
	PostErr    error
}

func (m *mockRuleSet) Evaluate(stage core.Stage, q core.Query, a *core.Answer) (core.RuleResult, error) {
	if stage == core.Pre {
		return m.PreResult, m.PreErr
	}
	return m.PostResult, m.PostErr
}

func TestSandwichRun(t *testing.T) {
	ctx := context.Background()
	query := core.Query{Text: "what is manglekit?"}

	t.Run("successful run with all components", func(t *testing.T) {
		opts := core.Options{
			Retriever: &mockRetriever{Result: retrieve.Result{Docs: []core.Doc{{ID: "1", Text: "doc1"}}}},
			Reranker:  &mockReranker{},
			LLM:       &mockLLM{Result: llm.Response{Text: "final answer"}},
			Rules:     &mockRuleSet{PreResult: core.RuleResult{Allowed: true}, PostResult: core.RuleResult{Allowed: true}},
			TopK:      1,
		}
		orchestrator, err := pipeline.NewSandwich(opts)
		require.NoError(t, err)

		answer, err := orchestrator.Run(ctx, query)
		require.NoError(t, err)
		assert.Equal(t, "final answer", answer.Text)
		assert.Len(t, answer.Citations, 1)
		assert.Equal(t, 1.0, answer.Citations[0].Score)
		assert.Contains(t, answer.Meta, "retrieve_ms")
		assert.Contains(t, answer.Meta, "llm_ms")
	})

	t.Run("retrieval error", func(t *testing.T) {
		opts := core.Options{
			Retriever: &mockRetriever{Err: errors.New("db is down")},
			LLM:       &mockLLM{},
		}
		orchestrator, err := pipeline.NewSandwich(opts)
		require.NoError(t, err)

		_, err = orchestrator.Run(ctx, query)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "retrieve failed: db is down")
	})

	t.Run("pre-rule denies request", func(t *testing.T) {
		opts := core.Options{
			Retriever: &mockRetriever{},
			LLM:       &mockLLM{},
			Rules:     &mockRuleSet{PreResult: core.RuleResult{Allowed: false, Reason: "policy violation"}},
		}
		orchestrator, err := pipeline.NewSandwich(opts)
		require.NoError(t, err)

		_, err = orchestrator.Run(ctx, query)
		require.Error(t, err)
		assert.ErrorIs(t, err, core.ErrDenied)
		assert.Contains(t, err.Error(), "policy violation")
	})

	t.Run("fallback threshold not met", func(t *testing.T) {
		opts := core.Options{
			Retriever:         &mockRetriever{Result: retrieve.Result{Docs: []core.Doc{{ID: "1", Text: "doc1"}}}},
			Reranker:          &mockReranker{Result: []rerank.ScoredDoc{{Doc: core.Doc{ID: "1"}, Score: 0.4}}},
			LLM:               &mockLLM{},
			FallbackThreshold: 0.5,
		}
		orchestrator, err := pipeline.NewSandwich(opts)
		require.NoError(t, err)

		_, err = orchestrator.Run(ctx, query)
		require.Error(t, err)
		assert.ErrorIs(t, err, core.ErrNoEvidence)
	})
}
