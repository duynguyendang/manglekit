package embedders

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
)

// Handler is the component handler for Embedders.
type Handler struct{}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindEmbedder
}

// BuildComponent builds the Embedder component and assigns it to the resolved map.
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
		return nil, fmt.Errorf("invalid builder DI type for Embedder handler")
	}

	deps := diapi.EmbedderDeps{Genkit: b.Genkit()}
	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for Embedder handler")
	}
	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindEmbedder, name, err)
	}

	embedder, ok := built.(ai.Embedder)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid Embedder", name)
	}
	resolved.Embedders[name] = embedder

	if c, ok := built.(interface{ Close(context.Context) error }); ok {
		return c.Close, nil
	}
	return core.NopCloser, nil
}
