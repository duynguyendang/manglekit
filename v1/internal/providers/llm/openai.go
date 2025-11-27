package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/duynguyendang/manglekit/v1/internal/adapters"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
)

// OpenAIOptions provides typed configuration for OpenAI and compatible language
// models (e.g., Groq). It specifies the model to use and how to authenticate.
type OpenAIOptions struct {
	// APIKey is the API key for authenticating with the OpenAI or a compatible service.
	// If not set here, it is often read from an environment variable (e.g., OPENAI_API_KEY).
	APIKey string `json:"apiKey,omitempty"`
	// Model is the identifier for the specific model to be used for completions,
	// for example, "gpt-4-turbo" or "llama3-8b-8192".
	Model string `json:"model,omitempty"`
	// Temperature controls the randomness of the model's output.
	Temperature float32 `json:"temperature,omitempty"`
	// MaxOutputTokens is the maximum number of tokens to generate in the response.
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
	// BaseURL is an optional override for the OpenAI API base URL. This is useful
	// for pointing the client to a different compatible endpoint, such as Groq's.
	BaseURL string `yaml:"base_url,omitempty"`
	// SkipModelCheck is a test-only option to bypass the live model validation
	// call that genkit performs on initialization.
	SkipModelCheck bool `yaml:"skip_model_check,omitempty"`
}

func (o *OpenAIOptions) ProviderName() string    { return "openai" }
func (o *OpenAIOptions) ProviderKind() core.Kind { return core.KindLLM }
func (o *OpenAIOptions) GetAPIKey() string       { return o.APIKey }
func (o *OpenAIOptions) GetBaseURL() string      { return o.BaseURL }

func RegisterOpenAI(r *manglekit.Registry) {
	manglekit.Register(r, &OpenAIOptions{},
		func(ctx context.Context, deps diapi.LLMDeps, cfg *OpenAIOptions) (core.LLMClient, error) {
			if deps.Genkit == nil {
				return nil, fmt.Errorf("missing required dependency 'genkit'")
			}

			if cfg.Model == "" {
				return nil, fmt.Errorf("openai provider: model must be specified")
			}

			// 1. Initialize the OpenAI Model via Genkit Plugin.
			// APIKey is used if provided, otherwise read from OPENAI_API_KEY environment variable.
			opts := []option.RequestOption{}
			if cfg.APIKey != "" {
				opts = append(opts, option.WithAPIKey(cfg.APIKey))
			}
			if cfg.BaseURL != "" {
				opts = append(opts, option.WithBaseURL(cfg.BaseURL))
			}

			client := &openai.OpenAI{APIKey: cfg.APIKey, Opts: opts}

			var model ai.Model
			if !cfg.SkipModelCheck {
				model = client.Model(deps.Genkit, cfg.Model)
				if model == nil {
					return nil, fmt.Errorf("failed to initialize OpenAI model '%s'", cfg.Model)
				}
			}

			// 2. Return the Universal Adapter configured with this model.
			// The adapter handles all LLM completion logic, response handling, and token usage tracking.
			return adapters.NewGenkitLLMAdapter(
				deps.Genkit,
				model,
				"openai/"+cfg.Model,
				core.LLMOptions{
					Temperature:     cfg.Temperature,
					MaxOutputTokens: cfg.MaxOutputTokens,
				},
			), nil
		},
	)
}
