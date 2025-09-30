package retrieval

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"

	"github.com/firebase/genkit/go/ai"
)

func cosineSimilarity32(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		va := float64(a[i])
		vb := float64(b[i])
		dot += va * vb
		normA += va * va
		normB += vb * vb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func localVecDocID(doc *ai.Document) (string, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshal document: %w", err)
	}
	sum := md5.Sum(data)
	return fmt.Sprintf("%02x", sum), nil
}
