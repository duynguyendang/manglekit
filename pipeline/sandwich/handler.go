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

	// 2. Type-assert the factory to the GenericFactory interface.
	genericFactory, ok := factory.(core.GenericFactory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type: %T; expected core.GenericFactory", factory)
	}

	// 3. Call the factory's Build method, passing the fully resolved struct
	// as the dependency, which is the special contract for orchestrators.
	instance, err := genericFactory.Build(ctx, *resolved, cfg)
	if err != nil {
		return nil, err
	}

	// 4. Type-assert the resulting instance to a core.Orchestrator.
	orch, ok := instance.(core.Orchestrator)
	if !ok {
		return nil, fmt.Errorf("factory for %q returned type %T, but expected core.Orchestrator", name, instance)
	}

	// 5. Store the new orchestrator in the resolved map.
	resolved.Orchestrators[name] = orch

	return orch.Close, nil
}
