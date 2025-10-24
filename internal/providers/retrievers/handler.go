package retrievers

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler is the component handler for Retrievers.
type Handler struct{}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindRetriever
}

// BuildComponent builds the Retriever component and assigns it to the resolved map.
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
		return nil, fmt.Errorf("invalid builder DI type for Retriever handler: got %T", builderDI)
	}

	deps := diapi.RetrieverDeps{}
	if typedCfg, ok := cfg.(diapi.EmbedderDep); ok {
		embedder, err := b.GetEmbedder(typedCfg.GetEmbedder())
		if err != nil {
			return nil, err
		}
		deps.Embedder = embedder
	}
	if typedCfg, ok := cfg.(diapi.VectorStoreDep); ok {
		vs, err := b.GetVectorStore(typedCfg.GetVectorStore())
		if err != nil {
			return nil, err
		}
		deps.VectorStore = vs
	}
	deps.RetrieverResolver = b

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for Retriever handler")
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindRetriever, name, err)
	}

	retriever, ok := built.(core.Retriever)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid Retriever", name)
	}
	resolved.Retrievers[name] = retriever
	return core.NopCloser, nil
}
