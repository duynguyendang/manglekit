package genkit

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/adapters"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// GenericOptions allows using any model registered in the Genkit instance.
// This provider is useful for custom or dynamically-registered Genkit models
// that don't have a dedicated Manglekit provider.
//
// To use a custom model with this provider:
// 1. Ensure the model is registered in your Genkit instance
// 2. Set the provider to "genkit-llm"
// 3. Set the model name to the model's identifier
//
// Example YAML:
//
//	llm:
//	  provider: genkit-llm
//	  model: custom/my-model
//	  temperature: 0.7
type GenericOptions struct {
	// ModelProvider is the provider plugin name (e.g., "custom", for a custom-registered model).
	// This is used internally if a specific provider client needs to be instantiated.
	ModelProvider string `json:"modelProvider,omitempty"`
	// ModelName is the name/identifier of the model to use.
	ModelName string `json:"model"`
	// Temperature controls the randomness of the model's output.
	Temperature float32 `json:"temperature,omitempty"`
	// MaxOutputTokens is the maximum number of tokens to generate in the response.
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

func (o *GenericOptions) ProviderName() string    { return "genkit-llm" }
func (o *GenericOptions) ProviderKind() core.Kind { return core.KindLLM }

// Register registers the generic Genkit LLM provider factory.
func Register(r *manglekit.Registry) error {
	manglekit.Register(r, &GenericOptions{},
		func(ctx context.Context, deps diapi.LLMDeps, cfg *GenericOptions) (core.LLMClient, error) {
			if deps.Genkit == nil {
				return nil, fmt.Errorf("missing required dependency 'genkit'")
			}

			if cfg.ModelName == "" {
				return nil, fmt.Errorf("genkit-llm provider: model name must be specified")
			}

			// Attempt to get a model from Genkit using ai.Model.
			// Since Genkit doesn't expose a model registry lookup directly,
			// this approach uses ai.Model if passed in from elsewhere, or requires
			// a specific provider plugin to be configured.
			//
			// For production use cases, create a dedicated thin-factory provider
			// (like google.go or openai.go) instead of using this generic provider.
			//
			// This generic provider is primarily useful for:
			// 1. Testing with mock models
			// 2. Custom internal models registered within your application
			// 3. Rapid prototyping before creating a dedicated provider
			//
			// TODO: If Genkit exposes a model registry in future versions,
			// we can improve this to support dynamic model lookup.

			// For now, we return an error with guidance
			return nil, fmt.Errorf(
				"genkit-llm provider: dynamic model lookup is not yet supported in Genkit. " +
					"Please use a dedicated provider (e.g., 'google', 'openai') or create a custom thin-factory provider. " +
					"See internal/providers/llm/google.go for an example.",
			)
		},
	)

	return nil
}

// CustomModelLLMAdapter creates an adapter for a custom-registered Genkit model.
// This is a helper function for applications that register models directly.
// Usage:
//
//	model := myCustomGenkitPlugin.GetModel("my-model")
//	client := genkit.CustomModelLLMAdapter(genkitInstance, model, "custom/my-model", opts)
func CustomModelLLMAdapter(g *genkit.Genkit, model ai.Model, name string, opts core.LLMOptions) core.LLMClient {
	return adapters.NewGenkitLLMAdapter(g, model, name, opts)
}
