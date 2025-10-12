package hybrid_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/hybrid"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRetriever is a simple mock for testing retriever components.
type mockRetriever struct {
	docs []core.Doc
	err  error
}

func (m *mockRetriever) Retrieve(ctx context.Context, req retrieve.Request) (retrieve.Result, error) {
	if m.err != nil {
		return retrieve.Result{}, m.err
	}
	return retrieve.Result{Docs: m.docs}, nil
}

func TestHybrid_Retrieve(t *testing.T) {
	// 1. Setup mock dependencies.
	mockBM25 := &mockRetriever{
		docs: []core.Doc{
			{ID: "doc1", Text: "shared document"},
			{ID: "doc2", Text: "bm25 only"},
		},
	}
	mockDense := &mockRetriever{
		docs: []core.Doc{
			{ID: "doc1", Text: "shared document"},
			{ID: "doc3", Text: "dense only"},
		},
	}

	// 2. Create the Hybrid Retriever by passing the mocks directly.
	retriever, err := hybrid.New(retrieve.HybridOptions{
		BM25Retriever:  mockBM25,
		DenseRetriever: mockDense,
	})
	require.NoError(t, err)

	// 3. Perform the retrieval.
	req := retrieve.Request{Query: "test", TopK: 5}
	result, err := retriever.Retrieve(context.Background(), req)
	require.NoError(t, err)

	// 4. Assert the results.
	// Check the length.
	assert.Len(t, result.Docs, 3, "Should have combined and deduplicated results")

	// Check the RRF order. doc1 should be first as it's in both lists.
	assert.Equal(t, "doc1", result.Docs[0].ID, "doc1 should be ranked first by RRF")

	// The order of doc2 and doc3 is not guaranteed as they have the same RRF score.
	// So, we just check for their presence in the remaining slice.
	remainingIDs := make(map[string]bool)
	for _, doc := range result.Docs[1:] {
		remainingIDs[doc.ID] = true
	}
	assert.True(t, remainingIDs["doc2"], "doc2 should be in the results")
	assert.True(t, remainingIDs["doc3"], "doc3 should be in the results")
}
