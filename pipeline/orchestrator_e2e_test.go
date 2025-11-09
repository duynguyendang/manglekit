//go:build testhooks
// +build testhooks

package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/require"
)

// e2eMockLLMOptions provides a dummy options struct for the mock LLM.
type e2eMockLLMOptions struct{}

func (o *e2eMockLLMOptions) ProviderName() string { return "mock-llm" }
func (o *e2eMockLLMOptions) ProviderKind() core.Kind   { return core.KindLLM }

// e2eMockLLM is a mock implementation of core.LLMClient for E2E testing.
type e2eMockLLM struct{}

func (l *e2eMockLLM) Complete(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
	return core.LLMResponse{Text: "mock response"}, nil
}

func newTestRegistry() *manglekit.Registry {
	r := manglekit.NewRegistry()

	// Register Sandwich Orchestrator
	for _, handler := range orchestrators.Handlers() {
		r.RegisterHandler(handler)
	}
	sandwich.Register(r)

	// Register mock llm
	manglekit.Register(r, &e2eMockLLMOptions{},
		func(ctx context.Context, deps diapi.LLMDeps, cfg *e2eMockLLMOptions) (core.LLMClient, error) {
			return &e2eMockLLM{}, nil
		})
	r.RegisterHandler(&llm.Handler{})

	// Register mock retriever
	manglekit.Register(r, &mockRetrieverOptions{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
			return &mockRetriever{}, nil
		})
	r.RegisterHandler(retrievers.NewHandler())
	return r
}

func newTestRegistryMissingLLM() *manglekit.Registry {
	r := manglekit.NewRegistry()

	// Register Sandwich Orchestrator
	for _, handler := range orchestrators.Handlers() {
		r.RegisterHandler(handler)
	}
	sandwich.Register(r)

	// Register mock retriever
	manglekit.Register(r, &mockRetrieverOptions{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
			return &mockRetriever{}, nil
		})
	r.RegisterHandler(retrievers.NewHandler())
	return r
}

func TestOrchestratorE2E_SuccessfulLoad(t *testing.T) {
	reg := newTestRegistry()
	yaml, err := os.ReadFile(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	orch, err := sdk.LoadWithRegistry(context.Background(), yaml, reg)
	require.NoError(t, err)
	require.NotNil(t, orch)
}

func TestOrchestratorE2E_MissingComponentError(t *testing.T) {
	reg := newTestRegistryMissingLLM()
	yaml, err := os.ReadFile(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	_, err = sdk.LoadWithRegistry(context.Background(), yaml, reg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `could not find options type for kind=llm, type=mock-llm`)
}
