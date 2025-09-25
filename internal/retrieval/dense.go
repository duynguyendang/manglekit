// Package retrieval provides dense retrieval functionality.
package retrieval

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/localvec"
	"ndduy.dev/manglekit/internal/types"
)

const (
	collectionName = "manglekit-collection"
)

// Dense implements the DenseRetriever interface.
type Dense struct {
	retriever ai.Retriever
	generator *genkit.Genkit
}

// NewDense creates a new Dense retriever.
func NewDense(ctx context.Context, g *genkit.Genkit, embedder ai.Embedder, docs []*ai.Document) (*Dense, error) {
	retOpts := &ai.RetrieverOptions{
		Label: collectionName,
	}
	docStore, retriever, err := localvec.DefineRetriever(g, collectionName, localvec.Config{Embedder: embedder}, retOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to define retriever: %w", err)
	}

	if err := localvec.Index(ctx, docs, docStore); err != nil {
		return nil, fmt.Errorf("failed to index documents: %w", err)
	}

	return &Dense{
		retriever: retriever,
		generator: g,
	}, nil
}

// Retrieve performs a search using dense vectors.
func (d *Dense) Retrieve(ctx context.Context, query string, cfg types.DenseConfig) ([]string, error) {
	dRequest := ai.DocumentFromText(query, nil)
	response, err := genkit.Retrieve(ctx, d.generator,
		ai.WithRetriever(d.retriever),
		ai.WithDocs(dRequest),
		ai.WithConfig(&localvec.RetrieverOptions{K: cfg.TopK}))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	var results []string
	for _, doc := range response.Documents {
		results = append(results, doc.Content[0].Text)
	}
	return results, nil
}
