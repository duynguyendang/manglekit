package google

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

// Config holds configuration for the Google AI provider.
type Config struct {
	APIKey     string
	ModelName  string
	Project    string // Vertex AI project (enables Vertex mode)
	Location   string // Vertex AI region (e.g. "us-central1", "us", "eu", "global")
	APIVersion string // API version override ("v1", "v1beta"; "v1alpha" for GoogleAI)
	BaseURL    string // endpoint override — a proxy, a regional host, a mock

	// Platform selects the backend explicitly: "googleai" (default) or
	// "vertex". Set it to "vertex" WITHOUT a Project to use Vertex AI
	// Express Mode: API-key auth, no project, no location, no ADC.
	// Leaving it empty keeps the historical rule (Project set => Vertex).
	Platform string
}

// isVertex reports which backend this config selects. An EXPLICIT Platform
// wins; only when it is empty does the historical rule apply (Project set =>
// Vertex), so configs written before Platform existed keep their behavior.
func (c Config) isVertex() bool {
	switch strings.ToLower(strings.TrimSpace(c.Platform)) {
	case "vertex", "vertexai", "vertex_express":
		return true
	case "googleai", "google", "gemini":
		return false
	}
	return c.Project != ""
}

// expressVertex reports the Vertex Express Mode case specifically: API key
// auth with no project or location to resolve.
func (c Config) expressVertex() bool {
	return c.isVertex() && c.Project == ""
}

// Init sets up the Google provider using a Proxy Pattern.
// It isolates the Google plugin in a local instance and registers a proxy in the global registry.
// logger receives a warning whenever the proxied request's GenerationConfig is dropped (see the
// workaround below) so the loss of Temperature/MaxOutputTokens is observable rather than silent.
func Init(ctx context.Context, globalG *genkit.Genkit, apiKey string, modelName string, logger core.Logger) (string, error) {
	return InitWithConfig(ctx, globalG, Config{APIKey: apiKey, ModelName: modelName}, logger)
}

// InitWithConfig sets up the Google provider with full configuration including Vertex AI multi-region support.
func InitWithConfig(ctx context.Context, globalG *genkit.Genkit, cfg Config, logger core.Logger) (string, error) {
	apiKey := cfg.APIKey
	modelName := cfg.ModelName

	if cfg.isVertex() {
		return initVertexAI(ctx, globalG, cfg, logger)
	}

	// Google AI (API key) mode
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("google provider: API Key is required")
	}

	plugin := &googlegenai.GoogleAI{
		APIKey:     apiKey,
		APIVersion: cfg.APIVersion, // empty keeps the genai SDK default
		BaseURL:    cfg.BaseURL,    // proxy / mock endpoint override
	}
	localG := genkit.Init(context.Background(), genkit.WithPlugins(plugin))
	if localG == nil {
		return "", fmt.Errorf("failed to init local genkit sandbox for google")
	}

	realModel := genkit.LookupModel(localG, modelName)
	if realModel == nil {
		realModel = genkit.LookupModel(localG, "googleai/"+modelName)
	}
	if realModel == nil {
		return "", fmt.Errorf("model '%s' not found in local google plugin", modelName)
	}

	globalName := "googleai/" + modelName
	if genkit.LookupModel(globalG, globalName) != nil {
		return globalName, nil
	}

	registerProxy(globalG, globalName, modelName, realModel, logger)
	return globalName, nil
}

// initVertexAI sets up the Vertex AI provider: ADC/project mode (the
// historical path) or Express Mode — API key only, no project, no location,
// no ADC — when Project is empty (genkit >= v1.12).
func initVertexAI(ctx context.Context, globalG *genkit.Genkit, cfg Config, logger core.Logger) (string, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_CLOUD_API_KEY")
	}

	if cfg.expressVertex() {
		if apiKey == "" {
			return "", fmt.Errorf("google provider: Vertex Express Mode needs an API key (cfg.APIKey or GOOGLE_CLOUD_API_KEY) or a Project")
		}
		return registerVertex(ctx, globalG, cfg, logger, &googlegenai.VertexAI{
			APIKey:     apiKey,
			APIVersion: cfg.APIVersion,
			BaseURL:    cfg.BaseURL,
		})
	}

	location := cfg.Location
	if location == "" {
		location = os.Getenv("GOOGLE_CLOUD_LOCATION")
	}
	if location == "" {
		location = os.Getenv("GOOGLE_CLOUD_REGION")
	}
	if location == "" {
		location = "us-central1"
	}

	return registerVertex(ctx, globalG, cfg, logger, &googlegenai.VertexAI{
		ProjectID:  cfg.Project,
		Location:   location,
		APIVersion: cfg.APIVersion,
		BaseURL:    cfg.BaseURL,
	})
}

// registerVertex builds the plugin, resolves the model in a local Genkit
// sandbox and proxies it into the global registry.
func registerVertex(ctx context.Context, globalG *genkit.Genkit, cfg Config, logger core.Logger, plugin *googlegenai.VertexAI) (string, error) {
	localG := genkit.Init(context.Background(), genkit.WithPlugins(plugin))
	if localG == nil {
		return "", fmt.Errorf("failed to init local genkit sandbox for vertex ai")
	}

	realModel := genkit.LookupModel(localG, cfg.ModelName)
	if realModel == nil {
		realModel = genkit.LookupModel(localG, "vertexai/"+cfg.ModelName)
	}
	if realModel == nil {
		return "", fmt.Errorf("model '%s' not found in vertex ai plugin (project=%q, express=%t)", cfg.ModelName, cfg.Project, cfg.expressVertex())
	}

	globalName := "vertexai/" + cfg.ModelName
	if genkit.LookupModel(globalG, globalName) != nil {
		return globalName, nil
	}

	registerProxy(globalG, globalName, cfg.ModelName, realModel, logger)
	return globalName, nil
}

// registerProxy registers a proxy model in the global registry. The Genkit
// streaming callback (cb) is forwarded verbatim to the underlying Google
// model, so streaming generation works through this proxy: the googlegenai
// plugin implements the Genkit Model streaming contract natively, and
// adapters/ai (genkitAdapter.Stream) requests it via ai.WithStreaming.
// TODO(T-013): no provider-level google LLM client exists yet (enhancement
// E6); once one is added, its Stream method should mirror providers/openai.
func registerProxy(globalG *genkit.Genkit, globalName, modelName string, realModel ai.Model, logger core.Logger) {
	meta := &ai.ModelOptions{
		Label: modelName,
		Supports: &ai.ModelSupports{
			Multiturn: true, SystemRole: true, Tools: false, Media: false,
		},
	}

	genkit.DefineModel(globalG, globalName, meta,
		func(ctx context.Context, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
			if req.Config != nil {
				if logger != nil {
					logger.Warn("google proxy dropping GenerationConfig; upstream plugin rejects config on proxied models",
						"model", modelName)
				}
				req.Config = nil
			}
			return realModel.Generate(ctx, req, cb)
		},
	)
}
