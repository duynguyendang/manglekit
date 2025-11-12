package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// GoogleOptions provides typed configuration for Google language models,
// such as those in the Gemini family.
type GoogleOptions struct {
	// Model is the identifier for the specific Google model to be used for
	// completions, for example, "gemini-1.5-flash".
	Model string `json:"model"`
	// PromptTemplate is an optional custom Go template string for formatting the
	// final prompt that is sent to the LLM. If this is empty, a default
	// prompt template will be used by the client.
	PromptTemplate string `json:"promptTemplate"`
	// Temperature controls the randomness of the model's output.
	Temperature float32 `json:"temperature,omitempty"`
	// MaxOutputTokens is the maximum number of tokens to generate in the response.
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

func (o *GoogleOptions) ProviderName() string    { return "google" }
func (o *GoogleOptions) ProviderKind() core.Kind { return core.KindLLM }

func RegisterGoogle(r *manglekit.Registry) {
	manglekit.Register(r, &GoogleOptions{},
		func(ctx context.Context, deps diapi.LLMDeps, cfg *GoogleOptions) (core.LLMClient, error) {
			if deps.Genkit == nil {
				return nil, fmt.Errorf("missing required dependency 'genkit' of type *genkit.Genkit")
			}

			// Try to get or create the Google AI model
			model := googlegenai.GoogleAIModel(deps.Genkit, cfg.Model)
			if model == nil {
				return nil, fmt.Errorf("failed to initialize Google model '%s': ensure GOOGLE_API_KEY environment variable is set", cfg.Model)
			}

			return NewGoogle(*cfg, model, deps.Genkit)
		},
	)
}

// Google is a wrapper around a genkit AI model.
type Google struct {
	opts   GoogleOptions
	model  ai.Model
	genkit *genkit.Genkit
}

// NewGoogle creates a new Google LLM client.
func NewGoogle(opts GoogleOptions, model ai.Model, g *genkit.Genkit) (core.LLMClient, error) {
	return &Google{opts: opts, model: model, genkit: g}, nil
}

// Complete implements the core.LLMClient interface.
func (g *Google) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	if g.model == nil {
		return core.LLMResponse{}, fmt.Errorf("google llm client not initialized with a model")
	}

	// Use the functional options pattern from the new genkit API.
	// Don't pass temperature/config as the Google plugin may not support it
	opts := []ai.GenerateOption{
		ai.WithModel(g.model),
		ai.WithPrompt(req.Prompt),
	}

	// Add temperature if set
	if g.opts.Temperature > 0 {
		// Try to use GenerateConfig directly
		opts = append(opts, ai.WithConfig(map[string]any{
			"temperature":     float64(g.opts.Temperature),
			"maxOutputTokens": g.opts.MaxOutputTokens,
		}))
	}

	res, err := genkit.Generate(ctx, g.genkit, opts...)

	if err != nil {
		return core.LLMResponse{}, fmt.Errorf("google: llm completion failed: %w", err)
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

func (g *Google) Model() string {
	return g.opts.Model
}

func (g *Google) GetName() string {
	return "google"
}
