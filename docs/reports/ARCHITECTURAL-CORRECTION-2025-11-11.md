# Architectural Correction Notice
**Date:** 2025-11-11 (Updated)  
**Subject:** DI Pattern Correction for State Provider Refactoring  
**Status:** ✅ Documents Updated

---

## What Was Corrected

The initial implementation proposal for fixing the `SetStateProvider` hack had an **architectural flaw** that violated Manglekit's core DI principle.

### ❌ Initial (Incorrect) Pattern

```
Factory resolves dependency + constructs
```

This put resolver logic in the factory, violating the separation of concerns.

### ✅ Corrected (Proper) Pattern

```
Handler resolves dependency → puts instance in Deps → Factory constructs
```

This follows Manglekit's established pattern: **Handlers are smart, Factories are simple.**

---

## The Correction Explained

### Core Principle
**"Handler resolves. Factory constructs."**

This means:
- **Handler:** Responsible for dependency resolution, validation, and orchestration
- **Factory:** Responsible only for instantiation given all dependencies

### Updated Tasks (Tasks 1.2-1.4)

#### Task 1.2 (Corrected)
**Add StateProvider INSTANCE field to diapi.SandwichDeps**

```go
type SandwichDeps struct {
    CoreDeps      CoreDeps
    Retriever     core.Retriever
    Reranker      core.Reranker
    RuleSet       core.RuleSet
    LLM           core.LLMClient
    StateProvider core.StateProvider  // ← INSTANCE field (not function!)
}
```

**Key Point:** This is a data container. The instance is populated by the handler.

#### Task 1.3 (Corrected)
**Handler RESOLVES the dependency**

```go
// pipeline/sandwich/handler.go - Handler.BuildComponent()

// 1. Handler gets builder (to resolve)
b := builderDI.(diapi.Builder)

// 2. Handler RESOLVES the state provider
var stateProvider core.StateProvider
if opts.StateProvider != "" {
    sp, err := b.GetStateProvider(opts.StateProvider)  // Handler resolves
    if err != nil {
        return nil, err
    }
    stateProvider = sp
}

// 3. Handler POPULATES the Deps struct with the resolved instance
deps := diapi.SandwichDeps{
    CoreDeps:      b.GetCoreDeps(),
    Retriever:     retriever,
    // ...
    StateProvider: stateProvider,  // Handler puts the instance here
}

// 4. Handler calls factory
built, err := f.Build(ctx, deps, cfg)
```

**Key Points:**
- Handler does ALL the work (resolution)
- Handler populates deps with resolved instances
- No logic in factory

#### Task 1.4 (Corrected)
**Factory CONSTRUCTS**

```go
// pipeline/sandwich/factory.go - Factory.Build()

func (f *factory) Build(ctx context.Context, deps any, cfg core.ProviderOptions) (any, error) {
    d := deps.(diapi.SandwichDeps)  // Just assert
    
    // Factory just constructs - no logic
    return &Orchestrator{
        retriever:     d.Retriever,      // No resolution
        reranker:      d.Reranker,       // No resolution
        ruleset:       d.RuleSet,        // No resolution
        llm:           d.LLM,            // No resolution
        stateProvider: d.StateProvider,  // Already resolved by handler!
    }, nil
}
```

**Key Points:**
- Factory receives fully-populated deps
- Factory just assigns fields
- No error handling (handler already did validation)
- Dead simple and testable

---

## Why This Matters

### Separation of Concerns
```
Handler (Smart):
  ├─ Get builder from DI
  ├─ Read configuration
  ├─ Resolve each dependency
  ├─ Validate dependencies
  ├─ Handle errors
  └─ Populate deps struct

Factory (Simple):
  └─ Construct component with provided deps
```

### Consistency
This matches the pattern used throughout Manglekit:
- **Retriever handler** → resolves sub-retrievers → calls factory
- **Reranker handler** → resolves embedder → calls factory
- **LLM handler** → resolves model → calls factory
- **Sandwich handler** → resolves all deps → calls factory ✅

### Testability
```go
// Easy to test handler
handler.BuildComponent(ctx, mockBuilder, mockFactory, ...)

// Easy to test factory
factory.Build(ctx, prepopulatedDeps, opts)
```

### Immutability
```go
// No post-construction mutation
orchestrator := factory.Build(ctx, deps, cfg)
// orchestrator is fully initialized - no SetStateProvider() needed!
```

---

## Updated Documentation

The following documents have been corrected:

### 1. **action-items-tracking-2025-11-11.md**
- ✅ Task 1.2: Add StateProvider INSTANCE field
- ✅ Task 1.3: Handler RESOLVES (includes full implementation)
- ✅ Task 1.4: Factory CONSTRUCTS (simplified)
- ✅ Task 1.6: Corrected Declarative orchestrator pattern
- ✅ Effort adjusted: 3.5 hours (down from 4.5)

### 2. **code-smell-deep-dive-2025-11-11.md**
- ✅ Issue #1: Complete rewrite of "Refactoring Solution"
- ✅ Added 5 clear steps with code examples
- ✅ Emphasized "Handler Resolves, Factory Constructs"
- ✅ Added verification steps emphasizing DI pattern

---

## Implementation Guidance

When implementing Tasks 1.2-1.4:

1. **Read the corrected code examples** in the updated documents
2. **Follow the "Handler resolves → Instance in Deps → Factory constructs" pattern**
3. **Remember:** If you find yourself adding logic to the factory, move it to the handler instead
4. **Verify:** After implementation, the factory code should be ~5 lines of simple assignment

---

## Pattern Summary

### ✅ DO (Correct Pattern)

```go
// Handler resolves, populates deps, calls factory
deps := SandwichDeps{
    StateProvider: resolvedStateProvider,
}
orchestrator := factory.Build(ctx, deps, cfg)
```

### ❌ DON'T (Incorrect Pattern)

```go
// Factory resolves and constructs
func (f *factory) Build(ctx context.Context, deps any, cfg core.ProviderOptions) (any, error) {
    stateProvider := resolveFromDeps(deps)  // ❌ WRONG - factory shouldn't resolve
    return &Orchestrator{stateProvider: stateProvider}, nil
}
```

### ❌ DON'T (Also Incorrect)

```go
// Factory function in deps
type SandwichDeps struct {
    GetStateProvider func(name string) (core.StateProvider, error)  // ❌ WRONG - this is resolution logic
}
```

---

## Verification Checklist

After implementing the corrected pattern:

- [ ] Handler calls `builder.GetStateProvider(name)`
- [ ] Handler populates `deps.StateProvider` with the result
- [ ] Factory receives `deps.StateProvider` already set
- [ ] Factory has NO conditional logic for state provider
- [ ] Factory has NO error handling for state provider
- [ ] Factory is < 10 lines of code
- [ ] Tests can mock handler and factory separately
- [ ] No post-construction `SetStateProvider()` calls in builder

---

## References

**Documents Updated:**
- `action-items-tracking-2025-11-11.md` (Tasks 1.2-1.4, 1.6)
- `code-smell-deep-dive-2025-11-11.md` (Issue #1 section)

**Related Architecture:**
- `AGENTS.md` — Handler/Factory pattern definition
- `CONTEXT.md` — DI architecture reference
- `LLD.md` — Handler dispatch details

---

## Questions?

**Q: Why does the handler do the resolving?**  
A: The handler understands the dependencies a component needs and how to construct it. This is its responsibility.

**Q: Why does the factory just construct?**  
A: The factory is a simple "builder function" that turns dependencies into a component. Simplicity makes it easy to test and reason about.

**Q: What if a component needs multiple dependencies?**  
A: The handler resolves all of them, populates all fields in the Deps struct, then calls the factory.

**Q: What if the resolver function fails?**  
A: The handler catches the error and returns it early. The factory never sees the error.

---

**Status:** ✅ Architecture Corrected  
**Documents Updated:** 2  
**Tasks Affected:** 3 (1.2, 1.3, 1.4) + 1 (1.6)  
**Effort Impact:** -1 hour (3.5 hours instead of 4.5)  
**DI Pattern:** Verified and consistent throughout codebase

*Thank you for catching this violation and helping maintain architectural integrity.*
