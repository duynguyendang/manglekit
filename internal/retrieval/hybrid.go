// Package retrieval provides hybrid retrieval functionality.
package retrieval

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	"ndduy.dev/manglekit/internal/types"
)

// HybridRetriever combines results from BM25 and Dense retrievers.
type HybridRetriever struct {
	bm25  BM25Retriever
	dense DenseRetriever
}

// NewHybridRetriever creates a new HybridRetriever.
func NewHybridRetriever(bm25 BM25Retriever, dense DenseRetriever) (*HybridRetriever, error) {
	if bm25 == nil {
		return nil, fmt.Errorf("BM25 retriever is required")
	}
	if dense == nil {
		return nil, fmt.Errorf("dense retriever is required")
	}
	return &HybridRetriever{
		bm25:  bm25,
		dense: dense,
	}, nil
}

// Retrieve performs a hybrid search.
func (h *HybridRetriever) Retrieve(ctx context.Context, query string, bm25Cfg types.BM25Config, denseCfg types.DenseConfig) ([]string, error) {
	var g errgroup.Group
	var bm25Results, denseResults []string

	g.Go(func() error {
		var err error
		bm25Results, err = h.bm25.Retrieve(ctx, query, bm25Cfg)
		return err
	})

	g.Go(func() error {
		var err error
		denseResults, err = h.dense.Retrieve(ctx, query, denseCfg)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Combine and deduplicate results
	allResults := append(bm25Results, denseResults...)
	seen := make(map[string]struct{})
	var uniqueResults []string
	for _, r := range allResults {
		if _, ok := seen[r]; !ok {
			seen[r] = struct{}{}
			uniqueResults = append(uniqueResults, r)
		}
	}

	return uniqueResults, nil
}
