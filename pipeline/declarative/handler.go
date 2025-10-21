package declarative

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

type declarativeHandler struct{}

var Handler core.ComponentHandler = &declarativeHandler{}

func (h *declarativeHandler) Kind() core.Kind {
	return core.KindOrchestrator
}

func (h *declarativeHandler) BuildComponent(
	ctx context.Context,
	builder any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	f, ok := factory.(func(context.Context, core.Resolved, Options) (core.Orchestrator, error))
	if !ok {
		return nil, fmt.Errorf("invalid factory type for declarative orchestrator")
	}

	opts, ok := cfg.(Options)
	if !ok {
		return nil, fmt.Errorf("invalid options type for declarative orchestrator")
	}

	orch, err := f(ctx, *resolved, opts)
	if err != nil {
		return nil, err
	}
	resolved.Orchestrators[name] = orch
	return orch.Close, nil
}
