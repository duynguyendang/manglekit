package llm

import (
	"context"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/adapters"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMProviders_Integration(t *testing.T) {
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")

	if openaiAPIKey == "" {
		t.Skip("skipping openai integration test: OPENAI_API_KEY not set")
	}
	if googleAPIKey == "" {
		t.Skip("skipping google integration test: GOOGLE_API_KEY not set")
	}

	ctx := context.Background()
	// In a real application, the genkit instance would be managed by the DI container.
	// For this integration test, we will create a temporary one.
	g := genkit.Init(ctx, nil)

	t.Run("openai", func(t *testing.T) {
		// Create OpenAI client via Genkit
		opts := []option.RequestOption{option.WithAPIKey(openaiAPIKey)}
		openAIClient := &openai.OpenAI{APIKey: openaiAPIKey, Opts: opts}

		// Get the model from OpenAI client
		model := openAIClient.Model(g, "gpt-3.5-turbo")
		require.NotNil(t, model)

		// Create adapter for the model
		client := adapters.NewGenkitLLMAdapter(
			g,
			model,
			"openai/gpt-3.5-turbo",
			core.LLMOptions{Temperature: 0.7, MaxOutputTokens: 100},
		)

		req := core.LLMRequest{
			Prompt:    "hello",
			MaxTokens: 5,
		}

		resp, err := client.Complete(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Text)
		assert.NotNil(t, resp.Usage)
		assert.Contains(t, resp.Usage, "prompt")
		assert.Contains(t, resp.Usage, "completion")
		assert.Contains(t, resp.Usage, "total")
	})

	t.Run("google", func(t *testing.T) {
		// The model is normally resolved by genkit, but we create it manually for the test.
		model := googlegenai.GoogleAIModel(g, "gemini-1.5-flash-latest")
		require.NotNil(t, model)

		// Create adapter for the model
		client := adapters.NewGenkitLLMAdapter(
			g,
			model,
			"google/gemini-1.5-flash-latest",
			core.LLMOptions{Temperature: 0.7, MaxOutputTokens: 100},
		)

		req := core.LLMRequest{
			Prompt:    "hello",
			MaxTokens: 5,
		}

		resp, err := client.Complete(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Text)
		assert.NotNil(t, resp.Usage)
		assert.Contains(t, resp.Usage, "prompt")
		assert.Contains(t, resp.Usage, "completion")
		assert.Contains(t, resp.Usage, "total")
	})
}
