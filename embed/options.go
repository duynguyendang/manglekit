package embed

// GoogleEmbedderOptions provides typed configuration for Google embedding models,
// such as those available through the `generative-ai-go` library.
type GoogleEmbedderOptions struct {
	// Model is the identifier for the specific Google embedding model to be used,
	// for example, "embedding-001".
	Model string `json:"model,omitempty"`
}

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
