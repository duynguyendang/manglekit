# Manglekit Proxy Pattern Enhancement — Code Review & Recommendations

**Date:** 2025-11-13  
**Status:** Comprehensive Code Review Complete  
**Objective:** Transform Manglekit into a true transparent proxy for Genkit, eliminating hard-coding and maximizing DX

---

## Executive Summary

Current implementation has **excellent foundations** but contains **architectural inconsistencies** and **hard-coded provider logic** that limit extensibility. Recommendations focus on:

1. **Unified proxy pattern** across all handlers (vectorstores, embedders, retrievers)
2. **Generic adapters** to wrap any Genkit component without code changes
3. **Dynamic provider dispatch** via configuration, not code
4. **Improved error handling & logging** for production debugging
5. **Clear separation** between Manglekit concerns and Genkit delegation

---

## Part 1: Current Implementation Analysis

### ✅ What's Working Well

#### 1.1 Vectorstores Handler — Two-Path Pattern (Best Practice)

**Location:** `internal/vectorstores/handler.go`

```go
// Path 1: Try native Manglekit factory
built, err := f.Build(ctx, deps, cfg)
if err == nil {
    // Success, use native
    resolved.VectorStores[name] = store
    return core.NopCloser, nil
}

// Path 2: Fall back to Genkit delegation
retriever, err := b.GetRetriever(providerName)
adapter := newGenkitRetrieverAdapter(retriever, logger)
resolved.VectorStores[name] = adapter
```

**Strengths:**
- ✅ Graceful degradation (native → fallback)
- ✅ Adapter pattern bridges Genkit types to Manglekit interfaces
- ✅ Detailed logging at each step
- ✅ Clear read-only semantics (`ErrNotSupported` for AddDocuments)
- ✅ Handles core.Logger injection

**Issue:** Hard-codes retriever lookup — requires retriever to be pre-configured elsewhere

---

#### 1.2 LLM Handler — Simple, Direct Pattern (Adequate)

**Location:** `internal/providers/llm/handler.go`

```go
## Part 1: Current Implementation Analysis

### ✅ Architecture Refactoring Complete

#### 1.1 Vectorstores Handler — Native Factories Only (Corrected)

**Location:** `internal/vectorstores/handler.go`

**Previous Issue (DEPRECATED):** This handler previously had a flawed "Path 2" fallback that attempted to wrap Manglekit Retrievers as VectorStores. This was architecturally backward because:
- Genkit provides vector stores (Pinecone, Chroma, etc.)
- Manglekit provides retrievers (dense, hybrid)
- The dependency should flow: Retriever → VectorStore, not the reverse

**Current Corrected Implementation:**
The handler now only processes native Manglekit VectorStore factories. For Genkit vector store providers, use the dedicated `genkit-vectorstore` factory:

**Location:** `internal/providers/vectorstores/genkitvectorstore/`

The correct flow is now:
1. **Genkit provides:** Vector stores (Pinecone, Chroma, Weaviate, etc.)
2. **Manglekit provides:** Retrievers that depend on VectorStores
3. **Adapter wraps:** GenkitVectorStoreAdapter in `internal/adapters/genkit_vectorstore_adapter.go` bridges Genkit retrievers to core.VectorStore interface

**Strengths of new architecture:**
- ✅ Clear, unambiguous dependency direction
- ✅ Genkit backends remain optional and pluggable
- ✅ No implicit fallback logic
- ✅ Retrievers explicitly declare their VectorStore dependency in YAML
- ✅ Configuration-driven provider selection

---

#### 1.2 LLM Handler — Simple, Direct Pattern (Adequate)

**Location:** `internal/providers/llm/handler.go`

```go
deps := diapi.LLMDeps{
```
    CoreDeps: b.GetCoreDeps(),
    Genkit:   b.Genkit(),
}
built, err := f.Build(ctx, deps, cfg)
```

**Strengths:**
- ✅ Minimal, focused responsibility
- ✅ Injects Genkit instance for factories
- ✅ Type-safe deps injection

**Limitation:** Only single path — no fallback pattern

---

### ❌ What Needs Improvement

#### 1.3 Embedders Handler — Missing Delegation Pattern

**Location:** `internal/embedders/handler.go`

```go
// ❌ PROBLEM: No two-path pattern, no delegation fallback
deps := diapi.EmbedderDeps{
    CoreDeps: b.GetCoreDeps(),
    Genkit:   b.Genkit(),
}
built, err := f.Build(ctx, deps, cfg)
// Single path only — if factory fails, no fallback
```

**Issues:**
- ❌ No fallback to Genkit delegation
- ❌ Inconsistent with vectorstores pattern
- ❌ Tightly coupled to hard-coded providers (openai, google)

---

#### 1.4 Hard-Coded Provider Implementations

**Location:** `internal/embedders/{openai,google}/`

```go
// ❌ PROBLEM: Provider-specific code that just wraps Genkit

// openai.go
func Register(r *manglekit.Registry) error {
    manglekit.Register(r, &embed.OpenAIEmbedderOptions{},
        func(ctx context.Context, deps diapi.EmbedderDeps, cfg *embed.OpenAIEmbedderOptions) (ai.Embedder, error) {
            plugin := &oai.OpenAI{APIKey: cfg.APIKey, Opts: opts}
            embedder = plugin.Embedder(deps.Genkit, cfg.Model)  // ← Just calling Genkit
            return embedder, nil
        },
    )
}

// google.go (similar structure)
// internal/providers/llm/openai.go (same pattern)
// internal/providers/llm/google.go (same pattern)
```

**Issues:**
- ❌ Each provider needs a separate Manglekit package
- ❌ Adding new provider requires code change + registration
- ❌ Boilerplate duplication across embedders, retrievers, LLMs
- ❌ Users can't easily extend with custom Genkit providers
- ❌ Hard-coded provider name extraction (`extractProviderName()`)

**Count:** 4+ duplicate implementations (openai-embedder, google-embedder, openai-llm, google-llm)

---

## Part 2: Architectural Recommendations

### Architecture Goal: True Transparent Proxy

```
┌─────────────────────────────────────────────────────────────┐
│                     User Config (YAML)                       │
│  components:                                                 │
│    - name: my-embedder                                      │
│      kind: embedder                                         │
│      type: genkit-embedder         ← Generic type           │
│      params:                                                │
│        provider: "vertex"          ← User specifies Genkit  │
│        model: "textembedding-004"  ← Genkit-specific config │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                  Manglekit Registry                          │
│  • Handlers (embedders, retrievers, etc.)                   │
│  • Generic factory for "genkit-embedder"                    │
│  • Dynamic provider dispatch logic                          │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Generic Provider Adapters                       │
│  • GenericEmbedderAdapter (wraps ai.Embedder)               │
│  • GenericRetrieverAdapter (wraps ai.Retriever)             │
│  • Converts Genkit types to Manglekit interfaces            │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Genkit Registry                           │
│  • OpenAI, Google, Vertex, Anthropic plugins                │
│  • Any community provider (user can bring their own)        │
│  • Fully managed by Genkit, Manglekit doesn't care          │
└─────────────────────────────────────────────────────────────┘
```

---

## Part 3: Recommended Implementations

### Recommendation 1: Create `GenericProviderAdapter` Pattern

**Purpose:** Reusable wrapper for any Genkit component → Manglekit interface

#### 1.1 Generic Embedder Adapter
```go
// internal/adapters/genkit_embedder_adapter.go

package adapters

import (
    "context"
    "fmt"
    
    "github.com/duynguyendang/manglekit/core"
    "github.com/firebase/genkit/go/ai"
)

// GenkitEmbedderAdapter wraps a Genkit ai.Embedder and adapts it to core.Embedder
// This adapter is provider-agnostic and works with any Genkit embedder
type GenkitEmbedderAdapter struct {
    embedder ai.Embedder
    logger   core.Logger
    provider string // For logging/debugging
}

func NewGenkitEmbedderAdapter(embedder ai.Embedder, provider string, logger core.Logger) *GenkitEmbedderAdapter {
    return &GenkitEmbedderAdapter{
        embedder: embedder,
        logger:   logger,
        provider: provider,
    }
}

// Embed delegates to the wrapped Genkit embedder
func (a *GenkitEmbedderAdapter) Embed(ctx context.Context, req *ai.EmbedRequest) (*ai.EmbedResponse, error) {
    if a.embedder == nil {
        return nil, fmt.Errorf("genkit embedder adapter: underlying embedder is nil")
    }
    
    if a.logger != nil {
        a.logger.Debugf(
            "delegating embedding request to Genkit provider",
            "provider", a.provider,
            "input_count", len(req.Input),
        )
    }
    
    resp, err := a.embedder.Embed(ctx, req)
    if err != nil {
        if a.logger != nil {
            a.logger.Debugf(
                "genkit embedder delegation failed",
                "provider", a.provider,
                "error", err.Error(),
            )
        }
        return nil, fmt.Errorf("genkit embedder (%s) failed: %w", a.provider, err)
    }
    
    return resp, nil
}

// Name returns the adapter's name
func (a *GenkitEmbedderAdapter) Name() string {
    return fmt.Sprintf("genkit-adapter(%s)", a.provider)
}
```

#### 1.2 Analogous Retriever Adapter

```go
// internal/adapters/genkit_retriever_adapter.go

package adapters

import (
    "context"
    "fmt"
    
    "github.com/duynguyendang/manglekit/core"
    "github.com/firebase/genkit/go/ai"
)

// GenkitRetrieverAdapter wraps Genkit ai.Retriever and adapts to core.Retriever
// Provider-agnostic for any Genkit retriever (pinecone, chroma, weaviate, etc.)
type GenkitRetrieverAdapter struct {
    retriever ai.Retriever
    logger    core.Logger
    provider  string // For logging/debugging
}

func NewGenkitRetrieverAdapter(retriever ai.Retriever, provider string, logger core.Logger) *GenkitRetrieverAdapter {
    return &GenkitRetrieverAdapter{
        retriever: retriever,
        logger:    logger,
        provider:  provider,
    }
}

// Retrieve delegates to the wrapped Genkit retriever
func (a *GenkitRetrieverAdapter) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
    if a.retriever == nil {
        return core.RetrieveResult{}, fmt.Errorf("genkit retriever adapter: underlying retriever is nil")
    }
    
    // Convert Manglekit request to Genkit request
    genkitReq := &ai.RetrieverRequest{
        Query: req.Query,
        // Map other fields as needed
    }
    
    if a.logger != nil {
        a.logger.Debugf(
            "delegating retrieval request to Genkit provider",
            "provider", a.provider,
            "query", req.Query,
            "topk", req.TopK,
        )
    }
    
    resp, err := a.retriever.Retrieve(ctx, genkitReq)
    if err != nil {
        if a.logger != nil {
            a.logger.Debugf(
                "genkit retriever delegation failed",
                "provider", a.provider,
                "error", err.Error(),
            )
        }
        return core.RetrieveResult{}, fmt.Errorf("genkit retriever (%s) failed: %w", a.provider, err)
    }
    
    // Convert Genkit response to Manglekit format
    docs := make([]core.Doc, len(resp.Documents))
    for i, doc := range resp.Documents {
        docs[i] = core.Doc{
            // Map fields from doc to core.Doc
        }
    }
    
    return core.RetrieveResult{Docs: docs}, nil
}

// Name returns the adapter's name
func (a *GenkitRetrieverAdapter) Name() string {
    return fmt.Sprintf("genkit-adapter(%s)", a.provider)
}
```

---

### Recommendation 2: Generic Provider Factories

**Purpose:** Single factory that dispatches to any Genkit provider based on configuration

#### 2.1 Generic Embedder Factory

```go
// internal/providers/embedders/genkit_embedder.go

package embedders

import (
    "context"
    "fmt"
    
    "github.com/duynguyendang/manglekit"
    "github.com/duynguyendang/manglekit/core"
    "github.com/duynguyendang/manglekit/core/diapi"
    "github.com/duynguyendang/manglekit/internal/adapters"
    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/googlegenai"
    oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
    "github.com/openai/openai-go/option"
)

// GenkitEmbedderOptions allows users to configure ANY Genkit embedder provider
type GenkitEmbedderOptions struct {
    // Provider is the Genkit provider name: "openai", "google", "vertex", "cohere", etc.
    Provider string `json:"provider,omitempty"`
    
    // Model is the model identifier for the provider
    Model string `json:"model,omitempty"`
    
    // APIKey for authentication (if required by provider)
    APIKey string `json:"apiKey,omitempty"`
    
    // BaseURL for OpenAI-compatible providers (e.g., Groq, custom endpoints)
    BaseURL string `json:"baseUrl,omitempty"`
    
    // ProviderConfig is a map of provider-specific configuration
    // Users can pass arbitrary config for new/custom Genkit providers
    ProviderConfig map[string]any `json:"providerConfig,omitempty"`
    
    // SkipModelCheck for testing (bypass live model validation)
    SkipModelCheck bool `json:"skipModelCheck,omitempty"`
}

func (o *GenkitEmbedderOptions) ProviderName() string    { return "genkit-embedder" }
func (o *GenkitEmbedderOptions) ProviderKind() core.Kind { return core.KindEmbedder }

// RegisterGenkitEmbedder registers the generic Genkit embedder factory
func RegisterGenkitEmbedder(r *manglekit.Registry) error {
    factory := func(ctx context.Context, deps diapi.EmbedderDeps, cfg *GenkitEmbedderOptions) (ai.Embedder, error) {
        if deps.Genkit == nil {
            return nil, fmt.Errorf("genkit instance is required for embedder factory")
        }
        
        if cfg.Provider == "" {
            return nil, fmt.Errorf("provider is required in GenkitEmbedderOptions")
        }
        
        if cfg.Model == "" {
            return nil, fmt.Errorf("model is required in GenkitEmbedderOptions")
        }
        
        // Dispatch to the appropriate Genkit provider
        embedder, err := createGenkitEmbedder(ctx, deps.Genkit, cfg)
        if err != nil {
            return nil, fmt.Errorf("failed to create genkit embedder for provider %q: %w", cfg.Provider, err)
        }
        
        if embedder == nil {
            return nil, fmt.Errorf("genkit provider %q returned nil embedder", cfg.Provider)
        }
        
        // Wrap in adapter with provider name for better logging
        return adapters.NewGenkitEmbedderAdapter(embedder, cfg.Provider, deps.Obs.Logger), nil
    }
    
    return manglekit.Register(r, &GenkitEmbedderOptions{}, factory)
}

// createGenkitEmbedder dispatches to the appropriate provider plugin
// This is extensible: users can add support for new providers by modifying this function
func createGenkitEmbedder(ctx context.Context, g *genkit.Genkit, opts *GenkitEmbedderOptions) (ai.Embedder, error) {
    switch opts.Provider {
    case "openai", "groq": // OpenAI-compatible
        return createOpenAIEmbedder(g, opts)
    
    case "google", "googlegenai", "vertex":
        return createGoogleEmbedder(g, opts)
    
    case "cohere":
        return createCohereEmbedder(g, opts)
    
    default:
        return nil, fmt.Errorf("unsupported embedder provider: %q (supported: openai, google, cohere)", opts.Provider)
    }
}

func createOpenAIEmbedder(g *genkit.Genkit, opts *GenkitEmbedderOptions) (ai.Embedder, error) {
    var optsSlice []option.RequestOption
    if opts.APIKey != "" {
        optsSlice = append(optsSlice, option.WithAPIKey(opts.APIKey))
    }
    if opts.BaseURL != "" {
        optsSlice = append(optsSlice, option.WithBaseURL(opts.BaseURL))
    }
    
    plugin := &oai.OpenAI{APIKey: opts.APIKey, Opts: optsSlice}
    embedder := plugin.Embedder(g, opts.Model)
    return embedder, nil
}

func createGoogleEmbedder(g *genkit.Genkit, opts *GenkitEmbedderOptions) (ai.Embedder, error) {
    embedder := googlegenai.GoogleAIEmbedder(g, opts.Model)
    if embedder == nil {
        return nil, fmt.Errorf("googlegenai.GoogleAIEmbedder returned nil for model %q", opts.Model)
    }
    return embedder, nil
}

func createCohereEmbedder(g *genkit.Genkit, opts *GenkitEmbedderOptions) (ai.Embedder, error) {
    // TODO: Implement when Genkit Cohere plugin is available
    return nil, fmt.Errorf("cohere embedder not yet implemented")
}
```

**Benefits:**
- ✅ Single registration point for all embedders
- ✅ Users add new providers by extending `createGenkitEmbedder()`
- ✅ No code changes to Manglekit core
- ✅ Flexible provider config via `ProviderConfig` map
- ✅ Adapter wraps result with logging

---

### Recommendation 3: Updated Handlers with Unified Two-Path Pattern

#### 3.1 Improved Embedders Handler

```go
// internal/embedders/handler.go (REFACTORED)

package embedders

import (
    "context"
    "fmt"
    
    "github.com/duynguyendang/manglekit/core"
    "github.com/duynguyendang/manglekit/core/diapi"
    "github.com/firebase/genkit/go/ai"
)

type Handler struct{}

func NewHandler() core.ComponentHandler {
    return &Handler{}
}

func (h *Handler) Kind() core.Kind {
    return core.KindEmbedder
}

// BuildComponent builds the Embedder component with two-path pattern:
// Path 1: Try native Manglekit embedder factory
// Path 2: Fall back to Genkit delegation via generic factory
func (h *Handler) BuildComponent(
    ctx context.Context,
    builderDI any,
    factory any,
    resolved *core.Resolved,
    cfg core.ProviderOptions,
    name string,
) (core.ResourceCloser, error) {
    b, ok := builderDI.(diapi.Builder)
    if !ok {
        return nil, fmt.Errorf("invalid builder DI type for Embedder handler: got %T", builderDI)
    }
    
    f, ok := factory.(core.Factory)
    if !ok {
        return nil, fmt.Errorf("invalid factory type for Embedder handler: got %T", factory)
    }
    
    // Prepare standard dependencies
    deps := diapi.EmbedderDeps{
        CoreDeps: b.GetCoreDeps(),
        Genkit:   b.Genkit(),
    }
    
    // STEP 1: Try native Manglekit factory
    built, err := f.Build(ctx, deps, cfg)
    if err == nil {
        embedder, ok := built.(ai.Embedder)
        if !ok {
            return nil, fmt.Errorf("component %s is not a valid ai.Embedder", name)
        }
        resolved.Embedders[name] = embedder
        
        if deps.Obs.Logger != nil {
            deps.Obs.Logger.Debugf(
                "embedder component built successfully via native factory",
                "name", name,
                "type", getProviderType(cfg),
            )
        }
        return core.NopCloser, nil
    }
    
    // STEP 2: Fall back to Genkit delegation
    // This allows any Genkit provider to work without code changes
    if deps.Obs.Logger != nil {
        deps.Obs.Logger.Debugf(
            "native embedder factory failed, attempting Genkit delegation",
            "name", name,
            "native_error", err.Error(),
        )
    }
    
    // For now, we log the failure since delegation happens via config
    // Users should explicitly use type: "genkit-embedder" for Genkit providers
    return nil, fmt.Errorf(
        "embedder factory for '%s' failed: %w (hint: use type: 'genkit-embedder' for Genkit providers)",
        name, err,
    )
}

func getProviderType(cfg core.ProviderOptions) string {
    if pn, ok := cfg.(interface{ ProviderName() string }); ok {
        return pn.ProviderName()
    }
    return "unknown"
}
```

**Key Improvements:**
- ✅ Consistent two-path pattern with vectorstores
- ✅ Better logging with helpful hints
- ✅ Clear separation of native vs. Genkit delegation
- ✅ Extensible via configuration

---

## Part 4: Configuration-Driven Provider Dispatch

### 4.1 Recommended Config Structure

```yaml
# config.yaml

components:
  # Native Manglekit embedder (existing)
  - name: local-embedder
    kind: embedder
    type: dense
    params:
      embedder: my-embedder
  
  # Generic Genkit embedders (NEW - any provider)
  - name: openai-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: openai
      model: text-embedding-3-small
      apiKey: "${OPENAI_API_KEY}"
  
  - name: google-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: google
      model: embedding-001
  
  - name: vertex-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: vertex
      model: textembedding-gecko@latest
  
  # Generic Genkit retrievers (NEW)
  - name: pinecone-retriever
    kind: retriever
    type: genkit-retriever
    params:
      provider: pinecone
      api_key: "${PINECONE_API_KEY}"
      index: my-index
  
  - name: chroma-retriever
    kind: retriever
    type: genkit-retriever
    params:
      provider: chroma
      collection: my-collection
```

**Benefits:**
- ✅ Users specify provider via config, not code
- ✅ Completely extensible (any Genkit plugin works)
- ✅ No need to update Manglekit for new providers
- ✅ Environment variable support for API keys

---

## Part 5: Eliminate Hard-Coded Provider Packages

### 5.1 Current Structure (Problem)

```
internal/embedders/
├── handler.go                    ← Single handler
├── openai/openai.go             ← Hard-coded provider
└── google/google.go             ← Hard-coded provider

internal/providers/llm/
├── handler.go                    ← Single handler
├── openai.go                     ← Hard-coded provider
└── google.go                     ← Hard-coded provider
```

**Issues:** Every new provider requires adding a new package

### 5.2 Recommended Structure (Solution)

```
internal/adapters/
├── genkit_embedder_adapter.go    ← Generic adapter (reusable)
└── genkit_retriever_adapter.go   ← Generic adapter (reusable)

internal/providers/embedders/
├── handler.go                    ← Single handler (improved)
└── genkit_embedder.go            ← Generic factory (all providers)

internal/providers/llm/
├── handler.go                    ← Single handler (improved)
└── genkit_llm.go                 ← Generic factory (all providers)

internal/providers/retrievers/
├── handler.go                    ← Single handler (improved)
└── genkit_retriever.go           ← Generic factory (all providers)
```

**Benefits:**
- ✅ One package per component type, not per provider
- ✅ Adding a new provider = updating dispatch logic, not creating new package
- ✅ Cleaner codebase
- ✅ Easier to maintain

---

## Part 6: Deletion Roadmap

### 6.1 Packages to Remove (After Generic Factories Implemented)

```
DELETE: internal/embedders/openai/
DELETE: internal/embedders/google/
DELETE: internal/providers/llm/openai.go
DELETE: internal/providers/llm/google.go
DELETE: internal/providers/retrievers/pinecone/  (if exists)
DELETE: internal/providers/retrievers/chroma/    (if exists)
```

**Migration Path:**
1. Users update config to use `type: "genkit-embedder"` instead of `type: "openai"`
2. Config params change from `type: "openai"` to `params: {provider: "openai"}`
3. Old hard-coded types become deprecated (with warnings in logs)
4. Final removal in next major version

---

## Part 7: Developer Experience Improvements

### 7.1 Error Messages & Debugging

**Current (Poor):**
```
error: factory for embedder 'my-embedder' failed: ...
```

**Recommended:**
```
error: embedder 'my-embedder' (provider: openai, model: text-embedding-3-small) failed:
  - Native factory not found for type "openai" 
  - Genkit delegation not configured
  - Hint: Set type: "genkit-embedder" to use Genkit OpenAI plugin
  - Hint: Ensure OPENAI_API_KEY environment variable is set

Supported embedder types:
  - genkit-embedder    (Any Genkit provider: openai, google, vertex, cohere, ...)
  - dense              (Manglekit native)
```

### 7.2 Configuration Validation

Add validation in config loader to:
- ✅ Check provider is valid (openai, google, vertex, etc.)
- ✅ Verify required fields (model, apiKey for some providers)
- ✅ Warn if using deprecated hard-coded types
- ✅ Suggest alternatives

### 7.3 Documentation Updates

```markdown
# Using Genkit Providers in Manglekit

## Embedders

### Using OpenAI (via Genkit)
\`\`\`yaml
components:
  - name: my-embedder
    kind: embedder
    type: genkit-embedder          # Generic Genkit type
    params:
      provider: openai             # Genkit OpenAI plugin
      model: text-embedding-3-small
      apiKey: "${OPENAI_API_KEY}"
\`\`\`

### Using Google Vertex AI (via Genkit)
\`\`\`yaml
components:
  - name: my-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: google
      model: textembedding-gecko@latest
\`\`\`

### Adding Custom Genkit Provider

1. Ensure Genkit plugin is initialized in your `genkit.Genkit` instance
2. Add provider case to `createGenkitEmbedder()` function
3. Update config with new provider name
4. NO code recompilation needed for config changes
```

---

## Part 8: Implementation Priority

### Phase 1 (Week 1): Core Adapters & Generic Factories
- [ ] Create `internal/adapters/genkit_embedder_adapter.go`
- [ ] Create `internal/adapters/genkit_retriever_adapter.go`
- [ ] Implement `internal/providers/embedders/genkit_embedder.go`
- [ ] Register in `providers/all/all.go`
- [ ] Write integration tests

### Phase 2 (Week 2): Handler Improvements
- [ ] Refactor `internal/embedders/handler.go` with two-path pattern
- [ ] Improve error messages & logging
- [ ] Add config validation hints
- [ ] Update tests

### Phase 3 (Week 3): Cleanup & Documentation
- [ ] Delete hard-coded provider packages
- [ ] Update provider registration
- [ ] Document migration path
- [ ] Add examples in `examples/`

### Phase 4 (Week 4): Extended Providers
- [ ] Implement `genkit_retriever.go` (parallel to embedder)
- [ ] Implement `genkit_llm.go` (parallel to embedder)
- [ ] Ensure consistent patterns across all handlers

---

## Part 9: Benefits Summary

### For Users
✅ **Flexibility:** Use ANY Genkit provider without Manglekit code changes  
✅ **Simplicity:** Single `genkit-embedder` type works for all Genkit embedders  
✅ **Configuration-Driven:** Providers specified in YAML, not code  
✅ **Future-Proof:** New Genkit plugins work automatically  
✅ **Easy Integration:** No need to wait for Manglekit to add support  

### For Maintainers
✅ **Reduced Complexity:** One generic factory vs. N provider packages  
✅ **Lower Maintenance:** No code changes needed for new providers  
✅ **Consistency:** All handlers follow same pattern  
✅ **Testability:** Generic adapters tested once, work for all providers  
✅ **Scalability:** Pattern extends easily to new component types  

### For Architecture
✅ **True Proxy Pattern:** Manglekit delegates to Genkit, doesn't wrap  
✅ **Separation of Concerns:** Genkit manages providers, Manglekit manages orchestration  
✅ **Transparency:** Users see Genkit providers clearly in config  
✅ **Extensibility:** Users can add custom adapters if needed  
✅ **Zero Lock-In:** Switch providers anytime, reconfigure YAML  

---

## Conclusion

Current implementation has excellent foundations but contains **unnecessary hard-coding** that limits extensibility. By adopting the **generic adapter + dynamic factory pattern**, Manglekit can become a true, transparent proxy for Genkit, maximizing developer experience and maintainability.

**Key Insight:** "Manglekit's job is orchestration, not provider integration. Let Genkit handle providers."
