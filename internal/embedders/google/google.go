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
	manglekit.RegisterEmbedder("google", New)
}

// GoogleEmbedder implements both retrieve.Embedder and ai.Embedder interfaces.
type GoogleEmbedder struct {
	client *genai.EmbeddingModel
	model  string
	dim    int
}

// New creates a new GoogleEmbedder with explicit, typed dependencies.
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
		return nil, fmt.Errorf("google: unknown model %s, cannot determine dimension", model)
	}

	return &GoogleEmbedder{
		client: client.EmbeddingModel(model),
		model:  model,
		dim:    dim,
	}, nil
}


// Name returns the model name (for ai.Embedder).
func (e *GoogleEmbedder) Name() string {
	return e.model
}

// Embed generates embeddings for the given request (for ai.Embedder).
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

	// The response needs to be converted to genkit's format.
	var genkitEmbeddings []*ai.Embedding
	for _, emb := range res.Embeddings {
		genkitEmbeddings = append(genkitEmbeddings, &ai.Embedding{Embedding: emb.Values})
	}

	return &ai.EmbedResponse{
		Embeddings: genkitEmbeddings,
	}, nil
}

func (e *GoogleEmbedder) Register(r api.Registry) {
	//TODO: implement me
	panic("implement me")
}