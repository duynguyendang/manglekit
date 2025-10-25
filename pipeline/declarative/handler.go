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
	// Assert builderDI type for consistency, even if not used directly.
	if _, ok := builderDI.(diapi.Builder); !ok {
		return nil, fmt.Errorf("invalid builderDI type: %T", builderDI)
	}

	f, ok := factory.(func(context.Context, core.Resolved, Options) (core.Orchestrator, error))
	if !ok {
		return nil, fmt.Errorf("invalid factory type for declarative orchestrator, got %T", factory)
	}

	opts, ok := cfg.(Options)
	if !ok {
		return nil, fmt.Errorf("invalid options type for declarative orchestrator, got %T", cfg)
	}

	orch, err := f(ctx, *resolved, opts)
	if err != nil {
		return nil, err
	}

	resolved.Orchestrators[name] = orch
	return orch.Close, nil
}
