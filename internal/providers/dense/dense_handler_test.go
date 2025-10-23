package dense_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/embedders"
	"github.com/duynguyendang/manglekit/internal/providers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/internal/vectorstores"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrchestratorOptions provides a dummy options struct for the mock orchestrator.
type mockOrchestratorOptions struct{}

func (o mockOrchestratorOptions) ProviderName() string { return "mock-orch" }
func (o mockOrchestratorOptions) ProviderKind() core.Kind   { return core.KindOrchestrator }

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

// mockEmbedderOptions provides a dummy options struct for the mock embedder.
type mockEmbedderOptions struct{}

func (o mockEmbedderOptions) ProviderName() string { return "mock-embedder" }
func (o mockEmbedderOptions) ProviderKind() core.Kind   { return core.KindEmbedder }

// mockEmbedder is a mock implementation of ai.Embedder for testing.
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	return &ai.EmbedResponse{}, nil
}
func (m *mockEmbedder) Name() string                     { return "mock-embedder" }
func (m *mockEmbedder) Register(r api.Registry)          {}

// mockVectorStoreOptions provides a dummy options struct for the mock vector store.
type mockVectorStoreOptions struct{}

func (o mockVectorStoreOptions) ProviderName() string { return "mock-vs" }
func (o mockVectorStoreOptions) ProviderKind() core.Kind   { return core.KindVectorStore }

// mockVectorStore is a mock implementation of core.VectorStore for testing.
type mockVectorStore struct{}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []core.Doc) error { return nil }
func (m *mockVectorStore) Search(ctx context.Context, queryText string, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error) {
	return nil, nil
}

func newTestBuilder(t *testing.T) *manglekit.Builder {
	reg := manglekit.NewRegistry()
	b := manglekit.NewBuilder(reg)

	// Register mock orchestrator so the main Build() call can succeed.
	reg.RegisterHandler(&mockOrchestratorHandler{})
	manglekit.Register(reg, mockOrchestratorOptions{}, func(ctx context.Context, resolved core.Resolved, cfg mockOrchestratorOptions) (core.Orchestrator, error) {
		return &mockOrchestrator{}, nil
	})
	b.With("mock-orchestrator", mockOrchestratorOptions{})
	b.WithOrchestrator("mock-orchestrator")

	// Register mock dependencies for the dense retriever
	reg.RegisterHandler(&embedders.Handler{})
	manglekit.Register(reg, mockEmbedderOptions{}, func(ctx context.Context, deps any, cfg mockEmbedderOptions) (ai.Embedder, error) {
		return &mockEmbedder{}, nil
	})
	reg.RegisterHandler(&vectorstores.Handler{})
	manglekit.Register(reg, mockVectorStoreOptions{}, func(ctx context.Context, deps any, cfg mockVectorStoreOptions) (core.VectorStore, error) {
		return &mockVectorStore{}, nil
	})

	// Register the actual handlers and factories needed for the test.
	reg.RegisterHandler(&retrievers.Handler{})
	dense.Register(reg)

	return b
}

func TestDense_Handler_HappyPath(t *testing.T) {
	b := newTestBuilder(t)

	b.With("mock-embedder", mockEmbedderOptions{})
	b.With("mock-vs", mockVectorStoreOptions{})
	b.With("my-dense", dense.DenseOptions{
		Embedder:    "mock-embedder",
		VectorStore: "mock-vs",
	})

	_, _, err := b.Build(context.Background())
	require.NoError(t, err)
}

func TestDense_Handler_MissingDependency(t *testing.T) {
	b := newTestBuilder(t)

	b.With("mock-embedder", mockEmbedderOptions{})
	// VectorStore is not registered
	b.With("my-dense", dense.DenseOptions{
		Embedder:    "mock-embedder",
		VectorStore: "mock-vs", // This one is missing
	})

	_, _, err := b.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency not found: mock-vs")
}
