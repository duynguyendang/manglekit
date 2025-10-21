package vectorstores

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler is the component handler for VectorStores.
type Handler struct{}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindVectorStore
}

// BuildComponent builds the VectorStore component and assigns it to the resolved map.
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
		return nil, fmt.Errorf("invalid builder DI type for VectorStore handler")
	}

	deps := diapi.VectorStoreDeps{}
	if typedCfg, ok := cfg.(diapi.EmbedderDep); ok {
		embedder, err := b.GetEmbedder(typedCfg.GetEmbedder())
		if err != nil {
			return nil, err
		}
		deps.Embedder = embedder
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for VectorStore handler")
	}
	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindVectorStore, name, err)
	}

	vectorStore, ok := built.(core.VectorStore)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid VectorStore", name)
	}
	resolved.VectorStores[name] = vectorStore

	if c, ok := built.(interface{ Close(context.Context) error }); ok {
		return c.Close, nil
	}
	return core.NopCloser, nil
}
