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

// NewGoogle creates a new Google LLM client.
func NewGoogle(opts GoogleOptions, model ai.Model, g *genkit.Genkit) (core.LLMClient, error) {
	return NewGenkitLLMAdapter(
		g,
		model,
		"google",
		opts.Model,
		opts.Temperature,
		opts.MaxOutputTokens,
	), nil
}
