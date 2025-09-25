// Package orchestrator implements the main workflow orchestrator for Manglekit.
package orchestrator

import (
	"context"

	"ndduy.dev/manglekit/internal/llm"
	"ndduy.dev/manglekit/internal/types"
)

// Config holds the configuration for the Orchestrator.
type Config struct {
	MaxContextTokens  int             `yaml:"maxContextTokens"`
	FallbackThreshold float64         `yaml:"fallbackThreshold"`
	LLM               types.LLMConfig `yaml:"llm"`
}

// orchestrator implements the types.Orchestrator interface.
type orchestrator struct {
	retriever  types.Retriever
	llmGateway types.Gateway
}

// New creates a new Orchestrator.
func New(ctx context.Context, cfg Config, retriever types.Retriever) (types.Orchestrator, error) {
	llmGateway, err := llm.New(ctx, cfg.LLM)
	if err != nil {
		return nil, err
	}

	return &orchestrator{
		retriever:  retriever,
		llmGateway: llmGateway,
	}, nil
}

// RunFlow executes the complete Sandwich pattern workflow.
func (o *orchestrator) RunFlow(ctx context.Context, input *types.QueryInput) (*types.Response, error) {
	// This is a simplified flow that only calls the LLM Gateway.
	// A complete implementation would include Mangle-Pre, Retrieval, and Mangle-Post.
	expandedQuery := &types.ExpandedQuery{NormalizedQuery: input.Query}
	chunks, err := o.retriever.Search(ctx, expandedQuery, nil)
	if err != nil {
		return nil, err
	}

	return o.llmGateway.Generate(ctx, input.Query, chunks)
}