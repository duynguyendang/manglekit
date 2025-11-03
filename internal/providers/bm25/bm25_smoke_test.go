package bm25_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/bm25"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRetriever is a test helper to instantiate the BM25 retriever.
func newTestRetriever(t *testing.T, docs map[string]string) core.Retriever {
	t.Helper()
	tempDir := t.TempDir()
	for name, content := range docs {
		err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644)
		require.NoError(t, err)
	}
	opts := bm25.BM25Options{Path: tempDir}
	r, err := bm25.New(opts, diapi.NoopDeps{})
	require.NoError(t, err)
	return r
}

func TestRetriever_BasicRanking(t *testing.T) {
	// Fixture updated to test ranking between a full and partial match.
	docs := map[string]string{
		// D1 now contains one of the query terms ("keyword").
		"D1.md": "manglekit provides a useful keyword abstraction",
		// D2 contains both query terms ("keyword" and "search").
		"D2.md": "bm25 keyword search over local files",
		// D3 is a distractor document.
		"D3.md": "stateful chat uses a state provider",
	}
	r := newTestRetriever(t, docs)

	req := core.RetrieveRequest{Query: "keyword search", TopK: 2}
	res, err := r.Retrieve(context.Background(), req)
	require.NoError(t, err)

	// We expect exactly 2 results, with D2 ranked higher because it matches both terms.
	require.Len(t, res.Docs, 2)
	assert.Equal(t, "D2.md", res.Docs[0].ID, "D2 should be ranked first as it contains both query terms")
	assert.Equal(t, "D1.md", res.Docs[1].ID, "D1 should be ranked second as it contains one query term")
}

func TestRetriever_TopK_And_EmptyResult(t *testing.T) {
	docs := map[string]string{
		"D1.md": "manglekit provides typed dependency injection",
		"D2.md": "bm25 keyword search over local files",
		"D3.md": "stateful chat uses a state provider",
	}
	r := newTestRetriever(t, docs)

	t.Run("EmptyResult", func(t *testing.T) {
		req := core.RetrieveRequest{Query: "nonexistent token"}
		res, err := r.Retrieve(context.Background(), req)
		require.NoError(t, err)
		assert.Empty(t, res.Docs)
	})

	t.Run("TopK", func(t *testing.T) {
		req := core.RetrieveRequest{Query: "state provider", TopK: 1}
		res, err := r.Retrieve(context.Background(), req)
		require.NoError(t, err)
		require.Len(t, res.Docs, 1)
		assert.Equal(t, "D3.md", res.Docs[0].ID)
	})
}

func TestRetriever_Tiebreak_StableOrder(t *testing.T) {
	// Two docs with the same query term, plus a third distractor doc.
	// The third doc is necessary to ensure the IDF for "retrieval" is non-zero.
	docs := map[string]string{
		"doc_a.md": "retrieval with manglekit",
		"doc_b.md": "retrieval for search",
		"doc_c.md": "an unrelated document",
	}
	r := newTestRetriever(t, docs)

	// Run the query multiple times to check for stable order.
	var firstRunIDs []string
	for i := 0; i < 10; i++ {
		req := core.RetrieveRequest{Query: "retrieval"}
		res, err := r.Retrieve(context.Background(), req)
		require.NoError(t, err)
		// We expect the two documents containing "retrieval" to be returned.
		require.Len(t, res.Docs, 2, "Expected to find 'retrieval' in two documents")

		currentRunIDs := []string{res.Docs[0].ID, res.Docs[1].ID}
		if i == 0 {
			firstRunIDs = currentRunIDs
		} else {
			assert.Equal(t, firstRunIDs, currentRunIDs, fmt.Sprintf("run %d: order is not stable", i+1))
		}
	}
}
