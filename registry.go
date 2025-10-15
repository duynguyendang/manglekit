package manglekit

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
)

// Registry is the central store for all registered component constructors.
// It is the primary mechanism for extending MangleKit with new providers.
type Registry struct {
	Retrievers     map[string]RetrieverFactory
	LLMs           map[string]LLMFactory
	Embedders      map[string]EmbedderFactory
	Rerankers      map[string]RerankerFactory
	StateProviders map[string]StateProviderFactory
	VectorStores   map[string]VectorStoreFactory
	RuleSets       map[string]RuleSetFactory
	SchemaParsers  map[string]ComponentFactory // Keep generic for now
	// Component holds registered generic component constructors, such as `core.VectorStore`.
	Component map[string]ComponentFactory
	// Options holds registered options types for components.
	Options map[string]reflect.Type
	// ClientFactories holds registered client factory functions.
	ClientFactories map[string]any
}

// NewRegistry creates and returns a new, fully initialized Registry struct.
func NewRegistry() *Registry {
	return &Registry{
		Retrievers:     make(map[string]RetrieverFactory),
		LLMs:           make(map[string]LLMFactory),
		Embedders:      make(map[string]EmbedderFactory),
		Rerankers:      make(map[string]RerankerFactory),
		StateProviders: make(map[string]StateProviderFactory),
		VectorStores:   make(map[string]VectorStoreFactory),
		RuleSets:       make(map[string]RuleSetFactory),
		SchemaParsers:  make(map[string]ComponentFactory),
		Component:      make(map[string]ComponentFactory),
		Options:        make(map[string]reflect.Type),
		ClientFactories: make(map[string]any),
	}
}

// ClientFactory defines the contract for a function that creates a shared client
// for a provider family (e.g., Google, OpenAI). It takes the global `*Config`
// object and is responsible for extracting its own provider-specific settings.
// It returns the initialized client, a `core.ResourceCloser` for graceful shutdown,
// and an error.
type ClientFactory func(ctx context.Context, cfg *Config) (client any, closer core.ResourceCloser, err error)

// ToolFactory is a specialized factory for declarative tools, which may have
// a different dependency injection mechanism.
type ToolFactory func(options any, deps FactoryDeps) (any, error)

// Get retrieves a constructor function from the specified registry map. It is a
// helper used by the Builder to find and instantiate components.
//
// registry is the specific component map to search in (e.g., `r.Component`).
// name is the string name under which the component was registered.
// It returns the constructor as `any` or an error if the name is not found in the map.
func Get[T any](registry map[string]T, name string) (T, error) {
	c, ok := registry[name]
	if !ok {
		var zero T
		return zero, fmt.Errorf("unknown component: %s", name)
	}
	return c, nil
}

// RegisterRetriever adds a retriever constructor to the registry.
func (r *Registry) RegisterRetriever(name string, c RetrieverFactory) { r.Retrievers[name] = c }

// RegisterReranker adds a reranker constructor to the registry.
func (r *Registry) RegisterReranker(name string, c RerankerFactory) { r.Rerankers[name] = c }

// RegisterRuleSet adds a rules engine constructor to the registry.
func (r *Registry) RegisterRuleSet(name string, c RuleSetFactory) { r.RuleSets[name] = c }

// RegisterLLM adds a language model constructor to the registry.
func (r *Registry) RegisterLLM(name string, c LLMFactory) { r.LLMs[name] = c }

// RegisterEmbedder adds a text embedder constructor to the registry.
func (r *Registry) RegisterEmbedder(name string, c EmbedderFactory) { r.Embedders[name] = c }

// RegisterSchemaParser adds a schema parser constructor to the registry.
func (r *Registry) RegisterSchemaParser(name string, c ComponentFactory) { r.SchemaParsers[name] = c }

// RegisterVectorStore adds a vector store constructor to the registry.
func (r *Registry) RegisterVectorStore(name string, c VectorStoreFactory) { r.VectorStores[name] = c }

// Register adds a generic component constructor (e.g., a vector store) to the registry.
func (r *Registry) Register(name string, c ComponentFactory) { r.Component[name] = c }

// RegisterStateProvider adds a state provider constructor to the registry.
func (r *Registry) RegisterStateProvider(name string, c StateProviderFactory) {
	r.StateProviders[name] = c
}

// RegisterClientFactory adds a client factory function to the registry.
func (r *Registry) RegisterClientFactory(name string, c ClientFactory) {
	r.ClientFactories[name] = c
}
