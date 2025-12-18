package google

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// Enable returns a ClientOption that wires the Google provider.
// usage: client.New(..., google.Enable("key", "gemini-pro", "my_action"))
func Enable(apiKey, modelName, actionName string) sdk.ClientOption {
	return func(c *sdk.Client) error {
		// 1. Prepare Context (Use Background for init-time wiring)
		ctx := context.Background()

		// 2. Get Global Genkit Registry
		g := ai.GetGenkit(ctx)

		// 3. Call Internal Wiring (Proxy Pattern from plugin.go)
		// This handles the local instance creation and bug fixes.
		wiredName, err := Init(ctx, g, apiKey, modelName)
		if err != nil {
			return fmt.Errorf("google.Enable failed to init plugin: %w", err)
		}

		// 4. Create Mangle Action Adapter
		action, err := ai.NewGenkitAction(ctx, wiredName)
		if err != nil {
			return fmt.Errorf("google.Enable failed to create action: %w", err)
		}

		// 5. Inject into Client
		if gen, ok := action.(core.TextGenerator); ok {
			c.SetLLM(gen)
		} else {
			return fmt.Errorf("google action does not implement TextGenerator")
		}

		// Optional: Register as a named action for explicit lookup
		if actionName != "" {
			c.RegisterAction(actionName, action)
		}

		return nil
	}
}

// Factory implements the sdk.ProviderFactory interface for Config-driven loading.
func Factory(opts map[string]any) (sdk.ClientOption, error) {
	apiKey, _ := opts["api_key"].(string)
	model, _ := opts["model"].(string)

	// Validation
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("google factory: missing 'api_key'")
	}
	if model == "" {
		model = "gemini-1.5-flash" // Safe default
	}

	actionName, _ := opts["_action_name"].(string)

	// Delegate to Code-First logic
	return Enable(apiKey, model, actionName), nil
}

// Auto-register the factory with the SDK
func init() {
	sdk.RegisterProvider("google", Factory)
}
