package cosine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSmokeEmbedder is a deterministic embedder for smoke tests.
type mockSmokeEmbedder struct {
	vectors map[string][]float32
}

func (m *mockSmokeEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	if len(req.Input) == 0 || len(req.Input[0].Content) == 0 {
		return nil, errors.New("empty input")
	}
	text := req.Input[0].Content[0].Text
	vector, ok := m.vectors[text]
	if !ok {
		// Return a zero vector for unknown text to avoid panics and allow tests to assert on ranking.
		zeroVector := make([]float32, 3) // Assuming vector dimension is 3 for this test suite
		return &ai.EmbedResponse{
			Embeddings: []*ai.Embedding{{Embedding: zeroVector}},
		}, nil
	}
	return &ai.EmbedResponse{
		Embeddings: []*ai.Embedding{{Embedding: vector}},
	}, nil
}

func (m *mockSmokeEmbedder) Name() string          { return "mock-smoke-embedder" }
func (m *mockSmokeEmbedder) Register(r api.Registry) {}

func TestCosineReranker_BasicRanking(t *testing.T) {
	t.Parallel()

	embedder := &mockSmokeEmbedder{
		vectors: map[string][]float32{
			"keyword search":   {1, 1, 0},
			"typed dependency": {0, 1, 0},
			"keyword only":     {1, 0, 0},
		},
	}
	reranker, err := cosine.New(cosine.CosineOptions{}, diapi.RerankerDeps{Embedder: embedder})
	require.NoError(t, err)

	docs := []core.Doc{
		{ID: "D1", Text: "typed dependency"},
		{ID: "D2", Text: "keyword search"},
		{ID: "D3", Text: "keyword only"},
	}

	req := core.RerankRequest{
		Query: "keyword search",
		Docs:  docs,
		TopK:  2,
	}

	result, err := reranker.Rerank(context.Background(), req)
	require.NoError(t, err)

	// D2 is a perfect match (score 1.0)
	// D1 and D3 have the same partial match score (~0.707)
	// With stable sort, D1 should come before D3 due to its ID.
	assert.Len(t, result, 2)
	assert.Equal(t, "D2", result[0].Doc.ID)
	assert.Equal(t, "D1", result[1].Doc.ID)
	assert.GreaterOrEqual(t, result[0].Score, result[1].Score)
	assert.InDelta(t, 1.0, result[0].Score, 0.001)
}

func TestCosineReranker_EmptyAndTopK(t *testing.T) {
	t.Parallel()

	embedder := &mockSmokeEmbedder{
		vectors: map[string][]float32{
			"keyword search": {1, 1, 0},
			"doc1":           {0, 1, 0},
			"doc2":           {1, 1, 0},
			"doc3":           {1, 0, 0},
		},
	}
	reranker, err := cosine.New(cosine.CosineOptions{}, diapi.RerankerDeps{Embedder: embedder})
	require.NoError(t, err)
	ctx := context.Background()

	// Test empty candidate list
	emptyReq := core.RerankRequest{
		Query: "keyword search",
		Docs:  []core.Doc{},
	}
	emptyResult, err := reranker.Rerank(ctx, emptyReq)
	require.NoError(t, err)
	assert.Empty(t, emptyResult)

	// Test TopK=1
	docs := []core.Doc{
		{ID: "D1", Text: "doc1"},
		{ID: "D2", Text: "doc2"},
		{ID: "D3", Text: "doc3"},
	}
	topKReq := core.RerankRequest{
		Query: "keyword search",
		Docs:  docs,
		TopK:  1,
	}
	topKResult, err := reranker.Rerank(ctx, topKReq)
	require.NoError(t, err)
	assert.Len(t, topKResult, 1)
	assert.Equal(t, "D2", topKResult[0].Doc.ID)
}

func TestCosineReranker_TieBreak_StableOrder(t *testing.T) {
	t.Parallel()

	embedder := &mockSmokeEmbedder{
		vectors: map[string][]float32{
			"query": {1, 1, 0},
			"docA":  {1, 0, 0}, // cos = ~0.707
			"docB":  {0, 1, 0}, // cos = ~0.707
			"docC":  {1, 1, 0}, // perfect match
		},
	}
	reranker, err := cosine.New(cosine.CosineOptions{}, diapi.RerankerDeps{Embedder: embedder})
	require.NoError(t, err)

	// docA and docB have the same score, so they should be ordered by ID.
	docs := []core.Doc{
		{ID: "docB", Text: "docB"}, // This will be second
		{ID: "docA", Text: "docA"}, // This will be first
		{ID: "docC", Text: "docC"}, // This will be third
	}

	req := core.RerankRequest{
		Query: "query",
		Docs:  docs,
	}

	result, err := reranker.Rerank(context.Background(), req)
	require.NoError(t, err)

	assert.Len(t, result, 3)
	assert.Equal(t, "docC", result[0].Doc.ID)
	assert.Equal(t, "docA", result[1].Doc.ID)
	assert.Equal(t, "docB", result[2].Doc.ID)
}
