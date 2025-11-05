package cosine_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/embedders"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/rerank"
	"github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
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

// mockRetrieverOptions provides a dummy options struct for the mock retriever.
type mockRetrieverOptions struct{}

func (o *mockRetrieverOptions) ProviderName() string { return "mock-retriever" }
func (o *mockRetrieverOptions) ProviderKind() core.Kind   { return core.KindRetriever }
func (o *mockRetrieverOptions) GetProviderOptions() any   { return o }

// mockRetriever is a mock implementation of core.Retriever for testing.
type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{}, nil
}

func registerTestComponents(r *manglekit.Registry) {
	cosine.Register(r)
	sandwich.Register(r)
	manglekit.Register(r, &mockLLMOptions{}, func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
		return &mockLLM{}, nil
	})
	manglekit.Register(r, &mockEmbedderOptions{}, func(ctx context.Context, deps diapi.EmbedderDeps, cfg *mockEmbedderOptions) (ai.Embedder, error) {
		return &mockEmbedder{}, nil
	})
	manglekit.Register(r, &mockRetrieverOptions{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
		return &mockRetriever{}, nil
	})
	r.RegisterHandler(&retrievers.Handler{})
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(&llm.Handler{})
	r.RegisterHandler(&embedders.Handler{})
	r.RegisterHandler(rerank.NewHandler())
}

func TestCosine_Handler_HappyPath(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	configYAML := `
orchestrator: test-sandwich
components:
  - name: mock-embedder
    kind: embedder
    type: mock-embedder
  - name: my-cosine
    kind: reranker
    type: cosine
    params:
      embedder: mock-embedder
  - name: mock-retriever
    kind: retriever
    type: mock-retriever
  - name: mock-llm
    kind: llm
    type: mock-llm
  - name: test-sandwich
    kind: orchestrator
    type: sandwich
    params:
      retriever: mock-retriever
      llm: mock-llm
      reranker: my-cosine
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(configYAML), reg)
	require.NoError(t, err)
}

func TestCosine_Handler_MissingDependency(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	configYAML := `
orchestrator: test-sandwich
components:
  - name: my-cosine
    kind: reranker
    type: cosine
    params:
      embedder: mock-embedder # This embedder is registered, but not in the components list
  - name: mock-retriever
    kind: retriever
    type: mock-retriever
  - name: mock-llm
    kind: llm
    type: mock-llm
  - name: test-sandwich
    kind: orchestrator
    type: sandwich
    params:
      retriever: mock-retriever
      llm: mock-llm
      reranker: my-cosine
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(configYAML), reg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `dependency not found: mock-embedder`)
}
