// Package openai provides a MangleKit embedder implementation for OpenAI and
// other services that offer an OpenAI-compatible API (like Groq).
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
	// Register the same constructor for multiple provider names.
	manglekit.RegisterEmbedder("openai", New)
	manglekit.RegisterEmbedder("groq", New)
}

// openAIEmbedder implements the `ai.Embedder` interface for OpenAI and compatible services.
type openAIEmbedder struct {
	client     *openai.Client
	modelName  string
	dimensions int
}

// New is the constructor for the OpenAI-compatible embedder. It is registered
// with the MangleKit framework for both "openai" and "groq" providers.
//
// opts provides configuration such as the model name and optional embedding dimensions.
// client is the pre-configured `openai.Client` instance, injected by the builder.
// It returns an `ai.Embedder` or an error if the configuration is invalid.
func New(opts embed.OpenAIEmbedderOptions, client *openai.Client) (ai.Embedder, error) {
	if client == nil {
		return nil, fmt.Errorf("openai client is required")
	}
	modelName := opts.Model
	if modelName == "" {
		modelName = "text-embedding-ada-002" // A sensible default.
	}

	return &openAIEmbedder{
		client:     client,
		modelName:  modelName,
		dimensions: opts.Dimensions,
	}, nil
}

// Name returns the identifier of the underlying OpenAI-compatible embedding model.
// This method satisfies the `ai.Embedder` interface.
func (e *openAIEmbedder) Name() string {
	return e.modelName
}

// Embed generates embeddings for a batch of documents using the OpenAI API.
// It converts the Genkit `EmbedRequest` into an `openai` request, calls the API,
// and then converts the response back to the Genkit format, including casting
// the embedding values from float64 to float32.
// This method satisfies the `ai.Embedder` interface.
//
// ctx is the context for the API call.
// req is the embedding request from Genkit.
// It returns the embedding response or an error if the API call fails.
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

	// Add dimensions parameter only if it's specified, as not all models support it.
	if e.dimensions > 0 {
		paramReq.Dimensions = param.NewOpt(int64(e.dimensions))
	}

	resp, err := e.client.Embeddings.New(ctx, paramReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings with openai: %w", err)
	}

	// Convert the response back to genkit's format.
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

// Register is part of the `ai.Embedder` interface for use in Genkit flows,
// but is not used by the MangleKit builder. This is a no-op to satisfy the
// interface, as seen in mock implementations within the codebase (e.g., dense_test.go).
func (e *openAIEmbedder) Register(r api.Registry) {}
