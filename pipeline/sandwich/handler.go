package sandwich

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// handler implements the ComponentHandler for the "sandwich" orchestrator.
type handler struct{}

// NewHandler returns a new ComponentHandler for the sandwich orchestrator.
func NewHandler() core.ComponentHandler {
	return &handler{}
}

// Kind returns the kind of component this handler builds.
func (h *handler) Kind() core.Kind {
	return core.KindOrchestrator
}

// BuildComponent constructs a Sandwich orchestrator.
func (h *handler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	// 1. Type-assert the builderDI to the diapi.Builder interface.
	if _, ok := builderDI.(diapi.Builder); !ok {
		return nil, fmt.Errorf("invalid builderDI type: %T", builderDI)
	}

	// 2. Type-assert the options to the SandwichOptions type.
	sandwichOpts, ok := cfg.(*SandwichOptions)
	if !ok {
		return nil, fmt.Errorf("invalid options type: %T", cfg)
	}

	// 3. Type-assert the factory to the expected factory type.
	sandwichFactory, ok := factory.(func(context.Context, core.Resolved, SandwichOptions) (core.Orchestrator, error))
	if !ok {
		return nil, fmt.Errorf("invalid factory type: %T", factory)
	}

	// 4. Call the factory to create the orchestrator.
	orch, err := sandwichFactory(ctx, *resolved, *sandwichOpts)
	if err != nil {
		return nil, err
	}

	// 5. Store the new orchestrator in the resolved map.
	resolved.Orchestrators[name] = orch

	return orch.Close, nil
}
