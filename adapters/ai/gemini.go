package ai

import (
	"context"
	"os"
	"sync"

	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

type geminiAdapter struct {
	gk    *genkit.Genkit
	model ai.Model
}

var (
	gk       *genkit.Genkit
	initOnce sync.Once
)

// NewGemini initializes the Google AI plugin and returns a TextGenerator using the specified model.
func NewGemini(ctx context.Context, apiKey string, modelName string) (sdk.TextGenerator, error) {
	initOnce.Do(func() {
		// Set API Key if provided, otherwise rely on ENV
		if apiKey != "" {
			if os.Getenv("GOOGLE_API_KEY") == "" {
				os.Setenv("GOOGLE_API_KEY", apiKey)
			}
			if os.Getenv("GEMINI_API_KEY") == "" {
				os.Setenv("GEMINI_API_KEY", apiKey)
			}
		}

		// Create GoogleAI Plugin instance manually
		gai := &googlegenai.GoogleAI{
			APIKey: apiKey,
		}

		// Initialize Genkit with the plugin using proper Option
		gk = genkit.Init(context.Background(), genkit.WithPlugins(gai))
	})

	// Get the model from the initialized Genkit instance.
	// We rely on the implicit configuration of the googlegenai package via environment variables.
	model := googlegenai.GoogleAIModel(gk, modelName)

	return &geminiAdapter{
		gk:    gk,
		model: model,
	}, nil
}

// Complete generates text using the underlying Gemini model.
func (g *geminiAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	req := ai.NewModelRequest(
		nil, // Use default config
		ai.NewUserTextMessage(prompt),
	)

	resp, err := g.model.Generate(ctx, req, nil)
	if err != nil {
		return "", err
	}

	return resp.Text(), nil
}
