package google

import (
	"context"
	"fmt"
	"os"
	"strings"

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
		wiredName, err := Init(ctx, g, apiKey, modelName, c.Logger())
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
	cfg, err := ConfigFromOpts(opts)
	if err != nil {
		return nil, err
	}
	actionName, _ := opts["_action_name"].(string)
	return EnableWithConfig(cfg, actionName), nil
}

// ConfigFromOpts maps provider options (mangle.yaml / WithProviderConfig) onto
// a Config, applying the same env fallbacks the code-first path uses.
//
// Recognized keys: api_key, model, platform, project, location, api_version,
// base_url. Before this, project/location/api_version were unreachable from
// YAML — only code-first callers could select Vertex.
func ConfigFromOpts(opts map[string]any) (Config, error) {
	str := func(k string) string {
		v, _ := opts[k].(string)
		return strings.TrimSpace(v)
	}

	cfg := Config{
		APIKey:     str("api_key"),
		ModelName:  str("model"),
		Platform:   str("platform"),
		Project:    str("project"),
		Location:   str("location"),
		APIVersion: str("api_version"),
		BaseURL:    str("base_url"),
	}

	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("GOOGLE_API_KEY")
	}
	if cfg.APIKey == "" {
		return cfg, fmt.Errorf("google factory: missing 'api_key'")
	}
	if cfg.ModelName == "" {
		cfg.ModelName = "gemini-1.5-flash" // Safe default
	}
	return cfg, nil
}

// EnableWithConfig wires the provider from a full Config — Vertex project
// mode, Vertex Express Mode, and endpoint overrides included. Enable(...)
// stays as the minimal code-first entry point.
func EnableWithConfig(cfg Config, actionName string) sdk.ClientOption {
	return func(c *sdk.Client) error {
		ctx := context.Background()
		g := ai.GetGenkit(ctx)

		wiredName, err := InitWithConfig(ctx, g, cfg, c.Logger())
		if err != nil {
			return fmt.Errorf("google.EnableWithConfig failed to init plugin: %w", err)
		}
		action, err := ai.NewGenkitAction(ctx, wiredName)
		if err != nil {
			return fmt.Errorf("google.EnableWithConfig failed to create action: %w", err)
		}
		gen, ok := action.(core.TextGenerator)
		if !ok {
			return fmt.Errorf("google action does not implement TextGenerator")
		}
		c.SetLLM(gen)
		if actionName != "" {
			c.RegisterAction(actionName, action)
		}
		return nil
	}
}

// Auto-register the factory with the SDK
func init() {
	sdk.RegisterProvider("google", Factory)
}
