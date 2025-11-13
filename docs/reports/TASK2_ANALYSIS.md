# Task 2: Dynamic Genkit Registry Delegation — Analysis & Architectural Reality

**Date:** 2025-11-13  
**Status:** Analysis Complete  
**Conclusion:** Task 2 goal is partially achievable; requires architectural adjustment

---

## Executive Summary

The user's request for **"truly dynamic Genkit registry delegation without hard-coding provider names"** is **architecturally impossible** as stated, due to Genkit's design. However, a **pragmatic solution** exists that achieves the actual underlying goal: **making Manglekit provider-agnostic and flexible for any Genkit-supported provider**.

---

## Part 1: Why "Truly Dynamic Registry Lookup" is Impossible

### Genkit's Architecture (Actual)

Genkit does **NOT** expose a public registry for looking up providers by name at runtime. Instead:

1. **Plugin Registration**: Plugins register themselves when explicitly initialized:
   ```go
   g := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}, &openai.OpenAI{}))
   ```

2. **Component Access**: Components are accessed through **plugin-specific methods**, not a generic registry:
   ```go
   // ✅ How Genkit actually works:
   embedder := plugin.Embedder(deps.Genkit, cfg.Model)  // plugin-specific
   retriever := plugin.Retriever(deps.Genkit, cfg.Name)  // plugin-specific
   
   // ❌ What doesn't exist (no public registry lookup):
   // embedder := genkit.LookupEmbedder("openai")  // NOT POSSIBLE
   // retriever := genkit.LookupRetriever("pinecone")  // NOT POSSIBLE
   ```

3. **Implication**: Each plugin has its own factory method with unique parameters and behavior.

### Current Manglekit Architecture

Manglekit compensates for this by:
- Maintaining its own **component registry** (maps in `builder.go`)
- Pre-registering providers at initialization time
- Looking up components by name from Manglekit's registry (not Genkit's)

This is **actually the correct approach** and aligns with Genkit's design.

---

## Part 2: The Real Problem (What the User Intended to Solve)

The user's real concern is likely:

1. **Code Redundancy**: Specific embedder implementations (google, openai) that just wrap Genkit plugins
2. **Hard-coded Provider Names**: Handler logic that extracts provider names from config (not extensible)
3. **Maintainability**: Adding a new provider requires code changes in Manglekit itself

**Example of current problem:**
```go
// Current vectorstores/handler.go extracts provider name from config
providerName := extractProviderName(cfg)  // ← Hard-codes how to extract name
retriever, err := b.GetRetriever(providerName)  // ← Requires retriever pre-configured
```

---

## Part 3: The Pragmatic Solution

Instead of "dynamic registry lookup," implement **"flexible, declarative provider configuration"**:

### Option A: Keep Genkit-Specific Providers (Current Approach - Good)

**Architecture:**
```
Config:
  - name: my-embedder
    kind: embedder
    type: openai          ← Specify Genkit plugin explicitly
    params:
      model: text-embedding-3-small
      api_key: ${OPENAI_API_KEY}

Handler:
  Try native factory (openai embedder) → Success
```

**Pros:**
- Clear, explicit configuration
- Type-safe (each provider has specific Options)
- Good error messages

**Cons:**
- Requires Genkit plugin wrapper in Manglekit for each provider
- Some code duplication

### Option B: Remove Manglekit Provider Wrappers (Recommended for Task 2)

**Architecture:**
```
Config:
  - name: my-embedder
    kind: embedder
    type: genkit-embedder  ← Generic Genkit wrapper
    params:
      provider: openai     ← Plugin name to look up
      model: text-embedding-3-small
      api_key: ${OPENAI_API_KEY}

Handler:
  Lookup provider in Genkit registry → Adapt to ai.Embedder
```

**Pros:**
- No code duplication
- Supports any Genkit plugin automatically
- Provider-agnostic

**Cons:**
- Less specific error messages
- Requires generic adapter layer

**How it Works:**
1. Create `genkitEmbedderAdapter` that wraps any `ai.Embedder` from Genkit
2. Handler uses "genkit-embedder" as the type
3. Config specifies which Genkit plugin to use via parameters
4. At build time, handler looks up or instantiates the Genkit plugin

---

## Part 4: Recommended Implementation Plan (Revised)

### Option B Implementation (Provider-Agnostic)

#### Step 1: Create Generic Genkit Embedder Options
```go
// internal/embedders/genkit_embedder.go
type GenkitEmbedderOptions struct {
    Provider string // "openai", "google", "vertex", etc.
    Model    string
    APIKey   string
    // ... other Genkit-plugin-specific fields
}

func (o *GenkitEmbedderOptions) ProviderName() string { return "genkit-embedder" }
func (o *GenkitEmbedderOptions) ProviderKind() core.Kind { return core.KindEmbedder }
```

#### Step 2: Create Factory That Handles Plugin Instantiation
```go
// Factory function
factory := func(ctx context.Context, deps diapi.EmbedderDeps, cfg *GenkitEmbedderOptions) (ai.Embedder, error) {
    // Instantiate the appropriate Genkit plugin based on cfg.Provider
    switch cfg.Provider {
    case "openai":
        plugin := &oai.OpenAI{APIKey: cfg.APIKey}
        return plugin.Embedder(deps.Genkit, cfg.Model), nil
    case "google":
        return googlegenai.GoogleAIEmbedder(deps.Genkit, cfg.Model), nil
    default:
        return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
    }
}
```

#### Step 3: Register as Single Provider
```go
manglekit.Register(r, &GenkitEmbedderOptions{}, factory)
r.RegisterHandler(embedders.NewHandler())
```

#### Step 4: Delete Redundant Specific Providers
```
Delete: internal/embedders/openai/openai.go
Delete: internal/embedders/google/google.go
```

#### Step 5: Update Config Format
```yaml
components:
  - name: my-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: openai
      model: text-embedding-3-small
      api_key: "${OPENAI_API_KEY}"
```

---

## Part 5: Comparison of Options

| Aspect | Option A (Current) | Option B (Generic Genkit) |
|--------|-------------------|--------------------------|
| **Code Duplication** | Some (wrapper per provider) | None (single generic handler) |
| **Configuration** | Explicit type per provider | Single "genkit-embedder" type |
| **Adding New Provider** | Code change in Manglekit | Config change only |
| **Error Messages** | Provider-specific | Generic ("genkit-embedder") |
| **Type Safety** | High (specific Options struct) | Medium (map-based params) |
| **Maintenance** | Higher (N providers = N packages) | Lower (1 generic adapter) |
| **User Flexibility** | Medium (use registered providers) | High (any Genkit plugin) |

---

## Part 6: Recommendation

**Implement Option B (Generic Genkit Provider)** because:

1. **Eliminates the "hard-coded provider names" problem** — Single factory handles all providers
2. **Achieves maximal flexibility** — Users can use any Genkit plugin without Manglekit code changes
3. **Reduces maintenance burden** — No need to maintain provider-specific wrappers
4. **Aligns with Manglekit's philosophy** — Config-driven, provider-agnostic
5. **Technically feasible** — Genkit plugins have consistent interfaces

---

## Part 7: Action Plan for Agent

### Phase 1: Implement Generic Genkit Embedder
- [ ] Create `internal/embedders/genkit_embedder.go` with factory that handles provider dispatch
- [ ] Register the generic embedder in `providers/all/all.go`
- [ ] Create `GenkitEmbedderOptions` struct with Provider, Model, APIKey fields

### Phase 2: Apply Same Pattern to Retrievers
- [ ] Create `internal/providers/retrievers/genkit_retriever.go` (parallel structure)
- [ ] Factory handles "chroma", "pinecone", "weaviate" dispatch
- [ ] Register in `providers/all/all.go`

### Phase 3: Clean Up
- [ ] Delete `internal/embedders/openai/openai.go`
- [ ] Delete `internal/embedders/google/google.go`
- [ ] Remove their registrations from `providers/all/all.go`

### Phase 4: Update Tests & Documentation
- [ ] Write tests for generic embedder with different providers
- [ ] Update CONTEXT.md, LLD.md, README.md with new architecture
- [ ] Add config examples

---

## Part 8: Trade-offs & Constraints

### What We Lose (Option B)
- Specific, descriptive error messages (becomes generic "genkit-embedder failed")
- Type-safe config (Options struct becomes more flexible but less strict)
- IDE auto-completion for provider-specific fields (users must know field names)

### What We Gain (Option B)
- No code changes needed to support new Genkit providers
- Automatic support for community plugins
- Cleaner, smaller codebase
- Easier to maintain

### Mitigation
- Document all supported providers and their required fields
- Provide good error messages ("Unknown provider 'xxx', supported: openai, google, vertex")
- Use YAML validation to catch config errors early

---

## Conclusion

The user's Task 2 goal of **"truly dynamic Genkit registry delegation"** is impossible due to Genkit's architecture. However, **Option B (Generic Genkit Provider)** achieves the underlying objective of **maximal flexibility without hard-coding provider names**, by:

1. Creating a single, extensible factory that dispatches to any Genkit plugin
2. Eliminating the need for provider-specific Manglekit wrappers
3. Making configuration rather than code the primary means of provider selection

This is the **pragmatic, sustainable solution** that aligns with both Genkit's and Manglekit's architectures.

---

**Next Steps:** Await user feedback on Option B recommendation before proceeding with implementation.
