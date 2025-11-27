//go:build testhooks
// +build testhooks

package declarative_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/duynguyendang/manglekit/v1/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/v1/internal/providers/state"
	"github.com/duynguyendang/manglekit/v1/pipeline/declarative"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/require"
)

// Mock Retriever
type mockRetrieverOptions struct{}

func (o *mockRetrieverOptions) ProviderName() string    { return "mock-retriever" }
func (o *mockRetrieverOptions) ProviderKind() core.Kind { return core.KindRetriever }
func (o *mockRetrieverOptions) GetProviderOptions() any { return o }

type mockRetriever struct{}

func (r *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{}, nil
}

// Mock State Provider
type mockStateProviderOptions struct{}

func (o *mockStateProviderOptions) ProviderName() string    { return "mock-state" }
func (o *mockStateProviderOptions) ProviderKind() core.Kind { return core.KindStateProvider }
func (o *mockStateProviderOptions) GetProviderOptions() any { return o }

type mockStateProvider struct{}

func (p *mockStateProvider) Get(ctx context.Context, key string) (interface{}, error) {
	return nil, nil
}
func (p *mockStateProvider) Set(ctx context.Context, key string, value interface{}) error { return nil }
func (p *mockStateProvider) Delete(ctx context.Context, key string) error                 { return nil }
func (p *mockStateProvider) Close(ctx context.Context) error                              { return nil }

func registerTestDeps(r *manglekit.Registry) {
	// Register the handler for the component-under-test.
	r.RegisterHandler(declarative.NewHandler())
	declarative.Register(r)

	// Register mock retriever
	manglekit.Register(r, &mockRetrieverOptions{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
			return &mockRetriever{}, nil
		})
	r.RegisterHandler(retrievers.NewHandler())

	// Register mock state provider
	manglekit.Register(r, &mockStateProviderOptions{},
		func(ctx context.Context, deps diapi.StateProviderDeps, cfg *mockStateProviderOptions) (core.StateProvider, error) {
			return &mockStateProvider{}, nil
		})
	r.RegisterHandler(state.NewHandler())
}

const configYAML = `
orchestrator: my-declarative-orchestrator
components:
  - name: my-declarative-orchestrator
    kind: orchestrator
    type: declarative
    params:
      steps:
        - name: mock-retriever
  - name: mock-retriever
    kind: retriever
    type: mock-retriever
`

func TestDeclarativeOrchestrator_Handler(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestDeps(reg)

	orch, err := sdk.LoadWithRegistry(context.Background(), []byte(configYAML), reg)
	require.NoError(t, err)
	require.NotNil(t, orch)
}

const configWithState = `
orchestrator: my-declarative-orchestrator
components:
  - name: my-declarative-orchestrator
    kind: orchestrator
    type: declarative
    params:
      state_provider: mock-state
      steps:
        - name: mock-retriever
  - name: mock-retriever
    kind: retriever
    type: mock-retriever
  - name: mock-state
    kind: state_provider
    type: mock-state
`

func TestDeclarativeOrchestrator_WithStateProvider(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestDeps(reg)

	orch, err := sdk.LoadWithRegistry(context.Background(), []byte(configWithState), reg)
	require.NoError(t, err)
	require.NotNil(t, orch)

	declOrch, ok := orch.(*declarative.DeclarativeOrchestrator)
	require.True(t, ok)
	require.NotNil(t, declOrch.StateProvider)
}

const configMissingTool = `
orchestrator: my-declarative-orchestrator
components:
  - name: my-declarative-orchestrator
    kind: orchestrator
    type: declarative
    params:
      steps:
        - name: missing-tool
`

func TestDeclarativeOrchestrator_MissingTool(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestDeps(reg)

	_, err := sdk.LoadWithRegistry(context.Background(), []byte(configMissingTool), reg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `tool with name 'missing-tool' not found`)
}
