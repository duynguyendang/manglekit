# Code Smell Deep Dive Analysis
**Generated:** 2025-11-11  
**Purpose:** Detailed technical analysis of identified issues with code examples and verification steps

---

## Issue #1: SetStateProvider Hack Pattern — Detailed Analysis

### The Problem

**File:** `builder.go` (lines 216-222)

```go
if stateProviderName != "" {
    sp, ok := b.stateProviders[stateProviderName]
    if !ok {
        return nil, nil, fmt.Errorf("state provider %q not found", stateProviderName)
    }
    // This is a bit of a hack, but it's the only way to get the state provider to the orchestrator for now.
    // A better solution would be to have the orchestrator handler resolve its own dependencies.
    if orchWithState, ok := orchestrator.(interface{ SetStateProvider(core.StateProvider) }); ok {
        orchWithState.SetStateProvider(sp)
    }
}
```

### Why It's a Problem

1. **Runtime Type Assertion:** Uses `interface{}` duck typing to check if orchestrator supports `SetStateProvider`
2. **Post-Construction Mutation:** State provider is set after the orchestrator is fully constructed
3. **Implicit Dependency:** The builder must know about this pattern; not part of the orchestrator handler's responsibility
4. **Acknowledged Debt:** The comment admits this is a workaround

### Current Implementation

**Orchestrator Options Structure:**

```go
// pipeline/sandwich/options.go
type Options struct {
    PreRules              string `yaml:"pre_rules,omitempty"`
    Retriever             string `yaml:"retriever,omitempty"`
    Reranker              string `yaml:"reranker,omitempty"`
    LLM                   string `yaml:"llm,omitempty"`
    PostRules             string `yaml:"post_rules,omitempty"`
    StateProvider         string `yaml:"state_provider,omitempty"` // ← SET BUT NOT USED
    TopK                  int    `yaml:"topk,omitempty"`
    MaxTokens             int    `yaml:"max_tokens,omitempty"`
    FallbackThreshold     float64 `yaml:"fallback_threshold,omitempty"`
}
```

The `StateProvider` field exists in the options but is **never read by the handler**. Instead, the builder reads the config and calls `SetStateProvider()` afterward.

### Refactoring Solution

**CRITICAL PATTERN:** Handler Resolves, Factory Constructs

This fix follows Manglekit's core DI principle. The handler is responsible for resolving dependencies; the factory only constructs.

**Step 1: Add GetStateProvider to diapi.Builder**

```go
// core/diapi/di.go
type Builder interface {
    // ... existing methods ...
    GetStateProvider(name string) (core.StateProvider, error)
}
```

**Step 2: Add StateProvider instance field to diapi.SandwichDeps**

```go
// core/diapi/di.go
type SandwichDeps struct {
    CoreDeps      CoreDeps
    Retriever     core.Retriever
    Reranker      core.Reranker
    RuleSet       core.RuleSet
    LLM           core.LLMClient
    StateProvider core.StateProvider  // ← Add INSTANCE field (not a function!)
}
```

**Step 3: Handler RESOLVES the dependency (pipeline/sandwich/handler.go)**

```go
// The Handler does the RESOLUTION work
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
        return nil, fmt.Errorf("invalid builder DI type")
    }

    opts := cfg.(*sandwich.Options)
    
    // 1. Handler RESOLVES the state provider dependency
    var stateProvider core.StateProvider
    if opts.StateProvider != "" {
        sp, err := b.GetStateProvider(opts.StateProvider)
        if err != nil {
            return nil, fmt.Errorf("failed to get state provider '%s': %w", opts.StateProvider, err)
        }
        stateProvider = sp
    }

    // 2. Handler POPULATES the Deps struct with the resolved instance
    deps := diapi.SandwichDeps{
        CoreDeps:      b.GetCoreDeps(),
        Retriever:     retriever,
        Reranker:      reranker,
        RuleSet:       ruleset,
        LLM:           llm,
        StateProvider: stateProvider,  // ← Instance is set by handler
    }

    // 3. Handler calls factory (factory just constructs, no logic)
    built, err := factory.(core.Factory).Build(ctx, deps, cfg)
    if err != nil {
        return nil, fmt.Errorf("factory failed: %w", err)
    }

    orchestrator := built.(core.Orchestrator)
    if err := b.SetOrchestrator(name, orchestrator); err != nil {
        return nil, fmt.Errorf("failed to set orchestrator: %w", err)
    }
    return core.NopCloser, nil
}
```

**Step 4: Factory CONSTRUCTS (pipeline/sandwich/factory.go)**

```go
// The Factory just CONSTRUCTS - no logic
func NewFactory() core.Factory {
    return func(ctx context.Context, deps any, cfg core.ProviderOptions) (any, error) {
        d := deps.(diapi.SandwichDeps)  // Just assert the type
        opts := cfg.(*Options)

        // Factory does NOT resolve - just constructs with what the Handler provided
        return &Orchestrator{
            retriever:         d.Retriever,
            reranker:          d.Reranker,
            ruleset:           d.RuleSet,
            llm:               d.LLM,
            stateProvider:     d.StateProvider,  // <-- Already resolved by handler
            obs:               d.CoreDeps.Obs,
            topK:              opts.TopK,
            maxTokens:         opts.MaxTokens,
            fallbackThreshold: opts.FallbackThreshold,
        }, nil
    }
}
```

**Step 5: Update builder to remove the hack**

```go
// builder.go - REMOVE THIS SECTION
// if stateProviderName != "" {
//     sp, ok := b.stateProviders[stateProviderName]
//     if !ok {
//         return nil, nil, fmt.Errorf("state provider %q not found", stateProviderName)
//     }
//     if orchWithState, ok := orchestrator.(interface{ SetStateProvider(core.StateProvider) }); ok {
//         orchWithState.SetStateProvider(sp)
//     }
// }
```

### Verification Steps

After implementing this refactoring:

1. ✅ `builder.Build()` does NOT call `SetStateProvider()`
2. ✅ **Handler** reads `StateProvider` from options and resolves via `builder.GetStateProvider()`
3. ✅ **Handler** populates `SandwichDeps.StateProvider` with the resolved instance
4. ✅ **Factory** receives fully-populated deps and just constructs
5. ✅ No runtime type assertions for state provider setup
6. ✅ Orchestrator is fully immutable after construction
7. ✅ **DI Pattern verified:** Handler resolves → Instance in Deps → Factory constructs

---

## Issue #2: Non-Deterministic Map Iteration in Rules Module

### The Problem

**File:** `internal/providers/rules/mangle/rules.go`

Multiple locations iterate over maps without sorting:

```go
// Line 140
for p := range edbDecls {
    log.Debugf("mangle predicate registered", "predicate", p.Symbol, "arity", p.Arity)
}

// Line 289
for r := range denied {
    // Process denial rules
}

// Line 444
for id := range dropReasons {
    // Log drop reasons
}

// Line 911
for v := range results {
    // Aggregate results
}
```

### Why It's a Problem

Go's map iteration order is **explicitly randomized** per spec. This means:
- Debug logs will vary between runs (confusing for troubleshooting)
- Test output is non-deterministic (flaky tests)
- Reproducibility is compromised
- Difficult to generate consistent reports

### Current Implementation

```go
// Status: UNFIXED
// Maps are iterated directly without sorting
```


### Refactoring Solution

**Pattern: Extract, Sort, Iterate**

```go
// Line 140 - Before
for p := range edbDecls {
    log.Debugf("mangle predicate registered", "predicate", p.Symbol, "arity", p.Arity)
}

// Line 140 - After
var sortedPredicates []ast.PredicateSym
for p := range edbDecls {
    sortedPredicates = append(sortedPredicates, p)
}
sort.Slice(sortedPredicates, func(i, j int) bool {
    if sortedPredicates[i].Symbol != sortedPredicates[j].Symbol {
        return sortedPredicates[i].Symbol < sortedPredicates[j].Symbol
    }
    return sortedPredicates[i].Arity < sortedPredicates[j].Arity
})
for _, p := range sortedPredicates {
    log.Debugf("mangle predicate registered", "predicate", p.Symbol, "arity", p.Arity)
}
```

### Affected Locations and Fixes

| Line | Type | Keys | Sort Criterion |
|------|------|------|---|
| 140 | `edbDecls` | `ast.PredicateSym` | `Symbol, Arity` |
| 289 | `denied` | `string` | Lexicographic |
| 444 | `dropReasons` | `string` | Lexicographic |
| 911 | `results` | `*ast.Atom` | `String()` representation |

### Verification Steps

1. ✅ All map iterations have extract-sort-iterate pattern
2. ✅ Tests generate consistent output across runs
3. ✅ Debug logs are deterministic
4. ✅ No change to functionality, only iteration order

---

## Issue #3: Implicit Orchestrator State Injection Design Inconsistency

### The Problem

**Files:**
- `pipeline/declarative/orchestrator.go` (line 44-47)
- `pipeline/sandwich/sandwich.go` (line 29)

Both orchestrators declare a `StateProvider` field:

```go
// Declarative
type DeclarativeOrchestrator struct {
    steps               []executionStep
    StateProvider       core.StateProvider // ← Can be set post-construction
    obs                 core.Observability
    // ...
}

// Sandwich
type Orchestrator struct {
    retriever           core.Retriever
    stateProvider       core.StateProvider // ← Can be set post-construction
    // ...
}
```

But the **only way to set it** is via the builder hack:

```go
if orchWithState, ok := orchestrator.(interface{ SetStateProvider(core.StateProvider) }); ok {
    orchWithState.SetStateProvider(sp)
}
```

### Why It's a Problem

1. **Two paths to configure:** Options can specify state provider name, but it's not used
2. **Mutability:** Orchestrator is mutable after construction (violates immutability principle)
3. **Implicit API:** The `SetStateProvider()` method exists but is not documented in the options
4. **Inconsistent DI:** State provider is injected differently than other dependencies

### Design Inconsistency Pattern

| Dependency | Resolution Path | Timing |
|------------|---|---|
| Retriever | Options → Handler | Construction |
| Reranker | Options → Handler | Construction |
| RuleSet | Options → Handler | Construction |
| LLM | Options → Handler | Construction |
| **StateProvider** | **Builder hack → SetStateProvider()** | **Post-construction** ❌ |

### Comparison with Correct Pattern

**Correct (Retriever):**
```yaml
components:
  - name: my_retriever
    type: hybrid
    kind: retriever
    options:
      retrievers: [bm25, dense]  # ← Specified in options
```

```go
// Handler resolves it during construction
deps := diapi.RetrieverDeps{
    SubRetrievers: map[string]core.Retriever{...}
}
```

**Incorrect (StateProvider):**
```yaml
components:
  - name: my_sandwich
    type: sandwich
    kind: orchestrator
    options:
      state_provider: redis  # ← Specified but ignored by handler!
```

```go
// Builder ignores options.StateProvider and uses a hack instead
if orchWithState, ok := orchestrator.(interface{ SetStateProvider(core.StateProvider) }); ok {
    orchWithState.SetStateProvider(sp)  // ← Post-construction mutation
}
```

### Root Cause

The orchestrator handler was designed before the full DI system was in place. The state provider field was added to options but never integrated into the handler's dependency resolution.

### Refactoring Solution

Align with the pattern used for other dependencies:

**Step 1: Update options to include StateProvider explicitly**

```yaml
components:
  - name: my_sandwich
    type: sandwich
    kind: orchestrator
    options:
      retriever: bm25_retriever
      reranker: cosine_reranker
      pre_rules: mangle_pre
      post_rules: mangle_post
      llm: openai_gpt4
      state_provider: redis  # ← Explicit and honored
      topk: 10
```

**Step 2: Handler resolves it during construction (see Issue #1 solution)**

This is already partially solved by fixing Issue #1.

---

## Issue #4: Unhandled Resource Closer Edge Cases

### The Problem

**File:** `builder.go` (lines 229-239)

```go
func (b *builder) closeResources(ctx context.Context) error {
    closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    var combined error
    for i := len(b.opts.ResourceClosers) - 1; i >= 0; i-- {
        if err := b.opts.ResourceClosers[i](closeCtx); err != nil {
            combined = errors.Join(combined, err)
        }
    }
    return combined
}
```

### Issues

1. **Hard-coded 5-second timeout:** May be inappropriate for:
   - Quick in-memory closers (excessive waiting)
   - Long I/O operations (insufficient time)
   - External API closures (unpredictable)

2. **Silent individual failures:** If a closer fails partway through, that's only visible if the caller checks the error

3. **No logging:** Developers debugging resource leaks have no visibility into closer failures

### Current Behavior

✅ **Correct:**
- Uses `errors.Join()` to aggregate multiple failures
- Iterates in reverse order (LIFO — good practice for cleanup)
- Applies timeout to prevent indefinite hangs
- Returns combined error

⚠️ **Could be improved:**
- Timeout value is not configurable
- No per-closer logging
- Silent failures within the timeout

### Refactoring Solution

**Step 1: Add timeout configuration to builder options**

```go
type BuilderOptions struct {
    Obs               core.Observability
    ResourceClosers   []core.ResourceCloser
    CleanupTimeout    time.Duration  // ← NEW: Default 5 seconds
}
```

**Step 2: Add logging to closeResources**

```go
func (b *builder) closeResources(ctx context.Context) error {
    closeCtx, cancel := context.WithTimeout(ctx, b.opts.CleanupTimeout)
    defer cancel()
    
    var combined error
    for i := len(b.opts.ResourceClosers) - 1; i >= 0; i-- {
        closerIndex := i // Capture for logging
        if err := b.opts.ResourceClosers[closerIndex](closeCtx); err != nil {
            b.opts.Obs.Logger.Warnf(
                "resource closer failed",
                "closer_index", closerIndex,
                "error", err,
            )
            combined = errors.Join(combined, err)
        }
    }
    return combined
}
```

**Step 3: Document timeout expectations**

Add to CONTEXT.md or code comments:
```
Resource Cleanup Timeout (default 5s):
- In-memory closers should complete within 100ms
- Database connections should gracefully close within 2s
- External API clients should timeout after 3s
If your closer exceeds this timeout, it will be forcibly cancelled.
```

### Verification Steps

1. ✅ Timeout is configurable via builder options
2. ✅ Each closer failure is logged with index and error
3. ✅ Existing timeout behavior is preserved by default
4. ✅ Documentation clearly states cleanup expectations

---

## Issue #5: Handler Dispatch Non-Extensibility

### The Problem

**File:** `internal/providers/retrievers/handler.go` (lines 43-100)

```go
switch typedOpts := opts.(type) {
case diapi.SubRetrieversDep:
    hybridDeps := diapi.RetrieverDeps{
        CoreDeps:      coreDeps,
        SubRetrievers: make(map[string]core.Retriever),
    }
    for _, subName := range typedOpts.GetSubRetrievers() {
        r, err := b.GetRetriever(subName)
        if err != nil {
            return nil, fmt.Errorf("failed to get sub-retriever '%s': %w", subName, err)
        }
        hybridDeps.SubRetrievers[subName] = r
    }
    deps = hybridDeps

case diapi.EmbedderDep:
    embedder, err := b.GetEmbedder(typedOpts.GetEmbedder())
    if err != nil {
        return nil, fmt.Errorf("failed to get embedder: %w", err)
    }
    // ...
    deps = diapi.DenseRetrieverDeps{...}

default:
    deps = diapi.NoopDeps{CoreDeps: coreDeps}
}
```

### Why It's a Problem

1. **Handler modification required:** Adding a new retriever type requires modifying the handler
2. **Violates Open/Closed Principle:** Open for extension, closed for modification
3. **Type-switch hell:** Many cases make the code harder to read and maintain
4. **Not extensible without recompile:** New retriever types can't be added via plugins

### Current Architecture

```
Config (YAML)
    ↓
Handler.BuildComponent()
    ↓
Type-switch on options
    ├─ SubRetrieversDep → Hybrid
    ├─ EmbedderDep → Dense
    └─ Default → Noop
    ↓
Factory.Build()
    ↓
Retriever instance
```

### Potential Refactoring: Registry-Based Dispatch

This is **not a critical issue** because:
1. New retrievers are added infrequently
2. The type-switch pattern is explicit and clear
3. Performance is not affected
4. The solution adds complexity without clear benefit

### Alternative Pattern (if needed in future)

```go
// Register dependency resolvers per-type
type DepResolverRegistry struct {
    resolvers map[string]func(b diapi.Builder, opts any) (any, error)
}

// In handler:
resolver, ok := depRegistry.Get(opts.ProviderName())
if !ok {
    return nil, fmt.Errorf("no resolver for provider %s", opts.ProviderName())
}
deps, err := resolver(b, opts)
if err != nil {
    return nil, err
}
```

### Verdict

**No immediate action needed.** This is a design trade-off between extensibility and simplicity. The current approach is acceptable for a closed set of provider types.

---

## Summary Table

| Issue | Severity | Status | Fix Complexity | Priority |
|-------|----------|--------|---|---|
| SetStateProvider Hack | Medium | Recommended | Medium | 1 |
| Map Iteration Non-Determinism | Low | Recommended | Low | 3 |
| Orchestrator State Injection Inconsistency | Low | Consistency Issue | Medium | 2 |
| Resource Closer Hard-coded Timeout | Low | Enhancement | Low | 3 |
| Handler Dispatch Non-Extensibility | Low | Design Trade-off | High | 4 |

---

## Appendix: Verification Checklist

- [ ] **Issue #1:** State provider hack refactoring implemented and tested
- [ ] **Issue #2:** All map iterations in rules module use extract-sort-iterate
- [ ] **Issue #3:** Orchestrator state injection aligned with other dependencies
- [ ] **Issue #4:** Cleanup timeout is configurable with debug logging
- [ ] **Issue #5:** Document handler dispatch pattern and extensibility limitations

---

*Deep dive analysis generated by comprehensive code review*  
*Reference: code-smell-review-2025-11-11.md*
