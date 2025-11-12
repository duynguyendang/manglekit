# SDK Genkit Architecture & Provider Plugin Strategy

**Date:** 2025-11-12  
**Status:** Implemented & Tested

## Overview

The Manglekit SDK now uses a **smart, centralized approach** for genkit plugin initialization that:

1. **Supports all LLM providers** through a single configuration point
2. **Decouples SDK entry points** from LLM API key requirements
3. **Allows flexible initialization** based on use case (programmatic vs. config-based)
4. **Maintains backward compatibility** with existing code

## Problem Solved

### Before
- Genkit plugin initialization was scattered across multiple SDK functions
- Every SDK function had to handle Google LLM plugin initialization individually
- Tests without LLM components failed because plugins required API keys at SDK init time
- Adding new providers required changes in multiple places

### After
- **Centralized** plugin initialization in two functions: `InitGenkitWithProviders()` and `InitGenkitBasic()`
- **Config-based** loading (SDK.Load) uses basic genkit without forcing LLM plugins
- **Programmatic** loading (SDK.NewBuilder) pre-initializes LLM plugins for better DX
- **Easy extensibility**: Adding new providers only requires changes in one place
- **Tests work without API keys**: Tests that don't use LLM components pass cleanly

## Architecture

### Two-Tier Genkit Initialization Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                     SDK Entry Points                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Config-Based API              Programmatic API                  │
│  ├─ sdk.Load()                 └─ sdk.NewBuilder()               │
│  └─ sdk.LoadWithRegistry()         (pre-initializes plugins)     │
│      (uses basic genkit)                                         │
│                                                                   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                ┌──────────▼──────────┐
                │   Genkit Factory    │
                ├─────────────────────┤
                │ InitGenkitBasic()   │─► genkit.Init(ctx)
                │ InitGenkitWithProv()│─► genkit.Init(ctx, plugins...)
                └─────────────────────┘
                           │
            ┌──────────────▼──────────────┐
            │   Genkit Instance           │
            ├──────────────────────────────┤
            │  • No LLM plugins (basic)    │
            │  • All LLM plugins (with)    │
            └──────────────────────────────┘
```

### Functions

#### `InitGenkitBasic(ctx context.Context) *genkit.Genkit`

Creates a basic genkit instance **without** LLM provider plugins.

**When to use:**
- Config-based orchestrator loading (`FromConfig`)
- Tests without LLM components
- Configurations without LLM providers

**Advantages:**
- No API key requirements
- Lightweight initialization
- Allows LLM components to register plugins dynamically if needed

**Code location:** `sdk.go` line 44

#### `InitGenkitWithProviders(ctx context.Context) *genkit.Genkit`

Creates a genkit instance **with** all LLM provider plugins pre-initialized.

**When to use:**
- Programmatic API (`sdk.NewBuilder`)
- Examples with LLM components
- Applications guaranteed to use LLM providers

**Advantages:**
- Plugins ready immediately
- Better error messages for missing API keys
- Optimized for LLM-heavy workloads

**Requirements:**
- LLM provider API keys must be set (e.g., `GOOGLE_API_KEY`)

**Code location:** `sdk.go` line 32

### SDK Entry Points & Strategy

| Entry Point | Genkit Init | Plugins | Use Case |
|-------------|-----------|---------|----------|
| `Load()` | InitGenkitBasic | None | Simple config-based loading |
| `LoadWithRegistry()` | InitGenkitBasic | None | Config-based with custom registry |
| `FromConfig()` | InitGenkitBasic | None | Programmatic config processing |
| `sdk.NewBuilder()` | InitGenkitWithProviders | All | Programmatic builder (full app) |

### Extension Point: Adding New LLM Providers

To add support for a new LLM provider (e.g., Anthropic Claude):

**Step 1:** Import the provider plugin
```go
import (
    "github.com/firebase/genkit/go/plugins/claude"
)
```

**Step 2:** Add to `InitGenkitWithProviders()`
```go
func InitGenkitWithProviders(ctx context.Context) *genkit.Genkit {
    return genkit.Init(ctx, genkit.WithPlugins(
        &googlegenai.GoogleAI{},
        &claude.Claude{},  // Add new provider here
    ))
}
```

**Step 3:** Implement LLM provider package
```go
// internal/providers/llm/claude.go
func RegisterClaude(r *manglekit.Registry) {
    manglekit.Register(r, &ClaudeOptions{}, 
        func(ctx context.Context, deps diapi.LLMDeps, cfg *ClaudeOptions) (core.LLMClient, error) {
            // Implementation
        },
    )
}
```

**Result:**
- New provider automatically available in programmatic API
- Config-based API works without requiring new API keys if Claude isn't used
- Single change point in the codebase

## Design Principles

1. **Centralization**: All genkit initialization logic in one place
2. **Flexibility**: Support both API-key-heavy and API-key-light scenarios
3. **Extensibility**: Adding new providers doesn't require changes to SDK layer
4. **Backward Compatibility**: Existing code works without modification
5. **Clear Intent**: API names clearly indicate when plugins are loaded

## Test Impact

### Before
```
❌ TestBM25_Handler_HappyPath FAILS
   Error: Google AI requires GOOGLE_API_KEY environment variable
   Reason: Genkit plugins forced to initialize even though test doesn't use Google LLM
```

### After
```
✅ TestBM25_Handler_HappyPath PASSES
   Reason: Basic genkit used for config loading, no plugin initialization
   
✅ Example 01-programmatic-setup PASSES  
   Reason: Programmatic builder pre-initializes plugins for Google LLM
```

## Code Organization

### Root Package (`sdk.go`)
- **Exported:** `InitGenkitWithProviders()`, `InitGenkitBasic()`
- **Purpose:** Plugin initialization strategies
- **Usage:** Called by SDK entry points and user code

### SDK Package (`sdk/sdk.go`)
- **Functions:** `Load()`, `LoadWithRegistry()`, `NewBuilder()`
- **Purpose:** User-facing SDK API
- **Strategy:**
  - Config-based functions → `InitGenkitBasic()`
  - Programmatic function → `InitGenkitWithProviders()`

## Migration Guide

### For Library Users

**No changes required.** Existing code works as before:

```go
// Config-based - still works, now more flexible
orch, err := sdk.Load(ctx, configData)

// Programmatic - same API, now with better plugin support
builder, err := sdk.NewBuilder(ctx)
```

### For SDK Contributors

When adding new LLM provider support:

1. Add provider plugin to `InitGenkitWithProviders()` in `sdk.go`
2. Implement LLM provider in `internal/providers/llm/`
3. Add provider factory to registry
4. Update tests to skip when API key not set
5. Example automatically uses new provider

## Future Considerations

1. **Dynamic plugin loading**: Load plugins only when configuration requires them
2. **Plugin configuration**: Allow users to specify which plugins to load
3. **Plugin caching**: Reuse genkit instance across requests
4. **Provider-specific factories**: Different factory strategies per provider

## References

- `sdk.go` - Root SDK genkit initialization
- `sdk/sdk.go` - User-facing SDK entry points
- `internal/providers/llm/google.go` - Google LLM provider implementation
- `examples/01-programmatic-setup/main.go` - Programmatic API usage
- `internal/providers/retrievers/bm25/bm25_handler_test.go` - Config-based API testing
