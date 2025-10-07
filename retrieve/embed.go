package retrieve

import "context"

// Embedder abstracts the process of converting text into numerical vectors.
// Deprecated: This interface has been replaced by ai.Embedder from the Genkit
// framework. It is kept for backward compatibility but will be removed in a
// future version.
type Embedder interface {
	// EmbedTexts converts a slice of text strings into a slice of float64 vectors.
	//
	// ctx is the context for the operation.
	// texts is a slice of strings to be embedded.
	// It returns a slice of vectors or an error if the embedding process fails.
	EmbedTexts(ctx context.Context, texts []string) ([][]float64, error)
	// Dim returns the dimensionality of the embedding vectors produced by the model.
	Dim() int
}

// GoogleEmbedderOptions provides configuration for Google's embedding models.
// Deprecated: This has been moved to the embed package as embed.GoogleOptions.
type GoogleEmbedderOptions struct {
	// The name of the model to use, e.g., "embedding-001".
	Model string
}

// OpenAIEmbedderOptions provides configuration for OpenAI's embedding models.
// Deprecated: This has been moved to the embed package as embed.OpenAIOptions.
type OpenAIEmbedderOptions struct {
	// The name of the model to use, e.g., "text-embedding-3-small".
	Model string
	// The desired dimension of the output vectors.
	Dimensions int
}