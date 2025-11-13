package embed

import "github.com/duynguyendang/manglekit/core"

// GenkitEmbedderOptions provides universal configuration for ANY Genkit embedder provider.
// This single options struct replaces provider-specific options (OpenAI, Google, etc.)
// by allowing users to specify the provider in configuration rather than hard-coding it.
//
// Example usage in config.yaml:
//
//	components:
//	  - name: my-embedder
//	    kind: embedder
//	    type: genkit-embedder
//	    params:
//	      provider: openai
//	      model: text-embedding-3-small
//	      apiKey: "${OPENAI_API_KEY}"
type GenkitEmbedderOptions struct {
	// Provider is the Genkit embedder provider name.
	// Supported values: "openai", "google", "vertex", "cohere", "anthropic", etc.
	// Any Genkit-compatible embedder plugin can be specified here.
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`

	// Model is the model identifier for the provider.
	// Examples:
	//   - OpenAI: "text-embedding-3-small", "text-embedding-3-large"
	//   - Google: "embedding-001"
	//   - Vertex: "textembedding-gecko@latest"
	//   - Cohere: "embed-english-v3.0"
	Model string `json:"model,omitempty" yaml:"model,omitempty"`

	// APIKey is the API authentication key for providers that require it.
	// If not provided here, providers typically look for environment variables
	// (e.g., OPENAI_API_KEY, GOOGLE_API_KEY, COHERE_API_KEY).
	APIKey string `json:"apiKey,omitempty" yaml:"api_key,omitempty"`

	// BaseURL is an optional base URL override for API endpoints.
	// Useful for OpenAI-compatible APIs (e.g., Groq, custom local servers).
	// Example: "https://api.groq.com/openai/v1"
	BaseURL string `json:"baseUrl,omitempty" yaml:"base_url,omitempty"`

	// Dimensions is the desired dimensionality of output embeddings.
	// Only supported by some providers (e.g., OpenAI text-embedding-3).
	// If the provider doesn't support it, this is ignored.
	Dimensions int `json:"dimensions,omitempty" yaml:"dimensions,omitempty"`

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

func (o *GenkitEmbedderOptions) ProviderName() string    { return "genkit-embedder" }
func (o *GenkitEmbedderOptions) ProviderKind() core.Kind { return core.KindEmbedder }

// GetAPIKey implements the diapi.APIKeyProvider interface for generic API key access.
func (o *GenkitEmbedderOptions) GetAPIKey() string { return o.APIKey }

// GetBaseURL implements the diapi.BaseURLProvider interface for generic base URL access.
func (o *GenkitEmbedderOptions) GetBaseURL() string { return o.BaseURL }

// ShouldSkipModelCheck implements the diapi.SkipModelCheckProvider interface.
func (o *GenkitEmbedderOptions) ShouldSkipModelCheck() bool { return o.SkipModelCheck }
