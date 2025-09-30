package orchestrator

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"ndduy.dev/manglekit/internal/mangle"
	"ndduy.dev/manglekit/internal/types"
)

func TestRunFlowWithMockLLMAndFacts(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	rulesPath := filepath.Join("..", "..", "config", "mangle", "main.dlog")
	factsPath := filepath.Join("..", "..", "config", "mangle", "*", "*.dlog")

	processor, err := mangle.New(ctx, mangle.Config{RulesFile: rulesPath, FactsFile: factsPath})
	if err != nil {
		t.Fatalf("create mangle processor: %v", err)
	}

	mockLLM := &mockGateway{answer: "Mock answer with citations."}

	mockRetriever := &mockRetriever{chunks: []*types.Chunk{
		{
			ID:    "chunk-1",
			DocID: "doc-1",
			Title: "Hybrid retrieval",
			Text:  "parse_intent_ner: Genkit normalizes the query and extracts entities.",
			Score: 0.72,
			Metadata: map[string]any{
				"visibility": "public",
				"tenant":     "*",
			},
		},
		{
			ID:    "chunk-2",
			DocID: "doc-1",
			Title: "Hybrid retrieval",
			Text:  "hybrid_retrieval: BM25 must/should filters combine with MRL ANN top-k.",
			Score: 0.65,
			Metadata: map[string]any{
				"visibility": "public",
				"tenant":     "*",
			},
		},
	}}

	mockReranker := &mockReranker{
		results: []*types.Chunk{
			{
				ID:    "chunk-1",
				DocID: "doc-1",
				Title: "Hybrid retrieval",
				Text:  "parse_intent_ner: Genkit normalizes the query and extracts entities.",
				Score: 0.91,
				Metadata: map[string]any{
					"visibility": "public",
					"tenant":     "*",
				},
			},
		},
		explanations: []types.Explanation{{
			Type:   "rerank",
			Rule:   "mrl",
			Action: "retained",
			Reason: "chunk-1 reranked",
		}},
	}

	mockIntent := &types.IntentResult{
		Intent:     "question",
		Confidence: 0.9,
		Entities: map[string][]string{
			"topic":    {"bm25"},
			"artifact": {"pdf"},
		},
		Explanation: "detected informational question",
	}

	orch := &orchestrator{
		processor:  processor,
		llmGateway: mockLLM,
		retriever:  mockRetriever,
		reranker:   mockReranker,
		intent:     &mockIntentParser{result: mockIntent},
		config: Config{
			MaxContextTokens:  4000,
			FallbackThreshold: 0.1,
		},
	}

	input := &types.QueryInput{Query: "Explain bm25 ann pipeline"}

	expanded, err := processor.PreProcess(&types.QueryInput{Query: input.Query, Intent: mockIntent})
	if err != nil {
		t.Fatalf("preprocess input: %v", err)
	}

	wantTerms := []string{"ann", "approximate nearest neighbor", "best matching 25", "bm25"}
	for _, term := range wantTerms {
		if !slices.Contains(expanded.ExpansionTerms, term) {
			t.Fatalf("expected expansion term %q to be present, got %v", term, expanded.ExpansionTerms)
		}
	}

	if expanded.Constraints.Visibility != "public" {
		t.Fatalf("expected default visibility filter 'public', got %q", expanded.Constraints.Visibility)
	}

	response, err := orch.RunFlow(ctx, input)
	if err != nil {
		t.Fatalf("RunFlow error: %v", err)
	}

	if mockRetriever.lastQuery == nil {
		t.Fatalf("retriever did not receive expanded query")
	}

	if mockRetriever.lastQuery.NormalizedQuery != expanded.NormalizedQuery {
		t.Fatalf("retriever query mismatch: got %q want %q", mockRetriever.lastQuery.NormalizedQuery, expanded.NormalizedQuery)
	}

	if mockRetriever.lastFilters == nil || mockRetriever.lastFilters["visibility"] != "public" {
		t.Fatalf("expected retriever visibility filter to be public, got %q", mockRetriever.lastFilters["visibility"])
	}

	if len(mockReranker.lastCandidates) == 0 {
		t.Fatalf("reranker did not receive candidates")
	}

	if len(mockLLM.chunks) == 0 || mockLLM.chunks[0].ID != mockReranker.results[0].ID {
		t.Fatalf("expected reranked chunk to be passed to LLM, got %+v", mockLLM.chunks)
	}

	if response.Answer != mockLLM.answer {
		t.Fatalf("unexpected answer: got %q want %q", response.Answer, mockLLM.answer)
	}

	if len(response.Explanations) < 3 {
		t.Fatalf("expected rerank, intent, and mangle explanations to be included")
	}

	if response.Explanations[0].Type != "rerank" {
		t.Fatalf("expected rerank explanation first, got %+v", response.Explanations[0])
	}

	explanation := response.Explanations[len(response.Explanations)-1]
	if explanation.Type != "mangle-pre" || explanation.Rule != "expanded_query" {
		t.Fatalf("unexpected explanation metadata: %+v", explanation)
	}

	meta, ok := response.Metadata.(map[string]any)
	if !ok {
		t.Fatalf("expected metadata map, got %T", response.Metadata)
	}
	if _, ok := meta["intent"]; !ok {
		t.Fatalf("intent metadata missing: %+v", meta)
	}
	if _, ok := meta["expandedQuery"]; !ok {
		t.Fatalf("expanded query metadata missing: %+v", meta)
	}
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

type mockRetriever struct {
	chunks      []*types.Chunk
	lastQuery   *types.ExpandedQuery
	lastFilters map[string]string
}

func (m *mockRetriever) Search(_ context.Context, query *types.ExpandedQuery, filters map[string]string) ([]*types.Chunk, error) {
	m.lastQuery = query
	if filters != nil {
		m.lastFilters = make(map[string]string, len(filters))
		for k, v := range filters {
			m.lastFilters[k] = v
		}
	}
	out := make([]*types.Chunk, 0, len(m.chunks))
	for _, chunk := range m.chunks {
		clone := *chunk
		if chunk.Metadata != nil {
			clone.Metadata = make(map[string]any, len(chunk.Metadata))
			for k, v := range chunk.Metadata {
				clone.Metadata[k] = v
			}
		}
		out = append(out, &clone)
	}
	return out, nil
}

type mockReranker struct {
	results        []*types.Chunk
	explanations   []types.Explanation
	lastQuery      *types.ExpandedQuery
	lastCandidates []*types.Chunk
}

func (m *mockReranker) Rerank(_ context.Context, query *types.ExpandedQuery, candidates []*types.Chunk) ([]*types.Chunk, []types.Explanation, error) {
	m.lastQuery = query
	m.lastCandidates = append([]*types.Chunk(nil), candidates...)
	out := make([]*types.Chunk, 0, len(m.results))
	for _, chunk := range m.results {
		clone := *chunk
		if chunk.Metadata != nil {
			clone.Metadata = make(map[string]any, len(chunk.Metadata))
			for k, v := range chunk.Metadata {
				clone.Metadata[k] = v
			}
		}
		out = append(out, &clone)
	}
	return out, append([]types.Explanation(nil), m.explanations...), nil
}

type mockIntentParser struct {
	result *types.IntentResult
	err    error
}

func (m *mockIntentParser) Parse(_ context.Context, _ *types.QueryInput) (*types.IntentResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}
