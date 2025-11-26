package engine

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/duynguyendang/manglekit/core"
)

// PolicyEngine handles policy-based authorization and validation.
// It wraps the Datalog execution layer and provides observability through OTel tracing
// and structured logging.
type PolicyEngine struct {
	tracer trace.Tracer
	logger core.Logger
}

// New creates a new PolicyEngine without a tracer or logger (for backward compatibility).
// Prefer NewWithObservability for full observability.
func New() *PolicyEngine {
	return &PolicyEngine{
		logger: core.NopLogger{},
	}
}

// NewWithTracer creates a new PolicyEngine with OTel tracing enabled.
// Deprecated: Use NewWithObservability for full observability support.
func NewWithTracer(tracer trace.Tracer) *PolicyEngine {
	return &PolicyEngine{
		tracer: tracer,
		logger: core.NopLogger{},
	}
}

// NewWithObservability creates a new PolicyEngine with both tracing and logging enabled.
func NewWithObservability(tracer trace.Tracer, logger core.Logger) *PolicyEngine {
	if logger == nil {
		logger = core.NopLogger{}
	}
	return &PolicyEngine{
		tracer: tracer,
		logger: logger,
	}
}

// Logger returns the engine's configured Logger instance.
// This is used by the guard layer to inject the logger into the context.
func (e *PolicyEngine) Logger() core.Logger {
	if e.logger == nil {
		return core.NopLogger{}
	}
	return e.logger
}

// LoadFromPath loads policy rules from a file.
// It verifies the file exists and is readable.
//
// TODO: Replace with actual Mangle Engine parsing logic when ready.
// For now, this method validates file accessibility to prevent runtime errors.
func (e *PolicyEngine) LoadFromPath(path string) error {
	if path == "" {
		return nil
	}

	// Verify the file exists and is readable
	_, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	// Log successful load (optional debug message)
	if e.logger != nil {
		e.logger.Debug("policy loaded from file", "path", path)
	}

	return nil
}

// Authorize performs pre-execution policy checks (Pre-Check).
// It evaluates the Datalog rules to determine if the action is allowed to proceed.
func (e *PolicyEngine) Authorize(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	// If no tracer is configured, execute without tracing
	if e.tracer == nil {
		return e.authorizeInternal(ctx, actionMeta, input)
	}

	// Start a child span for the Datalog pre-check
	ctx, span := e.tracer.Start(ctx, "Datalog.PreCheck",
		trace.WithAttributes(
			attribute.String("policy.name", actionMeta.Name),
			attribute.String("policy.type", actionMeta.Type),
			attribute.String("decision.type", "authorize"),
		),
	)
	defer span.End()

	err := e.authorizeInternal(ctx, actionMeta, input)
	if err != nil {
		// Record the denial in the span
		span.SetStatus(codes.Error, "authorization denied")
		span.SetAttributes(attribute.String("outcome", "DENIED"))
		span.RecordError(err)
		return err
	}

	span.SetAttributes(attribute.String("outcome", "ALLOWED"))
	return nil
}

// authorizeInternal contains the actual authorization logic.
func (e *PolicyEngine) authorizeInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	// TODO: Implement actual Datalog policy evaluation
	// For now, all actions are allowed
	return nil
}

// Validate performs post-execution policy checks (Post-Check).
// It evaluates the Datalog rules to validate and potentially transform the output.
func (e *PolicyEngine) Validate(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	// If no tracer is configured, execute without tracing
	if e.tracer == nil {
		return e.validateInternal(ctx, actionMeta, output)
	}

	// Start a child span for the Datalog post-check
	ctx, span := e.tracer.Start(ctx, "Datalog.PostCheck",
		trace.WithAttributes(
			attribute.String("policy.name", actionMeta.Name),
			attribute.String("policy.type", actionMeta.Type),
			attribute.String("decision.type", "validate"),
		),
	)
	defer span.End()

	result, err := e.validateInternal(ctx, actionMeta, output)
	if err != nil {
		// Record the denial in the span
		span.SetStatus(codes.Error, "validation denied")
		span.SetAttributes(attribute.String("outcome", "DENIED"))
		span.RecordError(err)
		return core.Envelope{}, err
	}

	span.SetAttributes(attribute.String("outcome", "ALLOWED"))
	return result, nil
}

// validateInternal contains the actual validation logic.
func (e *PolicyEngine) validateInternal(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	// TODO: Implement actual Datalog policy evaluation
	// For now, all outputs are valid and passed through unchanged
	return output, nil
}
