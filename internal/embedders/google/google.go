// Package google provides a MangleKit embedder implementation that uses Google's
// generative AI models (e.g., embedding-001) via the `google/generative-ai-go` SDK.
package google

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/google/generative-ai-go/genai"
)

const (
	defaultEmbeddingModel = "embedding-001"
	defaultDim            = 768
)

func init() {
	manglekit.RegisterEmbedder("google-embedder", New)
}

// GoogleEmbedder implements the `ai.Embedder` interface from Genkit, providing
// a wrapper around the Google `genai.EmbeddingModel` client.
type GoogleEmbedder struct {
	client *genai.EmbeddingModel
	model  string
	dim    int
}

// New is the constructor for the GoogleEmbedder. It is registered with the
// MangleKit framework and called by the builder to create a new instance.
//
// opts provides the configuration, such as the specific model to use.
// client is the pre-configured `genai.Client` dependency, injected by the builder.
// It returns an `ai.Embedder` or an error if the configuration is invalid.
func New(opts embed.GoogleEmbedderOptions, client *genai.Client) (ai.Embedder, error) {
	if client == nil {
		return nil, fmt.Errorf("google: genai.Client is required")
	}

	model := opts.Model
	if model == "" {
		model = defaultEmbeddingModel
	}

	var dim int
	switch model {
	case "embedding-001":
		dim = defaultDim
	default:
		// A real implementation might query model capabilities instead of erroring.
		return nil, fmt.Errorf("google: unknown model %s, cannot determine dimension", model)
	}

	return &GoogleEmbedder{
		client: client.EmbeddingModel(model),
		model:  model,
		dim:    dim,
	}, nil
}

// Name returns the identifier of the underlying Google embedding model.
// This method satisfies the `ai.Embedder` interface.
func (e *GoogleEmbedder) Name() string {
	return e.model
}

// Embed generates embeddings for a batch of documents. It takes a Genkit
// `EmbedRequest`, converts it into a `genai` batch request, calls the Google API,
// and then converts the response back into the Genkit `EmbedResponse` format.
// This method satisfies the `ai.Embedder` interface.
//
// ctx is the context for the API call.
// req is the embedding request from Genkit, containing the documents to embed.
// It returns the embedding response or an error if the API call fails.
func (e *GoogleEmbedder) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
	if len(req.Input) == 0 {
		return &ai.EmbedResponse{}, nil
	}

	batch := e.client.NewBatch()
	for _, doc := range req.Input {
		// A more robust implementation might handle other content types.
		batch.AddContent(genai.Text(doc.Content[0].Text))
	}

	res, err := e.client.BatchEmbedContents(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("google: failed to embed contents: %w", err)
	}

	// The response needs to be converted back to genkit's format.
	var genkitEmbeddings []*ai.Embedding
	for _, emb := range res.Embeddings {
		genkitEmbeddings = append(genkitEmbeddings, &ai.Embedding{Embedding: emb.Values})
	}

	return &ai.EmbedResponse{
		Embeddings: genkitEmbeddings,
	}, nil
}

// Register is part of the `ai.Embedder` interface for use in Genkit flows,
// but is not used by the MangleKit builder.
func (e *GoogleEmbedder) Register(r api.Registry) {
	//TODO: implement me
	panic("implement me")
}
