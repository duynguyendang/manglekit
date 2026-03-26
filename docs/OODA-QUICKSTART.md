# OODA Quick Start Guide

**Get up and running with the OODA SDK in 5 minutes.**

---

## TL;DR

1. Create a `Registry` and register actions
2. Build a `CognitiveFrame` with `Builder`
3. Call `ooda.Run(ctx, frame)`
4. Check the result and `AuditTrail`

---

## 1. Minimal Setup

### Install

```bash
go get github.com/duynguyendang/manglekit/sdk/ooda
```

### Hello World

```go
package main

import (
    "context"
    "fmt"
    "github.com/duynguyendang/manglekit/sdk/ooda"
)

func main() {
    ctx := context.Background()

    // 1. Create action registry
    registry := ooda.NewRegistry()
    registry.MustRegister("greet", func(ctx context.Context, args map[string]interface{}) (string, error) {
        return "Hello, " + args["name"].(string) + "!", nil
    })

    // 2. Build frame
    frame := ooda.NewBuilder().
        WithInput("Greet the user").
        WithRegistry(registry).
        WithMaxRetries(2).
        Build()

    // 3. Run OODA loop
    // Note: Without a Brain, Decide is skipped and no action is dispatched.
    // For this demo, we'd need a Brain that returns a Decision with Action{Name: "greet"}.
    result, err := ooda.Run(ctx, frame)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Result: %v\n", result.ActionResult)
}
```

---

## 2. Core Concepts (30 Seconds Each)

### CognitiveFrame — The State

```go
frame := ooda.NewBuilder().
    WithInput("Generate BRD document").    // What to do
    WithBrain(policyEngine).               // How to decide (Datalog)
    WithRegistry(registry).                // How to execute (Go functions)
    WithMemory(mebAdapter).                // Long-term knowledge
    WithSessionID("sess-001").             // Multi-turn continuity
    WithMaxRetries(3).                     // Retry on failure
    WithTimeout(5 * time.Minute).          // Hard timeout
    Build()
```

### Registry — Map Actions to Go

```go
registry := ooda.NewRegistry()

registry.MustRegister("generate_document", func(ctx context.Context, args map[string]interface{}) (string, error) {
    docType := args["doc_type"].(string)
    // ... generate document logic ...
    return "Generated " + docType, nil
})

registry.MustRegister("validate", func(ctx context.Context, args map[string]interface{}) (string, error) {
    return "Validation passed", nil
})
```

### Brain — Decision Making

Implement the `Brain` interface to connect Datalog:

```go
type MyBrain struct {
    engine *manglekit.PolicyEngine
}

func (b *MyBrain) Evaluate(ctx context.Context, frame *ooda.CognitiveFrame) (*core.Decision, error) {
    // Query Datalog for decision
    results, _ := b.engine.Query("action_for_input(Action, \"" + frame.Input + "\")")
    
    return &core.Decision{
        Outcome: "proceed",
        Action: &core.ActionEnvelope{
            Name:      results[0]["Action"],
            Arguments: map[string]interface{}{"input": frame.Input},
        },
        AuditTrail: &core.AuditTrail{...},
    }, nil
}

func (b *MyBrain) Verify(ctx context.Context, frame *ooda.CognitiveFrame) (*core.AuditTrail, error) {
    // Validate action result against Datalog rules
    return &core.AuditTrail{...}, nil
}

func (b *MyBrain) LoadPolicy(ctx context.Context, rules string) error {
    return b.engine.LoadPolicy(ctx, rules)
}
```

---

## 3. Common Patterns

### Pattern 1: Simple Execute (No Brain)

```go
// Direct registry execution without OODA loop
registry := ooda.NewRegistry()
registry.MustRegister("process", func(ctx context.Context, args map[string]interface{}) (string, error) {
    return "processed: " + args["data"].(string), nil
})

result, err := registry.Execute(ctx, "process", map[string]interface{}{
    "data": "hello",
})
```

### Pattern 2: Full OODA with Brain + Memory

```go
frame := ooda.NewBuilder().
    WithInput(userInput).
    WithBrain(policyEngine).          // Datalog decisions
    WithMemory(mebAdapter).           // MEB long-term memory
    WithRegistry(registry).           // Go actions
    WithSessionID(sessionID).         // Session continuity
    Build()

result, err := ooda.Run(ctx, frame)
if err != nil {
    log.Printf("Failed: %v", err)
    log.Printf("Audit: %s", result.GetAuditSummary())
}
```

### Pattern 3: With Error Recovery

```go
frame := ooda.NewBuilder().
    WithInput(input).
    WithBrain(brain).
    WithRegistry(registry).
    WithMaxRetries(3).                // Up to 3 retries
    WithTimeout(2 * time.Minute).     // Hard timeout
    Build()

result, err := ooda.Run(ctx, frame)
if err != nil {
    log.Printf("Failed after %d retries: %v", result.RetryCount, err)
    log.Printf("Phase durations: %v", result.GetPhaseDurations())
}
```

### Pattern 4: Dispatcher with SafeStop

```go
registry := ooda.NewRegistry()
registry.MustRegister("known_action", myFunc)

// SafeStop is called for unknown actions
ooda.SafeStop = func(ctx context.Context, args map[string]interface{}) (string, error) {
    log.Printf("SafeStop triggered: action=%s", args["action"])
    return "STOPPED", nil
}

dispatcher := ooda.NewDispatcher(registry)
result, err := dispatcher.Dispatch(ctx, "unknown_action", nil)
// Logs: "SOVEREIGN VIOLATION: action 'unknown_action' not found"
// Calls SafeStop, returns "STOPPED"
```

---

## 4. Configuration Templates

### Fast Response (Low Latency)

```go
frame := ooda.NewBuilder().
    WithInput(input).
    WithBrain(brain).
    WithRegistry(registry).
    WithTimeout(30 * time.Second).
    WithMaxRetries(1).
    Build()
```

### High Quality (Multiple Iterations)

```go
frame := ooda.NewBuilder().
    WithInput(input).
    WithBrain(brain).
    WithRegistry(registry).
    WithTimeout(5 * time.Minute).
    WithMaxRetries(5).
    Build()
```

### Production (Balanced)

```go
frame := ooda.NewBuilder().
    WithInput(input).
    WithBrain(brain).
    WithRegistry(registry).
    WithTimeout(2 * time.Minute).
    WithMaxRetries(3).
    Build()
```

---

## 5. Debugging Tips

### Check Phase Durations

```go
result, _ := ooda.Run(ctx, frame)
for phase, dur := range result.GetPhaseDurations() {
    fmt.Printf("  %s: %v\n", phase, dur)
}
fmt.Printf("  total: %v\n", result.TotalDuration())
```

### Inspect Audit Trail

```go
// Human-readable summary
fmt.Println(result.GetAuditSummary())

// Detailed rules
for _, rule := range result.AuditTrail.MatchedRules {
    fmt.Printf("  [%s] %s: %s (from %s)\n", rule.Tier, rule.RuleName, rule.Predicate, rule.SourceFile)
}
```

### Check TransientStore State

```go
// Query session state
facts, _ := frame.TransientStore.ToAtoms(ctx, frame.SessionID)
for _, f := range facts {
    fmt.Printf("  %s.%s = %s\n", f.Subject, f.Predicate, f.Object)
}
```

---

## 6. Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `observe phase failed` | Input validation failed | Check `frame.Input` is set |
| `orient phase failed` | Memory/KnowledgeStore query failed | Verify MEB is initialized |
| `brain evaluation failed` | Datalog query error | Check rules loaded, query syntax |
| `action 'X' not found` | Action not in Registry | Register the action before `Run()` |
| `SafeStop invoked` | Unknown action dispatched | Add action to Registry or set fallback |
| `context deadline exceeded` | Timeout | Increase `WithTimeout()` |

---

## 7. Next Steps

- Read the full [OODA Multi-Agent Guide](./OODA-MULTI-AGENT-GUIDE.md)
- Explore [domain extensions](./DOMAIN-EXTENSION-GUIDE.md)
- Review [Architect Agent](https://github.com/duynguyendang/architect-agent) — reference implementation
- Check [Manglekit CONTEXT.md](./CONTEXT.md) for full architecture

---

**Questions?** Check the [Multi-Agent Guide](./OODA-MULTI-AGENT-GUIDE.md) or open an issue.
