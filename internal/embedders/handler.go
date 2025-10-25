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

// BuildComponent builds the Embedder component.
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
		return nil, fmt.Errorf("invalid builder DI type for Embedder handler: got %T", builderDI)
	}

	deps := diapi.EmbedderDeps{
		Genkit: b.Genkit(),
	}

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
	return core.NopCloser, nil
}
