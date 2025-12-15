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

// options holds configuration for the Google provider.
type options struct {
	apiKey string
	model  string
}

// Option is a functional option for configuration.
type Option func(*options)

// WithAPIKey sets the Google Cloud API Key.
func WithAPIKey(key string) Option {
	return func(o *options) {
		o.apiKey = key
	}
}

// WithModel sets the Gemini model name (e.g., "gemini-1.5-flash").
func WithModel(model string) Option {
	return func(o *options) {
		o.model = model
	}
}

// New initializes the Google GenAI TextGenerator using Functional Options.
func New(ctx context.Context, opts ...Option) (core.TextGenerator, error) {
	o := &options{
		model: "gemini-2.5-flash", // Default
	}

	// 1. Load Defaults from Env
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		o.apiKey = key
	}

	// 2. Apply Options
	for _, opt := range opts {
		opt(o)
	}

	// 3. Validation
	if o.apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is missing (set env or use WithAPIKey)")
	}

	// 4. Initialize Genkit Plugin
	gai := &googlegenai.GoogleAI{APIKey: o.apiKey}

	// Initialize the Genkit runtime with the plugin.
	gk := genkit.Init(ctx, genkit.WithPlugins(gai))
	if gk == nil {
		return nil, fmt.Errorf("failed to initialize genkit runtime")
	}

	// 5. Resolve Model
	model := googlegenai.GoogleAIModel(gk, o.model)
	if model == nil {
		return nil, fmt.Errorf("googlegenai model '%s' not found", o.model)
	}

	// 6. Wrap with Generic Adapter
	return ai.NewGenkitAdapter(model, gk), nil
}

// Register installs the Google GenAI provider into the Manglekit SDK registry.
func Register() {
	sdk.RegisterProvider("google", func(ctx context.Context, name string, cfg config.ActionConfig) (core.Action, error) {
		var opts []Option

		// Map Config Options to Functional Options
		if m, ok := cfg.Options["model"].(string); ok {
			opts = append(opts, WithModel(m))
		}
		if k, ok := cfg.Options["api_key"].(string); ok {
			opts = append(opts, WithAPIKey(k))
		}

		gen, err := New(ctx, opts...)
		if err != nil {
			return nil, err
		}
		return ai.NewLLMAction(name, gen)
	})
}
