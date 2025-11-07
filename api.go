package manglekit

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/retrieve"
)

// ProgrammaticBuilder defines the fluent interface for programmatically constructing
// a Manglekit orchestrator.
type ProgrammaticBuilder interface {
	// WithOptions configures a component using its specific options struct.
	// This is the generic way to add any provider to the builder.
	WithOptions(name string, opts core.ProviderOptions) ProgrammaticBuilder

	// Build constructs the final orchestrator based on the configured components.
	Build(ctx context.Context, orchestratorName, updatableName, stateProviderName string) (core.Orchestrator, retrieve.Updatable, error)
}
