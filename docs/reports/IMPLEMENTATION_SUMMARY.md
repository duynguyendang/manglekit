# Implementation Summary: Making Manglekit a True Genkit Proxy

**Status:** Code Review Complete, Adapters Created, Recommendations Documented  
**Date:** 2025-11-13  
**Files Created:**
- `ENHANCEMENT_RECOMMENDATIONS.md` — Comprehensive code review + enhancement roadmap
- `internal/adapters/genkit_embedder_adapter.go` — Reusable embedder adapter  
- `internal/adapters/genkit_retriever_adapter.go` — Reusable retriever adapter

---

## Current State Assessment

### ✅ What's Implemented & Working

1. **Vectorstores Handler (Excellent)**
   - Two-path pattern: native → genkit fallback
   - `genkitRetrieverAdapter` wraps Genkit retriever
   - Proper error logging
   - Read-only semantics enforced

2. **Embedders Handler (Basic)**
   - Genkit instance injected
   - Single-path (no fallback)
   - No delegation pattern

3. **LLM Handler (Basic)**
   - Genkit instance injected
   - Direct factory call
   - No fallback pattern

### ❌ Current Limitations

1. **Hard-Coded Provider Packages**
   ```
   internal/embedders/openai/    ← Wrapper just calls Genkit
   internal/embedders/google/     ← Wrapper just calls Genkit
   internal/providers/llm/openai.go  ← Wrapper just calls Genkit
   internal/providers/llm/google.go  ← Wrapper just calls Genkit
   ```
   Each provider = new package = code duplication = maintenance burden

2. **Inconsistent Patterns**
   - VectorStores: Two-path delegation ✓
   - Embedders: Single-path only ✗
   - LLMs: Single-path only ✗

3. **Hard-Coded Provider Name Extraction**
   ```go
   providerName := extractProviderName(cfg)  // ← Only extracts via ProviderName()
   ```

4. **Config Structure Misalignment**
   ```yaml
   # Current (tight coupling to provider)
   - type: openai
     params:
       model: text-embedding-3-small
   
   # Recommended (provider-agnostic)
   - type: genkit-embedder
     params:
       provider: openai
       model: text-embedding-3-small
   ```

---

## Enhancement Roadmap

### Phase 1: Create Reusable Adapters ✓ (DONE)

**Status:** Completed  
**Files:** `internal/adapters/genkit_*.go`

- `GenkitEmbedderAdapter` — Wraps `ai.Embedder` for any Genkit plugin
- `GenkitRetrieverAdapter` — Wraps `core.Retriever` with provider context

**Benefits:**
- Provider-agnostic (works with OpenAI, Google, Vertex, Cohere, Anthropic, etc.)
- Centralized logging & error handling
- Reusable across all handlers

---

### Phase 2: Create Generic Provider Factories (RECOMMENDED NEXT)

**Create:** `internal/providers/embedders/genkit_embedder.go`

```go
// Single factory supports ALL embedder providers
type GenkitEmbedderOptions struct {
    Provider string              // "openai", "google", "vertex", "cohere"
    Model    string              // Model-specific identifier
    APIKey   string              // For authentication
    BaseURL  string              // For OpenAI-compatible APIs
    ProviderConfig map[string]any // Extensible for custom params
    SkipModelCheck bool           // For testing
}

func RegisterGenkitEmbedder(r *Registry) error {
    // Single factory handles provider dispatch
    factory := func(ctx, deps, cfg) (ai.Embedder, error) {
        embedder, err := createGenkitEmbedder(ctx, deps.Genkit, cfg)
        return adapters.NewGenkitEmbedderAdapter(embedder, cfg.Provider, deps.Obs.Logger), nil
    }
}
```

**Benefits:**
- Single registration point (not 10 separate packages)
- Adding new provider = update `createGenkitEmbedder()` switch statement
- No Manglekit recompilation needed for config-only changes
- Fully extensible via `ProviderConfig` map

---

### Phase 3: Apply Consistent Two-Path Pattern (RECOMMENDED)

**Refactor:** `internal/embedders/handler.go`

```go
// STEP 1: Try native Manglekit factory
built, err := f.Build(ctx, deps, cfg)
if err == nil {
    // Success, use native
    return
}

// STEP 2: Fall back to Genkit delegation
// Log failure and suggest using "genkit-embedder" type
```

**Benefits:**
- Consistent with vectorstores pattern
- Clear error messages with migration hints
- Graceful degradation path

---

### Phase 4: Create Generic LLM Factory (OPTIONAL)

**Create:** `internal/providers/llm/genkit_llm.go`

Similar structure to embedder factory, supporting any Genkit LLM provider.

---

### Phase 5: Cleanup & Documentation (FINAL)

**Delete (AFTER factories implemented):**
```
DELETE: internal/embedders/openai/
DELETE: internal/embedders/google/
DELETE: internal/providers/llm/openai.go      (keep as reference, deprecate)
DELETE: internal/providers/llm/google.go      (keep as reference, deprecate)
DELETE: internal/vectorstores/handler.go      (move logic to adapters)
```

**Update:**
- `providers/all/all.go` — Register generic factories instead
- `docs/CONTEXT.md` — Document new architecture
- `docs/LLD.md` — Update handler dispatch section
- Examples in `examples/` — Show new config structure

---

## Configuration Migration Path

### Before (Hard-Coded Providers)
```yaml
components:
  - name: my-embedder
    kind: embedder
    type: openai              # ← Type is provider
    params:
      model: text-embedding-3-small
      apiKey: "${OPENAI_API_KEY}"
  
  - name: my-llm
    kind: llm
    type: openai              # ← Type is provider
    params:
      model: gpt-4-turbo
      apiKey: "${OPENAI_API_KEY}"
```

### After (Proxy Pattern)
```yaml
components:
  # Generic Genkit embedder (ANY provider)
  - name: my-embedder
    kind: embedder
    type: genkit-embedder     # ← Type is generic
    params:
      provider: openai        # ← Provider specified in config
      model: text-embedding-3-small
      apiKey: "${OPENAI_API_KEY}"
  
  # Generic Genkit LLM (ANY provider)
  - name: my-llm
    kind: llm
    type: genkit-llm          # ← Type is generic
    params:
      provider: openai        # ← Provider specified in config
      model: gpt-4-turbo
      apiKey: "${OPENAI_API_KEY}"
  
  # Switch to different provider - NO CODE CHANGE
  - name: my-embedder-vertex
    kind: embedder
    type: genkit-embedder
    params:
      provider: google        # ← Just change provider
      model: textembedding-gecko@latest
```

**Benefits:**
- Change provider by editing YAML
- No recompilation
- No code changes
- Extensible to any Genkit plugin

---

## Detailed Recommendations

See `ENHANCEMENT_RECOMMENDATIONS.md` for:
- **Part 1:** Current implementation analysis with code examples
- **Part 2:** Architectural recommendations with diagrams
- **Part 3:** Complete code examples for adapters and factories
- **Part 4:** Configuration-driven provider dispatch patterns
- **Part 5:** Elimination of hard-coded provider packages
- **Part 6:** Deletion roadmap
- **Part 7:** DX improvements (error messages, validation, documentation)
- **Part 8:** Implementation priority (4-phase plan)
- **Part 9:** Benefits summary

---

## Next Steps

### Immediate (This Session)
1. ✅ Code review completed
2. ✅ Adapters created (`internal/adapters/genkit_*.go`)
3. ✅ Comprehensive recommendations documented
4. 📋 Decision point: Proceed with Phase 2?

### If Approved (Phase 2+)
1. Create generic embedder factory
2. Refactor embedders handler with two-path pattern
3. Create generic LLM factory (optional)
4. Update tests and documentation
5. Delete hard-coded provider packages

### Timeline
- **Phase 1:** ✓ Complete (adapters)
- **Phase 2:** ~2-3 hours (generic factories + handler refactoring)
- **Phase 3:** ~2-3 hours (two-path pattern + testing)
- **Phase 4:** ~1-2 hours (LLM factory, optional)
- **Phase 5:** ~1-2 hours (cleanup & docs)

**Total:** ~6-11 hours for full implementation

---

## Key Decision Points

### Q1: Delete existing provider packages?
**Recommendation:** YES  
**Reason:** Hard-coded packages are redundant; generic factories replace them completely.  
**Migration:** Users update config (`type: "openai"` → `type: "genkit-embedder"`, add `provider: "openai"`)

### Q2: Mandate two-path pattern for all handlers?
**Recommendation:** YES  
**Reason:** Consistent, predictable behavior across codebase.  
**Implementation:** Embedders, LLMs should have fallback like VectorStores do.

### Q3: Make generic factories the default?
**Recommendation:** YES  
**Reason:** Native Manglekit providers (like `dense`) are kept but generic Genkit wrappers are primary interface.  
**Flexibility:** Users can still use native providers if preferred.

---

## Architecture Principle

> **"Manglekit orchestrates, Genkit integrates. Let Genkit handle providers."**

This means:
- ✅ Manglekit = Proxy layer (thin, transparent)
- ✅ Genkit = Provider layer (full control)
- ✅ Config = Control point (user drives behavior)
- ❌ Hard-coding = Anti-pattern (avoid)
- ❌ Wrapper per provider = Unnecessary duplication

---

## Summary

Manglekit's current implementation is **solid but inconsistent**. By implementing the recommended enhancements:

1. **Consistency:** All handlers use two-path pattern
2. **Simplicity:** One generic factory instead of N provider packages
3. **Extensibility:** Users add new Genkit providers without Manglekit changes
4. **Transparency:** Genkit providers directly visible in configuration
5. **Maintainability:** Reduced code duplication and complexity

The transformation makes Manglekit a **true transparent proxy** for Genkit, fully embracing its design philosophy.
