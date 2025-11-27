// Package reasoners provides the component handler for reasoner components.
package reasoners

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/duynguyendang/manglekit/v1/internal/providers/reasoners/mangle"
)

// Register registers all the reasoner providers.
func Register(r *manglekit.Registry) {
	r.RegisterHandler(NewHandler())
	mangle.Register(r)
}

// Handler is the component handler for Reasoners.
type Handler struct{}

// NewHandler returns a new ComponentHandler for Reasoners.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindReasoner
}

// BuildComponent builds the Reasoner component and assigns it to the resolved map.
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
		return nil, fmt.Errorf("invalid builder DI type for Reasoner handler: got %T", builderDI)
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for Reasoner handler: got %T", factory)
	}

	deps := diapi.RuleSetDeps{
		CoreDeps: b.GetCoreDeps(),
		Registry: b.Registry(),
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindReasoner, name, err)
	}

	reasoner, ok := built.(core.Reasoner)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid Reasoner", name)
	}
	resolved.Reasoners[name] = reasoner

	return core.NopCloser, nil
}
