package llm_test

import (
	"context"
	"os"
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

// The os package is used in TestLLM_DI_HappyPath to check for GOOGLE_API_KEY

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
	llm.Register(r)
	sandwich.Register(r)
	manglekit.Register(r, &mockRetrieverOptions{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
		return &mockRetriever{}, nil
	})
	r.RegisterHandler(&llm.Handler{})
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(&retrievers.Handler{})
}

func TestLLM_DI_HappyPath(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	t.Run("openai", func(t *testing.T) {
		const testConfig = `
orchestrator: "test-sandwich"
components:
- name: "my-openai"
  kind: "llm"
  type: "openai"
  params:
    apiKey: "fake-key"
    model: "gpt-3.5-turbo"
    skip_model_check: true
- name: "mock-retriever"
  kind: "retriever"
  type: "mock-retriever"
- name: "test-sandwich"
  kind: "orchestrator"
  type: "sandwich"
  params:
    llm: "my-openai"
    retriever: "mock-retriever"
`
		_, err := sdk.LoadWithRegistry(context.Background(), []byte(testConfig), reg)
		require.NoError(t, err)
	})

	t.Run("google", func(t *testing.T) {
		if os.Getenv("GOOGLE_API_KEY") == "" {
			t.Skip("skipping google LLM DI test: GOOGLE_API_KEY not set")
		}

		const testConfig = `
orchestrator: "test-sandwich"
components:
- name: "my-google"
  kind: "llm"
  type: "google"
  params:
    model: "gemini-1.5-flash-latest"
- name: "mock-retriever"
  kind: "retriever"
  type: "mock-retriever"
- name: "test-sandwich"
  kind: "orchestrator"
  type: "sandwich"
  params:
    llm: "my-google"
    retriever: "mock-retriever"
`
		_, err := sdk.LoadWithRegistry(context.Background(), []byte(testConfig), reg)
		require.NoError(t, err)
	})
}

func TestLLM_DI_MissingAPIKey(t *testing.T) {
	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	const testConfig = `
orchestrator: "test-sandwich"
components:
- name: "my-openai"
  kind: "llm"
  type: "openai"
  params:
    model: "gpt-3.5-turbo"
- name: "mock-retriever"
  kind: "retriever"
  type: "mock-retriever"
- name: "test-sandwich"
  kind: "orchestrator"
  type: "sandwich"
  params:
    llm: "my-openai"
    retriever: "mock-retriever"
`
	_, err := sdk.LoadWithRegistry(context.Background(), []byte(testConfig), reg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `authentication is required: apiKey must be provided for openai provider`)
}
