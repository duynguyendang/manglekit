package dense

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

func (m *mockEmbedder) Name() string { return "mock-embedder" }

func (m *mockEmbedder) Register(r api.Registry) {}

// mockVectorStore is a mock implementation of core.VectorStore for testing.
type mockVectorStore struct {
	AddDocumentsFunc func(ctx context.Context, docs []core.Doc) error
	SearchFunc       func(ctx context.Context, queryText string, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error)
}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []core.Doc) error {
	if m.AddDocumentsFunc != nil {
		return m.AddDocumentsFunc(ctx, docs)
	}
	return errors.New("AddDocumentsFunc not implemented")
}

func (m *mockVectorStore) Search(ctx context.Context, queryText string, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, queryText, queryVector, topK, filter)
	}
	return nil, errors.New("SearchFunc not implemented")
}

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		deps := diapi.DenseRetrieverDeps{
			Embedder:    &mockEmbedder{},
			VectorStore: &mockVectorStore{},
		}
		retriever, err := New(deps)
		require.NoError(t, err)
		assert.NotNil(t, retriever)
	})

	t.Run("nil_embedder", func(t *testing.T) {
		deps := diapi.DenseRetrieverDeps{
			VectorStore: &mockVectorStore{},
		}
		_, err := New(deps)
		assert.Error(t, err)
	})

	t.Run("nil_vectorstore", func(t *testing.T) {
		deps := diapi.DenseRetrieverDeps{
			Embedder: &mockEmbedder{},
		}
		_, err := New(deps)
		assert.Error(t, err)
	})
}

func TestDense_Retrieve(t *testing.T) {
	ctx := context.Background()
	mockEmbedding := []float32{1.0, 2.0, 3.0}
	mockDocs := []core.Doc{{ID: "doc1", Text: "hello world"}}

	embedder := &mockEmbedder{
		EmbedFunc: func(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
			return &ai.EmbedResponse{
				Embeddings: []*ai.Embedding{{Embedding: mockEmbedding}},
			}, nil
		},
	}

	vectorStore := &mockVectorStore{
		SearchFunc: func(ctx context.Context, queryText string, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error) {
			assert.Equal(t, mockEmbedding, queryVector)
			return mockDocs, nil
		},
	}

	deps := diapi.DenseRetrieverDeps{
		Embedder:    embedder,
		VectorStore: vectorStore,
	}
	retriever, err := New(deps)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		res, err := retriever.Retrieve(ctx, core.RetrieveRequest{Query: "test"})
		require.NoError(t, err)
		assert.Equal(t, mockDocs, res.Docs)
	})

	t.Run("embedder_error", func(t *testing.T) {
		embedder.EmbedFunc = func(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
			return nil, errors.New("embedder failed")
		}
		_, err := retriever.Retrieve(ctx, core.RetrieveRequest{Query: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "embedder failed")
	})

	t.Run("vectorstore_error", func(t *testing.T) {
		// Restore embedder to success state
		embedder.EmbedFunc = func(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
			return &ai.EmbedResponse{
				Embeddings: []*ai.Embedding{{Embedding: mockEmbedding}},
			}, nil
		}
		vectorStore.SearchFunc = func(ctx context.Context, queryText string, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error) {
			return nil, errors.New("vectorstore failed")
		}
		_, err := retriever.Retrieve(ctx, core.RetrieveRequest{Query: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "vectorstore failed")
	})
}

func TestDense_Factory(t *testing.T) {
	r := manglekit.NewRegistry()
	Register(r)

	factory, err := r.Get(core.KindRetriever, "dense")
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		deps := diapi.DenseRetrieverDeps{
			Embedder:    &mockEmbedder{},
			VectorStore: &mockVectorStore{},
		}
		retriever, err := factory.Build(context.Background(), deps, &DenseOptions{})
		require.NoError(t, err)
		assert.NotNil(t, retriever)
	})

	t.Run("missing_embedder", func(t *testing.T) {
		deps := diapi.DenseRetrieverDeps{
			VectorStore: &mockVectorStore{},
		}
		_, err := factory.Build(context.Background(), deps, &DenseOptions{})
		assert.Error(t, err)
	})

	t.Run("missing_vectorstore", func(t *testing.T) {
		deps := diapi.DenseRetrieverDeps{
			Embedder: &mockEmbedder{},
		}
		_, err := factory.Build(context.Background(), deps, &DenseOptions{})
		assert.Error(t, err)
	})
}
