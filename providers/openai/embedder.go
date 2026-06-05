package openai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbedder implements core.Embedder using OpenAI's embedding API.
type OpenAIEmbedder struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

// NewEmbedder creates a new embedder using OpenAI.
// Default model: "text-embedding-3-small" (1536 dimensions)
// Supported models: "text-embedding-3-small", "text-embedding-3-large", "text-embedding-ada-002"
func NewEmbedder(apiKey, baseURL, modelName string) (*OpenAIEmbedder, error) {
	if modelName == "" {
		modelName = "text-embedding-3-small"
	}

	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	return &OpenAIEmbedder{
		client: openai.NewClientWithConfig(config),
		model:  openai.EmbeddingModel(modelName),
	}, nil
}

// Embed generates a vector for a single text string.
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: e.model,
	})
	if err != nil {
		return nil, fmt.Errorf("openai embedding error: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return resp.Data[0].Embedding, nil
}

// EmbedBatch generates vectors for multiple strings.
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: e.model,
	})
	if err != nil {
		return nil, fmt.Errorf("openai batch embedding error: %w", err)
	}

	var results [][]float32
	for _, item := range resp.Data {
		results = append(results, item.Embedding)
	}
	return results, nil
}

// Dimension returns the vector size for the configured model.
// Known models as of 2025-05:
//
//	text-embedding-3-small  → 1536
//	text-embedding-3-large  → 3072
//	text-embedding-ada-002  → 1536
//
// Returns 0 for unknown models so callers can fail explicitly rather than
// silently mismatching dimensions.
func (e *OpenAIEmbedder) Dimension() int {
	switch modelStr := string(e.model); modelStr {
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-3-small", "text-embedding-ada-002":
		return 1536
	default:
		return 0
	}
}
