package vector

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/duynguyendang/manglekit-wip/core"
)

// SimpleStore is a thread-safe, in-memory vector store using Cosine Similarity.
// For <10k items, this is faster than HNSW overhead.
type SimpleStore struct {
	mu       sync.RWMutex
	embedder core.Embedder
	vectors  map[string][]float32
	docs     map[string]string
}

func NewSimpleStore(embedder core.Embedder) *SimpleStore {
	return &SimpleStore{
		embedder: embedder,
		vectors:  make(map[string][]float32),
		docs:     make(map[string]string),
	}
}

func (s *SimpleStore) Upsert(ctx context.Context, id string, content string) error {
	vec, err := s.embedder.Embed(ctx, content)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.vectors[id] = vec
	s.docs[id] = content
	return nil
}

type searchResult struct {
	id    string
	score float32
}

func (s *SimpleStore) Search(ctx context.Context, query string, topK int) ([]string, error) {
	qVec, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []searchResult
	for id, vec := range s.vectors {
		score := cosineSimilarity(qVec, vec)
		results = append(results, searchResult{id: id, score: score})
	}

	// Sort descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	var ids []string
	for i := 0; i < len(results) && i < topK; i++ {
		ids = append(ids, results[i].id)
	}
	return ids, nil
}

func (s *SimpleStore) Get(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	content, ok := s.docs[id]
	if !ok {
		return "", fmt.Errorf("document not found: %s", id)
	}
	return content, nil
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) { return 0 }
	var dot, magA, magB float32
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 { return 0 }
	return dot / (float32(math.Sqrt(float64(magA))) * float32(math.Sqrt(float64(magB))))
}
