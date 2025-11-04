//go:build testhooks
// +build testhooks

package sandwich_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/require"
)

// Mock LLM
type mockLLMOptions struct{}

func (o *mockLLMOptions) ProviderName() string { return "mock-llm" }
func (o *mockLLMOptions) ProviderKind() core.Kind   { return core.KindLLM }

type mockLLM struct{}

func (l *mockLLM) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	return core.LLMResponse{Text: "mock response"}, nil
}

// Mock Retriever
type mockRetrieverOptions struct{}

func (o *mockRetrieverOptions) ProviderName() string { return "mock-retriever" }
func (o *mockRetrieverOptions) ProviderKind() core.Kind   { return core.KindRetriever }
func (o *mockRetrieverOptions) GetProviderOptions() any  { return o }

type mockRetriever struct{}

func (r *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{}, nil
}

func registerTestDeps(r *manglekit.Registry) {
	// Register the handler for the component-under-test.
	r.RegisterHandler(sandwich.NewHandler())
	sandwich.Register(r)

	// Register mock llm
	manglekit.Register(r, &mockLLMOptions{},
		func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
			return &mockLLM{}, nil
		})
	r.RegisterHandler(&llm.Handler{})

	// Register mock retriever
	manglekit.Register(r, &mockRetrieverOptions{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
			return &mockRetriever{}, nil
		})
	r.RegisterHandler(retrievers.NewHandler())
}

const configYAML = `
orchestrator: my-sandwich
components:
  - name: my-sandwich
    kind: orchestrator
    type: sandwich
    params:
      retriever: mock-retriever
      llm: mock-llm
  - name: mock-retriever
    kind: retriever
    type: mock-retriever
  - name: mock-llm
    kind: llm
    type: mock-llm
`

func TestSandwichOrchestrator_Handler(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestDeps(reg)

	orch, err := sdk.LoadWithRegistry(context.Background(), []byte(configYAML), reg)
	require.NoError(t, err)
	require.NotNil(t, orch)
}
