// Package llm implements the Gateway interface for Large Language Models.
package llm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/firebase/genkit/go/plugins/ollama"
	"github.com/openai/openai-go/option"
	"ndduy.dev/manglekit/internal/types"
)

// gateway implements the types.Gateway interface.
type gateway struct {
	model ai.Model
}

// New creates a new LLM Gateway based on the provided configuration.
func New(ctx context.Context, cfg types.LLMConfig) (types.Gateway, error) {
	var model ai.Model

	switch cfg.Provider {
	case "openai":
		plugin := &compat_oai.OpenAICompatible{
			Opts:     []option.RequestOption{option.WithAPIKey(cfg.APIKey)},
			Provider: "openai",
		}
		plugin.Init(ctx)
		model = plugin.DefineModel("openai", cfg.Model, ai.ModelOptions{Supports: &compat_oai.BasicText})
	case "ollama":
		plugin := &ollama.Ollama{}
		g := genkit.Init(ctx, genkit.WithPlugins(plugin))
		model = ollama.Model(g, cfg.Model)
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

	req := ai.NewModelRequest(nil, ai.NewUserMessage(ai.NewTextPart(finalPrompt)))

	_, err := g.model.Generate(ctx, req, func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
		if len(chunk.Content) > 0 {
			finalAnswer.WriteString(chunk.Content[0].Text)
		}
		return nil
	})
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
	for i, chunk := range chunks {
		title := chunk.Title
		if title == "" {
			title = chunk.DocID
		}
		contextBuilder.WriteString(fmt.Sprintf("[%d] DocID: %s | Title: %s | Score: %.3f\n", i+1, chunk.DocID, title, chunk.Score))
		contextBuilder.WriteString(chunk.Text)
		contextBuilder.WriteString("\n\n")
	}

	// Simple prompt template
	return fmt.Sprintf(
		"Answer the following question based only on the provided context. Use the numeric citations in brackets (e.g. [1]) to attribute statements.\n\n"+
			"Context:\n%s\n"+
			"Question: %s\n\n"+
			"Answer:",
		contextBuilder.String(),
		userPrompt,
	)
}

// extractCitations extracts document IDs from chunks to be used as citations.
func extractCitations(chunks []*types.Chunk) []string {
	type info struct {
		Title string
		Score float64
	}
	citationSet := make(map[string]info)
	for _, chunk := range chunks {
		docID := chunk.DocID
		if docID == "" {
			docID = chunk.ID
		}
		title := chunk.Title
		if title == "" {
			title = docID
		}
		existing, ok := citationSet[docID]
		if !ok || chunk.Score > existing.Score {
			citationSet[docID] = info{Title: title, Score: chunk.Score}
		}
	}

	citations := make([]string, 0, len(citationSet))
	for docID, meta := range citationSet {
		citations = append(citations, fmt.Sprintf("%s (%s, score=%.2f)", docID, meta.Title, meta.Score))
	}
	sort.Strings(citations)
	return citations
}
