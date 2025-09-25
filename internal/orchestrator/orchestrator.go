// Package orchestrator implements the main workflow orchestrator for Manglekit.
package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"ndduy.dev/manglekit/internal/llm"
	"ndduy.dev/manglekit/internal/mangle"
	"ndduy.dev/manglekit/internal/types"
)

// Config holds the configuration for the Orchestrator.
type Config struct {
	MaxContextTokens  int             `yaml:"maxContextTokens"`
	FallbackThreshold float64         `yaml:"fallbackThreshold"`
	LLM               types.LLMConfig `yaml:"llm"`
	Mangle            mangle.Config   `yaml:"mangle"`
}

// orchestrator implements the types.Orchestrator interface.
type orchestrator struct {
	retriever  types.Retriever
	llmGateway types.Gateway
	rag        ragRetriever
	processor  types.Processor
}

type ragRetriever interface {
	Retrieve(ctx context.Context, query string) ([]string, error)
}

// New creates a new Orchestrator.
func New(ctx context.Context, cfg Config, retriever types.Retriever, rag ragRetriever) (types.Orchestrator, error) {
	llmGateway, err := llm.New(ctx, cfg.LLM)
	if err != nil {
		return nil, err
	}

	processor, err := mangle.New(ctx, cfg.Mangle)
	if err != nil {
		return nil, fmt.Errorf("create mangle processor: %w", err)
	}

	return &orchestrator{
		retriever:  retriever,
		llmGateway: llmGateway,
		rag:        rag,
		processor:  processor,
	}, nil
}

// RunFlow executes the complete Sandwich pattern workflow.
func (o *orchestrator) RunFlow(ctx context.Context, input *types.QueryInput) (*types.Response, error) {
	expanded, err := o.processor.PreProcess(input)
	if err != nil {
		return nil, err
	}

	ragQuery := expanded.NormalizedQuery
	if len(expanded.ExpansionTerms) > 0 {
		ragQuery = strings.Join(append([]string{expanded.NormalizedQuery}, expanded.ExpansionTerms...), " ")
	}

	results, err := o.rag.Retrieve(ctx, ragQuery)
	if err != nil {
		return nil, err
	}

	var chunks []*types.Chunk
	for _, r := range results {
		chunks = append(chunks, &types.Chunk{
			Text: r,
		})
	}

	postChunks, _ := o.processor.PostProcess(chunks, &types.Context{UserContext: input.UserContext})
	response, err := o.llmGateway.Generate(ctx, input.Query, postChunks)
	if err != nil {
		return nil, err
	}

	if expanded != nil && expanded.Explanation != "" {
		response.Explanations = append(response.Explanations, types.Explanation{
			Type:   "mangle-pre",
			Rule:   "expanded_query",
			Action: "applied",
			Reason: expanded.Explanation,
		})
	}

	return response, nil
}
