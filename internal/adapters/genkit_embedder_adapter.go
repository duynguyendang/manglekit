package adapters

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
)

// GenkitEmbedderAdapter wraps a Genkit ai.Embedder and adapts it to ai.Embedder interface.
// This adapter is provider-agnostic and works with any Genkit embedder plugin
// (OpenAI, Google, Vertex, Cohere, Anthropic, etc.).
type GenkitEmbedderAdapter struct {
	embedder ai.Embedder
	logger   core.Logger
	provider string // For logging and debugging
}

// NewGenkitEmbedderAdapter creates a new adapter wrapping a Genkit embedder.
// provider is used for logging/debugging to identify which Genkit plugin is being used.
func NewGenkitEmbedderAdapter(embedder ai.Embedder, provider string, logger core.Logger) *GenkitEmbedderAdapter {
	return &GenkitEmbedderAdapter{
		embedder: embedder,
		logger:   logger,
		provider: provider,
	}
}

// Embed delegates the embedding request to the wrapped Genkit embedder.
// It handles error logging and wraps errors with provider context.
func (a *GenkitEmbedderAdapter) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	if a.embedder == nil {
		return nil, fmt.Errorf("genkit embedder adapter (%s): underlying embedder is nil", a.provider)
	}

	if req == nil {
		return nil, fmt.Errorf("genkit embedder adapter (%s): embed request is nil", a.provider)
	}

	if a.logger != nil {
		a.logger.Debugf(
			"delegating embedding request to Genkit provider",
			"provider", a.provider,
			"input_count", len(req.Input),
		)
	}

	resp, err := a.embedder.Embed(ctx, req)
	if err != nil {
		if a.logger != nil {
			a.logger.Debugf(
				"genkit embedder delegation failed",
				"provider", a.provider,
				"error", err.Error(),
			)
		}
		return nil, fmt.Errorf("genkit embedder (%s) failed: %w", a.provider, err)
	}

	if a.logger != nil {
		a.logger.Debugf(
			"embedding request completed successfully",
			"provider", a.provider,
			"embedding_count", len(resp.Embeddings),
		)
	}

	return resp, nil
}

// Name returns a human-readable name for the adapter.
func (a *GenkitEmbedderAdapter) Name() string {
	return fmt.Sprintf("genkit-embedder(%s)", a.provider)
}
