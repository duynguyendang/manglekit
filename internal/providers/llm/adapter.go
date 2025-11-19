package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// GenkitLLMAdapter is a generic adapter that wraps any Genkit ai.Model
// and exposes it as a Manglekit core.LLMClient.
// This removes the need for per-provider wrapper structs (e.g., OpenAI, Google)
// that duplicate the same logic.
type GenkitLLMAdapter struct {
	model        ai.Model
	genkit       *genkit.Genkit
	providerName string
	modelName    string
	// Common configuration options
	temperature     float32
	maxOutputTokens int
}

// NewGenkitLLMAdapter creates a new adapter for a Genkit model.
func NewGenkitLLMAdapter(
	g *genkit.Genkit,
	model ai.Model,
	providerName string,
	modelName string,
	temperature float32,
	maxOutputTokens int,
) *GenkitLLMAdapter {
	return &GenkitLLMAdapter{
		genkit:          g,
		model:           model,
		providerName:    providerName,
		modelName:       modelName,
		temperature:     temperature,
		maxOutputTokens: maxOutputTokens,
	}
}

// Complete implements the core.LLMClient interface.
func (a *GenkitLLMAdapter) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	if a.model == nil {
		return core.LLMResponse{}, fmt.Errorf("llm client not initialized with a model")
	}

	// Use the functional options pattern from the new genkit API.
	opts := []ai.GenerateOption{
		ai.WithModel(a.model),
		ai.WithPrompt(req.Prompt),
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
	res, err := genkit.Generate(ctx, a.genkit, opts...)
	if err != nil {
		return core.LLMResponse{}, fmt.Errorf("%s: llm completion failed: %w", a.providerName, err)
	}

	// Extract token usage.
	usage := make(map[string]int)
	if res.Usage != nil {
		usage["prompt"] = int(res.Usage.InputTokens)
		usage["completion"] = int(res.Usage.OutputTokens)
		usage["total"] = int(res.Usage.TotalTokens)
	}

	return core.LLMResponse{
		Text:  res.Text(),
		Usage: usage,
	}, nil
}

// Model returns the model identifier.
func (a *GenkitLLMAdapter) Model() string {
	return a.modelName
}

// GetName returns the provider name.
func (a *GenkitLLMAdapter) GetName() string {
	return a.providerName
}
