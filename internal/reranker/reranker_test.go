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
	// We are mocking a 2-dimensional embedding where each dimension has a vector of size 3.
	// The vectors are concatenated into a single flat slice.
	for _, doc := range req.Input {
		var embedding *ai.Embedding
		switch doc.Content[0].Text {
		case "query":
			// Represents two vectors: [1.0, 0.0, 0.0] and [1.0, 0.0, 0.0]
			embedding = &ai.Embedding{Embedding: []float32{1.0, 0.0, 0.0, 1.0, 0.0, 0.0}}
		case "doc1": // Most similar: High similarity in both dimensions.
			// Represents: [0.9, 0.1, 0.0] and [0.8, 0.2, 0.0]
			embedding = &ai.Embedding{Embedding: []float32{0.9, 0.1, 0.0, 0.8, 0.2, 0.0}}
		case "doc2": // Least similar: Low similarity in both dimensions.
			// Represents: [0.1, 0.9, 0.0] and [0.2, 0.8, 0.0]
			embedding = &ai.Embedding{Embedding: []float32{0.1, 0.9, 0.0, 0.2, 0.8, 0.0}}
		case "doc3": // Middle similarity: High similarity in one, low in the other.
			// Represents: [0.9, 0.1, 0.0] and [0.1, 0.9, 0.0]
			embedding = &ai.Embedding{Embedding: []float32{0.9, 0.1, 0.0, 0.1, 0.9, 0.0}}
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

	// Expected order based on average multi-dimensional similarity:
	// doc1: high score in both dimensions -> highest avg score
	// doc3: high score in one, low in other -> middle avg score
	// doc2: low score in both dimensions -> lowest avg score
	expectedOrder := []string{"doc1", "doc3"}
	for i, doc := range rerankedDocs {
		if doc != expectedOrder[i] {
			t.Errorf("expected doc at index %d to be %s, but got %s", i, expectedOrder[i], doc)
		}
	}
}

func TestMultiDimCosineSimilarity(t *testing.T) {
	// Two dimensions, each of size 3.
	// v1: [1,0,0], [0,1,0]
	// v2: [1,0,0], [1,0,0]
	v1 := []float32{1, 0, 0, 0, 1, 0}
	v2 := []float32{1, 0, 0, 1, 0, 0}
	vectorDim := 3
	// Expected similarity for dim 1: 1.0
	// Expected similarity for dim 2: 0.0
	// Average: (1.0 + 0.0) / 2 = 0.5
	similarity, err := multiDimCosineSimilarity(v1, v2, vectorDim)
	if err != nil {
		t.Fatalf("multiDimCosineSimilarity failed: %v", err)
	}
	if similarity < 0.49 || similarity > 0.51 {
		t.Errorf("expected similarity around 0.5, but got %f", similarity)
	}
}