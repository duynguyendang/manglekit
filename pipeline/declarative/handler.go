package declarative

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// declarativeHandler implements the ComponentHandler for the declarative orchestrator.
type declarativeHandler struct{}

// NewHandler returns a new ComponentHandler for the declarative orchestrator.
func NewHandler() core.ComponentHandler {
	return &declarativeHandler{}
}

// Kind returns the kind of component this handler builds.
func (h *declarativeHandler) Kind() core.Kind {
	return core.KindOrchestrator
}

// BuildComponent constructs a declarative orchestrator.
func (h *declarativeHandler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	builder, ok := builderDI.(diapi.Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builderDI type: %T", builderDI)
	}

	opts, ok := cfg.(*Options)
	if !ok {
		return nil, fmt.Errorf("invalid options type for declarative orchestrator, got %T", cfg)
	}

	// The handler is responsible for resolving the state provider dependency.
	var stateProvider core.StateProvider
	if opts.StateProvider != "" {
		sp, err := builder.GetStateProvider(opts.StateProvider)
		if err != nil {
			return nil, fmt.Errorf("declarative orchestrator: failed to get state provider %q: %w", opts.StateProvider, err)
		}
		stateProvider = sp
	}

	// Now, build the orchestrator using the standard factory mechanism.
	genericFactory, ok := factory.(core.GenericFactory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type: %T; expected core.GenericFactory", factory)
	}
	instance, err := genericFactory.Build(ctx, *resolved, cfg)
	if err != nil {
		return nil, err
	}
	orch, ok := instance.(*DeclarativeOrchestrator)
	if !ok {
		return nil, fmt.Errorf("factory for %q returned type %T, but expected *DeclarativeOrchestrator", name, instance)
	}

	// Inject the state provider.
	orch.StateProvider = stateProvider

	resolved.Orchestrators[name] = orch
	return orch.Close, nil
}
