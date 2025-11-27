# Logging Architecture

Manglekit uses a structured logging system based on the `core.Logger` interface. This allows for consistent, vendor-neutral, and configurable logging throughout the library.

## Core Interface

The `core.Logger` interface is designed for structured logging, where every log entry consists of a message and a set of key-value pairs.

```go
type Logger interface {
    Debug(msg string, fields ...any)
    Info(msg string, fields ...any)
    Warn(msg string, fields ...any)
    Error(msg string, fields ...any)
    With(fields ...any) Logger
}
```

### Logging Levels

-   `Debug`: Verbose diagnostic information.
-   `Info`: High-level lifecycle events.
-   `Warn`: Recoverable issues.
-   `Error`: Failures requiring attention.

## Standard Usage

The standard usage pattern is to access the logger from the `context.Context` or inject it during initialization.

### 1. Context-Based Logging (Recommended)

Manglekit automatically injects the logger into the context during the `GuardedAction` lifecycle.

```go
func (c *MyComponent) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
    // Retrieve logger from context
    logger := core.LoggerFromContext(ctx)

    logger.Info("processing request",
        "component", "MyComponent",
        "input_id", input.ID,
    )

    // ... logic ...
}
```

### 2. Constructor Injection

You can also inject the logger when creating your component.

```go
type MyComponent struct {
    logger core.Logger
}

func NewMyComponent(logger core.Logger) *MyComponent {
    return &MyComponent{
        logger: logger.With("component", "MyComponent"),
    }
}
```

## Configuration

### Default Logger (`StdLogger`)

By default, Manglekit uses `internal/logger.StdLogger`, which writes structured logs to `stdout` using the standard library `log` package.

```go
// Initialize with default logger
client, err := manglekit.NewClient(ctx, "policy.dl",
    manglekit.WithLogger(logger.NewStdLogger()),
)
```

### Custom Logger (e.g., Zap)

You can inject any logger that satisfies the `core.Logger` interface. Manglekit provides a built-in adapter for Uber's Zap logger.

```go
import (
    "go.uber.org/zap"
    "github.com/duynguyendang/manglekit/internal/logger"
)

// Create a Zap logger
zapLogger, _ := zap.NewProduction()
sugar := zapLogger.Sugar()

// Adapt it to core.Logger
mkitLogger := logger.NewZapAdapter(sugar)

// Pass to client
client, err := manglekit.NewClient(ctx, "policy.dl",
    manglekit.WithLogger(mkitLogger),
)
```

## Context Propagation

The `core/logger_context.go` package provides helpers for propagating loggers through the context.

-   `LoggerWithContext(ctx, logger)`: Returns a new context with the logger attached.
-   `LoggerFromContext(ctx)`: Retrieves the logger. Returns a `NopLogger` (safe no-op) if none is found.

This ensures that your code never panics due to a missing logger.

## Best Practices

1.  **Use Structured Fields**: Avoid `fmt.Sprintf` in log messages. Use key-value pairs instead.
    *   **Bad**: `logger.Info(fmt.Sprintf("User %s logged in", user))`
    *   **Good**: `logger.Info("user logged in", "user", user)`
2.  **Context Awareness**: Always prefer `LoggerFromContext(ctx)` inside `Execute` methods to inherit request-scoped fields (like Trace ID).
3.  **No Global State**: Do not use global loggers. Always inject or retrieve from context.
