package openai

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// Enable returns a ClientOption that wires the OpenAI provider.
// usage: client.New(..., openai.Enable("key", "gpt-4o", "", "my_action"))
func Enable(apiKey, modelName, baseURL, actionName string) sdk.ClientOption {
	return func(c *sdk.Client) error {
		ctx := context.Background()

		// 1. Prepare Config
		cfg := Config{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}

		// 2. Call Internal Wiring (from plugin.go)
		// OpenAI's Init registers the model and returns error if failed.
		// FIX: Use global Genkit instance
		g := ai.GetGenkit(ctx)
		if err := Init(g, modelName, cfg); err != nil {
			return fmt.Errorf("openai.Enable failed to init plugin: %w", err)
		}

		// 3. Create Mangle Action Adapter
		// Convention: OpenAI plugin registers model as "openai/{modelName}"
		fullModelName := "openai/" + modelName
		action, err := ai.NewGenkitAction(ctx, fullModelName)
		if err != nil {
			return fmt.Errorf("openai.Enable failed to create action: %w", err)
		}

		// 4. Inject into Client
		if gen, ok := action.(core.TextGenerator); ok {
			c.SetLLM(gen)
		} else {
			return fmt.Errorf("openai action does not implement TextGenerator")
		}

		if actionName != "" {
			c.RegisterAction(actionName, action)
		}

		return nil
	}
}

// Factory implements the sdk.ProviderFactory interface.
func Factory(opts map[string]any) (sdk.ClientOption, error) {
	apiKey, _ := opts["api_key"].(string)
	baseURL, _ := opts["base_url"].(string)
	model, _ := opts["model"].(string)

	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if apiKey == "" && baseURL == "" {
		return nil, fmt.Errorf("openai factory: missing 'api_key' (or 'base_url' for local setup)")
	}
	if model == "" {
		model = "gpt-4o"
	}

	actionName, _ := opts["_action_name"].(string)

	return Enable(apiKey, model, baseURL, actionName), nil
}

// Auto-register the factory with the SDK
func init() {
	sdk.RegisterProvider("openai", Factory)
}
