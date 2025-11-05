package openai_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/embedders"
	"github.com/duynguyendang/manglekit/internal/embedders/openai"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/require"
)

// mockLLMOptions provides a dummy options struct for the mock LLM.
type mockLLMOptions struct{}

func (o *mockLLMOptions) ProviderName() string { return "mock-llm" }
func (o *mockLLMOptions) ProviderKind() core.Kind   { return core.KindLLM }
func (o *mockLLMOptions) GetProviderOptions() any   { return o }

// mockLLM is a mock implementation of core.LLMClient for testing.
type mockLLM struct{}

func (m *mockLLM) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	return core.LLMResponse{Text: "mock response"}, nil
}

// mockRetriever is a mock implementation of core.Retriever for testing.
type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{}, nil
}

type mockRetrieverOptions struct{}

func (o *mockRetrieverOptions) ProviderName() string { return "mock-retriever" }
func (o *mockRetrieverOptions) ProviderKind() core.Kind   { return core.KindRetriever }
func (o *mockRetrieverOptions) GetProviderOptions() any   { return o }

func registerTestComponents(r *manglekit.Registry) {
	openai.Register(r)
	sandwich.Register(r)
	llm.Register(r)
	manglekit.Register(r, &mockLLMOptions{}, func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
		return &mockLLM{}, nil
	})
	manglekit.Register(r, &mockRetrieverOptions{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
		return &mockRetriever{}, nil
	})
	r.RegisterHandler(&embedders.Handler{})
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(&llm.Handler{})
	r.RegisterHandler(&retrievers.Handler{})
}

func TestOpenAIEmbedder_DI(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	testConfig := fmt.Sprintf(`
orchestrator: "test-sandwich"
components:
- name: "my-openai-embedder"
  kind: "embedder"
  type: "openai"
  params:
    apiKey: "fake-key"
    model: "text-embedding-ada-002"
    skip_model_check: true
- name: "mock-retriever"
  kind: "retriever"
  type: "mock-retriever"
- name: "mock-llm"
  kind: "llm"
  type: "mock-llm"
- name: "test-sandwich"
  kind: "orchestrator"
  type: "sandwich"
  params:
    retriever: "mock-retriever"
    llm: "mock-llm"
`)

	_, err := sdk.LoadWithRegistry(context.Background(), []byte(testConfig), reg)
	require.NoError(t, err)
}

func TestOpenAIEmbedder_DI_MissingAPIKey(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	const testConfig = `
orchestrator: "test-sandwich"
components:
- name: "my-openai-embedder"
  kind: "embedder"
  type: "openai"
  params:
    model: "text-embedding-ada-002"
- name: "mock-retriever"
  kind: "retriever"
  type: "mock-retriever"
- name: "mock-llm"
  kind: "llm"
  type: "mock-llm"
- name: "test-sandwich"
  kind: "orchestrator"
  type: "sandwich"
  params:
    retriever: "mock-retriever"
    llm: "mock-llm"
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(testConfig), reg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `apiKey is required for openai embedder`)
}
