package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"go.uber.org/zap"
	"ndduy.dev/manglekit/internal/mangle"
	"ndduy.dev/manglekit/internal/types"
)

func TestRunFlowWithMockLLMAndFacts(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	// Create temporary rules and facts for a self-contained test.
	rulesDir := t.TempDir()
	rulesContent := `
		Decl query_token(Token).
		Decl expanded_query(Token, Expansion).
		Decl alias(Source, Target).
		Decl query_filter(Key, Value).
		Decl default_filter(Key, Value).
		expanded_query(Token, Token) :- query_token(Token).
		expanded_query(Token, Expansion) :- query_token(Token), alias(Token, Expansion).
		query_filter(Key, Value) :- default_filter(Key, Value).
	`
	factsContent := `
		alias("ann", "approximate nearest neighbor").
		alias("bm25", "best matching 25").
		default_filter("visibility", "public").
	`
	if err := os.WriteFile(filepath.Join(rulesDir, "rules.dlog"), []byte(rulesContent), 0o600); err != nil {
		t.Fatalf("write rules.dlog: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "facts.dlog"), []byte(factsContent), 0o600); err != nil {
		t.Fatalf("write facts.dlog: %v", err)
	}

	processor, err := mangle.New(ctx, mangle.Config{RulesFile: rulesDir})
	if err != nil {
		t.Fatalf("create mangle processor: %v", err)
	}

	mockRetriever := &mockHybridRetriever{results: []string{
		"parse_intent_ner: Genkit normalizes the query and extracts entities.",
		"hybrid_retrieval: BM25 must/should filters combine with MRL ANN top-k.",
	}}

	mockLLM := &mockGateway{answer: "Mock answer with citations."}

	mockIntent := &types.IntentResult{
		Intent:     "question",
		Confidence: 0.9,
		Entities: map[string][]string{
			"topic":    {"bm25"},
			"artifact": {"pdf"},
		},
		Explanation: "detected informational question",
	}

	logger, _ := zap.NewDevelopment()
	orch := &orchestrator{
		processor:  processor,
		llmGateway: mockLLM,
		retriever:  mockRetriever,
		reranker:   &mockReranker{},
		intent:     &mockIntentParser{result: mockIntent},
		cfg:        Config{}, // Use default config for test
		log:        logger,
	}

	input := &types.QueryInput{Query: "Explain bm25 ann pipeline"}

	expanded, err := processor.PreProcess(input)
	if err != nil {
		t.Fatalf("preprocess input: %v", err)
	}

	wantTerms := []string{"ann", "approximate nearest neighbor", "best matching 25", "bm25", "explain", "pipeline"}
	sort.Strings(expanded.ExpansionTerms)
	if !reflect.DeepEqual(expanded.ExpansionTerms, wantTerms) {
		t.Fatalf("unexpected expansion terms. got %v, want %v", expanded.ExpansionTerms, wantTerms)
	}

	if got := expanded.Filters["visibility"]; got != "public" {
		t.Fatalf("expected default visibility filter 'public', got %q", got)
	}

	response, err := orch.RunFlow(ctx, input)
	if err != nil {
		t.Fatalf("RunFlow error: %v", err)
	}

	expectedParts := []string{expanded.NormalizedQuery}
	if len(expanded.ExpansionTerms) > 0 {
		expectedParts = append(expectedParts, expanded.ExpansionTerms...)
	}
	keys := make([]string, 0, len(mockIntent.Entities))
	for key := range mockIntent.Entities {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		expectedParts = append(expectedParts, mockIntent.Entities[key]...)
	}
	expectedQuery := strings.Join(expectedParts, " ")

	if mockRetriever.lastQuery != expectedQuery {
		t.Fatalf("unexpected rag query. got %q want %q", mockRetriever.lastQuery, expectedQuery)
	}

	if mockRetriever.lastFilter["visibility"] != "public" {
		t.Fatalf("expected visibility filter to be 'public', got %q", mockRetriever.lastFilter["visibility"])
	}

	if len(mockLLM.chunks) == 0 || mockLLM.chunks[0].Text != mockRetriever.results[0] {
		t.Fatalf("expected first chunk to match RAG context %q, got %+v", mockRetriever.results[0], mockLLM.chunks)
	}

	if response.Answer != mockLLM.answer {
		t.Fatalf("unexpected answer: got %q want %q", response.Answer, mockLLM.answer)
	}

	if len(response.Explanations) < 2 {
		t.Fatalf("expected intent and mangle explanations to be included")
	}

	if response.Explanations[0].Type != "intent" {
		t.Fatalf("expected intent explanation first, got %+v", response.Explanations[0])
	}

	explanation := response.Explanations[1]
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
}

type mockHybridRetriever struct {
	results    []string
	lastQuery  string
	lastFilter map[string]string
}

func (m *mockHybridRetriever) Retrieve(_ context.Context, query string, filters map[string]string, _ types.BM25Config, _ types.DenseConfig) ([]string, error) {
	m.lastQuery = query
	m.lastFilter = filters
	return m.results, nil
}

type mockReranker struct{}

func (m *mockReranker) Rerank(_ context.Context, _ string, docs []string, _ types.RerankConfig) ([]string, error) {
	// Passthrough for testing
	return docs, nil
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