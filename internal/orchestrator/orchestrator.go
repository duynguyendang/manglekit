// Package orchestrator implements the main workflow orchestrator for Manglekit.
package orchestrator

import (
	"context"

	"ndduy.dev/manglekit/internal/llm"
	"ndduy.dev/manglekit/internal/rag"
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
	rag        *rag.RAG
}

// New creates a new Orchestrator.
func New(ctx context.Context, cfg Config, retriever types.Retriever, rag *rag.RAG) (types.Orchestrator, error) {
	llmGateway, err := llm.New(ctx, cfg.LLM)
	if err != nil {
		return nil, err
	}

	return &orchestrator{
		retriever:  retriever,
		llmGateway: llmGateway,
		rag:        rag,
	}, nil
}

// RunFlow executes the complete Sandwich pattern workflow.
func (o *orchestrator) RunFlow(ctx context.Context, input *types.QueryInput) (*types.Response, error) {
	// This is a simplified flow that only calls the LLM Gateway.
	// A complete implementation would include Mangle-Pre, Retrieval, and Mangle-Post.
	results, err := o.rag.Retrieve(ctx, input.Query)
	if err != nil {
		return nil, err
	}

	var chunks []*types.Chunk
	for _, r := range results {
		chunks = append(chunks, &types.Chunk{
			Text: r,
		})
	}

	return o.llmGateway.Generate(ctx, input.Query, chunks)
}