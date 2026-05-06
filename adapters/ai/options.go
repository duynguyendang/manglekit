package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/plugins/middleware"
)

// GenerationOption is a functional option for configuring generation with middleware.
type GenerationOption func(*GenerationSettings)

// GenerationSettings holds all generation configuration including middleware.
type GenerationSettings struct {
	Config     *core.GenerationConfig
	Middleware *MiddlewareConfig
}

// WithGenerationConfig sets the base generation config.
func WithGenerationConfig(cfg *core.GenerationConfig) GenerationOption {
	return func(s *GenerationSettings) {
		s.Config = cfg
	}
}

// WithRetryMiddleware enables retry middleware with default settings.
func WithRetryMiddleware(maxRetries int) GenerationOption {
	return func(s *GenerationSettings) {
		if s.Middleware == nil {
			s.Middleware = &MiddlewareConfig{}
		}
		s.Middleware.Retry = &middleware.Retry{
			MaxRetries:     maxRetries,
			InitialDelayMs: 1000,
			MaxDelayMs:     60000,
		}
	}
}

// WithFallbackMiddleware enables fallback to alternative models.
func WithFallbackMiddleware(models ...ai.ModelRef) GenerationOption {
	return func(s *GenerationSettings) {
		if s.Middleware == nil {
			s.Middleware = &MiddlewareConfig{}
		}
		s.Middleware.Fallback = &middleware.Fallback{
			Models: models,
		}
	}
}

// WithToolApprovalMiddleware requires approval for tool execution.
func WithToolApprovalMiddleware(allowedTools ...string) GenerationOption {
	return func(s *GenerationSettings) {
		if s.Middleware == nil {
			s.Middleware = &MiddlewareConfig{}
		}
		s.Middleware.ToolApproval = &middleware.ToolApproval{
			AllowedTools: allowedTools,
		}
	}
}

// WithFilesystemMiddleware enables scoped filesystem access.
func WithFilesystemMiddleware(rootDir string, allowWrite bool) GenerationOption {
	return func(s *GenerationSettings) {
		if s.Middleware == nil {
			s.Middleware = &MiddlewareConfig{}
		}
		s.Middleware.Filesystem = &middleware.Filesystem{
			RootDir:          rootDir,
			AllowWriteAccess: allowWrite,
		}
	}
}

// WithDatalogValidator adds Datalog validation middleware.
func WithDatalogValidator(validator func(ctx context.Context, phase string, req *ai.ModelRequest, resp *ai.ModelResponse) error) GenerationOption {
	return func(s *GenerationSettings) {
		if s.Middleware == nil {
			s.Middleware = &MiddlewareConfig{}
		}
		s.Middleware.CustomMiddlewares = append(s.Middleware.CustomMiddlewares, &DatalogValidator{
			Validator: validator,
		})
	}
}

// WithTelemetry adds telemetry collection middleware.
func WithTelemetry(onGenerate func(ctx context.Context, duration time.Duration, model string, inputTokens, outputTokens int)) GenerationOption {
	return func(s *GenerationSettings) {
		if s.Middleware == nil {
			s.Middleware = &MiddlewareConfig{}
		}
		s.Middleware.CustomMiddlewares = append(s.Middleware.CustomMiddlewares, &TelemetryMiddleware{
			OnGenerate: onGenerate,
		})
	}
}

// WithLogging adds logging middleware.
func WithLogging(logger core.Logger) GenerationOption {
	return func(s *GenerationSettings) {
		if s.Middleware == nil {
			s.Middleware = &MiddlewareConfig{}
		}
		s.Middleware.CustomMiddlewares = append(s.Middleware.CustomMiddlewares, &LoggingMiddleware{
			Logger: logger,
		})
	}
}

// BuildGenerateOptions converts GenerationSettings to core.GenerateOption slice.
func BuildGenerateOptions(opts ...GenerationOption) []core.GenerateOption {
	settings := &GenerationSettings{}
	for _, opt := range opts {
		opt(settings)
	}

	var result []core.GenerateOption

	if settings.Config != nil {
		result = append(result, func(o *core.GenerationConfig) {
			*o = *settings.Config
		})
	}

	if settings.Middleware != nil {
		result = append(result, WithMiddleware(settings.Middleware))
	}

	return result
}

// ComposeMiddleware creates a single MiddlewareConfig from multiple options.
// This is useful when you want to build middleware incrementally.
func ComposeMiddleware(opts ...GenerationOption) (*MiddlewareConfig, error) {
	settings := &GenerationSettings{}
	for _, opt := range opts {
		opt(settings)
	}

	if settings.Middleware == nil {
		return nil, fmt.Errorf("no middleware configured")
	}

	return settings.Middleware, nil
}
