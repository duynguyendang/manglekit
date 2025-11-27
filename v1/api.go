package manglekit

import (
	"context"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/retrieve"
)

// ProgrammaticBuilder defines the fluent interface for programmatically constructing
// a Manglekit orchestrator.
type ProgrammaticBuilder interface {
	// WithOptions configures a component using its specific options struct.
	// This is the generic way to add any provider to the builder.
	WithOptions(name string, opts core.ProviderOptions) ProgrammaticBuilder

	// Build constructs the final orchestrator based on the configured components.
	// The state provider is resolved by the orchestrator handler from its Options.StateProvider field.
	Build(ctx context.Context, orchestratorName, updatableName string) (core.Orchestrator, retrieve.Updatable, error)
}
