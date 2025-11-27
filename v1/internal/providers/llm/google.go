package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/duynguyendang/manglekit/v1/internal/adapters"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// GoogleOptions provides typed configuration for Google language models,
// such as those in the Gemini family.
type GoogleOptions struct {
	// Model is the identifier for the specific Google model to be used for
	// completions, for example, "gemini-1.5-flash".
	Model string `json:"model"`
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
				return nil, fmt.Errorf("missing required dependency 'genkit'")
			}

			if cfg.Model == "" {
				return nil, fmt.Errorf("google provider: model must be specified")
			}

			// 1. Initialize the Google GenAI Model via Genkit Plugin.
			// This uses the GOOGLE_API_KEY from the environment automatically.
			model := googlegenai.GoogleAIModel(deps.Genkit, cfg.Model)
			if model == nil {
				return nil, fmt.Errorf("failed to initialize Google model '%s'", cfg.Model)
			}

			// 2. Return the Universal Adapter configured with this model.
			// The adapter handles all LLM completion logic, message conversion, and response handling.
			return adapters.NewGenkitLLMAdapter(
				deps.Genkit,
				model,
				"google/"+cfg.Model,
				core.LLMOptions{
					Temperature:     cfg.Temperature,
					MaxOutputTokens: cfg.MaxOutputTokens,
				},
			), nil
		},
	)
}
