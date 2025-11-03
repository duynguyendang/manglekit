package bm25_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/bm25"
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
// This is needed to work around a bug in the real declarative.Handler.
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
	component, err := genericFactory.Build(ctx, core.Resolved{}, cfg) // Pass empty deps
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

// newTestBuilder creates a new Manglekit Builder pre-configured with a mock orchestrator
// and the necessary handlers for testing the bm25 provider.
func newTestBuilder(t *testing.T) *manglekit.Builder {
	reg := manglekit.NewRegistry()
	b, err := manglekit.NewBuilder(context.Background(), reg, core.Observability{Logger: &mockLogger{}}, &genkit.Genkit{})
	if err != nil {
		t.Fatalf("failed to create new builder: %v", err)
	}

	// Register the mock orchestrator and its CORRECT handler.
	reg.RegisterHandler(&mockOrchestratorHandler{})
	if err := manglekit.Register(reg, &mockOrchestratorOptions{}, func(ctx context.Context, resolved core.Resolved, cfg *mockOrchestratorOptions) (core.Orchestrator, error) {
		return &mockOrchestrator{}, nil
	}); err != nil {
		t.Fatalf("failed to register mock orchestrator: %v", err)
	}
	b.With("mock-orchestrator", &mockOrchestratorOptions{})

	// Register the actual handlers and factories needed for the test.
	reg.RegisterHandler(&retrievers.Handler{})
	bm25.Register(reg)

	return b
}

func TestBM25_Handler_HappyPath(t *testing.T) {
	b := newTestBuilder(t)

	tempDir, err := os.MkdirTemp("", "bm25_handler_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = os.WriteFile(filepath.Join(tempDir, "doc1.md"), []byte("test content"), 0644)
	require.NoError(t, err)

	b.With("my-bm25", &bm25.BM25Options{
		Path: tempDir,
	})

	_, _, err = b.Build(context.Background(), "mock-orchestrator", "", "")
	require.NoError(t, err)
}

func TestBM25_Handler_ConfigFailure(t *testing.T) {
	b := newTestBuilder(t)

	b.With("my-bm25", &bm25.BM25Options{
		// Path is missing, which should cause an error
	})

	_, _, err := b.Build(context.Background(), "mock-orchestrator", "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path option is required for bm25 retriever")
}
