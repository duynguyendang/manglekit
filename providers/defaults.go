package providers

import "github.com/duynguyendang/manglekit"

// RegisterDefaults registers all standard Manglekit providers.
func RegisterDefaults(r *manglekit.Registry) {
	NewSet().
		WithGoogleLLM().
		WithOpenAI().
		WithGoogleEmbedder().
		WithOpenAIEmbedder().
		WithInMemoryRetriever().
		WithBM25Retriever().
		WithDenseRetriever().
		WithHybridRetriever().
		WithCosineReranker().
		WithMangleRules().
		WithJSONSchemaParser().
		WithRDFSchemaParser().
		ApplyTo(r)
}
