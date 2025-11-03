package pipeline_test

import (
	"context"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRetriever is a simple mock for the retriever.
type MockRetriever struct{}

func (r *MockRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	return core.RetrieveResult{Docs: []core.Doc{{Text: "mock document"}}}, nil
}

func TestSandwich_Integration(t *testing.T) {
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		t.Skip("skipping sandwich integration test: OPENAI_API_KEY not set")
	}

	ctx := context.Background()
	g := genkit.Init(ctx, nil)

	// Create a real OpenAI client.
	llmClient, err := llm.NewOpenAI(llm.OpenAIOptions{APIKey: openaiAPIKey, Model: "gpt-3.5-turbo"}, g)
	require.NoError(t, err)

	// Create the orchestrator with the NewSandwich factory.
	resolved := &core.Resolved{
		LLMs:          map[string]core.LLMClient{"openai": llmClient},
		Retrievers:    map[string]core.Retriever{"mock": &MockRetriever{}},
		Orchestrators: make(map[string]core.Orchestrator),
		Obs:           core.Observability{Logger: logger.NewStdLogger()},
	}
	opts := &sandwich.Options{
		Retriever: "mock",
		LLM:       "openai",
	}

	builder, err := manglekit.NewBuilder(ctx, manglekit.NewRegistry(), resolved.Obs, g)
	require.NoError(t, err)

	handler := sandwich.NewHandler()
	closer, err := handler.BuildComponent(ctx, builder, nil, resolved, opts, "test_sandwich")
	require.NoError(t, err)
	require.NotNil(t, closer)

	orchestrator, ok := resolved.Orchestrators["test_sandwich"]
	require.True(t, ok)

	// Execute the pipeline.
	query := core.Query{
		Text: "hello",
	}
	answer, err := orchestrator.Execute(ctx, "test-session", query)
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
