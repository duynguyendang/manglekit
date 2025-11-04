package pipeline_test

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRetrieverOptions provides a dummy options struct for the mock retriever.
type mockRetrieverOptions struct{}

func (o *mockRetrieverOptions) ProviderName() string { return "mock-retriever" }
func (o *mockRetrieverOptions) ProviderKind() core.Kind   { return core.KindRetriever }
func (o *mockRetrieverOptions) GetProviderOptions() any   { return o }

// mockRetriever is a mock implementation of core.Retriever for testing.
type mockRetriever struct{}

func (m *mockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{Docs: []core.Doc{{Text: "mock document"}}}, nil
}

func registerTestComponents(r *manglekit.Registry) {
	sandwich.Register(r)
	llm.RegisterOpenAI(r)
	manglekit.Register(r, &mockRetrieverOptions{}, func(ctx context.Context, deps diapi.NoopDeps, cfg *mockRetrieverOptions) (core.Retriever, error) {
		return &mockRetriever{}, nil
	})
	r.RegisterHandler(&retrievers.Handler{})
	r.RegisterHandler(sandwich.NewHandler())
	r.RegisterHandler(&llm.Handler{})
}

func TestSandwich_Integration(t *testing.T) {
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		t.Skip("skipping sandwich integration test: OPENAI_API_KEY not set")
	}

	reg := manglekit.NewRegistry()
	registerTestComponents(reg)

	configYAML := `
orchestrator: test-sandwich
components:
  - name: openai
    kind: llm
    type: openai
    params:
      apiKey: ` + openaiAPIKey + `
      model: gpt-3.5-turbo
  - name: mock-retriever
    kind: retriever
    type: mock-retriever
  - name: test-sandwich
    kind: orchestrator
    type: sandwich
    params:
      retriever: mock-retriever
      llm: openai
`
	orchestrator, err := sdk.LoadWithRegistry(context.Background(), []byte(configYAML), reg)
	require.NoError(t, err)

	// Execute the pipeline.
	query := core.Query{
		Text: "hello",
	}
	answer, err := orchestrator.Execute(context.Background(), "test-session", query)
	require.NoError(t, err)

	// Verify the results.
	assert.NotEmpty(t, answer.Text)
	assert.NotNil(t, answer.Meta)
	assert.Contains(t, answer.Meta, "token_usage")

	usage, ok := answer.Meta["token_usage"].(map[string]int)
	require.True(t, ok)
	assert.Contains(t, usage, "prompt")
	assert.Contains(t, usage, "completion")
	assert.Contains(t, usage, "total")
}
