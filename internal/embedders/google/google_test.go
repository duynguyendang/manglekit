package google

import (
	"context"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/embed"
	"github.com/firebase/genkit/go/ai"
	"github.com/google/generative-ai-go/genai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func TestNewGoogleEmbedder(t *testing.T) {
	// This test requires a valid Google AI API key to be set in the environment.
	// You can get a key from https://aistudio.google.com/app/apikey
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		t.Skip("GOOGLE_API_KEY environment variable not set")
	}

	opts := embed.GoogleEmbedderOptions{
		Model: "embedding-001",
	}
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	require.NoError(t, err)

	embedder, err := New(opts, client)
	require.NoError(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, "embedding-001", embedder.Name())
}

func TestGoogleEmbedder_Embed(t *testing.T) {
	// This test requires a valid Google AI API key to be set in the environment.
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		t.Skip("GOOGLE_API_KEY environment variable not set")
	}

	opts := embed.GoogleEmbedderOptions{
		Model: "embedding-001",
	}
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	require.NoError(t, err)

	embedder, err := New(opts, client)
	require.NoError(t, err)

	req := &ai.EmbedRequest{
		Input: []*ai.Document{
			ai.DocumentFromText("hello", nil),
			ai.DocumentFromText("world", nil),
		},
	}

	resp, err := embedder.Embed(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Embeddings, 2)
	assert.Len(t, resp.Embeddings[0].Embedding, 768)
	assert.Len(t, resp.Embeddings[1].Embedding, 768)
}