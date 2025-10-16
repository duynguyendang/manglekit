package pipeline

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
)

// This file contains an end-to-end integration test for the Sandwich orchestrator,
// verifying that the stage-based runner correctly executes a full pipeline with mocks.

func TestSandwich_Execute_EndToEnd(t *testing.T) {
	// 1. Setup Mock Components
	mockRetriever := &mockRetriever{
		docs: []core.Doc{{ID: "1", Text: "retrieved doc"}},
	}
	mockReranker := &mockReranker{
		rerankedDocs: []rerank.ScoredDoc{{Doc: core.Doc{ID: "1"}, Score: 0.95}},
	}
	mockLLM := &mockLLMClient{
		response: llm.Response{Text: "final answer"},
	}
	mockRuleSet := &mockRuleSet{
		result: core.RuleResult{
			Allowed: true,
			Mutate: func(q *core.Query, a *core.Answer) {
				if q.Text == "original query" {
					q.Text = "mutated query" // Pre-rule mutation
				}
				if a.Text == "final answer" {
					a.Text = "mutated answer" // Post-rule mutation
				}
			},
		},
	}
	testLogger := logger.NewStdLogger()

	// 2. Configure Options for the Orchestrator
	opts := core.Options{
		Retriever: mockRetriever,
		Reranker:  mockReranker,
		LLM:       mockLLM,
		Rules:     mockRuleSet,
		Obs: core.Observability{
			Logger: testLogger,
		},
	}

	// 3. Create the Orchestrator
	orchestrator, err := NewSandwich(opts)
	if err != nil {
		t.Fatalf("Failed to create sandwich orchestrator: %v", err)
	}
	s, ok := orchestrator.(*Sandwich)
	if !ok {
		t.Fatal("NewSandwich did not return a *Sandwich")
	}

	// 4. Execute the pipeline
	query := core.Query{Text: "original query"}
	answer, err := s.Execute(context.Background(), "test-session", query)

	// 5. Assert the results
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expectedAnswer := "mutated answer"
	if answer.Text != expectedAnswer {
		t.Errorf("Expected final answer to be '%s', got '%s'", expectedAnswer, answer.Text)
	}

	if len(answer.Citations) != 1 {
		t.Fatalf("Expected 1 citation, got %d", len(answer.Citations))
	}
	if answer.Citations[0].Score != 0.95 {
		t.Errorf("Expected citation score to be 0.95, got %f", answer.Citations[0].Score)
	}
}
