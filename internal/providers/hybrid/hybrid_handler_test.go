package hybrid_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger is a no-op logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debugf(msg string, kv ...any) {}
func (m *mockLogger) Infof(msg string, kv ...any)  {}
func (m *mockLogger) Warnf(msg string, kv ...any)  {}
func (m *mockLogger) Errorf(msg string, kv ...any) {}
func (m *mockLogger) With(kv ...any) core.Logger   { return m }

// mockOrchestratorOptions provides a dummy options struct for the mock orchestrator.
type mockOrchestratorOptions struct{}

func (o *mockOrchestratorOptions) ProviderName() string { return "mock-orch" }
func (o *mockOrchestratorOptions) ProviderKind() core.Kind   { return core.KindOrchestrator }

// mockOrchestrator is a minimal implementation of core.Orchestrator for testing.
type mockOrchestrator struct{}

func (m *mockOrchestrator) Execute(context.Context, string, core.Query) (core.Answer, error) {
	return core.Answer{}, nil
}
func (m *mockOrchestrator) Close(context.Context) error { return nil }

// mockOrchestratorHandler is a corrected component handler for the mock orchestrator.
type mockOrchestratorHandler struct{}

func (h *mockOrchestratorHandler) Kind() core.Kind {
	return core.KindOrchestrator
}

func (h *mockOrchestratorHandler) BuildComponent(
	ctx context.Context,
	builder any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	genericFactory, ok := factory.(core.GenericFactory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type: %T", factory)
	}
	component, err := genericFactory.Build(ctx, resolved, cfg) // Pass resolved deps
	if err != nil {
		return nil, err
	}
	orch, ok := component.(core.Orchestrator)
	if !ok {
		return nil, fmt.Errorf("factory for %q returned %T, expected core.Orchestrator", name, component)
	}
	resolved.Orchestrators[name] = orch
	return orch.Close, nil
}

// mockRetrieverOptions provides a dummy options struct for the mock retriever.
type mockRetrieverOptions struct {
	Name string
}

func (o *mockRetrieverOptions) ProviderName() string { return o.Name }
func (o *mockRetrieverOptions) ProviderKind() core.Kind   { return core.KindRetriever }
func (o *mockRetrieverOptions) GetProviderOptions() any   { return o }

// mockRetriever is a mock implementation of core.Retriever for testing.
type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{}, nil
}

func newTestBuilder(t *testing.T) *manglekit.Builder {
	reg := manglekit.NewRegistry()
	b, err := manglekit.NewBuilder(context.Background(), reg, core.Observability{Logger: &mockLogger{}}, &genkit.Genkit{})
	if err != nil {
		t.Fatalf("failed to create new builder: %v", err)
	}

	// Register mock orchestrator so the main Build() call can succeed.
	reg.RegisterHandler(&mockOrchestratorHandler{})
	if err := manglekit.Register(reg, &mockOrchestratorOptions{}, func(ctx context.Context, resolved *core.Resolved, cfg *mockOrchestratorOptions) (core.Orchestrator, error) {
		return &mockOrchestrator{}, nil
	}); err != nil {
		t.Fatalf("failed to register mock orchestrator: %v", err)
	}
	b.With("mock-orchestrator", &mockOrchestratorOptions{})
	b.WithOrchestrator("mock-orchestrator")

	// Register mock sub-retrievers
	reg.RegisterHandler(&retrievers.Handler{})
	if err := manglekit.Register(reg, &mockRetrieverOptions{Name: "mock-r1"}, func(ctx context.Context, deps any, cfg *mockRetrieverOptions) (core.Retriever, error) {
		return &mockRetriever{}, nil
	}); err != nil {
		t.Fatalf("failed to register mock retriever r1: %v", err)
	}
	if err := manglekit.Register(reg, &mockRetrieverOptions{Name: "mock-r2"}, func(ctx context.Context, deps any, cfg *mockRetrieverOptions) (core.Retriever, error) {
		return &mockRetriever{}, nil
	}); err != nil {
		t.Fatalf("failed to register mock retriever r2: %v", err)
	}

	// Register the actual handlers and factories needed for the test.
	hybrid.Register(reg)

	return b
}

func TestHybrid_Handler_HappyPath(t *testing.T) {
	b := newTestBuilder(t)

	b.With("mock-r1", &mockRetrieverOptions{Name: "mock-r1"})
	b.With("mock-r2", &mockRetrieverOptions{Name: "mock-r2"})
	b.With("my-hybrid", &hybrid.HybridOptions{
		Retrievers: []string{"mock-r1", "mock-r2"},
	})

	_, _, err := b.Build(context.Background())
	require.NoError(t, err)
}

func TestHybrid_Handler_MissingDependency(t *testing.T) {
	b := newTestBuilder(t)

	b.With("mock-r1", &mockRetrieverOptions{Name: "mock-r1"})
	// mock-r2 is not registered with the builder
	b.With("my-hybrid", &hybrid.HybridOptions{
		Retrievers: []string{"mock-r1", "mock-r2"},
	})
	_, _, err := b.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get sub-retriever 'mock-r2'")
}
