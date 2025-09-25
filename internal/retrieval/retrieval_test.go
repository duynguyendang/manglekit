package retrieval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"ndduy.dev/manglekit/internal/types"
)

type mockBM25Retriever struct {
	results []string
}

func (m *mockBM25Retriever) Retrieve(ctx context.Context, query string, cfg types.BM25Config) ([]string, error) {
	return m.results, nil
}

type mockDenseRetriever struct {
	results []string
}

func (m *mockDenseRetriever) Retrieve(ctx context.Context, query string, cfg types.DenseConfig) ([]string, error) {
	return m.results, nil
}

func TestHybridRetriever(t *testing.T) {
	bm25 := &mockBM25Retriever{results: []string{"docA", "docB"}}
	dense := &mockDenseRetriever{results: []string{"docB", "docC"}}

	hybrid, err := NewHybridRetriever(bm25, dense)
	if err != nil {
		t.Fatalf("failed to create hybrid retriever: %v", err)
	}

	results, err := hybrid.Retrieve(context.Background(), "query", types.BM25Config{}, types.DenseConfig{})
	if err != nil {
		t.Fatalf("hybrid retrieve failed: %v", err)
	}

	expected := []string{"docA", "docB", "docC"}
	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(results))
	}

	for _, doc := range expected {
		found := false
		for _, res := range results {
			if doc == res {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find doc %s, but it was missing", doc)
		}
	}
}

func TestBM25Retriever(t *testing.T) {
	dir, err := os.MkdirTemp("", "bm25-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	docs := map[string]string{
		"doc1.md": "the quick brown fox",
		"doc2.md": "jumped over the lazy dog",
		"doc3.md": "the lazy brown dog",
	}

	for name, content := range docs {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	bm25, err := NewBM25(context.Background(), dir)
	if err != nil {
		t.Fatalf("failed to create bm25 retriever: %v", err)
	}

	results, err := bm25.Retrieve(context.Background(), "lazy dog", types.BM25Config{TopK: 2})
	if err != nil {
		t.Fatalf("bm25 retrieve failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0] != docs["doc3.md"] {
		t.Errorf("expected first result to be doc3, got %s", results[0])
	}
	if results[1] != docs["doc2.md"] {
		t.Errorf("expected second result to be doc2, got %s", results[1])
	}
}

type mockEmbedder struct{}

func (m *mockEmbedder) Name() string {
	return "mockEmbedder"
}

func (m *mockEmbedder) Register(r api.Registry) {
	// No-op for mock
}

func (m *mockEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	var embeddings []*ai.Embedding
	for _, doc := range req.Input {
		var embedding *ai.Embedding
		switch doc.Content[0].Text {
		case "query":
			embedding = &ai.Embedding{Embedding: []float32{1.0, 0.0}}
		case "docA":
			embedding = &ai.Embedding{Embedding: []float32{0.9, 0.1}}
		case "docB":
			embedding = &ai.Embedding{Embedding: []float32{0.1, 0.9}}
		default:
			return nil, fmt.Errorf("unknown document for embedding: %s", doc.Content[0].Text)
		}
		embeddings = append(embeddings, embedding)
	}
	return &ai.EmbedResponse{Embeddings: embeddings}, nil
}

func TestDenseRetriever(t *testing.T) {
	ctx := context.Background()
	g := genkit.Init(ctx)

	docs := []*ai.Document{
		ai.DocumentFromText("docA", nil),
		ai.DocumentFromText("docB", nil),
	}

	dense, err := NewDense(ctx, g, &mockEmbedder{}, docs)
	if err != nil {
		t.Fatalf("failed to create dense retriever: %v", err)
	}

	results, err := dense.Retrieve(ctx, "query", types.DenseConfig{TopK: 1})
	if err != nil {
		t.Fatalf("dense retrieve failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0] != "docA" {
		t.Errorf("expected result to be docA, got %s", results[0])
	}
}