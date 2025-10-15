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
	// Rules holds registered `core.RuleSet` constructors.
	Rules map[string]any
	// Component holds registered generic component constructors, such as `core.VectorStore`.
	Component map[string]ComponentFactory
	// Options holds registered options types for components.
	Options map[string]reflect.Type
	// StateProviders holds registered `core.StateProvider` constructors.
	StateProviders map[string]StateProviderFactory
	// ClientFactories holds registered client factory functions.
	ClientFactories map[string]any
}

// NewRegistry creates and returns a new, fully initialized Registry struct.
func NewRegistry() *Registry {
	return &Registry{
		Rules:           make(map[string]any),
		Component:       make(map[string]ComponentFactory),
		Options:         make(map[string]reflect.Type),
		StateProviders:  make(map[string]StateProviderFactory),
		ClientFactories: make(map[string]any),
	}
}

// ClientFactory defines the contract for a function that creates a shared client
// for a provider family (e.g., Google, OpenAI). It takes the global `*Config`
// object and is responsible for extracting its own provider-specific settings.
// It returns the initialized client, a `core.ResourceCloser` for graceful shutdown,
// and an error.
type ClientFactory func(ctx context.Context, cfg *Config) (client any, closer core.ResourceCloser, err error)

type FactoryDeps map[string]any

// StateProviderFactory defines the contract for a state provider constructor,
// which requires a `context.Context` for initialization.
type StateProviderFactory func(ctx context.Context, options any, deps FactoryDeps) (core.StateProvider, error)

// ComponentFactory is a generic function signature for component constructors.
// It accepts a provider-specific options struct (`options`) and a map of
// resolved dependencies (`deps`), returning the initialized component as `any`.
type ComponentFactory func(ctx context.Context, options any, deps FactoryDeps) (any, error)

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
func (r *Registry) RegisterRetriever(name string, c ComponentFactory) { r.Component[name] = c }

// RegisterReranker adds a reranker constructor to the registry.
func (r *Registry) RegisterReranker(name string, c ComponentFactory) { r.Component[name] = c }

// RegisterRules adds a rules engine constructor to the registry.
func (r *Registry) RegisterRules(name string, c any) { r.Rules[name] = c }

// RegisterLLM adds a language model constructor to the registry.
func (r *Registry) RegisterLLM(name string, c ComponentFactory) { r.Component[name] = c }

// RegisterEmbedder adds a text embedder constructor to the registry.
func (r *Registry) RegisterEmbedder(name string, c ComponentFactory) { r.Component[name] = c }

// RegisterSchemaParser adds a schema parser constructor to the registry.
func (r *Registry) RegisterSchemaParser(name string, c ComponentFactory) { r.Component[name] = c }

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
