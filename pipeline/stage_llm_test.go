package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
)

type mockLLMClient struct {
	response llm.Response
	err      error
}

func (m *mockLLMClient) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	if m.err != nil {
		return llm.Response{}, m.err
	}
	return m.response, nil
}

func TestLLMStage_Execute(t *testing.T) {
	testLogger := logger.NewStdLogger()
	finalDocs := []core.Doc{{ID: "1", Text: "doc1 text", Source: "s1"}, {ID: "2", Text: "doc2 text", Source: "s2"}}
	rerankedDocs := []rerank.ScoredDoc{{Doc: finalDocs[0], Score: 0.9}}

	t.Run("happy path with reranked docs", func(t *testing.T) {
		client := &mockLLMClient{
			response: llm.Response{Text: "llm says hi", Usage: map[string]int{"total_tokens": 10}},
		}
		stage := &LLMStage{LLM: client, Logger: testLogger}
		p := &PipelineContext{
			Ctx:          context.Background(),
			FinalDocs:    finalDocs,
			RerankedDocs: rerankedDocs,
			Answer:       core.Answer{Meta: make(map[string]any)},
		}

		err := stage.Execute(p)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if p.Response != "llm says hi" {
			t.Errorf("unexpected response: got '%s'", p.Response)
		}
		if p.Answer.Text != "llm says hi" {
			t.Errorf("unexpected answer text: got '%s'", p.Answer.Text)
		}
		if len(p.Citations) != 1 {
			t.Errorf("expected 1 citation, got %d", len(p.Citations))
		}
		if p.Citations[0].Score != 0.9 {
			t.Errorf("expected citation score to be 0.9, got %f", p.Citations[0].Score)
		}
		if p.LLMMS < 0 {
			t.Error("expected LLMMS to be non-negative")
		}
		if p.Answer.Meta["token_usage"] == nil {
			t.Error("expected token usage in answer meta")
		}
	})

	t.Run("happy path without reranked docs", func(t *testing.T) {
		client := &mockLLMClient{
			response: llm.Response{Text: "llm says hi"},
		}
		stage := &LLMStage{LLM: client, Logger: testLogger}
		p := &PipelineContext{
			Ctx:       context.Background(),
			FinalDocs: finalDocs,
			Answer:    core.Answer{Meta: make(map[string]any)},
		}

		err := stage.Execute(p)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(p.Citations) != 2 {
			t.Errorf("expected 2 citations, got %d", len(p.Citations))
		}
		if p.Citations[0].Score != 0 {
			t.Error("expected citation score to be 0")
		}
	})

	t.Run("llm client returns error", func(t *testing.T) {
		testErr := errors.New("llm error")
		client := &mockLLMClient{err: testErr}
		stage := &LLMStage{LLM: client, Logger: testLogger}
		p := &PipelineContext{Ctx: context.Background(), FinalDocs: finalDocs}

		err := stage.Execute(p)
		if !strings.Contains(err.Error(), testErr.Error()) {
			t.Errorf("expected error to contain '%v', got '%v'", testErr, err)
		}
	})

	t.Run("nil llm client", func(t *testing.T) {
		stage := &LLMStage{LLM: nil, Logger: testLogger}
		p := &PipelineContext{Ctx: context.Background()}

		err := stage.Execute(p)
		if err == nil {
			t.Error("expected an error for nil client")
		}
	})
}
