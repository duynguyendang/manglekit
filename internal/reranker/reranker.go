// Package reranker provides document reranking functionality.
package reranker

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"

	"github.com/firebase/genkit/go/ai"
	"ndduy.dev/manglekit/internal/types"
)

// MRLReranker re-scores documents based on multi-dimensional semantic similarity.
type MRLReranker struct {
	embedder ai.Embedder
}

// New creates a new MRLReranker.
func New(embedder ai.Embedder) (*MRLReranker, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}
	return &MRLReranker{
		embedder: embedder,
	}, nil
}

type scoredDoc struct {
	doc   string
	score float32
}

// Rerank re-scores a list of documents against a query using multi-dimensional embeddings.
func (r *MRLReranker) Rerank(ctx context.Context, query string, docs []string, cfg types.RerankConfig) ([]string, error) {
	queryRequest := &ai.EmbedRequest{
		Input: []*ai.Document{ai.DocumentFromText(query, nil)},
	}
	queryResponse, err := r.embedder.Embed(ctx, queryRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	if len(queryResponse.Embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	queryEmbedding := queryResponse.Embeddings[0].Embedding

	var docInputs []*ai.Document
	for _, doc := range docs {
		docInputs = append(docInputs, ai.DocumentFromText(doc, nil))
	}

	docRequest := &ai.EmbedRequest{
		Input: docInputs,
	}
	docResponse, err := r.embedder.Embed(ctx, docRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to embed documents: %w", err)
	}
	if len(docResponse.Embeddings) != len(docs) {
		return nil, fmt.Errorf("number of document embeddings does not match number of documents")
	}

	var scoredDocs []scoredDoc
	for i, doc := range docs {
		docEmbedding := docResponse.Embeddings[i].Embedding
		// This is a simplified example of MRL. We are assuming the embedder returns
		// a flat []float32 that is a concatenation of multiple embedding vectors, and
		// we know the dimension of each of those vectors.
		// For this example, let's assume a fixed dimension size of 3 for each vector.
		const vectorDim = 3
		score, err := multiDimCosineSimilarity(queryEmbedding, docEmbedding, vectorDim)
		if err != nil {
			log.Printf("failed to calculate similarity, skipping doc: %v", err)
			continue
		}
		scoredDocs = append(scoredDocs, scoredDoc{
			doc:   doc,
			score: score,
		})
	}

	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	var rerankedDocs []string
	for i := 0; i < len(scoredDocs) && i < cfg.TopK; i++ {
		rerankedDocs = append(rerankedDocs, scoredDocs[i].doc)
	}

	return rerankedDocs, nil
}

func multiDimCosineSimilarity(v1, v2 []float32, vectorDim int) (float32, error) {
	if len(v1) != len(v2) || len(v1)%vectorDim != 0 {
		return 0, fmt.Errorf("vectors must have the same total dimension and be divisible by vectorDim")
	}
	numVectors := len(v1) / vectorDim
	var totalSimilarity float32
	for i := 0; i < numVectors; i++ {
		start := i * vectorDim
		end := start + vectorDim
		subV1 := v1[start:end]
		subV2 := v2[start:end]

		var dotProduct float32
		var normV1, normV2 float32
		for j := 0; j < len(subV1); j++ {
			dotProduct += subV1[j] * subV2[j]
			normV1 += subV1[j] * subV1[j]
			normV2 += subV2[j] * subV2[j]
		}
		denominator := math.Sqrt(float64(normV1)) * math.Sqrt(float64(normV2))
		if denominator > 0 {
			totalSimilarity += float32(float64(dotProduct) / denominator)
		}
	}

	return totalSimilarity / float32(numVectors), nil
}