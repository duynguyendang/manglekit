package state

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler is the component handler for StateProviders.
type Handler struct{}

// NewHandler returns a new ComponentHandler for StateProviders.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindStateProvider
}

// BuildComponent builds the StateProvider component.
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
		return nil, fmt.Errorf("invalid builder DI type for StateProvider handler")
	}

	deps := diapi.NoopDeps{
		CoreDeps: b.GetCoreDeps(),
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for StateProvider handler")
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindStateProvider, name, err)
	}

	provider, ok := built.(core.StateProvider)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid StateProvider", name)
	}
	resolved.StateProviders[name] = provider
	return provider.Close, nil
}
