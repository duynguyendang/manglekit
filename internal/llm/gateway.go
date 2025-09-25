// Package llm implements the Gateway interface for Large Language Models.
package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/ollama"
	"github.com/yukinagae/genkit-go-plugins/plugins/openai"
	"ndduy.dev/manglekit/internal/types"
)


// gateway implements the types.Gateway interface.
type gateway struct {
	model *genkit.Model
}

// New creates a new LLM Gateway based on the provided configuration.
func New(ctx context.Context, cfg types.LLMConfig) (types.Gateway, error) {
	var model *genkit.Model
	var err error

	switch cfg.Provider {
	case "openai":
		if err = openai.Init(ctx, &openai.Config{
			APIKey: cfg.APIKey,
		}); err != nil {
			return nil, fmt.Errorf("failed to initialize openai plugin: %w", err)
		}
		model = openai.Model(cfg.Model)
	case "ollama":
		if err := ollama.Init(ctx, &ollama.Config{}); err != nil {
			return nil, fmt.Errorf("failed to initialize ollama plugin: %w", err)
		}
		model, err = ollama.DefineModel(ctx, &ollama.ModelDefinition{
			Name:  cfg.Model,
			Model: cfg.Model,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to define ollama model: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}

	if model == nil {
		return nil, fmt.Errorf("failed to load model: %s", cfg.Model)
	}

	return &gateway{model: model}, nil
}

// Generate creates a response using the provided context.
func (g *gateway) Generate(ctx context.Context, prompt string, chunks []*types.Chunk) (*types.Response, error) {
	finalPrompt := buildPrompt(prompt, chunks)
	var finalAnswer strings.Builder

	_, err := g.model.Generate(ctx, genkit.NewPart(finalPrompt), genkit.WithStreaming(func(chunk *genkit.GenerateResponseChunk) error {
		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Message.Content) > 0 {
			finalAnswer.WriteString(chunk.Candidates[0].Message.Content[0].Text)
		}
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	return &types.Response{
		Answer:    finalAnswer.String(),
		Citations: extractCitations(chunks),
	}, nil
}

// buildPrompt constructs the final prompt from a template, user query, and context chunks.
func buildPrompt(userPrompt string, chunks []*types.Chunk) string {
	var contextBuilder strings.Builder
	for _, chunk := range chunks {
		contextBuilder.WriteString(fmt.Sprintf("- %s\n", chunk.Text))
	}

	// Simple prompt template
	return fmt.Sprintf(
		"Answer the following question based on the provided context:\n\n"+
			"Context:\n%s\n\n"+
			"Question: %s\n\n"+
			"Answer:",
		contextBuilder.String(),
		userPrompt,
	)
}

// extractCitations extracts document IDs from chunks to be used as citations.
func extractCitations(chunks []*types.Chunk) []string {
	citationSet := make(map[string]struct{})
	for _, chunk := range chunks {
		citationSet[chunk.DocID] = struct{}{}
	}

	citations := make([]string, 0, len(citationSet))
	for docID := range citationSet {
		citations = append(citations, docID)
	}
	return citations
}