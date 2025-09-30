package embedder

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbedder is a custom embedder for OpenAI models that implements the ai.Embedder interface.
type OpenAIEmbedder struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

// NewOpenAIEmbedder creates a new OpenAIEmbedder.
func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		client: openai.NewClient(apiKey),
		model:  openai.EmbeddingModel(model),
	}
}

// Name returns the name of the embedder.
func (e *OpenAIEmbedder) Name() string {
	return "custom/openai"
}

// Register is required by the ai.Embedder interface.
func (e *OpenAIEmbedder) Register(r api.Registry) {
	// We are not using the action registry in this case.
}

// Embed computes the embedding for a given document.
func (e *OpenAIEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	texts := make([]string, 0, len(req.Input))
	for _, doc := range req.Input {
		if len(doc.Content) > 0 {
			// Simple case: just use the first text part.
			texts = append(texts, doc.Content[0].Text)
		}
	}

	if len(texts) == 0 {
		return &ai.EmbedResponse{}, nil
	}

	apiReq := openai.EmbeddingRequest{
		Input: texts,
		Model: e.model,
	}

	apiResp, err := e.client.CreateEmbeddings(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	if len(apiResp.Data) != len(texts) {
		return nil, fmt.Errorf("number of embeddings (%d) does not match number of documents (%d)", len(apiResp.Data), len(texts))
	}

	embeddings := make([]*ai.Embedding, len(apiResp.Data))
	for i, data := range apiResp.Data {
		embeddings[i] = &ai.Embedding{
			Embedding: data.Embedding,
		}
	}

	return &ai.EmbedResponse{Embeddings: embeddings}, nil
}