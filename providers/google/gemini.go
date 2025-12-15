package google

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// Register installs the Google GenAI provider into the Manglekit SDK registry.
// Users simply call google.Register() in their main function.
func Register() {
	sdk.RegisterProvider("google", func(ctx context.Context, name string, cfg config.ActionConfig) (core.Action, error) {
		// 1. Check API Key
		apiKey := os.Getenv("GOOGLE_GENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GOOGLE_GENAI_API_KEY environment variable is not set")
		}

		// 2. Initialize Genkit Plugin
		// We create a new instance of the plugin configuration.
		gai := &googlegenai.GoogleAI{APIKey: apiKey}

		// Initialize the Genkit runtime with the plugin.
		// Note: genkit.Init returns the instance, not an error.
		gk := genkit.Init(ctx, genkit.WithPlugins(gai))
		if gk == nil {
			return nil, fmt.Errorf("failed to initialize genkit runtime")
		}

		// 3. Resolve Model
		modelName := "gemini-1.5-flash" // Default
		if m, ok := cfg.Options["model"].(string); ok {
			modelName = m
		}

		// In v1.2.0, we get the model from the provider specific function using the genkit instance
		model := googlegenai.GoogleAIModel(gk, modelName)
		if model == nil {
			return nil, fmt.Errorf("googlegenai model '%s' not found", modelName)
		}

		// 4. Wrap with Generic Adapter
		adapter := ai.NewGenkitAdapter(model, gk)

		return ai.NewLLMAction(name, adapter)
	})
}
