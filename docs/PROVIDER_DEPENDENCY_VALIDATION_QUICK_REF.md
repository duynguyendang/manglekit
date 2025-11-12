# Provider Dependency Validation — Quick Reference

## What is It?

The builder now **automatically validates** that providers have their required environment variables set when you configure them via `WithOptions()`.

## When Does It Run?

- **At Configuration Time**: When you call `builder.WithOptions("provider_name", opts)`
- **Reported at Build Time**: When you call `builder.Build(ctx, orchestrator, updatable)`

## How Do I Use It?

### Normal Usage (No Changes Required)

```go
import (
    "context"
    "github.com/duynguyendang/manglekit/sdk"
    llm "github.com/duynguyendang/manglekit/internal/providers/llm/openai"
)

// Set your API key
os.Setenv("OPENAI_API_KEY", "sk-...")

ctx := context.Background()
builder, _ := sdk.NewBuilder(ctx)

// These calls are automatically validated
builder.
    WithOptions("openai", &llm.Options{Model: "gpt-4"})

// If validation passes, build succeeds
orch, _, err := builder.Build(ctx, "sandwich", "")
if err != nil {
    log.Fatal(err)  // Would fail if OPENAI_API_KEY was missing
}
```

## What Gets Validated?

| Provider | Requires |
|----------|----------|
| **google** (LLM) | GOOGLE_API_KEY |
| **openai** (LLM) | OPENAI_API_KEY |
| **bm25** (Retriever) | (none) |
| **hybrid** (Retriever) | (none) |
| **dense** (Retriever) | (none) |
| **mangle** (Rules) | (none) |
| **inmemory** (State) | (none) |
| **sandwich** (Orchestrator) | (none) |

## If Validation Fails

### Error Message

```
Build failed:
  missing required environment variable for llm provider 'google': GOOGLE_API_KEY
```

### How to Fix

1. **Set the missing environment variable**:
   ```bash
   export GOOGLE_API_KEY="AIzaSy..."
   ```

2. **Or use a different provider** that has its key set:
   ```go
   builder.WithOptions("openai", &llm.Options{Model: "gpt-4"})
   ```

3. **Or skip that component** (if optional):
   ```go
   builder.
       WithOptions("bm25", &retriever.Options{Path: "docs"}).
       WithOptions("sandwich", &orchestrator.Options{
           Retriever: "bm25",
           LLM: "",  // Skip LLM
       })
   ```

## Extending for Custom Providers

```go
// In your code, after creating the registry:
registry := core.NewProviderDependencyRegistry()

registry.Register(&core.ProviderDependency{
    Name:            "anthropic",
    Kind:            core.KindLLM,
    RequiredEnvVars: []string{"ANTHROPIC_API_KEY"},
    Description:     "Anthropic Claude LLM provider",
})

// Now validation will check ANTHROPIC_API_KEY
```

## Testing Your Setup

```bash
# With validation
export GOOGLE_API_KEY="your-key"
go run ./your-app  # ✅ Should work

# Without validation (to see error)
unset GOOGLE_API_KEY
go run ./your-app  # ❌ Will show validation error
```

## Is It Breaking?

**No.** Existing code continues to work. This is a new, non-breaking feature.

## Does It Affect Performance?

**No.** Validation is a simple environment variable check (at configuration time, not per-request).

## Can I Disable It?

Currently, no. Validation always runs. If you need this feature in the future, it can be added via a builder option.

## Where's the Full Documentation?

See `docs/PROVIDER_DEPENDENCY_VALIDATION.md` for:
- Detailed usage examples
- Error handling patterns
- API reference
- Architecture details
- Known limitations and future enhancements

## Summary

| What | Answer |
|------|--------|
| **Validates** | Required environment variables for providers |
| **When** | At `WithOptions()` time (reported at `Build()`) |
| **Error Handling** | Clear messages with provider name and required vars |
| **Backward Compatible** | ✅ Yes |
| **Performance Impact** | ✅ None (simple env var checks) |
| **Extensible** | ✅ Yes (via registry) |
