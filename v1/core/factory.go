package core

import "context"

// Factory is an interface for a component factory.
type Factory interface {
	Build(ctx context.Context, deps any, cfg any) (any, error)
}

// GenericFactory is a type-erased wrapper around a strongly-typed provider factory.
// It allows the registry to store factories for different component types in a
// single map while preserving type safety internally. The `Build` method uses
// `any` for its parameters, but the concrete implementation (`typedFactory`)
// immediately casts them back to their compile-time types.
type GenericFactory interface {
	Kind() Kind
	Name() string
	Factory
}

// typedFactory is the concrete implementation of GenericFactory. It holds the
// actual, strongly-typed factory function `fn`. This generic struct ensures that
// within a provider's ecosystem, all interactions (factory definition, dependency
// injection, configuration) are type-safe from end to end.
type typedFactory[T any, D any, O ProviderOptions] struct {
	kind Kind
	name string
	fn   func(ctx context.Context, deps D, cfg O) (T, error)
}

// NewTypedFactory creates a new type-safe, generic factory.
func NewTypedFactory[T any, D any, O ProviderOptions](
	kind Kind,
	name string,
	fn func(ctx context.Context, deps D, cfg O) (T, error),
) GenericFactory {
	return typedFactory[T, D, O]{
		kind: kind,
		name: name,
		fn:   fn,
	}
}

// Kind returns the component kind (e.g., "llm", "retriever").
func (f typedFactory[T, D, O]) Kind() Kind { return f.kind }

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
