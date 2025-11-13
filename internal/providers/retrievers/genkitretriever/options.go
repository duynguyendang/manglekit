package genkitretriever

import "github.com/duynguyendang/manglekit/core"

// GenkitRetrieverOptions provides universal configuration for ANY Genkit retriever provider.
// This single options struct allows users to specify the provider in configuration
// rather than hard-coding it, supporting Genkit vector store plugins like Pinecone, Chroma,
// Weaviate, Qdrant, Milvus, and others.
//
// This replaces the old "dense" retriever approach, which was merely an orchestrator
// combining an embedder + vector store. Genkit retrievers already do this internally,
// so we wrap them directly for a simpler, cleaner architecture.
//
// Example usage in config.yaml:
//
//	components:
//	  - name: my-retriever
//	    kind: retriever
//	    type: genkit-retriever
//	    params:
//	      provider: pinecone
//	      model: text-embedding-3-small
//	      apiKey: "${PINECONE_API_KEY}"
//	      projectId: "my-project"
//	      indexName: "my-index"
type GenkitRetrieverOptions struct {
	// Provider is the Genkit retriever provider name.
	// Supported values: "pinecone", "chroma", "weaviate", "qdrant", "milvus", etc.
	// Any Genkit-compatible retriever plugin can be specified here.
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`

	// Model is the embedding model identifier for the provider.
	// Examples:
	//   - "text-embedding-3-small" (OpenAI)
	//   - "embedding-001" (Google)
	//   - "all-MiniLM-L6-v2" (local embeddings)
	Model string `json:"model,omitempty" yaml:"model,omitempty"`

	// APIKey is the API authentication key for providers that require it.
	// If not provided here, providers typically look for environment variables
	// (e.g., PINECONE_API_KEY, CHROMA_API_KEY).
	APIKey string `json:"apiKey,omitempty" yaml:"api_key,omitempty"`

	// ProjectID is the project identifier for some providers (e.g., Pinecone).
	// This identifies the project or namespace within the vector store service.
	ProjectID string `json:"projectId,omitempty" yaml:"project_id,omitempty"`

	// IndexName or CollectionName is the index/collection name within the vector store.
	// Examples:
	//   - Pinecone: "my-index"
	//   - Chroma: "my-collection"
	//   - Weaviate: "MyCollection"
	IndexName string `json:"indexName,omitempty" yaml:"index_name,omitempty"`

	// Endpoint is the connection endpoint for the vector store (URL or address).
	// Optional; some providers derive this from environment or default endpoints.
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`

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

func (o *GenkitRetrieverOptions) ProviderName() string    { return "genkit-retriever" }
func (o *GenkitRetrieverOptions) ProviderKind() core.Kind { return core.KindRetriever }

// GetAPIKey provides generic API key access (implements diapi.APIKeyProvider interface).
func (o *GenkitRetrieverOptions) GetAPIKey() string { return o.APIKey }

// ShouldSkipModelCheck implements the diapi.SkipModelCheckProvider interface.
func (o *GenkitRetrieverOptions) ShouldSkipModelCheck() bool { return o.SkipModelCheck }
