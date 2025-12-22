package core

import "context"

// Embedder defines the contract for converting text to vector representations.
type Embedder interface {
	// Embed generates a vector for a single text string.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates vectors for multiple strings.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension returns the vector size (e.g., 768 for Google GenAI).
	Dimension() int
}

// VectorStore defines the storage and retrieval contract.
type VectorStore interface {
	Upsert(ctx context.Context, id string, content string) error
	Search(ctx context.Context, query string, topK int) ([]string, error)
	Get(ctx context.Context, id string) (string, error)
}

// NopEmbedder is a no-op implementation of Embedder.
type NopEmbedder struct{}

func (n NopEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}

func (n NopEmbedder) EmbedBatch(_ context.Context, _ []string) ([][]float32, error) {
	return nil, nil
}

func (n NopEmbedder) Dimension() int { return 0 }

// NopVectorStore is a no-op implementation of VectorStore.
type NopVectorStore struct{}

func (n NopVectorStore) Upsert(_ context.Context, _ string, _ string) error { return nil }

func (n NopVectorStore) Search(_ context.Context, _ string, _ int) ([]string, error) {
	return nil, nil
}

func (n NopVectorStore) Get(_ context.Context, _ string) (string, error) { return "", nil }
