// Package retrieval defines interfaces for document retrieval.
package retrieval

import (
	"context"

	"ndduy.dev/manglekit/internal/types"
)

// BM25Retriever defines the interface for a BM25 retriever.
type BM25Retriever interface {
	Retrieve(ctx context.Context, query string, cfg types.BM25Config) ([]string, error)
}

// DenseRetriever defines the interface for a dense retriever.
type DenseRetriever interface {
	Retrieve(ctx context.Context, query string, cfg types.DenseConfig) ([]string, error)
}
