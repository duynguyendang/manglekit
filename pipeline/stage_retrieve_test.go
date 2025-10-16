package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/retrieve"
)

type mockRetriever struct {
	docs []core.Doc
	err  error
}

func (m *mockRetriever) Retrieve(ctx context.Context, req retrieve.Request) (retrieve.Result, error) {
	if m.err != nil {
		return retrieve.Result{}, m.err
	}
	return retrieve.Result{Docs: m.docs}, nil
}

func TestRetrieveStage_Execute(t *testing.T) {
	testLogger := logger.NewStdLogger()

	t.Run("happy path", func(t *testing.T) {
		retriever := &mockRetriever{
			docs: []core.Doc{{ID: "1", Text: "doc1"}},
		}
		stage := &RetrieveStage{
			Retriever: retriever,
			TopK:      1,
			Logger:    testLogger,
		}
		p := &PipelineContext{Ctx: context.Background()}

		err := stage.Execute(p)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(p.OriginalDocs) != 1 {
			t.Errorf("expected 1 doc, got %d", len(p.OriginalDocs))
		}
		if p.OriginalDocs[0].ID != "1" {
			t.Errorf("expected doc ID '1', got %s", p.OriginalDocs[0].ID)
		}
		if p.RetrieveMS < 0 {
			t.Error("expected RetrieveMS to be non-negative")
		}
	})

	t.Run("retriever returns error", func(t *testing.T) {
		testErr := errors.New("retriever error")
		retriever := &mockRetriever{err: testErr}
		stage := &RetrieveStage{
			Retriever: retriever,
			Logger:    testLogger,
		}
		p := &PipelineContext{Ctx: context.Background()}

		err := stage.Execute(p)

		if !strings.Contains(err.Error(), testErr.Error()) {
			t.Errorf("expected error to contain '%v', got '%v'", testErr, err)
		}
	})

	t.Run("retriever returns no documents", func(t *testing.T) {
		retriever := &mockRetriever{docs: []core.Doc{}}
		stage := &RetrieveStage{
			Retriever: retriever,
			Logger:    testLogger,
		}
		p := &PipelineContext{Ctx: context.Background()}

		err := stage.Execute(p)

		if !errors.Is(err, core.ErrNoEvidence) {
			t.Errorf("expected error %v, got %v", core.ErrNoEvidence, err)
		}
	})

	t.Run("nil retriever", func(t *testing.T) {
		stage := &RetrieveStage{
			Retriever: nil,
			Logger:    testLogger,
		}
		p := &PipelineContext{Ctx: context.Background()}

		err := stage.Execute(p)

		if !errors.Is(err, core.ErrNoEvidence) {
			t.Errorf("expected error %v, got %v", core.ErrNoEvidence, err)
		}
	})
}
