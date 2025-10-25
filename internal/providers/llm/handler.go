package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler is the component handler for LLMs.
type Handler struct{}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindLLM
}

// BuildComponent builds the LLM component and assigns it to the resolved map.
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
		return nil, fmt.Errorf("invalid builder DI type for LLM handler")
	}

	var deps any
	var err error

	switch c := cfg.(type) {
	default:
		// Default case for LLMs that only need the Genkit instance.
		deps = diapi.LLMDeps{Genkit: b.Genkit()}
	case diapi.ProviderWithOptions:
		// This case allows for more complex dependency resolution if needed in the future,
		// by inspecting the underlying options. For now, it defaults to basic LLMDeps.
		underlyingOpts := c.GetProviderOptions()
		switch underlyingOpts.(type) {
		// Add cases here for specific option types that require different dependencies.
		// e.g., case *MyAdvancedLLMOptions:
		default:
			deps = diapi.LLMDeps{Genkit: b.Genkit()}
		}
	}

	if err != nil {
		return nil, err
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for LLM handler")
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindLLM, name, err)
	}

	llm, ok := built.(core.LLMClient)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid LLM", name)
	}
	resolved.LLMs[name] = llm

	if c, ok := built.(interface{ Close(context.Context) error }); ok {
		return c.Close, nil
	}
	return core.NopCloser, nil
}
