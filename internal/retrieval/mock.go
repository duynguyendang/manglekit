// Package retrieval implements a mock retriever for Manglekit.
package retrieval

import (
	"context"

	"ndduy.dev/manglekit/internal/types"
)

// MockRetriever is a mock implementation of the types.Retriever interface.
type MockRetriever struct{}

// NewMock creates a new MockRetriever.
func NewMock() types.Retriever {
	return &MockRetriever{}
}

// Search returns a predefined set of chunks.
func (r *MockRetriever) Search(ctx context.Context, query *types.ExpandedQuery, filters map[string]string) ([]*types.Chunk, error) {
	return []*types.Chunk{
		{
			ID:    "chunk-1",
			DocID: "doc-1",
			Text:  "Manglekit is a lightweight, embeddable Go framework for Retrieval-Augmented Generation (RAG) workflows.",
		},
		{
			ID:    "chunk-2",
			DocID: "doc-1",
			Text:  "It integrates declarative rules and semantic search.",
		},
	}, nil
}
