package dense_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/dense"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmbedder simulates the behavior of an ai.Embedder.
type mockEmbedder struct{}

func (m *mockEmbedder) Name() string            { return "mockEmbedder" }
func (m *mockEmbedder) Register(r api.Registry) {}
func (m *mockEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	var embeddings []*ai.Embedding
	for _, doc := range req.Input {
		text := doc.Content[0].Text
		var embedding *ai.Embedding
		switch text {
		case "query":
			embedding = &ai.Embedding{Embedding: []float32{1.0, 0.1, 0.1}}
		default:
			embedding = &ai.Embedding{Embedding: []float32{0.5, 0.5, 0.5}}
		}
		embeddings = append(embeddings, embedding)
	}
	return &ai.EmbedResponse{Embeddings: embeddings}, nil
}

// mockVectorStore simulates the behavior of a core.VectorStore.
type mockVectorStore struct {
	SearchFunc func(ctx context.Context, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error)
}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []core.Doc) error {
	return nil // Not needed for this test.
}

func (m *mockVectorStore) Search(ctx context.Context, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, queryVector, topK, filter)
	}
	return []core.Doc{{ID: "default-doc", Text: "default text"}}, nil
}

func TestDense_Retrieve(t *testing.T) {
	// 1. Setup mocks.
	mockEmb := &mockEmbedder{}
	mockVS := &mockVectorStore{}
	mockVS.SearchFunc = func(ctx context.Context, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error) {
		// Assert that the vector passed to the search is the one from the embedder.
		assert.Equal(t, []float32{1.0, 0.1, 0.1}, queryVector)
		// Return a sample document.
		return []core.Doc{
			{ID: "doc1", Text: "This is document one."},
		}, nil
	}

	// 2. Instantiate the retriever with direct, typed dependencies.
	retriever, err := dense.New(mockEmb, mockVS)
	require.NoError(t, err)

	// 3. Perform the retrieval.
	req := retrieve.Request{
		Query: "query",
		TopK:  1,
	}
	result, err := retriever.Retrieve(context.Background(), req)
	require.NoError(t, err)

	// 4. Assert the results.
	assert.Len(t, result.Docs, 1)
	assert.Equal(t, "doc1", result.Docs[0].ID)
	assert.Equal(t, "This is document one.", result.Docs[0].Text)
}
