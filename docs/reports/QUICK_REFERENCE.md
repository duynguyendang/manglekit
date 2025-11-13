# Quick Reference: Proxy Pattern Implementation Guide

**Date:** 2025-11-13  
**Status:** Planning Phase Complete, Ready for Implementation

---

## Files Created This Session

### 1. `ENHANCEMENT_RECOMMENDATIONS.md` (Comprehensive Review)
- **Purpose:** Code review with detailed analysis of current implementation
- **Sections:**
  - Part 1: Current implementation analysis (what's working, what needs improvement)
  - Part 2: Architectural recommendations (with diagrams)
  - Part 3: Recommended implementations (complete code examples)
  - Part 4: Configuration-driven provider dispatch
  - Part 5: Elimination of hard-coding
  - Part 6: Deletion roadmap
  - Part 7: DX improvements
  - Part 8: Implementation priority (4-phase plan)
  - Part 9: Benefits summary

### 2. `IMPLEMENTATION_SUMMARY.md` (Action Plan)
- **Purpose:** Executive summary with clear next steps
- **Contents:**
  - Current state assessment
  - Enhancement roadmap (5 phases)
  - Configuration migration path (before/after)
  - Detailed recommendations
  - Decision points with recommendations
  - Architecture principle

### 3. `internal/adapters/genkit_embedder_adapter.go` (Code)
- **Purpose:** Reusable adapter for ANY Genkit embedder plugin
- **Features:**
  - Provider-agnostic (works with OpenAI, Google, Vertex, Cohere, etc.)
  - Centralized error logging
  - Wraps errors with provider context
  - Name() method for debugging

### 4. `internal/adapters/genkit_retriever_adapter.go` (Code)
- **Purpose:** Reusable adapter for ANY Genkit retriever
- **Features:**
  - Wraps core.Retriever with provider context
  - Read-only semantics (IndexDocuments/UpdateDocuments return ErrNotSupported)
  - Centralized logging
  - Works with any Genkit backend (Pinecone, Chroma, Weaviate, etc.)

---

## Phase-by-Phase Implementation Guide

### Phase 1: Create Reusable Adapters ✓ DONE
**Time:** ~1 hour  
**Deliverable:** Generic adapters for reuse across all handlers

```
internal/adapters/
├── genkit_embedder_adapter.go  ✓
└── genkit_retriever_adapter.go ✓
```

---

### Phase 2: Create Generic Provider Factories (NEXT)
**Time:** ~2-3 hours  
**Deliverable:** Single factory per component type

#### 2.1 Generic Embedder Factory
**File:** `internal/providers/embedders/genkit_embedder.go`

```go
type GenkitEmbedderOptions struct {
    Provider string              // "openai", "google", "vertex", "cohere"
    Model    string              // Model identifier
    APIKey   string              // For authentication
    BaseURL  string              // For OpenAI-compatible APIs
    ProviderConfig map[string]any // Extensible config
    SkipModelCheck bool           // For testing
}

func createGenkitEmbedder(ctx, genkit, opts) ai.Embedder {
    switch opts.Provider {
    case "openai", "groq":
        return createOpenAIEmbedder(...)
    case "google", "vertex":
        return createGoogleEmbedder(...)
    case "cohere":
        return createCohereEmbedder(...)
    default:
        return nil  // error: unsupported provider
    }
}
```

**Benefits:**
- Single factory, any provider
- Adding new provider = update switch statement
- Fully extensible via ProviderConfig

#### 2.2 Generic LLM Factory (Optional)
**File:** `internal/providers/llm/genkit_llm.go`

Similar structure to embedder factory, supporting any Genkit LLM provider.

#### 2.3 Generic Retriever Factory (Optional)
**File:** `internal/providers/retrievers/genkit_retriever.go`

Similar structure, supporting any Genkit retriever backend.

---

### Phase 3: Handler Refactoring
**Time:** ~2-3 hours  
**Deliverable:** Consistent two-path pattern across all handlers

#### 3.1 Refactor Embedders Handler
**File:** `internal/embedders/handler.go`

```go
// STEP 1: Try native Manglekit factory
built, err := f.Build(ctx, deps, cfg)
if err == nil {
    embedder, ok := built.(ai.Embedder)
    if !ok { return nil, fmt.Errorf("not a valid embedder") }
    resolved.Embedders[name] = embedder
    return core.NopCloser, nil
}

// STEP 2: Fall back to Genkit delegation
// Log the failure and suggest using "genkit-embedder" type
return nil, fmt.Errorf(
    "embedder '%s' failed: %w\n"+
    "Hint: Set type: 'genkit-embedder' for Genkit providers\n"+
    "Supported Genkit providers: openai, google, vertex, cohere",
    name, err,
)
```

#### 3.2 Refactor LLM Handler (Similar)
**File:** `internal/providers/llm/handler.go`

---

### Phase 4: Configuration Validation & Documentation
**Time:** ~2-3 hours  
**Deliverable:** Updated examples and migration guide

#### 4.1 New Config Structure
```yaml
components:
  - name: embedder-openai
    kind: embedder
    type: genkit-embedder
    params:
      provider: openai
      model: text-embedding-3-small
      apiKey: "${OPENAI_API_KEY}"
  
  - name: embedder-vertex
    kind: embedder
    type: genkit-embedder
    params:
      provider: google
      model: textembedding-gecko@latest
  
  - name: llm-openai
    kind: llm
    type: genkit-llm
    params:
      provider: openai
      model: gpt-4-turbo
      apiKey: "${OPENAI_API_KEY}"
```

#### 4.2 Update `providers/all/all.go`
```go
// Register generic factories instead of provider-specific packages
func Register(r *Registry) {
    // ...existing native providers...
    
    // Generic Genkit providers (new)
    if err := embedders.RegisterGenkitEmbedder(r); err != nil {
        log.Printf("WARNING: Genkit embedder registration: %v\n", err)
    }
    if err := llm.RegisterGenkitLLM(r); err != nil {
        log.Printf("WARNING: Genkit LLM registration: %v\n", err)
    }
    // ...rest of registration...
}
```

---

### Phase 5: Cleanup & Documentation
**Time:** ~2-3 hours  
**Deliverable:** Reduced codebase, comprehensive documentation

#### 5.1 Delete Redundant Packages
```bash
# Remove hard-coded provider wrappers
rm -rf internal/embedders/openai
rm -rf internal/embedders/google
rm internal/providers/llm/openai.go    # (archive as reference)
rm internal/providers/llm/google.go    # (archive as reference)
```

#### 5.2 Update Documentation
- **`docs/CONTEXT.md`:** Update Provider Families section
- **`docs/LLD.md`:** Update handler dispatch section
- **`docs/README.md`:** Add Genkit provider examples
- **`examples/`:** Create config examples for each provider

#### 5.3 Migration Guide
Document how to migrate from hard-coded types to generic types:

```markdown
# Migration Guide: From Hard-Coded Providers to Generic Proxy

## Before
\`\`\`yaml
- type: openai
  params:
    model: text-embedding-3-small
\`\`\`

## After
\`\`\`yaml
- type: genkit-embedder
  params:
    provider: openai
    model: text-embedding-3-small
\`\`\`

## Benefit
- Switch providers by editing YAML, no code recompilation
- Support any Genkit provider automatically
```

---

## Architecture Comparison

### Before (Hard-Coded)
```
Config                Handler              Factory              Provider Package
↓                     ↓                    ↓                    ↓
type: "openai" → Handler.Build() → factory.Build() → OpenAIEmbedder (wrapper)
type: "google" → Handler.Build() → factory.Build() → GoogleEmbedder (wrapper)
type: "cohere" → Handler.Build() → factory.Build() → ??? (NOT SUPPORTED)
```

### After (Generic Proxy)
```
Config                Handler              Factory              Genkit Plugin
↓                     ↓                    ↓                    ↓
provider: "openai" → Handler.Build() → factory.Build() → OpenAI plugin
provider: "google" → Handler.Build() → factory.Build() → Google plugin
provider: "cohere" → Handler.Build() → factory.Build() → Cohere plugin
provider: "custom" → Handler.Build() → factory.Build() → Custom plugin
```

**Key Differences:**
- ✅ Single factory handles all providers
- ✅ Provider selection via configuration, not code
- ✅ New providers don't require code changes
- ✅ Adapter layer is thin, transparent
- ✅ Genkit fully in control

---

## Testing Strategy

### Unit Tests
```go
// Test adapter independently
TestGenkitEmbedderAdapter_Embed()
TestGenkitEmbedderAdapter_ErrorHandling()

// Test factory dispatch
TestCreateGenkitEmbedder_OpenAI()
TestCreateGenkitEmbedder_Google()
TestCreateGenkitEmbedder_InvalidProvider()
```

### Integration Tests
```go
// Test full pipeline with config
TestEmbedderHandler_GenkitEmbedder_Success()
TestEmbedderHandler_GenkitEmbedder_Fallback()
TestEmbedderHandler_ErrorMessages()
```

### E2E Tests
```yaml
# config.yaml test case
components:
  - name: my-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: openai
      model: text-embedding-3-small
      apiKey: "${OPENAI_API_KEY}"
```

---

## Decision Matrix

| Aspect | Hard-Coded (Before) | Generic Proxy (After) |
|--------|---------------------|----------------------|
| **Adding new provider** | Create new package | Update switch statement |
| **Provider selection** | Code & recompile | Config YAML only |
| **Support any Genkit plugin** | No | Yes |
| **Codebase size** | Large (1 package per provider) | Small (1 generic factory) |
| **Maintenance burden** | High | Low |
| **User flexibility** | Limited | High |
| **Consistency** | VectorStores different | All handlers same |
| **Extensibility** | Hard-coded | Configuration-driven |

---

## Rollout Strategy

### Backward Compatibility
- Keep old hard-coded types (openai, google) working
- Add deprecation warnings in logs
- Recommend migration in error messages
- Remove in next major version

### Deprecation Warnings
```go
if cfg.ProviderName() == "openai" {
    logger.Warnf(
        "DEPRECATED: type 'openai' is deprecated. Use type 'genkit-embedder' with provider: 'openai' instead",
        "component", name,
    )
}
```

### Migration Timeline
- **Week 1:** Deploy new generic factories (backward compatible)
- **Week 2:** Add deprecation warnings
- **Week 3:** Update documentation & examples
- **Month 2:** Mark old packages as deprecated
- **v2.0:** Remove hard-coded providers

---

## Success Criteria

✅ **Code Quality**
- No hard-coded provider names (except in dispatch logic)
- DRY: Single factory vs. N provider packages
- Consistent patterns across all handlers

✅ **Developer Experience**
- Adding new provider = config change, not code change
- Clear error messages with migration hints
- Examples for common providers (OpenAI, Google, etc.)

✅ **Maintainability**
- 40% less code in provider packages
- Easier to debug (centralized adapters)
- Reduced test maintenance

✅ **Extensibility**
- Users can add custom Genkit providers
- No Manglekit recompilation needed
- Configuration-driven everything

---

## Next Steps

### To Proceed
1. Review `ENHANCEMENT_RECOMMENDATIONS.md`
2. Review `IMPLEMENTATION_SUMMARY.md`
3. Approve the architecture direction
4. Proceed with Phase 2 implementation

### Questions to Address
1. Should old hard-coded types be deprecated immediately or gradually?
2. Should all providers (embedders, LLMs, retrievers) be refactored or just embedders?
3. Should we maintain backward compatibility or make a breaking change?

---

## References

- **Full Details:** `ENHANCEMENT_RECOMMENDATIONS.md`
- **Action Plan:** `IMPLEMENTATION_SUMMARY.md`
- **Adapters (Code):** `internal/adapters/genkit_*.go`
- **Current Implementation:** `internal/vectorstores/handler.go` (reference for two-path pattern)

---

**Recommendation:** Implement all 5 phases to achieve maximum DX and code quality. The effort is well-invested for the long-term maintainability gain.
