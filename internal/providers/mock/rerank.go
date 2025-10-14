package mock

import (
	"context"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/rerank"
)

// MockReranker is a mock reranker for testing.
type MockReranker struct{}

// NewMockReranker creates a new mock reranker.
func NewMockReranker(_ any) (rerank.Reranker, error) {
	return &MockReranker{}, nil
}

// Rerank returns the documents as is.
func (r *MockReranker) Rerank(ctx context.Context, req rerank.Request) ([]rerank.ScoredDoc, error) {
	var scoredDocs []rerank.ScoredDoc
	for _, doc := range req.Docs {
		scoredDocs = append(scoredDocs, rerank.ScoredDoc{Doc: doc, Score: 1.0})
	}
	return scoredDocs, nil
}

func init() {
	manglekit.RegisterReranker("mock", NewMockReranker)
}
