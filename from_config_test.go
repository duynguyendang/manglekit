package manglekit_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilderFromConfig_HappyPath(t *testing.T) {
	// 1. Create a registry and register mock providers.
	reg := manglekit.NewRegistry()
	mock.Register(reg)
	orchestrators.Register(reg)

	// 2. Create a config object that uses the mock providers.
	cfg := &config.Config{
		LLM: &config.LLMConfig{
			Provider: "mock-llm",
			Options:  map[string]any{"model": "test-model"},
		},
		Embedder: &config.EmbedderConfig{
			Provider: "mock-embedder",
		},
		Retrieve: &config.RetrieveConfig{
			Provider: "mock-retriever",
			Options:  map[string]any{"pairs": map[string]string{"hello": "world"}},
		},
		Rerank: &config.RerankConfig{
			Provider: "mock-reranker",
			Options:  map[string]any{"passthrough": map[string]bool{"hello": true}},
		},
		TopK: 42,
	}

	// 3. Call the function under test.
	builder, err := manglekit.NewBuilderFromConfig(context.Background(), cfg, reg)
	require.NoError(t, err)
	require.NotNil(t, builder)

	// 4. Build the orchestrator to confirm wiring was successful.
	orch, err := builder.Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, orch)

	// 5. Execute the orchestrator and verify the output from the mock components.
	query := "hello"
	resp, err := orch.Execute(context.Background(), "session1", core.Query{Text: query})
	require.NoError(t, err)

	// The mock LLM should return a specific string.
	// The mock retriever should have returned "world".
	// The mock reranker should have passed it through.
	// The final prompt to the LLM will be a combination of these.
	// For this test, we'll just check that the response contains the model name
	// and the retrieved document, which proves the components were wired correctly.
	assert.Contains(t, resp.Text, "test-model")
	// The mock LLM's response is `fmt.Sprintf("model: %s prompt: %s", l.model, fullPrompt.String())`
	// and the prompt will contain the retrieved docs.
	assert.Contains(t, resp.Text, "prompt: hello context: world")
}

func TestNewBuilderFromConfig_ValidationErrors(t *testing.T) {
	reg := manglekit.NewRegistry()
	mock.Register(reg)
	orchestrators.Register(reg)

	// Test case: Invalid config (missing LLM)
	cfg := &config.Config{
		Embedder: &config.EmbedderConfig{Provider: "mock-embedder"},
		Retrieve: &config.RetrieveConfig{Provider: "mock-retriever"},
	}

	_, err := manglekit.NewBuilderFromConfig(context.Background(), cfg, reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid configuration: llm configuration is required")

	// Test case: Unknown provider
	cfg.LLM = &config.LLMConfig{Provider: "unknown-provider"}
	_, err = manglekit.NewBuilderFromConfig(context.Background(), cfg, reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no options type registered for provider \"unknown-provider\"")
}