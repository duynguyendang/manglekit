package retrieve

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
)

// NoopRetriever is a retriever that does nothing.
type NoopRetriever struct{}

// Retrieve implements the Retriever interface.
func (r *NoopRetriever) Retrieve(ctx context.Context, req Request) (Result, error) {
	return Result{Docs: []core.Doc{}}, nil
}
