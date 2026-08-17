package ai

// Package ai provides Genkit integration for Manglekit with full middleware support.
//
// Genkit 1.7 Middleware Features
//
// This package now supports all Genkit 1.7 middleware features:
//
// # Retry Middleware
//
// Automatically retries failed model calls with exponential backoff:
//
//	import mkai "github.com/duynguyendang/manglekit/adapters/ai"
//
//	resp, err := gen.Generate(ctx, prompt,
//	    mkai.WithRetry(3), // retry up to 3 times
//	)
//
// # Fallback Middleware
//
// Falls back to alternative models when the primary fails:
//
//	resp, err := gen.Generate(ctx, prompt,
//	    mkai.WithFallback([]ai.ModelRef{
//	        googlegenai.ModelRef("googleai/gemini-flash", nil),
//	        googlegenai.ModelRef("googleai/gemini-pro", nil),
//	    }),
//	)
//
// # Tool Approval Middleware
//
// Requires explicit approval for tool execution:
//
//	resp, err := gen.Generate(ctx, prompt,
//	    mkai.WithToolApproval([]string{"read_file", "query_database"}),
//	)
//
// # Filesystem Middleware
//
// Enables scoped file system access for the model:
//
//	resp, err := gen.Generate(ctx, prompt,
//	    mkai.WithFilesystem("/app/data", true), // allow read/write
//	)
//
// # Datalog Validator Middleware
//
// Validates generation input/output using Datalog rules:
//
//	resp, err := gen.Generate(ctx, prompt,
//	    mkai.WithDatalogValidator(func(ctx, phase, req, resp) error {
//	        // Validate with Datalog engine
//	        return nil
//	    }),
//	)
//
// # Composing Multiple Middleware
//
// All middleware can be composed together:
//
//	resp, err := gen.Generate(ctx, prompt,
//	    mkai.WithRetry(3),
//	    mkai.WithFallback(fallbackModels),
//	    mkai.WithToolApproval(allowedTools),
//	    mkai.WithFilesystem("/app/data", false),
//	    mkai.WithTelemetry(func(ctx, duration, model, in, out) {
//	        metrics.Record(duration, model, in, out)
//	    }),
//	)
//
// # OODA Loop Integration
//
// The OODA loop can be configured to use middleware for LLM-based actions:
//
//	frame := ooda.NewCognitiveFrame(input, intent, taskType).
//	    WithGenerateOptions(
//	        mkai.WithRetry(3),
//	        mkai.WithFallback(fallbackModels),
//	    )
//
// # MCP Tool Approval
//
// MCP tools can require approval before execution:
//
//	loader := mcp.NewLoader(cfg).
//	    WithMiddleware(mkai.WithToolApproval(allowedTools))
//
// The middleware is applied at the generation level, so any tool calls
// made by the model will go through the approval gate.

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
)

// Example: Complete generation with all middleware
type Example struct{}

// GenerateWithFullMiddleware demonstrates using all middleware features.
func (e *Example) GenerateWithFullMiddleware(
	ctx context.Context,
	gen core.TextGenerator,
	prompt string,
) (*core.LLMResponse, error) {
	return gen.Generate(ctx, prompt,
		// Retry failed API calls up to 3 times
		WithRetry(3),

		// Fallback to alternative models
		WithFallback([]ai.ModelRef{
			// Add fallback models here
		}),

		// Require approval for sensitive tools
		WithToolApproval([]string{"read_file"}),

		// Allow model to access files in /app/data (read-only)
		WithFilesystem("/app/data", false),
	)
}
