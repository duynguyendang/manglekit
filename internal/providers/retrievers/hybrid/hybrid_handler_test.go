package hybrid_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/hybrid"
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

type mockR1Options struct{}

func (o *mockR1Options) ProviderName() string { return "mock-r1" }
func (o *mockR1Options) ProviderKind() core.Kind   { return core.KindRetriever }
func (o *mockR1Options) GetProviderOptions() any   { return o }

type mockR2Options struct{}

func (o *mockR2Options) ProviderName() string { return "mock-r2" }
func (o *mockR2Options) ProviderKind() core.Kind   { return core.KindRetriever }
func (o *mockR2Options) GetProviderOptions() any   { return o }

func registerTestComponents(r *manglekit.Registry) {
	// 1. Register main provider
	hybrid.Register(r)
	sandwich.Register(r) // Orchestrator

	// 2. Register mock dependencies (Options + Factory)
	manglekit.Register(r, &mockLLMOptions{}, func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
		return &mockLLM{}, nil
	})
	manglekit.Register(r, &mockR1Options{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockR1Options) (core.Retriever, error) {
		return &mockRetriever{}, nil
	})
	manglekit.Register(r, &mockR2Options{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockR2Options) (core.Retriever, error) {
		return &mockRetriever{}, nil
	})

	// 3. Register all necessary handlers
	r.RegisterHandler(&retrievers.Handler{})
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(&llm.Handler{})
}

func TestHybrid_Handler_HappyPath(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	const testConfig = `
orchestrator: test-sandwich
components:
- name: "mock-r1"
  kind: "retriever"
  type: "mock-r1"

- name: "mock-r2"
  kind: "retriever"
  type: "mock-r2"

- name: "my-hybrid"
  kind: "retriever"
  type: "hybrid"
  params:
    retrievers:
    - "mock-r1"
    - "mock-r2"

- name: "mock-llm"
  kind: "llm"
  type: "mock-llm"

- name: "test-sandwich"
  kind: "orchestrator"
  type: "sandwich"
  params:
    retriever: "my-hybrid"
    llm: "mock-llm"
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(testConfig), reg)
	require.NoError(t, err)
}

func TestHybrid_Handler_MissingDependency(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg) // Registers r1 and r2

	const testConfig = `
orchestrator: test-sandwich
components:
- name: "mock-r1"
  kind: "retriever"
  type: "mock-r1"

- name: "my-hybrid"
  kind: "retriever"
  type: "hybrid"
  params:
    retrievers:
    - "mock-r1"
    - "mock-r3" # This one is not registered

- name: "mock-llm"
  kind: "llm"
  type: "mock-llm"

- name: "test-sandwich"
  kind: "orchestrator"
  type: "sandwich"
  params:
    retriever: "my-hybrid"
    llm: "mock-llm"
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(testConfig), reg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `failed to get sub-retriever 'mock-r3' for hybrid retriever 'my-hybrid': dependency not found: mock-r3`)
}
