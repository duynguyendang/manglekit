package llm

import "github.com/duynguyendang/manglekit/core"

// GenkitLLMOptions provides universal configuration for ANY Genkit LLM provider.
// This single options struct replaces provider-specific options (OpenAI, Google, etc.)
// by allowing users to specify the provider in configuration rather than hard-coding it.
//
// Example usage in config.yaml:
//
//	components:
//	  - name: my-llm
//	    kind: llm
//	    type: genkit-llm
//	    params:
//	      provider: openai
//	      model: gpt-4-turbo
//	      apiKey: "${OPENAI_API_KEY}"
type GenkitLLMOptions struct {
	// Provider is the Genkit LLM provider name.
	// Supported values: "openai", "google", "anthropic", "vertexai", etc.
	// Any Genkit-compatible LLM plugin can be specified here.
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`

	// Model is the model identifier for the provider.
	// Examples:
	//   - OpenAI: "gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"
	//   - Google: "gemini-1.5-pro", "gemini-1.5-flash"
	//   - Anthropic: "claude-3-opus", "claude-3-sonnet"
	//   - Vertex AI: "gemini-pro"
	Model string `json:"model,omitempty" yaml:"model,omitempty"`

	// APIKey is the API authentication key for providers that require it.
	// If not provided here, providers typically look for environment variables
	// (e.g., OPENAI_API_KEY, GOOGLE_API_KEY, ANTHROPIC_API_KEY).
	APIKey string `json:"apiKey,omitempty" yaml:"api_key,omitempty"`

	// BaseURL is an optional base URL override for API endpoints.
	// Useful for OpenAI-compatible APIs (e.g., Groq, custom local servers).
	// Example: "https://api.groq.com/openai/v1"
	BaseURL string `json:"baseUrl,omitempty" yaml:"base_url,omitempty"`

	// Temperature controls the randomness of the model's output.
	// Typical range: 0.0 (deterministic) to 2.0 (most random).
	// Lower values make output more focused and deterministic.
	Temperature float32 `json:"temperature,omitempty" yaml:"temperature,omitempty"`

	// MaxOutputTokens is the maximum number of tokens to generate in the response.
	// If unset, the provider's default limit applies.
	MaxOutputTokens int `json:"maxOutputTokens,omitempty" yaml:"max_output_tokens,omitempty"`

	// PromptTemplate is an optional custom Go template string for formatting the
	// final prompt that is sent to the LLM. If this is empty, a default
	// prompt template will be used by the client.
	PromptTemplate string `json:"promptTemplate,omitempty" yaml:"prompt_template,omitempty"`

	// SkipModelCheck bypasses live model validation when true.
	// Useful for testing or when the model is not immediately available.
	SkipModelCheck bool `json:"skipModelCheck,omitempty" yaml:"skip_model_check,omitempty"`

	// ProviderConfig is a map of arbitrary provider-specific configuration.
	// This allows passing additional parameters for new or custom Genkit providers
	// without requiring code changes to Manglekit.
	// Example:
	//   providerConfig:
	//     custom_param_1: "value1"
	//     custom_param_2: 42
	ProviderConfig map[string]any `json:"providerConfig,omitempty" yaml:"provider_config,omitempty"`
}

func (o *GenkitLLMOptions) ProviderName() string    { return "genkit-llm" }
func (o *GenkitLLMOptions) ProviderKind() core.Kind { return core.KindLLM }

// GetAPIKey implements the diapi.APIKeyProvider interface for generic API key access.
func (o *GenkitLLMOptions) GetAPIKey() string { return o.APIKey }

// GetBaseURL implements the diapi.BaseURLProvider interface for generic base URL access.
func (o *GenkitLLMOptions) GetBaseURL() string { return o.BaseURL }

// ShouldSkipModelCheck implements the diapi.SkipModelCheckProvider interface.
func (o *GenkitLLMOptions) ShouldSkipModelCheck() bool { return o.SkipModelCheck }
