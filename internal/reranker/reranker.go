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

// Reranker re-scores documents based on semantic similarity.
type Reranker struct {
	embedder ai.Embedder
}

// New creates a new Reranker.
func New(embedder ai.Embedder) (*Reranker, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}
	return &Reranker{
		embedder: embedder,
	}, nil
}

type scoredDoc struct {
	doc   string
	score float32
}

// Rerank re-scores a list of documents against a query.
func (r *Reranker) Rerank(ctx context.Context, query string, docs []string, cfg types.RerankConfig) ([]string, error) {
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
	queryEmbedding := queryResponse.Embeddings[0]

	var scoredDocs []scoredDoc
	for _, doc := range docs {
		docRequest := &ai.EmbedRequest{
			Input: []*ai.Document{ai.DocumentFromText(doc, nil)},
		}
		docResponse, err := r.embedder.Embed(ctx, docRequest)
		if err != nil {
			log.Printf("failed to embed document, skipping: %v", err)
			continue
		}
		if len(docResponse.Embeddings) == 0 {
			log.Printf("no embedding returned for doc, skipping: %s", doc)
			continue
		}
		docEmbedding := docResponse.Embeddings[0]

		score, err := cosineSimilarity(queryEmbedding.Embedding, docEmbedding.Embedding)
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

func cosineSimilarity(v1, v2 []float32) (float32, error) {
	if len(v1) != len(v2) {
		return 0, fmt.Errorf("vectors must have the same dimension")
	}
	var dotProduct float32
	var normV1, normV2 float32
	for i := 0; i < len(v1); i++ {
		dotProduct += v1[i] * v2[i]
		normV1 += v1[i] * v1[i]
		normV2 += v2[i] * v2[i]
	}
	denominator := math.Sqrt(float64(normV1)) * math.Sqrt(float64(normV2))
	if denominator == 0 {
		return 0, nil
	}
	return float32(float64(dotProduct) / denominator), nil
}