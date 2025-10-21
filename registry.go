package manglekit

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
)

// Registry is the central store for all registered component constructors.
// It has been refactored to use a single, generic, and type-safe mechanism.
type Registry struct {
	factories         map[core.Kind]map[string]core.GenericFactory
	handlers          map[core.Kind]core.ComponentHandler
	OptionsTypeToName map[reflect.Type]string
	OptionsTypeToKind map[reflect.Type]core.Kind
}

// NewRegistry creates and returns a new, fully initialized Registry struct.
func NewRegistry() *Registry {
	return &Registry{
		factories:         make(map[core.Kind]map[string]core.GenericFactory),
		handlers:          make(map[core.Kind]core.ComponentHandler),
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
		r.factories[kind] = make(map[string]core.GenericFactory)
	}

	factory := core.NewTypedFactory[T, D, O](kind, name, fn)
	r.factories[kind][name] = factory

	t := reflect.TypeOf(optsSample)
	r.OptionsTypeToName[t] = name
	r.OptionsTypeToKind[t] = kind
}

// Get retrieves a generic factory from the registry by its kind and name.
func (r *Registry) Get(kind core.Kind, name string) (core.GenericFactory, error) {
	if kindMap, ok := r.factories[kind]; ok {
		if factory, ok := kindMap[name]; ok {
			return factory, nil
		}
	}
	return nil, fmt.Errorf("unknown %s provider: %s", kind, name)
}

// RegisterHandler registers a component handler for a specific kind.
func (r *Registry) RegisterHandler(handler core.ComponentHandler) {
	r.handlers[handler.Kind()] = handler
}

// GetHandler retrieves a component handler for a specific kind.
func (r *Registry) GetHandler(kind core.Kind) (core.ComponentHandler, error) {
	handler, ok := r.handlers[kind]
	if !ok {
		return nil, fmt.Errorf("no handler registered for kind %q", kind)
	}
	return handler, nil
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
