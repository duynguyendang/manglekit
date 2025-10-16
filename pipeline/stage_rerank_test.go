package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/rerank"
)

type mockReranker struct {
	rerankedDocs []rerank.ScoredDoc
	err          error
}

func (m *mockReranker) Rerank(ctx context.Context, req rerank.Request) ([]rerank.ScoredDoc, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rerankedDocs, nil
}

func TestRerankStage_Execute(t *testing.T) {
	testLogger := logger.NewStdLogger()
	originalDocs := []core.Doc{{ID: "1", Text: "doc1"}, {ID: "2", Text: "doc2"}}

	t.Run("happy path", func(t *testing.T) {
		reranker := &mockReranker{
			rerankedDocs: []rerank.ScoredDoc{
				{Doc: core.Doc{ID: "2"}, Score: 0.9},
				{Doc: core.Doc{ID: "1"}, Score: 0.8},
			},
		}
		stage := &RerankStage{
			Reranker: reranker,
			Logger:   testLogger,
		}
		p := &PipelineContext{Ctx: context.Background(), OriginalDocs: originalDocs}

		err := stage.Execute(p)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(p.RerankedDocs) != 2 {
			t.Fatalf("expected 2 reranked docs, got %d", len(p.RerankedDocs))
		}
		if p.RerankedDocs[0].Doc.ID != "2" {
			t.Errorf("expected first doc to be '2', got '%s'", p.RerankedDocs[0].Doc.ID)
		}
		if p.BestScore != 0.9 {
			t.Errorf("expected best score to be 0.9, got %f", p.BestScore)
		}
		if len(p.FinalDocs) != 2 {
			t.Errorf("expected 2 final docs, got %d", len(p.FinalDocs))
		}
		if p.RerankMS < 0 {
			t.Error("expected RerankMS to be non-negative")
		}
	})

	t.Run("nil reranker (no-op)", func(t *testing.T) {
		stage := &RerankStage{Reranker: nil, Logger: testLogger}
		p := &PipelineContext{Ctx: context.Background(), OriginalDocs: originalDocs}

		err := stage.Execute(p)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(p.RerankedDocs) != 0 {
			t.Error("expected no reranked docs")
		}
		if len(p.FinalDocs) != 2 || p.FinalDocs[0].ID != "1" {
			t.Error("expected final docs to be the original docs")
		}
	})

	t.Run("reranker returns error", func(t *testing.T) {
		testErr := errors.New("reranker error")
		reranker := &mockReranker{err: testErr}
		stage := &RerankStage{Reranker: reranker, Logger: testLogger}
		p := &PipelineContext{Ctx: context.Background(), OriginalDocs: originalDocs}

		err := stage.Execute(p)

		if !strings.Contains(err.Error(), testErr.Error()) {
			t.Errorf("expected error to contain '%v', got '%v'", testErr, err)
		}
	})

	t.Run("fallback threshold not met", func(t *testing.T) {
		reranker := &mockReranker{
			rerankedDocs: []rerank.ScoredDoc{{Doc: core.Doc{ID: "1"}, Score: 0.5}},
		}
		stage := &RerankStage{
			Reranker:          reranker,
			FallbackThreshold: 0.6,
			Logger:            testLogger,
		}
		p := &PipelineContext{Ctx: context.Background(), OriginalDocs: originalDocs}

		err := stage.Execute(p)

		if !errors.Is(err, core.ErrNoEvidence) {
			t.Errorf("expected error %v, got %v", core.ErrNoEvidence, err)
		}
		if p.BestScore != 0.5 {
			t.Errorf("expected best score to be 0.5, got %f", p.BestScore)
		}
	})
}
