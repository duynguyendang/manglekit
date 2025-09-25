package retrieval

import (
	"hash/fnv"
	"math"
	"strings"
)

func embedText(text string, dims int) []float64 {
	if dims <= 0 {
		return nil
	}
	vec := make([]float64, dims)
	tokens := strings.Fields(strings.ToLower(text))
	if len(tokens) == 0 {
		return vec
	}
	for _, token := range tokens {
		h := fnv.New32a()
		_, _ = h.Write([]byte(token))
		idx := int(h.Sum32()) % dims
		if idx < 0 {
			idx = -idx
		}
		vec[idx] += 1
	}
	normalize(vec)
	return vec
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot float64
	for i := range a {
		dot += a[i] * b[i]
	}
	return dot
}

func normalize(vec []float64) {
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm == 0 {
		return
	}
	inv := 1 / math.Sqrt(norm)
	for i := range vec {
		vec[i] *= inv
	}
}
