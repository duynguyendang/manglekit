package llm

import (
	"context"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
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
		client, err := NewOpenAI(OpenAIOptions{APIKey: openaiAPIKey, Model: "gpt-3.5-turbo"}, g)
		require.NoError(t, err)

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

		client, err := NewGoogle(GoogleOptions{}, model, g)
		require.NoError(t, err)

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
