package genkitvectorstore

import "github.com/duynguyendang/manglekit/core"

// GenkitVectorStoreOptions provides universal configuration for ANY Genkit vector store provider.
// This single options struct replaces provider-specific options (Pinecone, Chroma, Weaviate, etc.)
// by allowing users to specify the provider in configuration rather than hard-coding it.
//
// Example usage in config.yaml:
//
//components:
//  - name: my-vectorstore
//    kind: vector_store
//    type: genkit-vectorstore
//    params:
//      provider: pinecone
//      index: documents
//      namespace: prod
//      apiKey: "${PINECONE_API_KEY}"
type GenkitVectorStoreOptions struct {
// Provider is the Genkit vector store provider name.
// Supported values: "pinecone", "chroma", "weaviate", "qdrant", "milvus", etc.
// Any Genkit-compatible vector store plugin can be specified here.
Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`

// Index is the index or collection name within the vector store.
// Examples:
//   - Pinecone: "documents", "embeddings"
//   - Chroma: "collection_name"
//   - Weaviate: "Document", "Product"
//   - Qdrant: "documents"
Index string `json:"index,omitempty" yaml:"index,omitempty"`

// Namespace is an optional namespace or partition within the vector store.
// Useful for multi-tenancy or logical separation of data.
Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

// APIKey is the API authentication key for providers that require it.
APIKey string `json:"apiKey,omitempty" yaml:"api_key,omitempty"`

// BaseURL is an optional base URL override for API endpoints.
BaseURL string `json:"baseUrl,omitempty" yaml:"base_url,omitempty"`

// Dimensions specifies the vector dimensionality expected by the vector store.
Dimensions int `json:"dimensions,omitempty" yaml:"dimensions,omitempty"`

// AdditionalConfig is a catch-all for provider-specific configuration parameters
AdditionalConfig map[string]any `json:"additionalConfig,omitempty" yaml:"additional_config,omitempty"`
}

// ProviderName returns the provider name ("genkit-vectorstore").
func (o *GenkitVectorStoreOptions) ProviderName() string {
return "genkit-vectorstore"
}

// ProviderKind returns the component kind.
func (o *GenkitVectorStoreOptions) ProviderKind() core.Kind {
return core.KindVectorStore
}
