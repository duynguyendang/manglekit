package embed

import "github.com/duynguyendang/manglekit/core"

// GoogleEmbedderOptions provides typed configuration for Google embedding models,
// such as those available through the `generative-ai-go` library.
type GoogleEmbedderOptions struct {
	// Model is the identifier for the specific Google embedding model to be used,
	// for example, "embedding-001".
	Model string `json:"model,omitempty"`
}

func (o *GoogleEmbedderOptions) ProviderName() string { return "google" }
func (o *GoogleEmbedderOptions) ProviderKind() core.Kind   { return core.KindEmbedder }

// OpenAIEmbedderOptions provides typed configuration for OpenAI and compatible
// embedding models.
type OpenAIEmbedderOptions struct {
	// APIKey is the API key for authenticating with the OpenAI or a compatible service.
	APIKey string `json:"apiKey,omitempty"`
	// Model is the identifier for the specific OpenAI model to be used,
	// for example, "text-embedding-3-small".
	Model string `json:"model,omitempty"`
	// Dimensions is the desired dimensionality of the output vectors. This is
	// supported by newer OpenAI models (like text-embedding-3) and allows for
	// trading off performance and accuracy. If omitted, the provider's default
	// dimensionality is used.
	Dimensions int `json:"dimensions,omitempty"`
}

func (o *OpenAIEmbedderOptions) ProviderName() string { return "openai" }
func (o *OpenAIEmbedderOptions) ProviderKind() core.Kind   { return core.KindEmbedder }

// GroqEmbedderOptions is an alias for OpenAIEmbedderOptions, but for the Groq provider.
type GroqEmbedderOptions struct {
	OpenAIEmbedderOptions
}

func (o *GroqEmbedderOptions) ProviderName() string { return "groq" }
func (o *GroqEmbedderOptions) ProviderKind() core.Kind   { return core.KindEmbedder }
