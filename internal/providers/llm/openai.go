package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
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
}

func (o OpenAIOptions) ProviderName() string { return "openai" }
func (o OpenAIOptions) ProviderKind() core.Kind   { return core.KindLLM }
func (o OpenAIOptions) GetAPIKey() string       { return o.APIKey }
func (o OpenAIOptions) GetBaseURL() string      { return o.BaseURL }

func RegisterOpenAI(r *manglekit.Registry) {
	// Factory function for OpenAI
	openAIFactory := func(ctx context.Context, d struct {
		diapi.LLMDeps
		diapi.OpenAIClientProvider
	}, cfg OpenAIOptions) (core.LLMClient, error) {
		if d.Genkit == nil {
			return nil, fmt.Errorf("missing required dependency 'genkit'")
		}
		if d.OpenAIClient() == nil {
			return nil, fmt.Errorf("missing required dependency 'OpenAIClient'")
		}
		model := d.OpenAIClient().Model(d.Genkit, cfg.Model)
		if model == nil {
			return nil, fmt.Errorf("failed to get openai model %q from genkit", cfg.Model)
		}
		return NewOpenAI(cfg, model, d.Genkit), nil
	}
	manglekit.Register(r, OpenAIOptions{}, openAIFactory)
}

// OpenAI is a wrapper around a genkit AI model from the OpenAI plugin.
type OpenAI struct {
	opts   OpenAIOptions
	model  ai.Model
	genkit *genkit.Genkit
}

// NewOpenAI is the constructor for the OpenAI client wrapper.
func NewOpenAI(opts OpenAIOptions, model ai.Model, g *genkit.Genkit) core.LLMClient {
	return &OpenAI{
		opts:   opts,
		model:  model,
		genkit: g,
	}
}

// Complete implements the core.LLMClient interface.
func (o *OpenAI) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	if o.model == nil {
		return core.LLMResponse{}, fmt.Errorf("openai llm client not initialized with a model")
	}

	// Start with default provider options.
	config := &ai.GenerationCommonConfig{
		Temperature:     float64(o.opts.Temperature),
		MaxOutputTokens: o.opts.MaxOutputTokens,
	}

	// Override with request-specific options if provided.
	if req.MaxTokens > 0 {
		config.MaxOutputTokens = req.MaxTokens
	}

	// Use the standard genkit.Generate function.
	res, err := genkit.Generate(ctx, o.genkit,
		ai.WithModel(o.model),
		ai.WithPrompt(req.Prompt),
		ai.WithConfig(config),
	)
	if err != nil {
		return core.LLMResponse{}, err
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
