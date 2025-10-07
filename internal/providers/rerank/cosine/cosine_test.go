package cosine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEmbedder struct{}

func (m *mockEmbedder) Name() string          { return "mockEmbedder" }
func (m *mockEmbedder) Register(r api.Registry) {}
func (m *mockEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	var embeddings []*ai.Embedding
	for _, doc := range req.Input {
		var embedding *ai.Embedding
		switch doc.Content[0].Text {
		case "query":
			embedding = &ai.Embedding{Embedding: []float32{1.0, 0.0, 0.0, 1.0, 0.0, 0.0}}
		case "doc1": // Most similar
			embedding = &ai.Embedding{Embedding: []float32{0.9, 0.1, 0.0, 0.8, 0.2, 0.0}}
		case "doc2": // Least similar
			embedding = &ai.Embedding{Embedding: []float32{0.1, 0.9, 0.0, 0.2, 0.8, 0.0}}
		case "doc3": // Middle similarity
			embedding = &ai.Embedding{Embedding: []float32{0.9, 0.1, 0.0, 0.1, 0.9, 0.0}}
		default:
			return nil, fmt.Errorf("unknown document for embedding: %s", doc.Content[0].Text)
		}
		embeddings = append(embeddings, embedding)
	}
	return &ai.EmbedResponse{Embeddings: embeddings}, nil
}

func TestRerank(t *testing.T) {
	opts := rerank.CosineOptions{
		TopK:      2,
		VectorDim: 3,
	}
	mockEmb := &mockEmbedder{}
	reranker, err := cosine.New(opts, mockEmb)
	require.NoError(t, err)

	req := rerank.Request{
		Query: "query",
		Docs: []core.Doc{
			{ID: "doc2", Text: "doc2"},
			{ID: "doc3", Text: "doc3"},
			{ID: "doc1", Text: "doc1"},
		},
		TopK: 2,
	}

	rerankedDocs, err := reranker.Rerank(req)
	require.NoError(t, err)
	assert.Len(t, rerankedDocs, 2)

	// Expected order based on average multi-dimensional similarity:
	// doc1: high score in both dimensions -> highest avg score
	// doc3: high score in one, low in other -> middle avg score
	// doc2: low score in both dimensions -> lowest avg score
	expectedOrder := []string{"doc1", "doc3"}
	for i, doc := range rerankedDocs {
		assert.Equal(t, expectedOrder[i], doc.Doc.ID)
	}
}

// The multiDimCosineSimilarity function is not exported, so we cannot test it directly here.
// Its behavior is tested indirectly via the TestRerank function.