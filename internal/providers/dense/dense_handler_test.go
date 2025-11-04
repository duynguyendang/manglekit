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
	"github.com/duynguyendang/manglekit/internal/vectorstores"
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

// mockLLM is a mock implementation of core.LLMClient for testing.
type mockLLM struct{}

func (m *mockLLM) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	return core.LLMResponse{Text: "mock response"}, nil
}

// mockEmbedderOptions provides a dummy options struct for the mock embedder.
type mockEmbedderOptions struct{}

func (o *mockEmbedderOptions) ProviderName() string { return "mock-embed" }
func (o *mockEmbedderOptions) ProviderKind() core.Kind   { return core.KindEmbedder }

// mockEmbedder is a mock implementation of ai.Embedder for testing.
type mockEmbedder struct{}

func (m *mockEmbedder) Name() string { return "mock-embed" }
func (m *mockEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	return &ai.EmbedResponse{}, nil
}
func (m *mockEmbedder) Register(r api.Registry) {}

// mockVectorStoreOptions provides a dummy options struct for the mock vector store.
type mockVectorStoreOptions struct {
	Embedder string `json:"embedder,omitempty"`
}

func (o *mockVectorStoreOptions) ProviderName() string { return "mock-vs" }
func (o *mockVectorStoreOptions) ProviderKind() core.Kind   { return core.KindVectorStore }
func (o *mockVectorStoreOptions) GetEmbedderName() string   { return o.Embedder }

// mockVectorStore is a mock implementation of core.VectorStore for testing.
type mockVectorStore struct{}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []core.Doc) error { return nil }
func (m *mockVectorStore) Search(ctx context.Context, queryText string, queryVector []float32, topK int, filter map[string]any) ([]core.Doc, error) {
	return nil, nil
}

func TestDense_Handler_HappyPath(t *testing.T) {
	reg := manglekit.NewRegistry()

	// 1. Register main provider
	dense.Register(reg)
	sandwich.Register(reg) // Orchestrator

	// 2. Register mock dependencies (Options + Factory)
	manglekit.Register(reg, &mockLLMOptions{}, func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
		return &mockLLM{}, nil
	})
	manglekit.Register(reg, &mockEmbedderOptions{}, func(ctx context.Context, deps diapi.EmbedderDeps, cfg *mockEmbedderOptions) (ai.Embedder, error) {
		return &mockEmbedder{}, nil
	})
	manglekit.Register(reg, &mockVectorStoreOptions{}, func(ctx context.Context, deps diapi.VectorStoreDeps, cfg *mockVectorStoreOptions) (core.VectorStore, error) {
		return &mockVectorStore{}, nil
	})

	// 3. Register all necessary handlers
	reg.RegisterHandler(&retrievers.Handler{})
	reg.RegisterHandler(sandwich.NewHandler())
	reg.RegisterHandler(&llm.Handler{})
	reg.RegisterHandler(&embedders.Handler{})
	reg.RegisterHandler(&vectorstores.Handler{})

	const testConfig = `
orchestrator: test-sandwich
components:
- name: "mock-embedder"
  kind: "embedder"
  type: "mock-embed"

- name: "mock-vs"
  kind: "vector_store"
  type: "mock-vs"
  params:
    embedder: "mock-embedder"

- name: "my-dense-retriever"
  kind: "retriever"
  type: "dense"
  params:
    embedder: "mock-embedder"
    vectorStore: "mock-vs"

- name: "test-sandwich"
  kind: "orchestrator"
  type: "sandwich"
  params:
    retriever: "my-dense-retriever"
    llm: "mock-llm"

- name: "mock-llm"
  kind: "llm"
  type: "mock-llm"
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(testConfig), reg)
	require.NoError(t, err)
}
