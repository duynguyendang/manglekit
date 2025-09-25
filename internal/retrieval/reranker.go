package retrieval

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"ndduy.dev/manglekit/internal/types"
)

type mrlReranker struct {
	dims []int
	topK int
}

// NewMRLReranker creates a cosine-based reranker that evaluates multiple embedding dimensions.
func NewMRLReranker(cfg RerankConfig) types.Reranker {
	dims := cfg.MRL.Dimensions
	if len(dims) == 0 {
		dims = []int{512, 768}
	}
	topK := cfg.MRL.TopK
	if topK < 0 {
		topK = 0
	}
	return &mrlReranker{dims: dims, topK: topK}
}

func (r *mrlReranker) Rerank(ctx context.Context, query *types.ExpandedQuery, candidates []*types.Chunk) ([]*types.Chunk, []types.Explanation, error) {
	if len(candidates) == 0 {
		return nil, nil, nil
	}
	_ = ctx

	queryParts := []string{query.NormalizedQuery}
	queryParts = append(queryParts, query.Constraints.Terms.Must...)
	queryParts = append(queryParts, query.Constraints.Terms.Should...)
	queryParts = append(queryParts, flattenEntityValues(query.Entities)...)
	queryText := strings.Join(queryParts, " ")

	queryEmbeddings := make(map[int][]float64, len(r.dims))
	for _, dim := range r.dims {
		if dim <= 0 {
			continue
		}
		queryEmbeddings[dim] = embedText(queryText, dim)
	}
	if len(queryEmbeddings) == 0 {
		queryEmbeddings[512] = embedText(queryText, 512)
	}

	type scored struct {
		chunk   *types.Chunk
		score   float64
		average float64
		detail  []string
	}

	scoredChunks := make([]scored, 0, len(candidates))
	for _, chunk := range candidates {
		if chunk == nil {
			continue
		}
		total := 0.0
		var detail []string
		for dim, qVec := range queryEmbeddings {
			emb := embedText(chunk.Text, dim)
			sim := cosineSimilarity(qVec, emb)
			total += sim
			detail = append(detail, fmt.Sprintf("%dd=%.3f", dim, sim))
		}
		avg := total / float64(len(queryEmbeddings))
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
