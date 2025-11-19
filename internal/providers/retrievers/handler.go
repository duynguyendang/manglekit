package retrievers

import (
	"context"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler is the component handler for Retrievers.
type Handler struct {
	resolverOnce sync.Once
	resolver     *diapi.ResolverRegistry
}

// NewHandler returns a new ComponentHandler for Retrievers.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindRetriever
}

// ensureResolverInitialized initializes the resolver registry on first use.
// This is done lazily to avoid circular dependency issues during init.
func (h *Handler) ensureResolverInitialized() {
	h.resolverOnce.Do(func() {
		h.resolver = diapi.NewResolverRegistry()
		// Register resolvers in priority order.
		// SubRetrieverResolver must come before GenkitRetrieverResolver.
		// GenkitRetrieverResolver must come before NoopRetrieverResolver.
		h.resolver.Register(core.KindRetriever, diapi.NewSubRetrieverResolver(nil))
		h.resolver.Register(core.KindRetriever, diapi.NewGenkitRetrieverResolver())
		h.resolver.Register(core.KindRetriever, diapi.NewNoopRetrieverResolver())
	})
}

// BuildComponent builds the Retriever component by delegating to registered resolvers.
// This design is extensible—new retriever types can be supported by registering
// new resolvers without modifying this handler.
func (h *Handler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	h.ensureResolverInitialized()

	b, ok := builderDI.(diapi.Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder DI type for Retriever handler: got %T", builderDI)
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for Retriever handler: got %T", factory)
	}

	providerWithOptions, ok := cfg.(diapi.ProviderWithOptions)
	if !ok {
		return nil, fmt.Errorf("provider options for %s do not implement diapi.ProviderWithOptions", name)
	}
	opts := providerWithOptions.GetProviderOptions()

	// Resolve dependencies using the registry.
	deps, err := h.resolver.Resolve(ctx, core.KindRetriever, builderDI, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies for retriever '%s': %w", name, err)
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindRetriever, name, err)
	}

	retriever, ok := built.(core.Retriever)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid Retriever", name)
	}
	if err := b.SetRetriever(name, retriever); err != nil {
		return nil, fmt.Errorf("failed to set retriever '%s': %w", name, err)
	}
	return core.NopCloser, nil
}
