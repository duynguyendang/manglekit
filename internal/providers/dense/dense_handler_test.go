package dense_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/embedders"
	"github.com/duynguyendang/manglekit/internal/providers/dense"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/stretchr/testify/require"
)

// mockLLMOptions provides a dummy options struct for the mock LLM.
type mockLLMOptions struct{}

func (o *mockLLMOptions) ProviderName() string { return "mock-llm" }
func (o *mockLLMOptions) ProviderKind() core.Kind   { return core.KindLLM }
func (o *mockLLMOptions) GetProviderOptions() any { return o }

// mockLLM is a mock implementation of core.LLMClient for testing.
type mockLLM struct{}

func (m *mockLLM) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	return core.LLMResponse{Text: "mock response"}, nil
}

// mockEmbedderOptions provides a dummy options struct for the mock embedder.
type mockEmbedderOptions struct{}

func (o *mockEmbedderOptions) ProviderName() string { return "mock-embedder" }
func (o *mockEmbedderOptions) ProviderKind() core.Kind   { return core.KindEmbedder }
func (o *mockEmbedderOptions) GetProviderOptions() any { return o }

// mockEmbedder is a mock implementation of ai.Embedder for testing.
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	return &ai.EmbedResponse{}, nil
}
func (m *mockEmbedder) Name() string                     { return "mock-embedder" }
func (m *mockEmbedder) Register(r api.Registry)          {}

// mockVectorStoreOptions provides a dummy options struct for the mock vector store.
type mockVectorStoreOptions struct{}

func (o *mockVectorStoreOptions) ProviderName() string { return "mock-vs" }
func (o *mockVectorStoreOptions) ProviderKind() core.Kind   { return core.KindVectorStore }
func (o *mockVectorStoreOptions) GetEmbedderName() string   { return "mock-embedder" }
func (o *mockVectorStoreOptions) GetProviderOptions() any { return o }

// mockVectorStore is a mock implementation of core.VectorStore for testing.
type mockVectorStore struct{}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []core.Doc) error { return nil }
func (m *mockVectorStore) Search(ctx context.Context, queryText string, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error) {
	return nil, nil
}

type mockVectorStoreHandler struct{}

func (h *mockVectorStoreHandler) Kind() core.Kind { return core.KindVectorStore }
func (h *mockVectorStoreHandler) BuildComponent(ctx context.Context, builderDI any, factory any, resolved *core.Resolved, cfg core.ProviderOptions, name string) (core.ResourceCloser, error) {
	f, ok := factory.(core.Factory)
	if !ok {
		return nil, nil
	}
	vs, err := f.Build(ctx, diapi.NoopDeps{}, cfg)
	if err != nil {
		return nil, err
	}
	resolved.VectorStores[name] = vs.(core.VectorStore)
	return core.NopCloser, nil
}

func TestDense_Handler_HappyPath(t *testing.T) {
	reg := manglekit.NewRegistry()
	dense.Register(reg)
	sandwich.Register(reg)
	manglekit.Register(reg, &mockLLMOptions{}, func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
		return &mockLLM{}, nil
	})
	manglekit.Register(reg, &mockEmbedderOptions{}, func(ctx context.Context, deps diapi.EmbedderDeps, cfg *mockEmbedderOptions) (ai.Embedder, error) {
		return &mockEmbedder{}, nil
	})
	manglekit.Register(reg, &mockVectorStoreOptions{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockVectorStoreOptions) (core.VectorStore, error) {
		return &mockVectorStore{}, nil
	})
	reg.RegisterHandler(&retrievers.Handler{})
	reg.RegisterHandler(sandwich.NewHandler())
	reg.RegisterHandler(&llm.Handler{})
	reg.RegisterHandler(&embedders.Handler{})
	reg.RegisterHandler(&mockVectorStoreHandler{})

	configYAML := `
orchestrator: test-sandwich
components:
  - name: mock-embedder
    kind: embedder
    type: mock-embedder
  - name: mock-vs
    kind: vectorStore
    type: mock-vs
  - name: my-dense
    kind: retriever
    type: dense
    params:
      embedder: mock-embedder
      vectorStore: mock-vs
  - name: mock-llm
    kind: llm
    type: mock-llm
  - name: test-sandwich
    kind: orchestrator
    type: sandwich
    params:
      retriever: my-dense
      llm: mock-llm
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(configYAML), reg)
	require.NoError(t, err)
}
