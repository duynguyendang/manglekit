# OTel Tracing Refactoring - Guard as Transaction Boundary

**Date:** November 2025  
**Author:** Principal Go Engineer & Observability Specialist  
**Status:** ✅ COMPLETED

## Executive Summary

This document describes the refactored OTel tracing architecture for Manglekit, where the **GuardedAction (guard/)** defines the transaction boundary (Parent Span), and the **PolicyEngine (engine/)** defines internal logic steps (Child Spans).

The architecture ensures a strict hierarchical trace structure:

```
Action.MyAction (Guard)
├── Datalog.PreCheck (Engine.Authorize)
├── Function.Execute / Genkit.Call (Inner Action)
└── Datalog.PostCheck (Engine.Validate)
```

## Architecture Overview

### Layers of Responsibility

| Layer | Component | Role | Tracing Responsibility |
|-------|-----------|------|------------------------|
| **Guard** | `GuardedAction` | Transaction boundary wrapper | Parent Span: `Action.*` |
| **Engine** | `PolicyEngine` | Policy enforcement logic | Child Spans: `Datalog.PreCheck`, `Datalog.PostCheck` |
| **Inner Action** | User code / Adapters | Business logic execution | No span creation (passes context through) |
| **Adapters** | `ai/`, `vector/`, `func/` | External system integration | No span creation (passes context through) |

### Span Hierarchy

**Parent Span (Guard):**
- Name: `Action.{action.name}` (e.g., `Action.MyStockCheck`)
- Attributes:
  - `action.name`: The action's metadata name
  - `action.type`: The action's metadata type
- Wraps the entire execution flow
- Records errors at the top level

**Child Span 1 (PreCheck):**
- Name: `Datalog.PreCheck`
- Attributes:
  - `policy.name`: The action's metadata name
  - `policy.type`: The action's metadata type
  - `decision.type`: `"authorize"`
  - `outcome`: `"ALLOWED"` or `"DENIED"`
- Executes before inner action

**Child Span 2 (PostCheck):**
- Name: `Datalog.PostCheck`
- Attributes:
  - `policy.name`: The action's metadata name
  - `policy.type`: The action's metadata type
  - `decision.type`: `"validate"`
  - `outcome`: `"ALLOWED"` or `"DENIED"`
- Executes after inner action

## Implementation Details

### 1. Kernel (manglekit.go)

The `Client` struct initializes and propagates the tracer throughout the system.

```go
type Client struct {
	engine *engine.PolicyEngine
	tracer trace.Tracer
	logger core.Logger
}

// NewClient initializes the Manglekit system
func NewClient(ctx context.Context, policyFile string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		logger: core.NopLogger{},
	}

	// Apply functional options (including WithTracerProvider)
	for _, opt := range opts {
		opt(c)
	}

	// Initialize the PolicyEngine with both tracer and logger
	c.engine = engine.NewWithObservability(c.tracer, c.logger)

	return c, nil
}

// Protect wraps an action with tracing
func (c *Client) Protect(action core.Action) core.Action {
	if c.tracer != nil {
		return guard.NewWithTracer(action, c.engine, c.tracer)
	}
	return guard.New(action, c.engine)
}
```

**Key Points:**
- `tracer` is initialized via `WithTracerProvider` option
- Tracer is passed to both the PolicyEngine and the Guard
- Guard decides whether to create spans based on tracer availability

### 2. Guard (guard/guard.go)

The `GuardedAction` owns the transaction boundary span and coordinates the execution flow.

```go
type GuardedAction struct {
	inner  core.Action
	engine *engine.PolicyEngine
	tracer trace.Tracer
}

// Execute runs the action through the policy engine's checks
func (g *GuardedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// If no tracer is configured, execute without tracing
	if g.tracer == nil {
		return g.executeInternal(ctx, input)
	}

	// Start the main transaction span
	meta := g.inner.Metadata()
	ctx, span := g.tracer.Start(ctx, fmt.Sprintf("Action.%s", meta.Name),
		trace.WithAttributes(
			attribute.String("action.name", meta.Name),
			attribute.String("action.type", meta.Type),
		),
	)
	defer span.End()

	result, err := g.executeInternal(ctx, input)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return core.Envelope{}, err
	}

	span.SetStatus(codes.Ok, "action completed successfully")
	return result, nil
}

// executeInternal contains the actual execution logic
func (g *GuardedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Inject logger into context for downstream access
	ctx = core.LoggerWithContext(ctx, g.engine.Logger())

	// Pre-execution authorization check (creates child span)
	if err := g.engine.Authorize(ctx, g.inner.Metadata(), input); err != nil {
		return core.Envelope{}, fmt.Errorf("authorization failed: %w", err)
	}

	// Execute the inner action (passes context for span continuity)
	result, err := g.inner.Execute(ctx, input)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("action execution failed: %w", err)
	}

	// Post-execution validation check (creates child span)
	validatedResult, err := g.engine.Validate(ctx, g.inner.Metadata(), result)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("validation failed: %w", err)
	}

	return validatedResult, nil
}
```

**Key Points:**
- Guard creates the parent span `Action.*`
- All downstream operations receive the context with the active span
- Child spans are automatically linked to the parent
- Logger is injected into context for all downstream access

### 3. Engine (engine/policy.go)

The `PolicyEngine` creates child spans for specific policy checks.

```go
type PolicyEngine struct {
	tracer trace.Tracer
	logger core.Logger
}

// Authorize performs pre-execution policy checks
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
		span.SetStatus(codes.Error, "authorization denied")
		span.SetAttributes(attribute.String("outcome", "DENIED"))
		span.RecordError(err)
		return err
	}

	span.SetAttributes(attribute.String("outcome", "ALLOWED"))
	return nil
}

// Validate performs post-execution policy checks
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
		span.SetStatus(codes.Error, "validation denied")
		span.SetAttributes(attribute.String("outcome", "DENIED"))
		span.RecordError(err)
		return core.Envelope{}, err
	}

	span.SetAttributes(attribute.String("outcome", "ALLOWED"))
	return result, nil
}
```

**Key Points:**
- PreCheck span name is `Datalog.PreCheck` (not `Datalog.PolicyCheck`)
- PostCheck span name is `Datalog.PostCheck` (distinct from PreCheck)
- Both spans set `outcome` attribute: `"ALLOWED"` or `"DENIED"`
- Errors are recorded in the span with appropriate status

### 4. Adapters (adapters/*)

Adapters (AI, Vector, Function) do NOT create their own spans. They pass the context through unchanged.

```go
// adapters/ai/adapter.go
func (a *LLMAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	prompt, ok := input.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("%w: invalid input type", core.ErrSystemError)
	}

	// Pass context directly to the generator (Genkit handles internal spans)
	resp, err := a.generator.Complete(ctx, prompt)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("llm generation failed: %w", err)
	}

	output := core.NewEnvelope(resp)
	return output, nil
}
```

**Key Points:**
- No `tracer.Start()` calls in adapters
- Context is passed through unchanged
- External systems (Genkit, etc.) handle their own instrumentation

## Testing

### Trace Hierarchy Tests

The `guard/trace_test.go` file contains comprehensive tests validating the trace hierarchy:

1. **TestTraceHierarchy**: Validates the complete span structure
   - Checks for presence of all required spans
   - Verifies span attributes (action.name, action.type, decision.type)
   - Ensures PreCheck and PostCheck have correct decision types

2. **TestTraceHierarchyWithoutTracer**: Validates graceful degradation
   - Execution succeeds even without a tracer
   - No OTel SDK calls are made

3. **TestTraceErrorHandling**: Validates error recording
   - Errors are properly recorded in spans
   - Span status is set to `codes.Error`
   - Error messages are captured in span status description

### Running Tests

```bash
# Run trace tests
go test -v ./guard -run TestTrace

# Run all tests
go test -v ./guard ./engine ./adapters/...
```

## Usage Example

```go
package main

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/core"
)

func main() {
	ctx := context.Background()

	// Setup OTel tracing
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
	defer tp.Shutdown(ctx)

	// Create Manglekit client with tracing
	client, err := manglekit.NewClient(ctx, "policy.dl",
		manglekit.WithTracerProvider(tp),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create an LLM action
	generator := &MyGenerator{} // Your LLM implementation
	llmAction := ai.NewLLMAction("stock-check", generator)

	// Protect the action with policies
	protectedAction := client.Protect(llmAction)

	// Execute the action
	input := core.NewEnvelope("What is the stock price of AAPL?")
	result, err := protectedAction.Execute(ctx, input)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	// The trace will show:
	// Action.stock-check
	// ├── Datalog.PreCheck (outcome: ALLOWED)
	// ├── Function.Execute (Genkit's internal spans)
	// └── Datalog.PostCheck (outcome: ALLOWED)

	log.Printf("Result: %v", result)
}
```

## Configuration

### WithTracerProvider

The `WithTracerProvider` option allows you to configure a custom OTel `TracerProvider`:

```go
import (
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/exporters/jaeger"
)

// Create an exporter
jaegerExporter, err := jaeger.New(...)
if err != nil {
	log.Fatal(err)
}

// Create a tracer provider
tp := trace.NewTracerProvider(
	trace.WithBatcher(jaegerExporter),
)

// Pass it to the client
client, err := manglekit.NewClient(ctx, "policy.dl",
	manglekit.WithTracerProvider(tp),
)
```

### Default Behavior (No Tracing)

If no `WithTracerProvider` option is provided, the client operates without tracing:

```go
// No tracing
client, err := manglekit.NewClient(ctx, "policy.dl")

// Execution proceeds normally, but no spans are created
```

## Verification Checklist

- ✅ Guard creates parent span `Action.*`
- ✅ Authorize creates child span `Datalog.PreCheck`
- ✅ Validate creates child span `Datalog.PostCheck`
- ✅ Child spans have `decision.type` attribute
- ✅ All spans have `outcome` attribute
- ✅ Errors are recorded in spans
- ✅ Adapters pass context through without creating spans
- ✅ Graceful degradation when no tracer is configured
- ✅ All existing tests pass
- ✅ New trace hierarchy tests pass

## Known Limitations

1. **Nested GuardedActions**: If a GuardedAction's inner action is itself a GuardedAction, each creates its own parent span. This is by design—nested governance layers should be visible in traces.

2. **Context Propagation**: Ensure that all downstream systems respect the trace context. External systems (Genkit, LLMs, etc.) should be configured to propagate OTel context.

3. **Sampling**: Apply OTel sampling policies at the TracerProvider level, not at individual span creation points.

## Future Enhancements

1. **Custom Span Names**: Allow actions to specify custom span names via metadata
2. **Additional Attributes**: Record policy decision reasons in span attributes
3. **Metrics Integration**: Emit metrics for policy decision outcomes
4. **Trace Export Configuration**: Provide convenience helpers for common export backends (Jaeger, Datadog, etc.)

## References

- [OpenTelemetry Go Documentation](https://pkg.go.dev/go.opentelemetry.io/otel)
- [OTel Tracing Specification](https://opentelemetry.io/docs/concepts/signals/traces/)
- [Guard Implementation](../guard/guard.go)
- [Engine Implementation](../engine/policy.go)
- [Trace Tests](../guard/trace_test.go)
