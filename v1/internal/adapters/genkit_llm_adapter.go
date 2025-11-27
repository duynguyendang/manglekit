package adapters

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// GenkitLLMAdapter wraps a Genkit ai.Model to implement the Manglekit core.LLMClient interface.
// This adapter translates between Manglekit's core.LLMRequest/Response and Genkit's
// generation API, enabling any Genkit-registered model (Google, OpenAI, Ollama, etc.)
// to be used seamlessly within the Manglekit pipeline.
type GenkitLLMAdapter struct {
	g               *genkit.Genkit
	model           ai.Model
	modelName       string
	temperature     float32
	maxOutputTokens int
}

// NewGenkitLLMAdapter creates a new adapter instance.
//
// g is the Genkit runtime instance needed for generation operations.
// model is the Genkit ai.Model to wrap (e.g., from GoogleAI, OpenAI plugin).
// name is a human-readable name for the model (e.g., "google/gemini-1.5-flash").
// opts contains configuration such as temperature and max output tokens.
func NewGenkitLLMAdapter(g *genkit.Genkit, model ai.Model, name string, opts core.LLMOptions) *GenkitLLMAdapter {
	return &GenkitLLMAdapter{
		g:               g,
		model:           model,
		modelName:       name,
		temperature:     opts.Temperature,
		maxOutputTokens: opts.MaxOutputTokens,
	}
}

// Complete implements the core.LLMClient interface.
// It translates Manglekit's LLMRequest to Genkit's generation API format,
// delegates execution to the underlying Genkit model, and translates the response back.
func (a *GenkitLLMAdapter) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	if a.model == nil {
		return core.LLMResponse{}, fmt.Errorf("genkit llm adapter: model not initialized")
	}

	// Construct the prompt with context if provided
	prompt := req.Prompt
	if len(req.Context) > 0 {
		// Prepend context to the prompt as background information
		prompt = "Context:\n" + joinStrings(req.Context, "\n---\n") + "\n\n" + prompt
	}

	// Use the functional options pattern from Genkit's API.
	opts := []ai.GenerateOption{
		ai.WithModel(a.model),
		ai.WithPrompt(prompt),
	}

	// Add temperature/config if set
	if a.temperature > 0 || a.maxOutputTokens > 0 {
		config := make(map[string]any)
		if a.temperature > 0 {
			config["temperature"] = float64(a.temperature)
		}
		if a.maxOutputTokens > 0 {
			config["maxOutputTokens"] = a.maxOutputTokens
		}
		opts = append(opts, ai.WithConfig(config))
	}

	// Use the standard genkit.Generate function.
	res, err := genkit.Generate(ctx, a.g, opts...)
	if err != nil {
		return core.LLMResponse{}, fmt.Errorf("genkit generate error for model %s: %w", a.modelName, err)
	}

	// Extract token usage.
	usage := make(map[string]int)
	if res.Usage != nil {
		usage["prompt"] = res.Usage.InputTokens
		usage["completion"] = res.Usage.OutputTokens
		if res.Usage.TotalTokens > 0 {
			usage["total"] = res.Usage.TotalTokens
		}
	}

	return core.LLMResponse{
		Text:  res.Text(),
		Usage: usage,
	}, nil
}

// joinStrings joins a slice of strings with a separator.
// This is a simple utility to handle context concatenation.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
