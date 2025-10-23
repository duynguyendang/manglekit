package bm25

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBM25(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bm25_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = os.WriteFile(filepath.Join(tempDir, "doc1.md"), []byte("---\ntitle: doc1\n---\nHello world"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "doc2.md"), []byte("This is a test document."), 0644)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		opts := BM25Options{
			Path: tempDir,
			TopK: 5,
		}
		retriever, err := New(opts)
		require.NoError(t, err)
		assert.NotNil(t, retriever)
	})

	t.Run("error_no_path", func(t *testing.T) {
		opts := BM25Options{}
		_, err := New(opts)
		assert.Error(t, err)
	})

	t.Run("error_invalid_path", func(t *testing.T) {
		opts := BM25Options{
			Path: "/invalid/path",
		}
		_, err := New(opts)
		assert.Error(t, err)
	})
}

func TestBM25Retrieve(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bm25_retrieve_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = os.WriteFile(filepath.Join(tempDir, "doc1.md"), []byte("---\ntitle: doc1\n---\nHello world, this is a test."), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "doc2.md"), []byte("Another test document."), 0644)
	require.NoError(t, err)

	opts := BM25Options{
		Path: tempDir,
		TopK: 2,
	}
	retriever, err := New(opts)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		req := core.RetrieveRequest{
			Query: "hello",
			TopK:  1,
		}
		result, err := retriever.Retrieve(context.Background(), req)
		require.NoError(t, err)
		assert.Len(t, result.Docs, 1)
		assert.Equal(t, "doc1.md", result.Docs[0].Meta["doc_id"])
	})

	t.Run("no_results", func(t *testing.T) {
		req := core.RetrieveRequest{
			Query: "nonexistent",
			TopK:  1,
		}
		result, err := retriever.Retrieve(context.Background(), req)
		require.NoError(t, err)
		assert.Len(t, result.Docs, 0)
	})
}

func TestParseFrontMatter(t *testing.T) {
	t.Run("with_front_matter", func(t *testing.T) {
		content := []byte("---\ntitle: test\n---\nThis is the content.")
		metadata, body := parseFrontMatter(content, nil)
		assert.NotNil(t, metadata)
		assert.Equal(t, "test", metadata["title"])
		assert.Equal(t, "This is the content.", string(body))
	})

	t.Run("without_front_matter", func(t *testing.T) {
		content := []byte("This is the content.")
		metadata, body := parseFrontMatter(content, nil)
		assert.Nil(t, metadata)
		assert.Equal(t, "This is the content.", string(body))
	})

	t.Run("invalid_front_matter", func(t *testing.T) {
		content := []byte("---\ninvalid-yaml\n---\nContent")
		metadata, body := parseFrontMatter(content, nil)
		assert.Nil(t, metadata)
		assert.Equal(t, string(content), string(body))
	})
}

func TestBM25_Factory(t *testing.T) {
	r := manglekit.NewRegistry()
	Register(r)

	tempDir, err := os.MkdirTemp("", "bm25_factory_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = os.WriteFile(filepath.Join(tempDir, "doc1.md"), []byte("test content"), 0644)
	require.NoError(t, err)

	opts := BM25Options{
		Path: tempDir,
	}

	factory, err := r.Get(core.KindRetriever, "bm25")
	require.NoError(t, err)

	retriever, err := factory.Build(context.Background(), diapi.RetrieverDeps{}, opts)
	require.NoError(t, err)
	assert.NotNil(t, retriever)
}
