// Package diapi provides the type-safe dependency injection contracts for Manglekit.
package diapi

import (
	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// Builder defines the dependency injection interface for the builder.
// It provides methods for handlers to look up already-built components by name.
type Builder interface {
	GetEmbedder(name string) (ai.Embedder, error)
	GetLLMClient(name string) (core.LLMClient, error)
	GetVectorStore(name string) (core.VectorStore, error)
	GetRetriever(name string) (core.Retriever, error)
	GetReranker(name string) (core.Reranker, error)
	GetStateProvider(name string) (core.StateProvider, error)
	GetRuleSet(name string) (core.RuleSet, error)
	Genkit() *genkit.Genkit
}

// APIKeyProvider is an interface for provider options that expose an API key.
type APIKeyProvider interface {
	GetAPIKey() string
}

// BaseURLProvider is an interface for provider options that expose a base URL.
type BaseURLProvider interface {
	GetBaseURL() string
}

// EmbedderDep is an interface for components that depend on an `ai.Embedder`.
type EmbedderDep interface {
	GetEmbedder() string
}

// VectorStoreDep is an interface for components that depend on a `core.VectorStore`.
type VectorStoreDep interface {
	GetVectorStore() string
}

// SubRetrieversDep is an interface for components that depend on a list of
// sub-retrievers, identified by their names.
type SubRetrieversDep interface {
	GetSubRetrievers() []string
}

// ProviderWithOptions is an interface for provider options that expose the underlying options.
type ProviderWithOptions interface {
	GetProviderOptions() any
}

// RetrieverDeps provides dependencies for a retriever that depends on other sub-retrievers.
type RetrieverDeps struct {
	SubRetrievers map[string]core.Retriever
}

// DenseRetrieverDeps provides dependencies for a dense retriever.
type DenseRetrieverDeps struct {
	Embedder    ai.Embedder
	VectorStore core.VectorStore
}

// LLMDeps provides all possible dependencies that an LLM factory might need.
type LLMDeps struct {
	Genkit *genkit.Genkit
	Client any // Provider-specific client, e.g., *openai.Client
}

// EmbedderDeps provides all possible dependencies that an embedder factory might need.
type EmbedderDeps struct {
	Genkit *genkit.Genkit
	Client any // Provider-specific client, e.g., *genai.EmbedderClient
}

// VectorStoreDeps provides all possible dependencies that a vector store factory might need.
type VectorStoreDeps struct {
	Embedder ai.Embedder
}

// RerankerDeps provides all possible dependencies that a reranker factory might need.
type RerankerDeps struct {
	Embedder ai.Embedder
}

// StateProviderDeps provides dependencies for a state provider factory.
// Currently, it has no dependencies.
type StateProviderDeps struct{}

// RuleSetDeps provides dependencies for a ruleset factory.
// Currently, it has no dependencies.
type RuleSetDeps struct{}

// NoopDeps is a placeholder for factories that have no dependencies.
type NoopDeps struct{}
