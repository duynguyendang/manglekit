package bm25_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit/internal/providers/bm25"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBM25_Retrieve(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "doc1.md"), []byte("the quick brown fox"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "doc2.md"), []byte("jumps over the lazy dog"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "doc3.md"), []byte("the lazy brown dog"), 0o600))

	retriever, err := bm25.New(retrieve.BM25Options{
		Path: dir,
	})
	require.NoError(t, err)

	req := retrieve.Request{
		Query: "lazy dog",
		TopK:  2,
	}
	result, err := retriever.Retrieve(context.Background(), req)
	require.NoError(t, err)

	assert.Len(t, result.Docs, 2)
	// doc3 should be more relevant because it contains both "lazy" and "dog".
	assert.Equal(t, "doc3.md", result.Docs[0].ID)
	assert.Equal(t, "doc2.md", result.Docs[1].ID)
}
