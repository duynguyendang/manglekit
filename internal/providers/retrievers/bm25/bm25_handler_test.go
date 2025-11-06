package bm25_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/assert"
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

func registerTestComponents(r *manglekit.Registry) {
	bm25.Register(r)
	sandwich.Register(r)
	manglekit.Register(r, &mockLLMOptions{}, func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
		return &mockLLM{}, nil
	})
	r.RegisterHandler(&retrievers.Handler{})
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(&llm.Handler{})
}

func TestBM25_Handler_HappyPath(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	tempDir, err := os.MkdirTemp("", "bm25_handler_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = os.WriteFile(filepath.Join(tempDir, "doc1.md"), []byte("test content"), 0644)
	require.NoError(t, err)

	configYAML := fmt.Sprintf(`
orchestrator: test-sandwich
components:
  - name: my-bm25
    kind: retriever
    type: bm25
    params:
      path: %s
  - name: mock-llm
    kind: llm
    type: mock-llm
  - name: test-sandwich
    kind: orchestrator
    type: sandwich
    params:
      retriever: my-bm25
      llm: mock-llm
`, tempDir)

	_, err = sdk.LoadWithRegistry(context.Background(), []byte(configYAML), reg)
	require.NoError(t, err)
}

func TestBM25_Handler_ConfigFailure(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	configYAML := `
orchestrator: test-sandwich
components:
  - name: my-bm25
    kind: retriever
    type: bm25
    params:
      # Path is missing, which should cause an error
  - name: mock-llm
    kind: llm
    type: mock-llm
  - name: test-sandwich
    kind: orchestrator
    type: sandwich
    params:
      retriever: my-bm25
      llm: mock-llm
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(configYAML), reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either path or documents option is required for bm25 retriever")
}
