package retrieve

import (
	"context"

	"github.com/duynguyendang/manglekit/v1/core"
)

// NoopRetriever is a retriever that does nothing.
type NoopRetriever struct{}

// Retrieve implements the Retriever interface.
func (r *NoopRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{Docs: []core.Doc{}}, nil
}
