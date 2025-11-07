package manglekit

import (
	"context"
	"fmt"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/genkit"
)

// FromConfig is a high-level function that loads a Manglekit orchestrator from a YAML
// configuration byte slice. It requires a pre-configured registry with all necessary
// component handlers already registered.
func FromConfig(ctx context.Context, data []byte, registry *Registry) (core.Orchestrator, retrieve.Updatable, error) {
	l := logger.NewStdLogger()
	obs := core.Observability{Logger: l}
	g := genkit.Init(ctx)

	// 2. Create a new builder and register all component handlers.
	b, err := NewBuilder(ctx, registry, obs, g)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new builder: %w", err)
	}

	// The public NewBuilder returns an interface. We need to cast it back to the
	// internal struct type to access the internal fromConfig method.
	internalBuilder, ok := b.(*builder)
	if !ok {
		return nil, nil, fmt.Errorf("internal error: builder is not of the expected type *builder")
	}

	// 3. Build the orchestrator from the configuration.
	orch, up, err := internalBuilder.fromConfig(ctx, data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build orchestrator from config: %w", err)
	}
	return orch, up, nil
}
