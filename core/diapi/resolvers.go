package diapi

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// ResolverRegistry manages component-specific dependency resolvers.
// This registry enables handlers to be extended without modifying their code.
type ResolverRegistry struct {
	resolvers map[core.Kind][]DependencyResolver
}

// NewResolverRegistry creates a new resolver registry.
func NewResolverRegistry() *ResolverRegistry {
	return &ResolverRegistry{
		resolvers: make(map[core.Kind][]DependencyResolver),
	}
}

// Register adds a resolver for a specific component kind.
// Resolvers are tried in registration order until one matches.
func (r *ResolverRegistry) Register(kind core.Kind, resolver DependencyResolver) {
	r.resolvers[kind] = append(r.resolvers[kind], resolver)
}

// Resolve attempts to resolve dependencies using registered resolvers.
// It tries each resolver in order until one matches the provider options.
// Returns an error if no resolver matches or if resolution fails.
func (r *ResolverRegistry) Resolve(ctx context.Context, kind core.Kind, builderDI any, cfg any) (any, error) {
	resolvers, ok := r.resolvers[kind]
	if !ok {
		return nil, fmt.Errorf("no resolvers registered for component kind %s", kind)
	}

	for _, resolver := range resolvers {
		if resolver.Matches(cfg) {
			return resolver.Resolve(ctx, builderDI, cfg)
		}
	}

	return nil, fmt.Errorf("no resolver matched provider options %T for component kind %s", cfg, kind)
}

// ============================================================================
// Built-in Resolvers for Retrievers
// ============================================================================

// SubRetrieverResolver resolves dependencies for retriever types that use
// sub-retrievers (e.g., hybrid retriever).
type SubRetrieverResolver struct {
	builder Builder
}

// NewSubRetrieverResolver creates a new resolver for sub-retriever types.
func NewSubRetrieverResolver(builder Builder) *SubRetrieverResolver {
	return &SubRetrieverResolver{builder: builder}
}

// Matches returns true if the options implement SubRetrieversDep.
func (r *SubRetrieverResolver) Matches(opts any) bool {
	_, ok := opts.(SubRetrieversDep)
	return ok
}

// Resolve builds RetrieverDeps by looking up each sub-retriever.
func (r *SubRetrieverResolver) Resolve(ctx context.Context, builderDI any, cfg any) (any, error) {
	b, ok := builderDI.(Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder DI type: got %T", builderDI)
	}

	typedOpts, ok := cfg.(SubRetrieversDep)
	if !ok {
		return nil, fmt.Errorf("options do not implement SubRetrieversDep: got %T", cfg)
	}

	hybridDeps := RetrieverDeps{
		CoreDeps:      b.GetCoreDeps(),
		SubRetrievers: make(map[string]core.Retriever),
	}

	for _, subName := range typedOpts.GetSubRetrievers() {
		r, err := b.GetRetriever(subName)
		if err != nil {
			return nil, fmt.Errorf("failed to get sub-retriever '%s': %w", subName, err)
		}
		hybridDeps.SubRetrievers[subName] = r
	}

	return hybridDeps, nil
}

// DenseRetrieverResolver resolves dependencies for dense (embedding-based)
// retriever types.
type DenseRetrieverResolver struct{}

// NewDenseRetrieverResolver creates a new resolver for dense retriever types.
func NewDenseRetrieverResolver() *DenseRetrieverResolver {
	return &DenseRetrieverResolver{}
}

// Matches returns true if the options implement both EmbedderDep and VectorStoreDep.
func (r *DenseRetrieverResolver) Matches(opts any) bool {
	_, embedderOk := opts.(EmbedderDep)
	_, vectorStoreOk := opts.(VectorStoreDep)
	return embedderOk && vectorStoreOk
}

// Resolve builds DenseRetrieverDeps by looking up the embedder and vector store.
func (r *DenseRetrieverResolver) Resolve(ctx context.Context, builderDI any, cfg any) (any, error) {
	b, ok := builderDI.(Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder DI type: got %T", builderDI)
	}

	embedderDep, ok := cfg.(EmbedderDep)
	if !ok {
		return nil, fmt.Errorf("options do not implement EmbedderDep: got %T", cfg)
	}

	vectorStoreDep, ok := cfg.(VectorStoreDep)
	if !ok {
		return nil, fmt.Errorf("options do not implement VectorStoreDep: got %T", cfg)
	}

	embedder, err := b.GetEmbedder(embedderDep.GetEmbedder())
	if err != nil {
		return nil, fmt.Errorf("failed to get embedder: %w", err)
	}

	vs, err := b.GetVectorStore(vectorStoreDep.GetVectorStore())
	if err != nil {
		return nil, fmt.Errorf("failed to get vector store: %w", err)
	}

	return DenseRetrieverDeps{
		CoreDeps:    b.GetCoreDeps(),
		Embedder:    embedder,
		VectorStore: vs,
	}, nil
}

// NoopRetrieverResolver resolves dependencies for retrievers that have no special
// dependencies (beyond CoreDeps).
type NoopRetrieverResolver struct{}

// NewNoopRetrieverResolver creates a new resolver for noop retriever types.
func NewNoopRetrieverResolver() *NoopRetrieverResolver {
	return &NoopRetrieverResolver{}
}

// Matches always returns true (catch-all for any other retriever type).
func (r *NoopRetrieverResolver) Matches(opts any) bool {
	return true
}

// Resolve builds NoopDeps.
func (r *NoopRetrieverResolver) Resolve(ctx context.Context, builderDI any, cfg any) (any, error) {
	b, ok := builderDI.(Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder DI type: got %T", builderDI)
	}

	return NoopDeps{CoreDeps: b.GetCoreDeps()}, nil
}
