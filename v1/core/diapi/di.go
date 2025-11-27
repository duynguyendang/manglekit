// Package diapi provides the type-safe dependency injection contracts for Manglekit.
package diapi

import (
	"context"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// Builder defines the dependency injection interface for the builder.
// It provides methods for handlers to look up already-built components by name.
type Builder interface {
	GetEmbedder(name string) (ai.Embedder, error)
	GetLLMClient(name string) (core.LLMClient, error)
	GetRetriever(name string) (core.Retriever, error)
	GetReranker(name string) (core.Reranker, error)
	GetStateProvider(name string) (core.StateProvider, error)
	GetRuleSet(name string) (core.RuleSet, error)
	GetSchemaParser(name string) (core.SchemaParser, error)
	GetReasoner(name string) (core.Reasoner, error)
	GetPlanner(name string) (core.Planner, error)
	Genkit() *genkit.Genkit
	GetCoreDeps() CoreDeps
	Registry() any // This is not ideal, but it's the only way to get the registry to the handler.

	SetRetriever(name string, retriever core.Retriever) error
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

// SubRetrieversDep is an interface for components that depend on a list of
// sub-retrievers, identified by their names.
type SubRetrieversDep interface {
	GetSubRetrievers() []string
}

// ProviderWithOptions is an interface for provider options that expose the underlying options.
type ProviderWithOptions interface {
	GetProviderOptions() any
}

// SkipModelCheckProvider is an interface for provider options that can skip model validation.
type SkipModelCheckProvider interface {
	ShouldSkipModelCheck() bool
}

// RetrieverDeps provides dependencies for a retriever that depends on other sub-retrievers.
type RetrieverDeps struct {
	CoreDeps
	SubRetrievers map[string]core.Retriever
}

// LLMDeps provides all possible dependencies that an LLM factory might need.
type LLMDeps struct {
	CoreDeps
	Genkit *genkit.Genkit
	Client any // Provider-specific client, e.g., *openai.Client
}

// EmbedderDeps provides all possible dependencies that an embedder factory might need.
type EmbedderDeps struct {
	CoreDeps
	Genkit *genkit.Genkit
	Client any // Provider-specific client, e.g., *genai.EmbedderClient
}

// RerankerDeps provides all possible dependencies that a reranker factory might need.
type RerankerDeps struct {
	CoreDeps
	Embedder ai.Embedder
}

// StateProviderDeps provides dependencies for a state provider factory.
// Currently, it has no dependencies.
type StateProviderDeps struct {
	CoreDeps
}

// RuleSetDeps provides dependencies for a ruleset factory.
type RuleSetDeps struct {
	CoreDeps
	Registry any
}

// NoopDeps is a placeholder for factories that have no dependencies.
type NoopDeps struct {
	CoreDeps
}

// GenkitRetrieverDeps provides dependencies for Genkit-based retrievers.
// It includes the embedder from Manglekit's registry to be used by Genkit providers.
type GenkitRetrieverDeps struct {
	CoreDeps
	Embedder ai.Embedder
}

// SandwichDeps provides all dependencies required by the sandwich orchestrator.
type SandwichDeps struct {
	CoreDeps
	Action        core.Action
	SubActions    map[string]core.Action
	Reranker      core.Reranker
	LLM           core.LLMClient
	StateProvider core.StateProvider
	RuleSet       core.RuleSet
}

// DeclarativeOrchestratorDeps provides dependencies for the declarative orchestrator.
type DeclarativeOrchestratorDeps struct {
	CoreDeps
	StateProvider core.StateProvider
	Tools         map[string]core.Tool
}

// PlannerDeps provides the dependencies required by a planner component.
// It includes access to reasoning and execution capabilities.
type PlannerDeps struct {
	CoreDeps
	Tools     map[string]core.Tool
	Reasoners map[string]core.Reasoner
}

// CoreDeps provides access to core framework services.
type CoreDeps struct {
	Obs core.Observability
}

// DependencyResolver is an interface for resolving component-specific dependencies
// based on the provider options. This pattern allows handlers to be extended
// without modifying their code—new providers simply register their own resolvers.
//
// Implementations of this interface are registered per component Kind and can
// be used to build dependencies for specific provider types (e.g., hybrid vs. dense
// retrievers). This eliminates the need for type-switch statements in handlers.
type DependencyResolver interface {
	// Matches returns true if this resolver can handle the given provider options.
	Matches(opts any) bool
	// Resolve builds the dependencies for the provider options.
	// The builderDI parameter is the diapi.Builder, and cfg contains the raw options.
	// Returns the built dependencies (e.g., RetrieverDeps, DenseRetrieverDeps) or error.
	Resolve(ctx context.Context, builderDI any, cfg any) (any, error)
}
