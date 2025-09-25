package reranker

import (
	"context"
	"fmt"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"ndduy.dev/manglekit/internal/types"
)

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
			embedding = &ai.Embedding{Embedding: []float32{1.0, 0.0, 0.0}}
		case "doc1": // Most similar
			embedding = &ai.Embedding{Embedding: []float32{0.9, 0.1, 0.0}}
		case "doc2": // Least similar
			embedding = &ai.Embedding{Embedding: []float32{0.1, 0.9, 0.0}}
		case "doc3": // Middle similarity
			embedding = &ai.Embedding{Embedding: []float32{0.5, 0.5, 0.0}}
		default:
			return nil, fmt.Errorf("unknown document for embedding: %s", doc.Content[0].Text)
		}
		embeddings = append(embeddings, embedding)
	}
	return &ai.EmbedResponse{Embeddings: embeddings}, nil
}

func TestRerank(t *testing.T) {
	ctx := context.Background()
	reranker, err := New(&mockEmbedder{})
	if err != nil {
		t.Fatalf("failed to create reranker: %v", err)
	}

	docs := []string{"doc2", "doc3", "doc1"}
	cfg := types.RerankConfig{TopK: 2}

	rerankedDocs, err := reranker.Rerank(ctx, "query", docs, cfg)
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}

	if len(rerankedDocs) != 2 {
		t.Fatalf("expected %d documents, but got %d", cfg.TopK, len(rerankedDocs))
	}

	expectedOrder := []string{"doc1", "doc3"}
	for i, doc := range rerankedDocs {
		if doc != expectedOrder[i] {
			t.Errorf("expected doc at index %d to be %s, but got %s", i, expectedOrder[i], doc)
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	v1 := []float32{1, 2, 3}
	v2 := []float32{4, 5, 6}
	// Expected: (1*4 + 2*5 + 3*6) / (sqrt(1*1+2*2+3*3) * sqrt(4*4+5*5+6*6))
	// = 32 / (sqrt(14) * sqrt(77)) = 32 / (3.74 * 8.77) = 32 / 32.8
	// approx 0.975
	similarity, err := cosineSimilarity(v1, v2)
	if err != nil {
		t.Fatalf("cosineSimilarity failed: %v", err)
	}
	if similarity < 0.97 || similarity > 0.98 {
		t.Errorf("expected similarity around 0.975, but got %f", similarity)
	}
}