package cosine

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmbedder is a mock implementation of ai.Embedder for testing.
type mockEmbedder struct {
	EmbedFunc func(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	if m.EmbedFunc != nil {
		return m.EmbedFunc(ctx, req)
	}
	return nil, errors.New("EmbedFunc not implemented")
}
func (m *mockEmbedder) Name() string          { return "mock-embedder" }
func (m *mockEmbedder) Register(r api.Registry) {}

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := diapi.RerankerDeps{Embedder: &mockEmbedder{}}
		reranker, err := New(CosineOptions{}, deps)
		require.NoError(t, err)
		assert.NotNil(t, reranker)
	})

	t.Run("nil_embedder", func(t *testing.T) {
		_, err := New(CosineOptions{}, diapi.RerankerDeps{})
		assert.Error(t, err)
	})
}

func TestRerank(t *testing.T) {
	ctx := context.Background()
	embedder := &mockEmbedder{
		EmbedFunc: func(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
			text := req.Input[0].Content[0].Text
			var embedding []float32
			switch text {
			case "query":
				embedding = []float32{1, 0}
			case "doc1": // a perfect match
				embedding = []float32{1, 0}
			case "doc2": // orthogonal
				embedding = []float32{0, 1}
			case "doc3": // opposite
				embedding = []float32{-1, 0}
			}
			return &ai.EmbedResponse{
				Embeddings: []*ai.Embedding{{Embedding: embedding}},
			}, nil
		},
	}
	deps := diapi.RerankerDeps{Embedder: embedder}
	reranker, err := New(CosineOptions{TopK: 2}, deps)
	require.NoError(t, err)

	req := core.RerankRequest{
		Query: "query",
		Docs: []core.Doc{
			{ID: "doc2", Text: "doc2"}, // Should be ranked second
			{ID: "doc3", Text: "doc3"}, // Should be ranked third (and cut by TopK)
			{ID: "doc1", Text: "doc1"}, // Should be ranked first
		},
	}

	result, err := reranker.Rerank(ctx, req)
	require.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Equal(t, "doc1", result[0].Doc.ID)
	assert.InDelta(t, 1.0, result[0].Score, 0.001)
	assert.Equal(t, "doc2", result[1].Doc.ID)
	assert.InDelta(t, 0.0, result[1].Score, 0.001)
}

func TestCosineSimilarity(t *testing.T) {
	assert.InDelta(t, 1.0, cosineSimilarity([]float32{1, 0}, []float32{1, 0}), 0.001)
	assert.InDelta(t, 0.0, cosineSimilarity([]float32{1, 0}, []float32{0, 1}), 0.001)
	assert.InDelta(t, -1.0, cosineSimilarity([]float32{1, 0}, []float32{-1, 0}), 0.001)
}

func TestFactory(t *testing.T) {
	r := manglekit.NewRegistry()
	Register(r)

	factory, err := r.Get(core.KindReranker, "cosine")
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		deps := diapi.RerankerDeps{
			Embedder: &mockEmbedder{},
		}
		reranker, err := factory.Build(context.Background(), deps, &CosineOptions{})
		require.NoError(t, err)
		assert.NotNil(t, reranker)
	})

	t.Run("missing_embedder", func(t *testing.T) {
		_, err := factory.Build(context.Background(), diapi.RerankerDeps{}, &CosineOptions{})
		assert.Error(t, err)
	})
}
