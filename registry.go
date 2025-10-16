package manglekit

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
)

// Dependency map for passing components.
type FactoryDeps map[string]any

// Define specific, type-safe factories.
type RetrieverFactory func(ctx context.Context, opts any, deps FactoryDeps) (retrieve.Retriever, error)
type LLMFactory func(ctx context.Context, opts any, deps FactoryDeps) (llm.Client, error)
type EmbedderFactory func(ctx context.Context, opts any, deps FactoryDeps) (ai.Embedder, error)
type RerankerFactory func(ctx context.Context, opts any, deps FactoryDeps) (rerank.Reranker, error)
type VectorStoreFactory func(ctx context.Context, opts any, deps FactoryDeps) (core.VectorStore, error)
type RuleSetFactory func(ctx context.Context, opts any, deps FactoryDeps) (core.RuleSet, error)
type StateProviderFactory func(ctx context.Context, options any, deps FactoryDeps) (core.StateProvider, error)
type SchemaParserFactory func(ctx context.Context, options any, deps FactoryDeps) (core.SchemaParser, error)
type FactConverterFactory func(ctx context.Context, options any, deps FactoryDeps) (core.FactConverter, error)

// OrchestratorFactory defines the signature for creating an orchestrator instance.
// It receives the fully built Options object containing all necessary components.
type OrchestratorFactory func(opts core.Options) (core.Orchestrator, error)

// Registry is the central store for all registered component constructors.
// It is the primary mechanism for extending MangleKit with new providers.
type Registry struct {
	// Strongly-typed factory maps
	Retrievers            map[string]RetrieverFactory
	LLMs                  map[string]LLMFactory
	Embedders             map[string]EmbedderFactory
	Rerankers             map[string]RerankerFactory
	VectorStores          map[string]VectorStoreFactory
	RuleSets              map[string]RuleSetFactory
	StateProviders        map[string]StateProviderFactory
	SchemaParsers         map[string]SchemaParserFactory
	FactConverters        map[string]FactConverterFactory
	OrchestratorFactories map[string]OrchestratorFactory

	// Options holds registered options types for components.
	Options map[string]reflect.Type
	// ClientFactories holds registered client factory functions.
	ClientFactories map[string]any

	// Type maps are now owned by the registry instance.
	nameToOptionsType map[string]reflect.Type
	optionsTypeToName map[reflect.Type]string
}

// NewRegistry creates and returns a new, fully initialized Registry struct.
func NewRegistry() *Registry {
	return &Registry{
		Retrievers:            make(map[string]RetrieverFactory),
		LLMs:                  make(map[string]LLMFactory),
		Embedders:             make(map[string]EmbedderFactory),
		Rerankers:             make(map[string]RerankerFactory),
		StateProviders:        make(map[string]StateProviderFactory),
		VectorStores:          make(map[string]VectorStoreFactory),
		RuleSets:              make(map[string]RuleSetFactory),
		SchemaParsers:         make(map[string]SchemaParserFactory),
		FactConverters:        make(map[string]FactConverterFactory),
		OrchestratorFactories: make(map[string]OrchestratorFactory),
		Options:               make(map[string]reflect.Type),
		ClientFactories:       make(map[string]any),

		// Initialize the instance-owned type maps.
		nameToOptionsType: make(map[string]reflect.Type),
		optionsTypeToName: make(map[reflect.Type]string),
	}
}

// RegisterOptions registers the **pointer-to-struct** options type for a provider.
// This is now a method on *Registry.
func (r *Registry) RegisterOptions(providerName string, typedNilPtr any) error {
	t := reflect.TypeOf(typedNilPtr)
	if t == nil {
		return fmt.Errorf("RegisterOptions %q: got nil; pass a typed nil pointer like (*T)(nil)", providerName)
	}
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("RegisterOptions %q: expected pointer to struct, got %v", providerName, t)
	}

	// These maps are guaranteed to be initialized by NewRegistry()
	r.nameToOptionsType[providerName] = t
	r.optionsTypeToName[t] = providerName
	return nil
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
func (r *Registry) RegisterSchemaParser(name string, c SchemaParserFactory) { r.SchemaParsers[name] = c }

// RegisterVectorStore adds a vector store constructor to the registry.
func (r *Registry) RegisterVectorStore(name string, c VectorStoreFactory) { r.VectorStores[name] = c }

// RegisterFactConverter adds a fact converter constructor to the registry.
func (r *Registry) RegisterFactConverter(name string, c FactConverterFactory) { r.FactConverters[name] = c }

// RegisterStateProvider adds a state provider constructor to the registry.
func (r *Registry) RegisterStateProvider(name string, c StateProviderFactory) {
	r.StateProviders[name] = c
}

// RegisterClientFactory adds a client factory function to the registry.
func (r *Registry) RegisterClientFactory(name string, c ClientFactory) {
	r.ClientFactories[name] = c
}

// RegisterOrchestrator adds an orchestrator factory to the registry.
func (r *Registry) RegisterOrchestrator(name string, c OrchestratorFactory) {
	r.OrchestratorFactories[name] = c
}