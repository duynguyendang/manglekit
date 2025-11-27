package mangle_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/duynguyendang/manglekit/v1/internal/providers/llm"
	"github.com/duynguyendang/manglekit/v1/internal/providers/retrievers"
	"github.com/duynguyendang/manglekit/v1/internal/providers/rules"
	"github.com/duynguyendang/manglekit/v1/internal/providers/rules/mangle"
	"github.com/duynguyendang/manglekit/v1/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/require"
)

// mockLLMOptions provides a dummy options struct for the mock LLM.
type mockLLMOptions struct{}

func (o *mockLLMOptions) ProviderName() string    { return "mock-llm" }
func (o *mockLLMOptions) ProviderKind() core.Kind { return core.KindLLM }
func (o *mockLLMOptions) GetProviderOptions() any { return o }

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

func (o *mockRetrieverOptions) ProviderName() string    { return "mock-retriever" }
func (o *mockRetrieverOptions) ProviderKind() core.Kind { return core.KindRetriever }
func (o *mockRetrieverOptions) GetProviderOptions() any { return o }

func registerTestComponents(r *manglekit.Registry) {
	mangle.Register(r)
	sandwich.Register(r)
	llm.Register(r)
	manglekit.Register(r, &mockLLMOptions{}, func(ctx context.Context, deps diapi.LLMDeps, cfg *mockLLMOptions) (core.LLMClient, error) {
		return &mockLLM{}, nil
	})
	manglekit.Register(r, &mockRetrieverOptions{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
		return &mockRetriever{}, nil
	})
	r.RegisterHandler(rules.NewHandler())
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(&llm.Handler{})
	r.RegisterHandler(&retrievers.Handler{})
}

func TestMangle_DI(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	tempDir, err := os.MkdirTemp("", "mangle_di_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ruleFile := filepath.Join(tempDir, "rules.dlog")
	err = os.WriteFile(ruleFile, []byte(`deny("test").`), 0644)
	require.NoError(t, err)

	testConfig := `
orchestrator: "test-sandwich"
components:
- name: "my-mangle-rules"
  kind: "rules"
  type: "mangle"
  params:
    path:
    - ` + tempDir + `
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
    ruleSet: "my-mangle-rules"
`
	_, err = sdk.LoadWithRegistry(context.Background(), []byte(testConfig), reg)
	require.NoError(t, err)
}
