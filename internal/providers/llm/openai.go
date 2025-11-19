package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
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
	// PromptTemplate is an optional custom Go template string for formatting the
	// final prompt that is sent to the LLM. If this is empty, a default
	// prompt template will be used by the client.
	PromptTemplate string `json:"promptTemplate,omitempty"`
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
	// Factory function for OpenAI
	openAIFactory := func(ctx context.Context, deps diapi.LLMDeps, cfg *OpenAIOptions) (core.LLMClient, error) {
		if deps.Genkit == nil {
			return nil, fmt.Errorf("missing required dependency 'genkit'")
		}
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("authentication is required: apiKey must be provided for openai provider")
		}
		client, err := NewOpenAI(*cfg, deps.Genkit)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	manglekit.Register(r, &OpenAIOptions{}, openAIFactory)
	r.RegisterHandler(NewHandler())
}

// NewOpenAI is the constructor for the OpenAI client wrapper.
func NewOpenAI(cfg OpenAIOptions, g *genkit.Genkit) (core.LLMClient, error) {

	opts := []option.RequestOption{option.WithAPIKey(cfg.GetAPIKey())}
	if cfg.GetBaseURL() != "" {
		opts = append(opts, option.WithBaseURL(cfg.GetBaseURL()))
	}
	client := &openai.OpenAI{APIKey: cfg.GetAPIKey(), Opts: opts}

	var model ai.Model
	if !cfg.SkipModelCheck {
		model = client.Model(g, cfg.Model)
		if model == nil {
			return nil, fmt.Errorf("failed to get openai model %q from genkit", cfg.Model)
		}
	}

	return NewGenkitLLMAdapter(
		g,
		model,
		"openai",
		cfg.Model,
		cfg.Temperature,
		cfg.MaxOutputTokens,
	), nil
}
