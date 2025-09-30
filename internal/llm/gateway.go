// Package llm implements the Gateway interface for Large Language Models.
package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/ollama"
	"github.com/openai/openai-go/option"
	"ndduy.dev/manglekit/internal/types"
)

const (
	// defaultPromptTemplate is used if no template is provided in the config.
	defaultPromptTemplate = "Answer the following question based on the provided context:\n\n" +
		"Context:\n%s\n\n" +
		"Question: %s\n\n" +
		"Answer:"
)

// gateway implements the types.Gateway interface.
type gateway struct {
	model ai.Model
	cfg   types.LLMConfig
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
	case "google":
		plugin := &googlegenai.GoogleAI{APIKey: cfg.APIKey}
		g := genkit.Init(ctx, genkit.WithPlugins(plugin))
		model = googlegenai.GoogleAIModel(g, cfg.Model)
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

	return &gateway{model: model, cfg: cfg}, nil
}

// Generate creates a response using the provided context.
func (g *gateway) Generate(ctx context.Context, prompt string, chunks []*types.Chunk) (*types.Response, error) {
	finalPrompt, finalChunks := g.buildPrompt(prompt, chunks)
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
		Citations: extractCitations(finalChunks),
	}, nil
}

// buildPrompt constructs the final prompt from a template, user query, and context chunks.
// It also truncates the chunks to respect the token limit.
func (g *gateway) buildPrompt(userPrompt string, chunks []*types.Chunk) (string, []*types.Chunk) {
	template := g.cfg.PromptTemplate
	if template == "" {
		template = defaultPromptTemplate
	}

	// Estimate tokens used by the template and the user prompt.
	// The '%s' for context and question are placeholders.
	templateTokens := countWords(fmt.Sprintf(template, "", userPrompt))

	var contextBuilder strings.Builder
	var finalChunks []*types.Chunk
	remainingTokens := g.cfg.MaxContextTokens - templateTokens

	for _, chunk := range chunks {
		chunkTokens := countWords(chunk.Text)
		if remainingTokens-chunkTokens < 0 {
			break // Not enough tokens left for this chunk
		}
		contextBuilder.WriteString(fmt.Sprintf("- %s\n", chunk.Text))
		finalChunks = append(finalChunks, chunk)
		remainingTokens -= chunkTokens
	}

	// Final prompt
	finalPrompt := fmt.Sprintf(
		template,
		contextBuilder.String(),
		userPrompt,
	)
	return finalPrompt, finalChunks
}

// countWords provides a simple estimation of token count.
func countWords(text string) int {
	return len(strings.Fields(text))
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