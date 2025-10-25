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

	opts, ok := cfg.(Options)
	if !ok {
		return nil, fmt.Errorf("invalid options type for declarative orchestrator, got %T", cfg)
	}

	var stateProvider core.StateProvider
	if opts.StateProvider != "" {
		sp, err := builder.GetStateProvider(opts.StateProvider)
		if err != nil {
			return nil, fmt.Errorf("declarative orchestrator: failed to get state provider %q: %w", opts.StateProvider, err)
		}
		stateProvider = sp
	} else if len(resolved.StateProviders) > 0 {
		return nil, fmt.Errorf("declarative orchestrator: 'state_provider' must be specified when state providers are available")
	}

	f, ok := factory.(func(context.Context, core.Resolved, core.StateProvider, Options) (core.Orchestrator, error))
	if !ok {
		return nil, fmt.Errorf("invalid factory type for declarative orchestrator, got %T", factory)
	}

	orch, err := f(ctx, *resolved, stateProvider, opts)
	if err != nil {
		return nil, err
	}

	resolved.Orchestrators[name] = orch
	return orch.Close, nil
}
