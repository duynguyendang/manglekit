package tools

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
)

type toolsHandler struct{}

func NewHandler() core.ComponentHandler {
	return &toolsHandler{}
}

func (h *toolsHandler) Kind() core.Kind {
	return core.KindTool
}

func (h *toolsHandler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	b, ok := builderDI.(diapi.Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder DI type for tool: got %T", builderDI)
	}

	deps := diapi.CoreDeps{
		Obs: b.GetCoreDeps().Obs,
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for tool: got %T", factory)
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindTool, name, err)
	}

	tool, ok := built.(core.Tool)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid tool", name)
	}

	resolved.Tools[name] = tool
	return core.NopCloser, nil
}
