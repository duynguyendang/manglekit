package retrieval

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"

	"ndduy.dev/manglekit/internal/types"
)

type mrlReranker struct {
	dims     []int
	topK     int
	embedder ai.Embedder
}

// NewMRLReranker creates a cosine-based reranker that evaluates multiple embedding dimensions.
func NewMRLReranker(embedder ai.Embedder, cfg RerankConfig) (types.Reranker, error) {
	if embedder == nil {
		return nil, errors.New("embedder is required for reranking")
	}
	dims := cfg.MRL.Dimensions
	if len(dims) == 0 {
		dims = []int{512, 768}
	}
	topK := cfg.MRL.TopK
	if topK < 0 {
		topK = 0
	}
	return &mrlReranker{dims: dims, topK: topK, embedder: embedder}, nil
}

func (r *mrlReranker) Rerank(ctx context.Context, query *types.ExpandedQuery, candidates []*types.Chunk) ([]*types.Chunk, []types.Explanation, error) {
	if len(candidates) == 0 {
		return nil, nil, nil
	}

	queryParts := []string{query.NormalizedQuery}
	queryParts = append(queryParts, query.Constraints.Terms.Must...)
	queryParts = append(queryParts, query.Constraints.Terms.Should...)
	queryParts = append(queryParts, flattenEntityValues(query.Entities)...)
	queryText := strings.Join(queryParts, " ")

	docs := make([]*ai.Document, 0, len(candidates)+1)
	docs = append(docs, ai.DocumentFromText(queryText, nil))
	validChunks := make([]*types.Chunk, 0, len(candidates))
	for _, chunk := range candidates {
		if chunk == nil {
			continue
		}
		docs = append(docs, ai.DocumentFromText(chunk.Text, nil))
		validChunks = append(validChunks, chunk)
	}
	if len(validChunks) == 0 {
		return nil, nil, nil
	}

	embedResp, err := r.embedder.Embed(ctx, &ai.EmbedRequest{Input: docs})
	if err != nil {
		return nil, nil, fmt.Errorf("embed reranker documents: %w", err)
	}
	if len(embedResp.Embeddings) != len(docs) {
		return nil, nil, fmt.Errorf("embedder returned %d vectors for %d documents", len(embedResp.Embeddings), len(docs))
	}

	queryVec := embedResp.Embeddings[0].Embedding
	chunkVectors := embedResp.Embeddings[1:]

	type scored struct {
		chunk   *types.Chunk
		score   float64
		average float64
		detail  []string
	}

	scoredChunks := make([]scored, 0, len(validChunks))
	for i, chunk := range validChunks {
		chunkVec := chunkVectors[i].Embedding
		sim := cosineSimilarity32(queryVec, chunkVec)
		avg := sim

		detail := make([]string, len(r.dims))
		for j, dim := range r.dims {
			detail[j] = fmt.Sprintf("%dd=%.3f", dim, sim)
		}
		if len(detail) == 0 {
			detail = []string{fmt.Sprintf("sim=%.3f", sim)}
		}
		final := 0.7*avg + 0.3*chunk.Score

		clone := *chunk
		if chunk.Metadata != nil {
			clone.Metadata = make(map[string]any, len(chunk.Metadata)+2)
			for k, v := range chunk.Metadata {
				clone.Metadata[k] = v
			}
		} else {
			clone.Metadata = map[string]any{}
		}
		clone.Metadata["rerankScore"] = avg
		clone.Metadata["rerankDetail"] = detail
		clone.Score = final

		scoredChunks = append(scoredChunks, scored{chunk: &clone, score: final, average: avg, detail: detail})
	}

	sort.Slice(scoredChunks, func(i, j int) bool {
		if scoredChunks[i].score == scoredChunks[j].score {
			return scoredChunks[i].chunk.ID < scoredChunks[j].chunk.ID
		}
		return scoredChunks[i].score > scoredChunks[j].score
	})

	topK := r.topK
	if topK == 0 || topK > len(scoredChunks) {
		topK = len(scoredChunks)
	}

	now := time.Now()
	var explanations []types.Explanation
	for idx, sc := range scoredChunks {
		action := "retained"
		reason := fmt.Sprintf("score=%.3f avg=%.3f detail=%s", sc.score, sc.average, strings.Join(sc.detail, ","))
		if idx >= topK {
			action = "dropped"
		}
		explanations = append(explanations, types.Explanation{
			Type:      "rerank",
			Rule:      "mrl",
			Action:    action,
			Reason:    fmt.Sprintf("chunk=%s %s", sc.chunk.ID, reason),
			Timestamp: now,
		})
	}

	reranked := make([]*types.Chunk, 0, topK)
	for i := 0; i < topK; i++ {
		reranked = append(reranked, scoredChunks[i].chunk)
	}
	return reranked, explanations, nil
}
