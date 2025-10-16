// Package diapi provides the type-safe dependency injection contracts for Manglekit.
//
// This package is designed to be a low-level, neutral dependency that is imported
// by provider factories and the core builder. It MUST NOT import the builder,
// registry, or any other high-level packages to avoid circular dependencies.
//
// Its purpose is to define the "what" (the dependency contracts), while the
// builder's responsibility is to provide the "how" (the concrete implementations).
package diapi

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// BuildRetrieverFunc defines a capability for building a retriever instance by name.
// This is used by components like the hybrid retriever to dynamically construct
// their sub-components without depending on the entire builder.
type BuildRetrieverFunc func(ctx context.Context, name string, params map[string]any) (retrieve.Retriever, error)

// RetrieverDeps provides all possible dependencies that a retriever factory might need.
type RetrieverDeps struct {
	Embedder          ai.Embedder
	VectorStore       core.VectorStore
	BuildSubRetriever BuildRetrieverFunc // Capability, not a Builder reference.
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