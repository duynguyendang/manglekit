package embedder

import (
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// NewGoogleEmbedder creates a new Google AI embedder.
func NewGoogleEmbedder(g *genkit.Genkit, modelName string) ai.Embedder {
	return googlegenai.GoogleAIEmbedder(g, modelName)
}