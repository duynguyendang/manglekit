package openai

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

func init() {
	manglekit.RegisterEmbedder("openai", New)
	manglekit.RegisterEmbedder("groq", New)
}

// openAIEmbedder implements the retrieve.Embedder interface for OpenAI.
type openAIEmbedder struct {
	client     *openai.Client
	modelName  string
	dimensions int
}

// New creates a new OpenAI embedder with explicit, typed dependencies.
func New(opts embed.OpenAIEmbedderOptions, client *openai.Client) (ai.Embedder, error) {
	if client == nil {
		return nil, fmt.Errorf("openai client is required")
	}
	modelName := opts.Model
	if modelName == "" {
		modelName = "text-embedding-ada-002" // default model
	}

	return &openAIEmbedder{
		client:     client,
		modelName:  modelName,
		dimensions: opts.Dimensions,
	}, nil
}

func (e *openAIEmbedder) Name() string {
	return e.modelName
}

// Embed generates embeddings for the given request (for ai.Embedder).
func (e *openAIEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	if len(req.Input) == 0 {
		return &ai.EmbedResponse{}, nil
	}

	texts := make([]string, len(req.Input))
	for i, doc := range req.Input {
		// A more robust implementation might handle other content types.
		texts[i] = doc.Content[0].Text
	}

	paramReq := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
		Model: openai.EmbeddingModel(e.modelName),
	}

	if e.dimensions > 0 {
		paramReq.Dimensions = param.NewOpt(int64(e.dimensions))
	}

	resp, err := e.client.Embeddings.New(ctx, paramReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings with openai: %w", err)
	}

	// Convert to genkit's format.
	genkitEmbeddings := make([]*ai.Embedding, len(resp.Data))
	for i, data := range resp.Data {
		// The openai-go library returns float64, but genkit expects float32.
		float32Embedding := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			float32Embedding[j] = float32(v)
		}
		genkitEmbeddings[i] = &ai.Embedding{Embedding: float32Embedding}
	}

	return &ai.EmbedResponse{
		Embeddings: genkitEmbeddings,
	}, nil
}

func (e *openAIEmbedder) Register(r api.Registry) {
	//TODO: implement me
	panic("implement me")
}