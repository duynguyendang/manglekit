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

// NewHandler returns a new ComponentHandler for Embedders.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindEmbedder
}

// BuildComponent builds the Embedder component with two-path pattern:
// Path 1: Try native Manglekit embedder factory (e.g., google, openai native implementations)
// Path 2: Fall back to suggesting Genkit delegation via "genkit-embedder" type
//
// This ensures consistent behavior with VectorStores and other components,
// providing clear error messages and migration guidance.
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

	// STEP 1: Try native Manglekit embedder factory
	built, err := f.Build(ctx, deps, cfg)
	if err == nil {
		// Native build succeeded
		embedder, ok := built.(ai.Embedder)
		if !ok {
			return nil, fmt.Errorf("component %s is not a valid Embedder", name)
		}
		resolved.Embedders[name] = embedder

		if deps.Obs.Logger != nil {
			deps.Obs.Logger.Debugf(
				"embedder component built successfully via native factory",
				"name", name,
				"type", getEmbedderType(cfg),
			)
		}

		return core.NopCloser, nil
	}

	// STEP 2: Native factory failed - provide helpful error message
	if deps.Obs.Logger != nil {
		deps.Obs.Logger.Debugf(
			"native embedder factory failed, genkit delegation recommended",
			"name", name,
			"native_error", err.Error(),
		)
	}

	// Return error with helpful hint about Genkit delegation
	return nil, fmt.Errorf(
		"embedder factory for '%s' failed: %w\n"+
			"\nHint: For Genkit providers (OpenAI, Google, Vertex, Cohere, etc.), use:\n"+
			"  type: 'genkit-embedder'\n"+
			"  params:\n"+
			"    provider: openai  # or google, vertex, cohere, anthropic, etc.\n"+
			"    model: text-embedding-3-small",
		name, err,
	)
}

// getEmbedderType extracts the provider type for logging
func getEmbedderType(cfg core.ProviderOptions) string {
	if pn, ok := cfg.(interface{ ProviderName() string }); ok {
		return pn.ProviderName()
	}
	return "unknown"
}
