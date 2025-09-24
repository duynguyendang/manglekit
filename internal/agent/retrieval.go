// Package retrieval implements the hybrid search layer for Manglekit.
// It combines vector and keyword search with metadata filtering.

package agent

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"ndduy.dev/manglekit/internal/types"
)

// RetrieverImpl is the concrete implementation of the Retriever interface.
type RetrieverImpl struct {
	// TODO: Vector index (FAISS or similar)
	// TODO: Keyword index (Bleve)
	mockDocs map[string]*types.Chunk // Placeholder mock data
}

// NewRetriever creates a new Retriever with mock data for development.
func NewRetriever() types.Retriever {
	return &RetrieverImpl{
		mockDocs: map[string]*types.Chunk{
			"d1": {
				ID:        "d1-ch1",
				DocID:     "d1",
				Text:      "PDF export crash in Go app due to renderer error. This affects version 2.1 on Ubuntu systems.",
				Embedding: []float64{0.1, 0.2, 0.3}, // Mock embedding
				Metadata:  map[string]interface{}{"tags": []string{"go", "bug"}, "access": "internal", "version": "2.1"},
				Score:     0.0, // To be computed during search
			},
			"d2": {
				ID:        "d2-ch1",
				DocID:     "d2",
				Text:      "Fix PDF output failure by updating library version to latest stable release.",
				Embedding: []float64{0.4, 0.5, 0.6},
				Metadata:  map[string]interface{}{"tags": []string{"pdf", "fix"}, "access": "public", "version": "latest"},
				Score:     0.0,
			},
			"d3": {
				ID:        "d3-ch1",
				DocID:     "d3",
				Text:      "General app crash troubleshooting guide for debugging common issues.",
				Embedding: []float64{0.7, 0.8, 0.9},
				Metadata:  map[string]interface{}{"tags": []string{"crash", "debug"}, "access": "internal", "version": "all"},
				Score:     0.0,
			},
		},
	}
}

// Search performs hybrid search: keyword + vector, with filters.
func (r *RetrieverImpl) Search(ctx context.Context, q *types.ExpandedQuery, filters map[string]string) ([]*types.Chunk, error) {
	if q == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}

	// Create a map to track unique chunks and their best scores
	chunkScores := make(map[string]*types.Chunk)

	// Mock query embedding for vector search
	mockQueryEmbedding := []float64{0.2, 0.3, 0.4}

	// Process each document
	for _, chunk := range r.mockDocs {
		// Apply metadata filters first
		if !r.passesFilters(chunk, filters) {
			continue
		}

		// Create a copy to avoid modifying original
		candidate := &types.Chunk{
			ID:        chunk.ID,
			DocID:     chunk.DocID,
			Text:      chunk.Text,
			Embedding: chunk.Embedding,
			Metadata:  chunk.Metadata,
			Score:     0.0,
		}

		// Keyword search score (0.0 to 1.0)
		keywordScore := r.calculateKeywordScore(candidate.Text, q.NormalizedQuery)

		// Vector similarity score (0.0 to 1.0)
		vectorScore := cosineSimilarity(mockQueryEmbedding, candidate.Embedding)

		// Hybrid score: weighted combination
		candidate.Score = 0.6*keywordScore + 0.4*vectorScore

		// Only include if above threshold
		if candidate.Score > 0.1 {
			// Generate snippet
			candidate.Snippet = r.generateSnippet(candidate.Text, q.NormalizedQuery)

			// Keep the best score for this chunk
			if existing, exists := chunkScores[candidate.ID]; !exists || candidate.Score > existing.Score {
				chunkScores[candidate.ID] = candidate
			}
		}
	}

	// Convert map to slice and sort by score
	var results []*types.Chunk
	for _, chunk := range chunkScores {
		results = append(results, chunk)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no relevant chunks found for query: %s", q.NormalizedQuery)
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top results (max 10)
	maxResults := 10
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// passesFilters checks if a chunk passes the given metadata filters.
func (r *RetrieverImpl) passesFilters(chunk *types.Chunk, filters map[string]string) bool {
	for filterKey, filterValue := range filters {
		switch filterKey {
		case "access_level":
			if metaAccess, ok := chunk.Metadata["access"].(string); ok {
				if metaAccess != filterValue && filterValue != "all" {
					return false
				}
			}
		case "version":
			if metaVersion, ok := chunk.Metadata["version"].(string); ok {
				if metaVersion != filterValue && metaVersion != "all" && filterValue != "all" {
					return false
				}
			}
		case "tags":
			if metaTags, ok := chunk.Metadata["tags"].([]string); ok {
				found := false
				for _, tag := range metaTags {
					if strings.Contains(strings.ToLower(tag), strings.ToLower(filterValue)) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		}
	}
	return true
}

// calculateKeywordScore computes a simple keyword matching score.
func (r *RetrieverImpl) calculateKeywordScore(text, query string) float64 {
	textLower := strings.ToLower(text)
	queryLower := strings.ToLower(query)

	// Simple exact match
	if strings.Contains(textLower, queryLower) {
		return 1.0
	}

	// Word-level matching
	queryWords := strings.Fields(queryLower)
	textWords := strings.Fields(textLower)

	if len(queryWords) == 0 {
		return 0.0
	}

	matches := 0
	for _, qWord := range queryWords {
		for _, tWord := range textWords {
			if strings.Contains(tWord, qWord) || strings.Contains(qWord, tWord) {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(queryWords))
}

// generateSnippet creates a snippet around the query terms.
func (r *RetrieverImpl) generateSnippet(text, query string) string {
	const maxSnippetLength = 150

	if len(text) <= maxSnippetLength {
		return text
	}

	// Find the query in the text
	queryLower := strings.ToLower(query)
	textLower := strings.ToLower(text)

	index := strings.Index(textLower, queryLower)
	if index == -1 {
		// Query not found, return beginning
		if len(text) > maxSnippetLength {
			return text[:maxSnippetLength] + "..."
		}
		return text
	}

	// Try to center the snippet around the query
	start := index - maxSnippetLength/4
	if start < 0 {
		start = 0
	}

	end := start + maxSnippetLength
	if end > len(text) {
		end = len(text)
		start = end - maxSnippetLength
		if start < 0 {
			start = 0
		}
	}

	snippet := text[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}

	return snippet
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (normA * normB)
}
