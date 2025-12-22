package google

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GoogleEmbedder struct {
	client *genai.Client
	model  string
}

// NewEmbedder creates a new embedder using Google GenAI.
// Default model: "embedding-001" or "text-embedding-004"
func NewEmbedder(ctx context.Context, apiKey string, modelName string) (*GoogleEmbedder, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create google genai client: %w", err)
	}
	if modelName == "" {
		modelName = "embedding-001"
	}
	return &GoogleEmbedder{
		client: client,
		model:  modelName,
	}, nil
}

func (e *GoogleEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	em := e.client.EmbeddingModel(e.model)
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}
	if res.Embedding == nil {
		return nil, fmt.Errorf("no embedding returned")
	}
	return res.Embedding.Values, nil
}

func (e *GoogleEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	em := e.client.EmbeddingModel(e.model)
	batch := em.NewBatch()
	for _, t := range texts {
		batch.AddContent(genai.Text(t))
	}

	res, err := em.BatchEmbedContents(ctx, batch)
	if err != nil {
		return nil, err
	}

	var results [][]float32
	for _, r := range res.Embeddings {
		results = append(results, r.Values)
	}
	return results, nil
}

func (e *GoogleEmbedder) Dimension() int {
	// Usually 768 for text-embedding-004
	return 768
}
