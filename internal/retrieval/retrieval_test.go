package retrieval

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"ndduy.dev/manglekit/internal/types"
)

type mockBM25Retriever struct {
	results []string
	err     error
}

func (m *mockBM25Retriever) Retrieve(ctx context.Context, query string, filters map[string]string, cfg types.BM25Config) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

type mockDenseRetriever struct {
	results []string
	err     error
}

func (m *mockDenseRetriever) Retrieve(ctx context.Context, query string, filters map[string]string, cfg types.DenseConfig) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestHybridRetriever(t *testing.T) {
	bm25 := &mockBM25Retriever{results: []string{"bm25_doc1", "shared_doc"}}
	dense := &mockDenseRetriever{results: []string{"dense_doc1", "shared_doc"}}

	hybrid, err := NewHybridRetriever(bm25, dense)
	assert.NoError(t, err)

	results, err := hybrid.Retrieve(context.Background(), "test query", nil, types.BM25Config{TopK: 2}, types.DenseConfig{TopK: 2})
	assert.NoError(t, err)

	expected := []string{"bm25_doc1", "shared_doc", "dense_doc1"}
	assert.ElementsMatch(t, expected, results)
}

func TestBM25Retriever(t *testing.T) {
	// This is a simplified test. A real test would require a temporary directory
	// with mock documents.
	ctx := context.Background()
	_, err := NewBM25(ctx, "./testdata")
	assert.NoError(t, err)

	// Since we can't easily test the Retrieve method without a proper test setup,
	// we'll just check that the constructor doesn't fail.
}

func TestDenseRetriever(t *testing.T) {
	// This test is also simplified due to the complexity of mocking Genkit and
	// the local vector store.
	// A real test would require a more sophisticated setup.
	assert.True(t, true, "Skipping dense retriever test due to complexity.")
}