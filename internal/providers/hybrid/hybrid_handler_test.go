package hybrid_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/hybrid"
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
func (o *mockLLMOptions) GetProviderOptions() any { return o }

// mockLLM is a mock implementation of core.LLMClient for testing.
type mockLLM struct{}

func (m *mockLLM) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	return core.LLMResponse{Text: "mock response"}, nil
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

func registerTestComponents(r *manglekit.Registry) {
	hybrid.Register(r)
	sandwich.Register(r)
	manglekit.Register(r, &mockLLMOptions{}, func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
		return &mockLLM{}, nil
	})
	manglekit.Register(r, &mockRetrieverOptions{Name: "mock-r1"}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
		return &mockRetriever{}, nil
	})
	manglekit.Register(r, &mockRetrieverOptions{Name: "mock-r2"}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
		return &mockRetriever{}, nil
	})
	r.RegisterHandler(&retrievers.Handler{})
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(&llm.Handler{})
}

func TestHybrid_Handler_HappyPath(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	configYAML := `
orchestrator: test-sandwich
components:
  - name: mock-r1
    kind: retriever
    type: mock-r1
  - name: mock-r2
    kind: retriever
    type: mock-r2
  - name: my-hybrid
    kind: retriever
    type: hybrid
    params:
      retrievers:
        - mock-r1
        - mock-r2
  - name: mock-llm
    kind: llm
    type: mock-llm
  - name: test-sandwich
    kind: orchestrator
    type: sandwich
    params:
      retriever: my-hybrid
      llm: mock-llm
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(configYAML), reg)
	require.NoError(t, err)
}
