package core

import (
	"context"
	"fmt"
)

// RetrieverAction adapts a core.Retriever to the core.Action interface.
type RetrieverAction struct {
	Retriever Retriever
	TopK      int
}

// Execute performs retrieval. Input is expected to be a core.Query.
func (ra *RetrieverAction) Execute(ctx context.Context, input any) (any, error) {
	q, ok := input.(Query)
	if !ok {
		return nil, fmt.Errorf("retriever action expects core.Query input, got %T", input)
	}

	req := RetrieveRequest{
		Query: q.Text,
		TopK:  ra.TopK,
		Meta:  q.Meta,
	}

	res, err := ra.Retriever.Retrieve(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GenkitRetrieverAdapter adapts a Genkit retriever to the core.Action interface.
// This ensures compatibility with Genkit-based components in the action-centric model.
type GenkitRetrieverAdapter struct {
	Retriever Retriever
	TopK      int
}

// Execute performs retrieval using the underlying Genkit retriever.
func (g *GenkitRetrieverAdapter) Execute(ctx context.Context, input any) (any, error) {
	q, ok := input.(Query)
	if !ok {
		return nil, fmt.Errorf("genkit retriever adapter expects core.Query input, got %T", input)
	}

	req := RetrieveRequest{
		Query: q.Text,
		TopK:  g.TopK,
		Meta:  q.Meta,
	}

	res, err := g.Retriever.Retrieve(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// HTTPToolAdapter is a proof of concept generic tool adapter implementing core.Action.
type HTTPToolAdapter struct {
	BaseURL string
}

// Execute simulates an HTTP request execution.
func (h *HTTPToolAdapter) Execute(ctx context.Context, input any) (any, error) {
	// In a real implementation, this would make an HTTP request.
	// For generic actions, the input is flexible.
	return fmt.Sprintf("Simulated HTTP request to %s with input: %v", h.BaseURL, input), nil
}
