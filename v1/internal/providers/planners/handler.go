package planners

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler implements the component handler for planners.
type Handler struct{}

// NewHandler creates a new planner handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindPlanner
}

// BuildComponent builds a planner component.
func (h *Handler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	builder, ok := builderDI.(diapi.Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder type: %T", builderDI)
	}

	typedFactory, ok := factory.(core.GenericFactory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type: %T", factory)
	}

	deps := diapi.PlannerDeps{
		CoreDeps:  builder.GetCoreDeps(),
		Tools:     resolved.Tools,
		Reasoners: resolved.Reasoners,
	}

	component, err := typedFactory.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build planner %s: %w", name, err)
	}

	planner, ok := component.(core.Planner)
	if !ok {
		return nil, fmt.Errorf("component %s is not a core.Planner", name)
	}

	if resolved.Planners == nil {
		resolved.Planners = make(map[string]core.Planner)
	}
	resolved.Planners[name] = planner
	return core.NopCloser, nil
}
