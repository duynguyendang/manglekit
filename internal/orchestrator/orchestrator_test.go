package orchestrator

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"ndduy.dev/manglekit/internal/mangle"
	"ndduy.dev/manglekit/internal/types"
)

func TestRunFlowWithMockLLMAndFacts(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	rulesPath := filepath.Join("..", "..", "rules.dlog")
	factsPath := filepath.Join("..", "..", "data")

	processor, err := mangle.New(ctx, mangle.Config{RulesFile: rulesPath, FactsFile: factsPath})
	if err != nil {
		t.Fatalf("create mangle processor: %v", err)
	}

	mockRAG := &mockRAGRetriever{results: []string{
		"parse_intent_ner: Genkit normalizes the query and extracts entities.",
		"hybrid_retrieval: BM25 must/should filters combine with MRL ANN top-k.",
	}}

	mockLLM := &mockGateway{answer: "Mock answer with citations."}

	orch := &orchestrator{
		processor:  processor,
		llmGateway: mockLLM,
		rag:        mockRAG,
	}

	input := &types.QueryInput{Query: "Explain bm25 ann pipeline"}

	expanded, err := processor.PreProcess(input)
	if err != nil {
		t.Fatalf("preprocess input: %v", err)
	}

	wantTerms := []string{"ann", "approximate nearest neighbor", "best matching 25", "bm25"}
	for _, term := range wantTerms {
		if !slices.Contains(expanded.ExpansionTerms, term) {
			t.Fatalf("expected expansion term %q to be present, got %v", term, expanded.ExpansionTerms)
		}
	}

	if got := expanded.Filters["visibility"]; got != "public" {
		t.Fatalf("expected default visibility filter 'public', got %q", got)
	}

	response, err := orch.RunFlow(ctx, input)
	if err != nil {
		t.Fatalf("RunFlow error: %v", err)
	}

	expectedQuery := expanded.NormalizedQuery
	if len(expanded.ExpansionTerms) > 0 {
		expectedQuery = expectedQuery + " " + strings.Join(expanded.ExpansionTerms, " ")
	}

	if mockRAG.lastQuery != expectedQuery {
		t.Fatalf("unexpected rag query. got %q want %q", mockRAG.lastQuery, expectedQuery)
	}

	if len(mockLLM.chunks) == 0 || mockLLM.chunks[0].Text != mockRAG.results[0] {
		t.Fatalf("expected first chunk to match RAG context %q, got %+v", mockRAG.results[0], mockLLM.chunks)
	}

	if response.Answer != mockLLM.answer {
		t.Fatalf("unexpected answer: got %q want %q", response.Answer, mockLLM.answer)
	}

	if len(response.Explanations) == 0 {
		t.Fatalf("expected mangle explanation to be included")
	}

	explanation := response.Explanations[0]
	if explanation.Type != "mangle-pre" || explanation.Rule != "expanded_query" {
		t.Fatalf("unexpected explanation metadata: %+v", explanation)
	}
}

type mockRAGRetriever struct {
	results   []string
	lastQuery string
}

func (m *mockRAGRetriever) Retrieve(_ context.Context, query string) ([]string, error) {
	m.lastQuery = query
	return m.results, nil
}

type mockGateway struct {
	answer string
	prompt string
	chunks []*types.Chunk
}

func (m *mockGateway) Generate(_ context.Context, prompt string, chunks []*types.Chunk) (*types.Response, error) {
	m.prompt = prompt
	m.chunks = append([]*types.Chunk(nil), chunks...)
	return &types.Response{
		Answer:    m.answer,
		Citations: []string{"doc-1"},
		Metadata: map[string]any{
			"chunkCount": len(chunks),
		},
	}, nil
}
