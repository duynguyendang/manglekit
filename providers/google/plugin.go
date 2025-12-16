package google

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// Init initializes the Google AI plugin (Gemini) and registers the specified model.
// It acts as a bridge, creating a local Genkit instance with the official plugin
// and proxying requests from the global Genkit instance.
func Init(g *genkit.Genkit, modelName string, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("google provider: API Key is required")
	}

	// 1. Initialize a local Genkit instance with the official Google AI plugin.
	// We do this because we cannot easily add plugins to an existing Genkit instance (g)
	// after it has been initialized.
	plugin := &googlegenai.GoogleAI{APIKey: apiKey}
	localG := genkit.Init(context.Background(), genkit.WithPlugins(plugin))
	if localG == nil {
		return fmt.Errorf("failed to initialize local genkit instance for google provider")
	}

	// 2. Lookup the model in the local instance.
	// The plugin usually registers models like "gemini-1.5-flash".
	localModel := googlegenai.GoogleAIModel(localG, modelName)
	if localModel == nil {
		return fmt.Errorf("model '%s' not found in googlegenai plugin", modelName)
	}

	// 3. Register a proxy model in the global Genkit registry (g).
	// We map "googleai/{modelName}" to the local model.
	fullName := "googleai/" + modelName

	meta := &ai.ModelOptions{
		Label: modelName,
		Supports: &ai.ModelSupports{
			Multiturn: true, SystemRole: true, Tools: false, Media: false,
		},
	}

	err := genkit.DefineModel(g, fullName, meta, func(ctx context.Context, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
		// Proxy the generation request to the local model
		return localModel.Generate(ctx, req, cb)
	})

	if err != nil {
		return fmt.Errorf("failed to register proxy model %s: %w", fullName, err)
	}

	return nil
}
