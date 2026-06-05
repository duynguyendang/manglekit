package google

import (
	"context"
	"fmt"
	"os"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// Init sets up the Google provider using a Proxy Pattern.
// It isolates the Google plugin in a local instance and registers a proxy in the global registry.
func Init(ctx context.Context, globalG *genkit.Genkit, apiKey string, modelName string) (string, error) {
	// 1. Fallback & Validation
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("google provider: API Key is required")
	}

	// 2. Setup Local Sandbox (Isolated Genkit Instance)
	// We init the plugin here to avoid polluting the global state or dealing with version conflicts.
	plugin := &googlegenai.GoogleAI{APIKey: apiKey}
	localG := genkit.Init(context.Background(), genkit.WithPlugins(plugin))
	if localG == nil {
		return "", fmt.Errorf("failed to init local genkit sandbox for google")
	}

	// 3. Lookup Real Model in Sandbox
	// Note: The plugin usually registers models simply by name or with 'googleai/' prefix.
	realModel := genkit.LookupModel(localG, modelName)
	if realModel == nil {
		// Try prefix if direct lookup fails
		realModel = genkit.LookupModel(localG, "googleai/"+modelName)
	}
	if realModel == nil {
		return "", fmt.Errorf("model '%s' not found in local google plugin", modelName)
	}

	// 4. Register Proxy in Global Registry
	// We use a standardized name in the global registry: "googleai/{modelName}"
	globalName := "googleai/" + modelName

	// Avoid re-registering if already exists
	if genkit.LookupModel(globalG, globalName) != nil {
		return globalName, nil
	}

	meta := &ai.ModelOptions{
		Label: modelName,
		Supports: &ai.ModelSupports{
			Multiturn: true, SystemRole: true, Tools: false, Media: false,
		},
	}

	// Define the Proxy
	// This function forwards calls from Global -> Local
	genkit.DefineModel(globalG, globalName, meta,
		func(ctx context.Context, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
			// WORKAROUND: The google plugin rejects standard GenerationConfig
			// (e.g. Temperature, MaxOutputTokens) on proxied models. Clearing
			// the config lets the request through to the real model which
			// applies its own defaults. This should be removed once the
			// upstream Genkit google plugin handles proxied configs correctly.
			req.Config = nil

			// Forward to the real model in the sandbox
			return realModel.Generate(ctx, req, cb)
		},
	)

	return globalName, nil
}
