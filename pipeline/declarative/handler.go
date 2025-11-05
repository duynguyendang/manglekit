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

	// The handler is responsible for resolving all dependencies and passing them
	// to the factory in a type-safe `Deps` struct.
	var stateProvider core.StateProvider
	if opts.StateProvider != "" {
		sp, err := builder.GetStateProvider(opts.StateProvider)
		if err != nil {
			return nil, fmt.Errorf("declarative orchestrator: failed to get state provider %q: %w", opts.StateProvider, err)
		}
		stateProvider = sp
	}

	tools := make(map[string]core.Tool)
	for _, step := range opts.Steps {
		tool, err := resolved.GetToolByName(step.Name)
		if err != nil {
			return nil, fmt.Errorf("declarative orchestrator: failed to get tool %q: %w", step.Name, err)
		}
		tools[step.Name] = tool
	}

	deps := diapi.DeclarativeOrchestratorDeps{
		CoreDeps:      builder.GetCoreDeps(),
		StateProvider: stateProvider,
		Tools:         tools,
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for declarative orchestrator, got %T", factory)
	}
	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindOrchestrator, name, err)
	}

	orchestrator, ok := built.(core.Orchestrator)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid orchestrator", name)
	}
	resolved.Orchestrators[name] = orchestrator
	return orchestrator.Close, nil
}
