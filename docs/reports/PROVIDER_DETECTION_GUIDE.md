# LLM Provider Detection Guide

**Last Updated:** 2025-11-12  
**Status:** Implemented

## Overview

The Manglekit SDK now uses **intelligent provider detection** in `InitGenkitWithProviders()` to determine which LLM provider plugins to register based on available environment variables.

Instead of forcing all plugins to initialize (which would fail if API keys are missing), the function now:

1. **Checks for each provider's API key** in the environment
2. **Only registers plugins for which API keys are available**
3. **Falls back to basic genkit** if no API keys are detected

This enables:
- ✅ **Flexible deployments**: Use Google in one environment, OpenAI in another
- ✅ **No forced dependencies**: If you don't have a provider's API key, its plugin won't be loaded
- ✅ **Better error messages**: Errors happen at actual LLM usage, not at SDK init
- ✅ **Graceful degradation**: Apps without LLM components work even without any API keys

---

## Current Behavior

### Environment Variable Detection

**Currently Supported:**
- `GOOGLE_API_KEY` → Enables Google GenAI plugin

**Planned Support:**
- `OPENAI_API_KEY` → Will enable OpenAI plugin (commented in code, ready to add)

### Detection Logic

```go
func InitGenkitWithProviders(ctx context.Context) *genkit.Genkit {
    // Register Google plugin if GOOGLE_API_KEY is set
    if os.Getenv("GOOGLE_API_KEY") != "" {
        return genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}))
    }

    // If no recognized API key found, return basic genkit
    return genkit.Init(ctx)
}
```

---

## Scenarios & Behavior

### Scenario 1: Using Google LLM Only

```bash
export GOOGLE_API_KEY="your-api-key"
unset OPENAI_API_KEY
```

**Result:** Google plugin registered ✅

```
InitGenkitWithProviders() →
  ├─ Detects GOOGLE_API_KEY ✓
  └─ Returns: genkit with GoogleAI plugin
```

**Example:**
```yaml
# config.yaml
orchestrator:
  kind: "sandwich"
  llm: "google"  # Will work!
```

---

### Scenario 2: Using OpenAI Only

```bash
unset GOOGLE_API_KEY
export OPENAI_API_KEY="sk-..."
```

**Result:** Currently falls back to basic genkit (OpenAI support pending)

```
InitGenkitWithProviders() →
  ├─ GOOGLE_API_KEY not set ✗
  └─ Returns: basic genkit (no plugins)
```

**Note:** When OpenAI support is added, it will work like Google:
```go
if os.Getenv("OPENAI_API_KEY") != "" {
    return genkit.Init(ctx, genkit.WithPlugins(&openai.OpenAI{}))
}
```

---

### Scenario 3: Both API Keys Available

```bash
export GOOGLE_API_KEY="your-api-key"
export OPENAI_API_KEY="sk-..."
```

**Result:** Both plugins will be registered (when OpenAI support is added)

```
InitGenkitWithProviders() →
  ├─ GOOGLE_API_KEY found → GoogleAI added
  ├─ OPENAI_API_KEY found → OpenAI added (future)
  └─ Returns: genkit with both plugins
```

---

### Scenario 4: No API Keys Available

```bash
unset GOOGLE_API_KEY
unset OPENAI_API_KEY
```

**Result:** Basic genkit (no LLM plugins)

```
InitGenkitWithProviders() →
  ├─ No recognized API keys found
  └─ Returns: basic genkit (graceful degradation)
```

**What Works:**
- ✅ Non-LLM orchestrators (using only retrievers, rules, rerankers)
- ✅ Config-based loading without LLM components

**What Fails:**
- ❌ Configs with `llm: "google"` (no plugin to handle it)
- ❌ Programmatic LLM component creation (missing plugin)

---

## How This Affects Different SDK Entry Points

### Config-Based API: `sdk.Load()`

Uses `InitGenkitBasic()` → **No plugin detection**

```go
func FromConfig(ctx context.Context, data []byte, registry *Registry) {
    g := InitGenkitBasic(ctx)  // Always basic genkit
    // ...
}
```

**Implication:** Config loading works regardless of API keys. Plugins are detected when needed:
- If config uses LLM: Plugin detection happens at component build time
- If config doesn't use LLM: No plugins needed

### Programmatic API: `sdk.NewBuilder()`

Uses `InitGenkitWithProviders()` → **Intelligent plugin detection**

```go
func NewBuilder(ctx context.Context) (ProgrammaticBuilder, error) {
    g := manglekit.InitGenkitWithProviders(ctx)  // Smart detection
    // ...
}
```

**Implication:** Builder startup behavior depends on available API keys:
- With Google API key: Google plugin ready immediately
- Without any API keys: Still builds (will fail when LLM component actually used)

---

## Decision Tree

```
Is this a Config-Based API?
├─ YES (Load, LoadWithRegistry, FromConfig)
│   └─ Uses InitGenkitBasic() → No plugin checks
│       └─ Plugins loaded dynamically if needed
│
└─ NO (Programmatic API, NewBuilder)
    └─ Uses InitGenkitWithProviders() → Intelligent detection
        ├─ GOOGLE_API_KEY set?
        │   └─ YES → Load Google plugin
        └─ Other API keys set? (future)
            └─ YES → Load respective plugin
```

---

## Testing Implications

### Tests Without LLM Components (e.g., BM25 tests)

**Before:** Needed `GOOGLE_API_KEY` even though no LLM was used
**After:** Works without any API keys

```bash
# This now works without GOOGLE_API_KEY
$ unset GOOGLE_API_KEY && go test ./internal/providers/retrievers/bm25 -v
# Result: ✅ PASS
```

### Tests With LLM Components

**Requirement:** Appropriate API key must be set

```bash
# Config-based LLM test
$ export GOOGLE_API_KEY="test-key" && go test ./internal/providers/llm

# Programmatic LLM usage
$ export GOOGLE_API_KEY="test-key" && go run ./examples/01-programmatic-setup
```

---

## Adding New Provider Support

### Current Status
- ✅ Google GenAI: Fully implemented
- ⏳ OpenAI: Code ready, waiting for import availability

### Adding OpenAI Support

**Step 1:** Import the plugin package
```go
import (
    "github.com/firebase/genkit/go/plugins/openai"
)
```

**Step 2:** Add detection in `InitGenkitWithProviders()`
```go
func InitGenkitWithProviders(ctx context.Context) *genkit.Genkit {
    if os.Getenv("GOOGLE_API_KEY") != "" {
        return genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}))
    }
    
    if os.Getenv("OPENAI_API_KEY") != "" {
        return genkit.Init(ctx, genkit.WithPlugins(&openai.OpenAI{}))
    }
    
    return genkit.Init(ctx)
}
```

**Step 3:** Test
```bash
export OPENAI_API_KEY="sk-..." && go test ./...
```

### Adding Multi-Provider Support (Future)

When supporting multiple plugins simultaneously, use genkit's plugin composition:

```go
func InitGenkitWithProviders(ctx context.Context) *genkit.Genkit {
    plugins := []any{}
    
    if os.Getenv("GOOGLE_API_KEY") != "" {
        plugins = append(plugins, &googlegenai.GoogleAI{})
    }
    
    if os.Getenv("OPENAI_API_KEY") != "" {
        plugins = append(plugins, &openai.OpenAI{})
    }
    
    if len(plugins) > 0 {
        return genkit.Init(ctx, genkit.WithPlugins(plugins...))
    }
    
    return genkit.Init(ctx)
}
```

---

## Environment Variable Reference

### For CI/CD

```bash
# Google Cloud
export GOOGLE_API_KEY="AIzaSy..."

# OpenAI
export OPENAI_API_KEY="sk-..."

# Local Development
echo "GOOGLE_API_KEY=test-key" > .env
echo "OPENAI_API_KEY=test-key" >> .env
source .env
```

### For Docker

```dockerfile
FROM golang:1.21

WORKDIR /app
COPY . .

# Set provider based on environment
ARG PROVIDER_API_KEY
ENV GOOGLE_API_KEY=${PROVIDER_API_KEY}

RUN go build -o app .
```

---

## Troubleshooting

### Error: "Plugin not available" when using LLM

**Cause:** API key for the provider is not set

**Solution:**
1. Check which LLM provider your config uses (e.g., `llm: "google"`)
2. Set the appropriate API key:
   ```bash
   export GOOGLE_API_KEY="your-key"
   ```
3. Re-run your application

### Error: "No plugins registered" when expecting LLM

**Cause:** `InitGenkitWithProviders()` detected no API keys

**Solution:**
1. Verify API key environment variable is set and not empty:
   ```bash
   echo $GOOGLE_API_KEY  # Should print your key
   ```
2. Restart your application after setting environment variables
3. If using containers, ensure environment is passed: `docker run -e GOOGLE_API_KEY=...`

### Test Failures After Adding New Provider

**Cause:** New provider plugin has different initialization

**Solution:**
1. Check if provider needs specific configuration
2. Update `InitGenkitWithProviders()` detection logic
3. Update test fixtures to set required environment variables

---

## Related Documentation

- **`docs/SDK_GENKIT_ARCHITECTURE.md`** — Two-tier initialization strategy overview
- **`docs/CONTEXT.md`** — Current SDK state and dependencies
- **`sdk.go`** — Implementation of `InitGenkitWithProviders()` and `InitGenkitBasic()`
- **`sdk/sdk.go`** — SDK entry points that use these functions

---

## Summary

| Aspect | Before | After |
|--------|--------|-------|
| **Plugin Registration** | Always all plugins | Only available plugins |
| **API Key Requirement** | Required all keys | Only required for used providers |
| **Test Compatibility** | Needed all keys even for non-LLM tests | Works without API keys for non-LLM |
| **Deployment Flexibility** | Single hardcoded plugin set | Environment-specific plugin set |
| **Error Messages** | Fails at SDK init if key missing | Fails at actual LLM usage if key missing |
| **Adding New Provider** | Modify multiple functions | Modify `InitGenkitWithProviders()` only |

---

**Last Updated:** 2025-11-12  
**Status:** ✅ Implemented and Tested
