package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
)

// Handler is the component handler for LLMs.
type Handler struct{}

// NewHandler returns a new ComponentHandler for LLMs.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindLLM
}

// BuildComponent builds the LLM component.
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
		return nil, fmt.Errorf("invalid builder DI type for LLM handler: got %T", builderDI)
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for LLM handler: got %T", factory)
	}

	deps := diapi.LLMDeps{
		CoreDeps: b.GetCoreDeps(),
		Genkit:   b.Genkit(),
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindLLM, name, err)
	}

	llm, ok := built.(core.LLMClient)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid LLMClient", name)
	}
	resolved.LLMs[name] = llm
	if closer, ok := built.(core.ResourceCloser); ok {
		return closer, nil
	}
	return core.NopCloser, nil
}
