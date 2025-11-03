package schemaparsers

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler is the component handler for SchemaParsers.
type Handler struct{}

// NewHandler returns a new ComponentHandler for SchemaParsers.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindSchemaParser
}

// BuildComponent builds the SchemaParser component.
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
		return nil, fmt.Errorf("invalid builder DI type for SchemaParser handler")
	}

	deps := diapi.NoopDeps{
		CoreDeps: b.GetCoreDeps(),
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for SchemaParser handler")
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindSchemaParser, name, err)
	}

	_, ok = built.(core.SchemaParser)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid SchemaParser", name)
	}
	// TODO: Add a map for schemaparsers to the resolved struct.
	resolved.SchemaParsers[name] = built.(core.SchemaParser)
	return core.NopCloser, nil
}
