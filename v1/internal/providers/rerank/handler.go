package rerank

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
)

// Handler is the component handler for Rerankers.
type Handler struct{}

// NewHandler returns a new ComponentHandler for Rerankers.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindReranker
}

// BuildComponent builds the Reranker component and assigns it to the resolved map.
func (h *Handler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	b, ok := builderDI.(diapi.Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder DI type for Reranker handler: got %T", builderDI)
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for Reranker handler: got %T", factory)
	}

	type embedderProvider interface {
		GetEmbedder() string
	}

	provider, ok := cfg.(embedderProvider)
	if !ok {
		return nil, fmt.Errorf("reranker options %T must provide GetEmbedder() string", cfg)
	}
	embedderName := provider.GetEmbedder()
	embedder, err := b.GetEmbedder(embedderName)
	if err != nil {
		return nil, err
	}

	deps := diapi.RerankerDeps{
		CoreDeps: b.GetCoreDeps(),
		Embedder: embedder,
	}
	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindReranker, name, err)
	}

	reranker, ok := built.(core.Reranker)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid Reranker", name)
	}
	resolved.Rerankers[name] = reranker
	return core.NopCloser, nil
}
