//go:build testhooks

package declarative_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/internal/providers/state"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/stretchr/testify/require"
)

// TestDeclarative_Handler_HappyPath verifies that the declarative orchestrator
// can be successfully built by the builder when provided with a valid configuration.
func TestDeclarative_Handler_HappyPath(t *testing.T) {
	reg := manglekit.NewRegistry()
	b := manglekit.NewBuilder(reg)

	// Register the declarative orchestrator and its handler.
	b.WithHandlers(declarative.NewHandler())
	require.NoError(t, manglekit.Register(reg, &declarative.Options{}, declarative.NewDeclarative))

	// Register the in-memory state provider and its handler.
	b.WithHandlers(state.NewHandler())
	require.NoError(t, manglekit.Register(reg, &inmemory.Options{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *inmemory.Options) (core.StateProvider, error) {
			return inmemory.New(*cfg)
		},
	))

	// Register a mock retriever that will be adapted into a tool.
	b.WithHandlers(retrievers.NewHandler())
	require.NoError(t, manglekit.Register(reg, &mock.RetrieverOptions{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *mock.RetrieverOptions) (core.Retriever, error) {
			return mock.NewRetriever(nil), nil
		},
	))

	// Configure the builder with the components.
	b.With("my-declarative", &declarative.Options{
		StateProvider: "my-inmemory",
		Steps: []declarative.ToolStepConfig{
			{Name: "noop-tool"},
		},
	})
	b.With("my-inmemory", &inmemory.Options{})
	b.With("noop-tool", &mock.RetrieverOptions{}) // Register the component that will be adapted.
	b.WithOrchestrator("my-declarative")

	// Build the orchestrator.
	_, _, err := b.Build(context.Background())
	require.NoError(t, err)
}
