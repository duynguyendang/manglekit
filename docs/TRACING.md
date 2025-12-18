# OTel Tracing Architecture

**Status:** Active
**Scope:** Core Kernel (Guard, Engine)
**Last Updated:** 2025-11-27

## Executive Summary

Manglekit uses a hierarchical OpenTelemetry (OTel) tracing architecture where the **GuardedAction** defines the transaction boundary (Parent Span) and the **PolicyEngine** defines the internal logic steps (Child Spans).

This ensures that every protected action produces a consistent trace structure:

```
Action.{ActionName}           [Guard]
├── Datalog.PreCheck          [Engine]
├── {InnerActionExecution}    [Adapter/Genkit]
└── Datalog.PostCheck         [Engine]
```

## Architecture Overview

### Layers of Responsibility

| Layer | Component | Role | Tracing Responsibility |
|-------|-----------|------|------------------------|
| **Guard** | `guard.GuardedAction` | Transaction Boundary | Starts Parent Span `Action.{Name}`. Records overall success/failure. |
| **Engine** | `engine.PolicyEngine` | Policy Logic | Starts Child Spans `Datalog.PreCheck` and `Datalog.PostCheck`. |
| **Adapters** | `adapters/*` | Execution | Pass context through. Do not start spans (rely on internal driver instrumentation). |

### Span Hierarchy & Attributes

#### 1. Parent Span (Guard)
*   **Name:** `Action.{ActionName}` (e.g., `Action.CheckStock`)
*   **Created By:** `guard.GuardedAction`
*   **Attributes:**
    *   `action.name`: The name of the action.
    *   `action.type`: The type of the action (e.g., `llm`, `func`).
    *   `outcome`: `"success"` (on success).
*   **Error Handling:** Records error and sets status if execution fails.

#### 2. Pre-Check Span (Engine)
*   **Name:** `Datalog.PreCheck`
*   **Created By:** `engine.PolicyEngine.Authorize`
*   **Attributes:**
    *   `outcome`: `"ALLOWED"` (if authorized).
*   **Error Handling:** Records error if authorization fails (implicitly DENIED).

#### 3. Post-Check Span (Engine)
*   **Name:** `Datalog.PostCheck`
*   **Created By:** `engine.PolicyEngine.Validate`
*   **Attributes:**
    *   `outcome`: `"ALLOWED"` (if validated).
*   **Error Handling:** Records error if validation fails (implicitly DENIED).

## Implementation Details

### Core Interfaces (`core/tracer.go`)

Manglekit abstracts OTel behind a simple interface to support no-op scenarios and easy testing.

```go
type Tracer interface {
    Start(ctx context.Context, name string) (context.Context, Span)
}

type Span interface {
    End()
    Error(err error)
    SetAttr(key string, value interface{})
}
```

### Guard Implementation (`guard/guard.go`)

The Guard wraps the execution in the parent span.

```go
func (g *GuardedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
    if g.tracer == nil {
        return g.executeInternal(ctx, input)
    }

    meta := g.inner.Metadata()
    // Start Parent Span
    ctx, span := g.tracer.Start(ctx, fmt.Sprintf("Action.%s", meta.Name))
    defer span.End()

    span.SetAttr("action.name", meta.Name)
    span.SetAttr("action.type", meta.Type)

    result, err := g.executeInternal(ctx, input)
    if err != nil {
        span.Error(err)
        return core.Envelope{}, err
    }

    span.SetAttr("outcome", "success")
    return result, nil
}
```

### Engine Implementation (`engine/policy.go`)

The Engine creates child spans for policy checks.

```go
func (e *PolicyEngine) Authorize(ctx context.Context, meta core.ActionMetadata, input core.Envelope) error {
    if e.tracer == nil {
        return e.authorizeInternal(ctx, meta, input)
    }

    // Start Child Span
    ctx, span := e.tracer.Start(ctx, "Datalog.PreCheck")
    defer span.End()

    err := e.authorizeInternal(ctx, meta, input)
    if err != nil {
        span.Error(err) // Records error, implies DENIED
    } else {
        span.SetAttr("outcome", "ALLOWED")
    }
    return err
}
```

## Configuration

Tracing is enabled by passing a `trace.TracerProvider` to the Client.

```go
import (
    "go.opentelemetry.io/otel/sdk/trace"
    "github.com/duynguyendang/manglekit"
)

// Initialize OTel TracerProvider
tp := trace.NewTracerProvider(...)

// Pass to Manglekit
client, err := manglekit.NewClient(ctx, "policy.dl",
    manglekit.WithTracerProvider(tp),
)
```

If `WithTracerProvider` is omitted, Manglekit uses a `NopTracer` with zero overhead.

## Testing

Trace hierarchy is verified in `guard/trace_test.go`. Tests ensure:
1.  Parent/Child relationship is preserved.
2.  Attributes are correctly set.
3.  Errors are propagated to spans.
4.  System degrades gracefully when no tracer is present.
