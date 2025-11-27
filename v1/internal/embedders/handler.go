package embedders

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/firebase/genkit/go/ai"
)

// Handler is the component handler for Embedders.
type Handler struct{}

// NewHandler returns a new ComponentHandler for Embedders.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindEmbedder
}

// BuildComponent builds the Embedder component.
// Embedders are registered as thin factories that delegate to Genkit plugins.
// Supported embedders: google, openai
func (h *Handler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	// Skip model check if requested (for testing)
	if p, ok := cfg.(diapi.SkipModelCheckProvider); ok {
		if p.ShouldSkipModelCheck() {
			if deps, ok := builderDI.(diapi.Builder); ok {
				if deps.GetCoreDeps().Obs.Logger != nil {
					deps.GetCoreDeps().Obs.Logger.Debugf(
						"skipping embedder model check",
						"name", name,
					)
				}
			}
			return core.NopCloser, nil
		}
	}

	b, ok := builderDI.(diapi.Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder DI type for Embedder handler: got %T", builderDI)
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for Embedder handler: got %T", factory)
	}

	deps := diapi.EmbedderDeps{
		CoreDeps: b.GetCoreDeps(),
		Genkit:   b.Genkit(),
	}

	// Build embedder via factory
	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("embedder factory for '%s' failed: %w", name, err)
	}

	// Verify result is a valid embedder
	embedder, ok := built.(ai.Embedder)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid Embedder", name)
	}

	resolved.Embedders[name] = embedder

	if deps.Obs.Logger != nil {
		deps.Obs.Logger.Debugf(
			"embedder component built successfully",
			"name", name,
			"type", getEmbedderType(cfg),
		)
	}

	return core.NopCloser, nil
}

// getEmbedderType extracts the provider type for logging
func getEmbedderType(cfg core.ProviderOptions) string {
	if pn, ok := cfg.(interface{ ProviderName() string }); ok {
		return pn.ProviderName()
	}
	return "unknown"
}
