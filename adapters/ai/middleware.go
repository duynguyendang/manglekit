package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/middleware"
)

// MiddlewareConfig holds genkit middleware settings for a generation call.
// Use this with GenerateOption to configure per-call middleware.
type MiddlewareConfig struct {
	// Retry configures automatic retry with exponential backoff.
	Retry *middleware.Retry

	// Fallback configures alternative model fallback on failure.
	Fallback *middleware.Fallback

	// ToolApproval requires explicit approval for tool execution.
	ToolApproval *middleware.ToolApproval

	// Filesystem enables scoped file system access for the model.
	Filesystem *middleware.Filesystem

	// CustomMiddlewares allows injecting arbitrary middleware.
	CustomMiddlewares []ai.Middleware
}

// WithMiddleware is a GenerateOption that injects genkit middleware into generation.
func WithMiddleware(cfg *MiddlewareConfig) core.GenerateOption {
	return func(o *core.GenerationConfig) {
		// Store middleware config in a special key for the adapter to pick up
		if o.Metadata == nil {
			o.Metadata = make(map[string]any)
		}
		o.Metadata["_genkit_middleware"] = cfg
	}
}

// WithRetry enables automatic retry with the specified configuration.
func WithRetry(maxRetries int) core.GenerateOption {
	return WithMiddleware(&MiddlewareConfig{
		Retry: &middleware.Retry{
			MaxRetries:     maxRetries,
			InitialDelayMs: 1000,
			MaxDelayMs:     60000,
		},
	})
}

// WithRetryAndBackoff enables retry with custom backoff parameters.
func WithRetryAndBackoff(maxRetries, initialDelayMs, maxDelayMs int) core.GenerateOption {
	return WithMiddleware(&MiddlewareConfig{
		Retry: &middleware.Retry{
			MaxRetries:     maxRetries,
			InitialDelayMs: initialDelayMs,
			MaxDelayMs:     maxDelayMs,
		},
	})
}

// WithFallback enables model fallback when the primary model fails.
func WithFallback(modelRefs []ai.ModelRef) core.GenerateOption {
	return WithMiddleware(&MiddlewareConfig{
		Fallback: &middleware.Fallback{
			Models: modelRefs,
		},
	})
}

// WithToolApproval requires explicit approval for all tool calls.
func WithToolApproval(allowedTools []string) core.GenerateOption {
	return WithMiddleware(&MiddlewareConfig{
		ToolApproval: &middleware.ToolApproval{
			AllowedTools: allowedTools,
		},
	})
}

// WithFilesystem enables scoped file system access for the model.
func WithFilesystem(rootDir string, allowWrite bool) core.GenerateOption {
	return WithMiddleware(&MiddlewareConfig{
		Filesystem: &middleware.Filesystem{
			RootDir:          rootDir,
			AllowWriteAccess: allowWrite,
		},
	})
}

// WithCustomMiddleware adds arbitrary genkit middleware to the generation.
func WithCustomMiddleware(mw ...ai.Middleware) core.GenerateOption {
	return WithMiddleware(&MiddlewareConfig{
		CustomMiddlewares: mw,
	})
}

// buildMiddlewareOptions converts MiddlewareConfig to ai.GenerateOption slices.
func buildMiddlewareOptions(cfg *MiddlewareConfig) []ai.GenerateOption {
	var opts []ai.GenerateOption

	if cfg == nil {
		return opts
	}

	if cfg.Retry != nil {
		opts = append(opts, ai.WithUse(cfg.Retry))
	}

	if cfg.Fallback != nil {
		opts = append(opts, ai.WithUse(cfg.Fallback))
	}

	if cfg.ToolApproval != nil {
		opts = append(opts, ai.WithUse(cfg.ToolApproval))
	}

	if cfg.Filesystem != nil {
		opts = append(opts, ai.WithUse(cfg.Filesystem))
	}

	for _, mw := range cfg.CustomMiddlewares {
		opts = append(opts, ai.WithUse(mw))
	}

	return opts
}

// extractMiddlewareConfig retrieves middleware config from GenerationConfig metadata.
func extractMiddlewareConfig(cfg *core.GenerationConfig) *MiddlewareConfig {
	if cfg == nil || cfg.Metadata == nil {
		return nil
	}
	if v, ok := cfg.Metadata["_genkit_middleware"].(*MiddlewareConfig); ok {
		return v
	}
	return nil
}

// --- Datalog Validator Middleware ---

// DatalogValidator is a custom middleware that validates generation input/output
// using Datalog rules before and after model calls.
type DatalogValidator struct {
	// Validator is called before and after generation to enforce Datalog rules.
	Validator func(ctx context.Context, phase string, req *ai.ModelRequest, resp *ai.ModelResponse) error
}

func (d *DatalogValidator) Name() string { return "manglekit/datalog-validator" }

func (d *DatalogValidator) New(ctx context.Context) (*ai.Hooks, error) {
	return &ai.Hooks{
		WrapGenerate: d.wrapGenerate,
	}, nil
}

func (d *DatalogValidator) wrapGenerate(ctx context.Context, params *ai.GenerateParams, next ai.GenerateNext) (*ai.ModelResponse, error) {
	// Pre-validation: Validate input before generation
	if d.Validator != nil && params.Request != nil {
		if err := d.Validator(ctx, "pre", params.Request, nil); err != nil {
			return nil, fmt.Errorf("datalog pre-validation failed: %w", err)
		}
	}

	// Execute generation
	resp, err := next(ctx, params)
	if err != nil {
		return nil, err
	}

	// Post-validation: Validate output after generation
	if d.Validator != nil && params.Request != nil {
		if err := d.Validator(ctx, "post", params.Request, resp); err != nil {
			return nil, fmt.Errorf("datalog post-validation failed: %w", err)
		}
	}

	return resp, nil
}

// --- Telemetry Middleware ---

// TelemetryMiddleware is a custom middleware that records generation metrics.
type TelemetryMiddleware struct {
	OnGenerate func(ctx context.Context, duration time.Duration, model string, inputTokens, outputTokens int)
}

func (t *TelemetryMiddleware) Name() string { return "manglekit/telemetry" }

func (t *TelemetryMiddleware) New(ctx context.Context) (*ai.Hooks, error) {
	start := time.Now()
	return &ai.Hooks{
		WrapModel: func(ctx context.Context, params *ai.ModelParams, next ai.ModelNext) (*ai.ModelResponse, error) {
			resp, err := next(ctx, params)
			if t.OnGenerate != nil {
				duration := time.Since(start)
				var inputTokens, outputTokens int
				if resp != nil && resp.Usage != nil {
					inputTokens = int(resp.Usage.InputTokens)
					outputTokens = int(resp.Usage.OutputTokens)
				}
				modelName := ""
				if params.Request != nil && len(params.Request.Messages) > 0 {
					// Try to get model name from request metadata or use a default
					modelName = params.Request.Messages[0].Text()
				}
				t.OnGenerate(ctx, duration, modelName, inputTokens, outputTokens)
			}
			return resp, err
		},
	}, nil
}

// --- Logging Middleware ---

// LoggingMiddleware is a custom middleware that logs generation details.
type LoggingMiddleware struct {
	Logger core.Logger
}

func (l *LoggingMiddleware) Name() string { return "manglekit/logging" }

func (l *LoggingMiddleware) New(ctx context.Context) (*ai.Hooks, error) {
	return &ai.Hooks{
		WrapGenerate: func(ctx context.Context, params *ai.GenerateParams, next ai.GenerateNext) (*ai.ModelResponse, error) {
			if l.Logger != nil {
				l.Logger.Info("Starting generation")
			}
			resp, err := next(ctx, params)
			if l.Logger != nil {
				if err != nil {
					l.Logger.Error("Generation failed", "error", err)
				} else {
					l.Logger.Info("Generation completed", "finish_reason", resp.FinishReason)
				}
			}
			return resp, err
		},
	}, nil
}

// Ensure interface compliance
var (
	_ ai.Middleware = (*DatalogValidator)(nil)
	_ ai.Middleware = (*TelemetryMiddleware)(nil)
	_ ai.Middleware = (*LoggingMiddleware)(nil)
)

// InitMiddlewarePlugin registers the middleware plugin with the Genkit instance.
// This enables middleware to be visible in the Dev UI.
func InitMiddlewarePlugin(ctx context.Context, g *genkit.Genkit) error {
	// The middleware plugin is automatically registered when using WithPlugins
	// This is a no-op for now since middleware can be used inline without plugin registration
	return nil
}
