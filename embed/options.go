package embed

// GoogleEmbedderOptions provides typed configuration for Google embedding models.
type GoogleEmbedderOptions struct {
	// Model is the identifier for the specific Google model to be used
	// (e.g., "embedding-001").
	Model string
}

// OpenAIEmbedderOptions provides typed configuration for OpenAI embedding models.
type OpenAIEmbedderOptions struct {
	// Model is the identifier for the specific OpenAI model to be used
	// (e.g., "text-embedding-3-small").
	Model string
	// Dimensions is the desired dimensionality of the output vectors. This is
	// only supported by certain models.
	Dimensions int
}