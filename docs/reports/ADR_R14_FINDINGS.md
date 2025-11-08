# ADR R14 Compliance: Findings & Recommendations

**Date:** November 8, 2025  
**Rule:** ADR R14 - "Factories must not accept `diapi.Builder` (typed deps only)"  
**Overall Status:** ✅ **FULLY COMPLIANT**

---

## Key Finding

The COMPREHENSIVE_EVALUATION.md report contains an **inaccurate statement** about ADR R14 compliance:

### Inaccurate Statement (Line 113)
```
"Some providers still accept generic `diapi.Builder` instead of typed deps 
(partially resolved per ADR R14)"
```

### Actual Status
✅ **100% COMPLIANT** - All 10 component handlers and factories follow the typed DI pattern correctly.

---

## What ADR R14 Actually Requires

**Rule:** Factories must not accept `diapi.Builder` (typed deps only)

### Correct Implementation Pattern

```go
// ✅ CORRECT: Handler accepts Builder interface (external client)
func (h *Handler) BuildComponent(
    ctx context.Context,
    builderDI any,  // Type-asserted to diapi.Builder
    factory any,
    resolved *core.Resolved,
    cfg core.ProviderOptions,
    name string,
) (core.ResourceCloser, error) {
    // Step 1: Type-assert to diapi.Builder interface
    b, ok := builderDI.(diapi.Builder)
    
    // Step 2: Resolve specific dependencies
    embedder, err := b.GetEmbedder(name)
    
    // Step 3: Construct typed dependency struct
    deps := diapi.RerankerDeps{
        CoreDeps: b.GetCoreDeps(),
        Embedder: embedder,
    }
    
    // Step 4: Pass TYPED deps to factory (NOT builder)
    built, err := f.Build(ctx, deps, cfg)  // ✅ deps is typed
}

// ✅ CORRECT: Factory receives typed dependencies
func NewReranker(
    ctx context.Context,
    deps diapi.RerankerDeps,  // Typed struct, not Builder
    cfg CosineOptions,
) (core.Reranker, error) {
    // Use deps.CoreDeps, deps.Embedder
    // Never receive the builder
}
```

### Incorrect Pattern (NOT FOUND IN CODEBASE)

```go
// ❌ WRONG: Factory accepts Builder directly
func NewReranker(
    ctx context.Context,
    deps diapi.Builder,  // ❌ This violates ADR R14
    cfg CosineOptions,
) (core.Reranker, error) {
    // This would be a violation
}
```

---

## Scan Results: All 10 Handlers Verified

| Handler | File | Status | Typed Deps Struct |
|---------|------|--------|-------------------|
| Retriever | `internal/providers/retrievers/handler.go` | ✅ | `RetrieverDeps`, `DenseRetrieverDeps`, `NoopDeps` |
| LLM | `internal/providers/llm/handler.go` | ✅ | `LLMDeps` |
| Embedder | `internal/embedders/handler.go` | ✅ | `EmbedderDeps` |
| Reranker | `internal/providers/rerank/handler.go` | ✅ | `RerankerDeps` |
| VectorStore | `internal/vectorstores/handler.go` | ✅ | `VectorStoreDeps`, `NoopDeps` |
| StateProvider | `internal/providers/state/handler.go` | ✅ | `StateProviderDeps` |
| RuleSet | `internal/providers/rules/handler.go` | ✅ | `RuleSetDeps` |
| SchemaParser | `internal/providers/schemaparsers/handler.go` | ✅ | `NoopDeps` |
| Sandwich Orchestrator | `pipeline/sandwich/handler.go` | ✅ | `SandwichDeps` |
| Declarative Orchestrator | `pipeline/declarative/handler.go` | ✅ | `DeclarativeOrchestratorDeps` |

**Result:** 10/10 handlers compliant ✅

---

## Why This Matters (Type Safety Benefits)

### 1. Compile-Time Safety
```go
// ✅ Compile-time error if factory signature changes
deps := diapi.RerankerDeps{
    CoreDeps: b.GetCoreDeps(),
    Embedder: embedder,
}
// If RerankerDeps changes, compiler catches it immediately
```

### 2. Self-Documenting Code
```go
// ✅ Clear what dependencies are needed
func NewReranker(ctx context.Context, deps diapi.RerankerDeps, cfg CosineOptions)
// Reader immediately knows: CoreDeps + Embedder are required
```

### 3. No Runtime Type Assertions in Factories
```go
// ❌ If factory accepted Builder, it would need:
embedder := deps.(diapi.Builder).GetEmbedder(name)  // Runtime assertion!

// ✅ With typed deps, no assertion needed:
embedder := deps.Embedder  // Direct field access
```

### 4. Testability
```go
// ✅ Easy to mock typed dependencies
mockDeps := diapi.RerankerDeps{
    CoreDeps: mockCoreDeps,
    Embedder: mockEmbedder,
}
reranker, err := NewReranker(ctx, mockDeps, opts)
```

---

## Documentation Update Recommendation

### Current Statement (COMPREHENSIVE_EVALUATION.md:113)
```markdown
**Weaknesses:**
- Some providers still accept generic `diapi.Builder` instead of typed deps 
  (partially resolved per ADR R14)
```

### Recommended Replacement
```markdown
**Strengths:**
- ✅ All providers receive typed dependency structs (full ADR R14 compliance)
- ✅ Handlers correctly resolve dependencies from diapi.Builder interface
- ✅ Factories never receive the builder directly
- ✅ Type-safe DI eliminates entire classes of runtime errors
```

### Recommended Addition to Section 1.3
```markdown
### 1.3 Dependency Injection System

**Rating: ⭐⭐⭐⭐⭐ (Excellent)**

The DI system is type-safe and fully compliant with ADR R14:

**Strengths:**
- ✅ All 10 handlers correctly use diapi.Builder interface
- ✅ All handlers resolve specific dependencies from builder
- ✅ All handlers construct typed diapi.*Deps structs
- ✅ All factories receive ONLY typed dependencies, never the builder
- ✅ Compile-time type safety throughout the DI chain
- ✅ No runtime type assertions in factories
- ✅ Self-documenting dependency requirements

**Evidence:**
- All handler implementations in internal/providers/* and pipeline/*
- Typed dependency structs in core/diapi/di.go
- Factory signatures accept typed deps, not Builder
```

---

## Verification Methodology

### Code Review Approach
1. ✅ Examined all 10 component handlers
2. ✅ Verified each handler's `BuildComponent` method
3. ✅ Confirmed type-assertion to `diapi.Builder` interface
4. ✅ Verified construction of typed dependency structs
5. ✅ Confirmed factories receive typed deps, not builder
6. ✅ Cross-referenced with core/diapi/di.go definitions

### Files Reviewed
- `internal/providers/retrievers/handler.go`
- `internal/providers/llm/handler.go`
- `internal/providers/rerank/handler.go`
- `internal/providers/state/handler.go`
- `internal/providers/rules/handler.go`
- `internal/providers/schemaparsers/handler.go`
- `internal/embedders/handler.go`
- `internal/vectorstores/handler.go`
- `pipeline/sandwich/handler.go`
- `pipeline/declarative/handler.go`
- `core/diapi/di.go` (dependency struct definitions)

---

## Recommendations

### Priority 1: Documentation Update (Immediate)
- [ ] Update COMPREHENSIVE_EVALUATION.md line 113 to reflect actual compliance
- [ ] Add detailed explanation of ADR R14 compliance to section 1.3
- [ ] Reference the new DI_COMPLIANCE_SCAN.md report

### Priority 2: Documentation Enhancement (Next Release)
- [ ] Create a "DI Pattern Guide" for developers implementing new providers
- [ ] Document the handler → typed deps → factory pattern
- [ ] Add code examples showing correct vs. incorrect patterns
- [ ] Include in CONTRIBUTING.md

### Priority 3: Automated Verification (Future)
- [ ] Add static analysis rule to verify factories don't accept `diapi.Builder`
- [ ] Add linter rule to check handler patterns
- [ ] Include in CI/CD pipeline

---

## Conclusion

**Manglekit's dependency injection system is production-ready and fully compliant with ADR R14.** The codebase demonstrates excellent type safety practices and should serve as a reference implementation for Go DI patterns.

The inaccuracy in COMPREHENSIVE_EVALUATION.md appears to be a documentation oversight rather than a code issue. All handlers and factories correctly implement the typed DI pattern.

---

## Related Documents

- [`docs/reports/DI_COMPLIANCE_SCAN.md`](DI_COMPLIANCE_SCAN.md) - Detailed handler-by-handler analysis
- [`docs/ADR.md`](../ADR.md) - Architecture Decision Records
- [`core/diapi/di.go`](../../core/diapi/di.go) - Typed dependency struct definitions
- [`docs/CONTEXT.md`](../CONTEXT.md) - Live architecture standard

---

**Report Generated:** November 8, 2025  
**Confidence Level:** 100% (all handlers manually reviewed)  
**Status:** ✅ No code changes required - documentation update recommended
