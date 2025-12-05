# Logging Architecture

Manglekit adopts a **"Batteries Included, Zero-Config"** logging philosophy using Go's modern `log/slog` library.

## 1. Zero-Config Experience

By default, Manglekit initializes with a thread-safe, structured logger based on `log/slog` (Standard Library). Developers **do not** need to configure anything.

```go
package main

import "github.com/duynguyendang/manglekit"

func main() {
    ctx := context.Background()

    // 🚀 AUTOMATIC: A default JSON/Text logger is injected automatically.
    // No need to call WithLogger().
    client, _ := manglekit.NewClient(ctx, manglekit.WithPolicyPath("policy.dl"))

    // The client will output system logs immediately:
    // time=2023-10-27T... level=INFO msg="Manglekit client initialized"
}
```

## 2. Core Interface

To remain vendor-neutral, Manglekit uses a minimal interface in `core/logger.go`. The default `slog` implementation satisfies this interface, as do adapters for Zap, Zerolog, etc.

```go
type Logger interface {
    Debug(msg string, fields ...any)
    Info(msg string, fields ...any)
    Warn(msg string, fields ...any)
    Error(msg string, fields ...any)
    With(fields ...any) Logger
}
```

## 3. Usage Patterns

### A. Implicit Logging (Recommended)

Follow the **"Invisible Governance"** principle. Do not clutter your business logic with entry/exit logs. The **Guard Middleware** handles this automatically.

  * **Action Start**: Logs input payload ID, Action Name.
  * **Action End**: Logs Duration (Latency), Output ID, or Error stack trace.
  * **Policy Checks**: Logs decisions (`ALLOW`, `DENY`, `RETRY`) and routing logic.

**Your Code:**

```go
func CheckStock(ctx context.Context, sku string) (int, error) {
    // Pure logic. No logs needed here.
    return 100, nil
}
```

**System Log Output:**

```text
level=INFO msg="Action started" action=check_stock input_id=abc-123
level=INFO msg="Action completed" action=check_stock latency=2ms result=success
```

### B. Explicit Logging (Context-Aware)

If you must log specific business events inside your Action, always retrieve the logger from the `context`. This ensures your logs carry the **Trace ID** and **Span ID**.

```go
import "github.com/duynguyendang/manglekit/core"

func Purchase(ctx context.Context, req Order) (Receipt, error) {
    // Retrieve the contextual logger
    log := core.LoggerFromContext(ctx)

    // This log will automatically inherit metadata from the parent action
    log.Info("processing payment", 
        "amount", req.Amount, 
        "currency", "USD",
    )

    return Receipt{}, nil
}
```

## 4. Customization & Overrides

While the default is robust, you can override it with your own logger (e.g., Uber Zap) for high-performance production environments.

### Injecting a Custom Logger

```go
import (
    "github.com/duynguyendang/manglekit"
    "github.com/duynguyendang/manglekit/adapters/zapadapter"
    "go.uber.org/zap"
)

func main() {
    // 1. Initialize your custom logger
    zapLog, _ := zap.NewProduction()
    
    // 2. Adapt and Inject
    // This overrides the default singleton logger.
    client, _ := manglekit.NewClient(ctx, 
        manglekit.WithLogger(zapadapter.New(zapLog)),
    )
}
```

## 5. Implementation Details (Internal)

  * **Location:** `manglekit/logger_std.go` (Root package).
  * **Engine:** `log/slog` (Go 1.21+).
  * **Pattern:** Lazy-loaded Singleton.
      * The first time `NewClient` or `NewStdLogger` is called, the system initializes the default logger via `sync.Once`.
  * **Format:** Defaults to `TextHandler` (human-readable) for development. Can be configured to `JSONHandler` via environment variables (future roadmap).

## 6. Helper Functions

The root `manglekit` package provides helpers to access the default logger without needing a client instance:

```go
// Create or retrieve the default singleton logger
log := manglekit.NewStdLogger()
log.Info("Application starting...")
```

## Best Practices Checklist

1.  ✅ **Trust the Middleware:** Let Manglekit log the "Start/End" of functions.
2.  ✅ **Use Context:** Always use `core.LoggerFromContext(ctx)` inside Actions.
3.  ✅ **Structure Data:** Use `"key", value` pairs. Never use `fmt.Sprintf` in log messages.
4.  ⛔ **No Globals:** Do not use `log.Println` or global variables. They break the tracing chain.