package embedder

import (
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"ndduy.dev/manglekit/internal/types"
)

// New creates a new embedder based on the provider specified in the config.
func New(cfg types.EmbedderConfig, g *genkit.Genkit) (ai.Embedder, error) {
	switch cfg.Provider {
	case "google":
		return NewGoogleEmbedder(g, cfg.Model), nil
	case "openai":
		return NewOpenAIEmbedder(cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unknown embedder provider: %s", cfg.Provider)
	}
}