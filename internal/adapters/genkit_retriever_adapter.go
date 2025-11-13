package adapters

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// GenkitRetrieverAdapter wraps a core.Retriever (which may delegate to Genkit) and tracks
// the provider for logging. This adapter is used when a Retriever is backed by a Genkit plugin.
//
// Note: Read-only constraint
// Genkit retrievers are designed as read-only systems. The adapter returns
// ErrNotSupported for any write operations (IndexDocuments, UpdateDocuments).
type GenkitRetrieverAdapter struct {
	retriever core.Retriever
	logger    core.Logger
	provider  string // For logging and debugging
}

// NewGenkitRetrieverAdapter creates a new adapter wrapping a core.Retriever.
// provider is used for logging/debugging to identify which Genkit plugin is being used.
func NewGenkitRetrieverAdapter(retriever core.Retriever, provider string, logger core.Logger) *GenkitRetrieverAdapter {
	return &GenkitRetrieverAdapter{
		retriever: retriever,
		logger:    logger,
		provider:  provider,
	}
}

// Retrieve delegates the retrieval request to the wrapped retriever.
// It handles error logging and wraps errors with provider context.
func (a *GenkitRetrieverAdapter) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	if a.retriever == nil {
		return core.RetrieveResult{}, fmt.Errorf("genkit retriever adapter (%s): underlying retriever is nil", a.provider)
	}

	if a.logger != nil {
		a.logger.Debugf(
			"delegating retrieval request to Genkit provider",
			"provider", a.provider,
			"query", req.Query,
			"topk", req.TopK,
		)
	}

	result, err := a.retriever.Retrieve(ctx, req)
	if err != nil {
		if a.logger != nil {
			a.logger.Debugf(
				"genkit retriever delegation failed",
				"provider", a.provider,
				"error", err.Error(),
			)
		}
		return core.RetrieveResult{}, fmt.Errorf("genkit retriever (%s) failed: %w", a.provider, err)
	}

	if a.logger != nil {
		a.logger.Debugf(
			"retrieval request completed successfully",
			"provider", a.provider,
			"doc_count", len(result.Docs),
		)
	}

	return result, nil
}

// IndexDocuments returns ErrNotSupported since Genkit retrievers are read-only.
func (a *GenkitRetrieverAdapter) IndexDocuments(ctx context.Context, docs []core.Doc) error {
	if a.logger != nil {
		a.logger.Debugf(
			"IndexDocuments called on read-only retriever (Genkit-delegated)",
			"provider", a.provider,
			"operation", "index_documents",
			"result", "not_supported",
		)
	}
	return fmt.Errorf("genkit retriever (%s): %w", a.provider, core.ErrNotSupported)
}

// UpdateDocuments returns ErrNotSupported since Genkit retrievers are read-only.
func (a *GenkitRetrieverAdapter) UpdateDocuments(ctx context.Context, docs []core.Doc) error {
	if a.logger != nil {
		a.logger.Debugf(
			"UpdateDocuments called on read-only retriever (Genkit-delegated)",
			"provider", a.provider,
			"operation", "update_documents",
			"result", "not_supported",
		)
	}
	return fmt.Errorf("genkit retriever (%s): %w", a.provider, core.ErrNotSupported)
}

// Name returns a human-readable name for the adapter.
func (a *GenkitRetrieverAdapter) Name() string {
	return fmt.Sprintf("genkit-retriever(%s)", a.provider)
}
