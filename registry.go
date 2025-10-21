package manglekit

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
)

// GenericFactory is a type-erased wrapper around a strongly-typed provider factory.
// It allows the registry to store factories for different component types in a
// single map while preserving type safety internally. The `Build` method uses
// `any` for its parameters, but the concrete implementation (`typedFactory`)
// immediately casts them back to their compile-time types.
type GenericFactory interface {
	Kind() core.Kind
	Name() string
	Build(ctx context.Context, deps any, cfg any) (any, error)
}

// typedFactory is the concrete implementation of GenericFactory. It holds the
// actual, strongly-typed factory function `fn`. This generic struct ensures that
// within a provider's ecosystem, all interactions (factory definition, dependency
// injection, configuration) are type-safe from end to end.
type typedFactory[T any, D any, O core.ProviderOptions] struct {
	kind core.Kind
	name string
	fn   func(ctx context.Context, deps D, cfg O) (T, error)
}

// Kind returns the component kind (e.g., "llm", "retriever").
func (f typedFactory[T, D, O]) Kind() core.Kind { return f.kind }

// Name returns the provider's unique name (e.g., "openai-chat").
func (f typedFactory[T, D, O]) Name() string { return f.name }

// Build executes the wrapped factory function. It performs the crucial type
// assertions that bridge the gap from the type-erased `any` parameters back
// to the concrete types `D` (dependencies) and `O` (options) required by the
// factory. This is the only place where such an assertion is needed for factories.
func (f typedFactory[T, D, O]) Build(ctx context.Context, deps any, cfg any) (any, error) {
	// Handle cases where dependencies or config might be nil.
	var d D
	if deps != nil {
		d = deps.(D)
	}

	var o O
	if cfg != nil {
		o = cfg.(O)
	}

	return f.fn(ctx, d, o)
}

// Registry is the central store for all registered component constructors.
// It has been refactored to use a single, generic, and type-safe mechanism.
type Registry struct {
	factories         map[core.Kind]map[string]GenericFactory
	OptionsTypeToName map[reflect.Type]string
	OptionsTypeToKind map[reflect.Type]core.Kind
}

// NewRegistry creates and returns a new, fully initialized Registry struct.
func NewRegistry() *Registry {
	return &Registry{
		factories:         make(map[core.Kind]map[string]GenericFactory),
		OptionsTypeToName: make(map[reflect.Type]string),
		OptionsTypeToKind: make(map[reflect.Type]core.Kind),
	}
}

// Register is the new generic, type-safe function for registering any provider.
// It infers the provider's name and kind directly from its Options type,
// which must implement `core.ProviderOptions`. This eliminates string literals
// and the possibility of mis-categorizing a provider.
//
// Example:
//
//	Register(
//	  registry,
//	  llm.OpenAIOptions{}, // The options struct itself carries the metadata.
//	  func(ctx context.Context, deps D, cfg llm.OpenAIOptions) (llm.Client, error) {
//	    // ... factory logic
//	  },
//	)
func Register[T any, D any, O core.ProviderOptions](
	r *Registry,
	optsSample O,
	fn func(ctx context.Context, deps D, cfg O) (T, error),
) {
	kind := optsSample.ProviderKind()
	name := optsSample.ProviderName()

	if _, ok := r.factories[kind]; !ok {
		r.factories[kind] = make(map[string]GenericFactory)
	}

	factory := typedFactory[T, D, O]{
		kind: kind,
		name: name,
		fn:   fn,
	}
	r.factories[kind][name] = factory

	t := reflect.TypeOf(optsSample)
	r.OptionsTypeToName[t] = name
	r.OptionsTypeToKind[t] = kind
}

// Get retrieves a generic factory from the registry by its kind and name.
func (r *Registry) Get(kind core.Kind, name string) (GenericFactory, error) {
	if kindMap, ok := r.factories[kind]; ok {
		if factory, ok := kindMap[name]; ok {
			return factory, nil
		}
	}
	return nil, fmt.Errorf("unknown %s provider: %s", kind, name)
}

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