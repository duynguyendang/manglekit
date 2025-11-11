# Pattern Comparison: Before vs. After

**Date:** 2025-11-11  
**Purpose:** Visual comparison of the old hack vs. the correct DI pattern

---

## Current State: Dual Paths (Hack + DI)

Currently, both patterns are in the codebase simultaneously:

### ❌ Path 1: Builder Hack (Old Way - To Be Removed)

```
builder.Build()
    ↓
Handler resolves ALL dependencies (retriever, llm, reranker, etc.)
    ↓
Handler calls Factory
    ↓
Factory constructs Orchestrator WITH state provider in constructor
    ↓
Orchestrator is FULLY INITIALIZED
    ↓
builder.BuildOrchestrator() continues...
    ↓
HACK: Builder reads state provider from config AGAIN
    ↓
Builder calls orchestrator.SetStateProvider() via duck typing
    ↓
POST-CONSTRUCTION MUTATION ❌
```

**File Location:** `builder.go:216-225`

**Code:**
```go
if stateProviderName != "" {
    sp, ok := b.stateProviders[stateProviderName]
    if !ok {
        return nil, nil, fmt.Errorf("state provider %q not found", stateProviderName)
    }
    // This is a bit of a hack...
    if orchWithState, ok := orchestrator.(interface{ SetStateProvider(core.StateProvider) }); ok {
        orchWithState.SetStateProvider(sp)
    }
}
```

**Problems:**
- Post-construction mutation (antipattern)
- Runtime duck typing (not type-safe)
- Builder has orchestrator-specific knowledge (violates ADR 7)
- Work is done twice (redundant)
- Immutability violated

---

### ✅ Path 2: Correct DI Pattern (Active Now - Will Be Exclusive)

```
builder.Build()
    ↓
Handler gets Builder from DI
    ↓
Handler calls b.GetStateProvider(opts.StateProvider)
    ↓
Handler receives resolved state provider
    ↓
Handler populates SandwichDeps.StateProvider with the instance
    ↓
Handler calls Factory with fully-populated deps
    ↓
Factory asserts deps type
    ↓
Factory assigns deps.StateProvider to orchestrator
    ↓
Orchestrator is FULLY INITIALIZED with all dependencies ✅
    ↓
No post-construction mutation needed
```

**File Locations:**
- `core/diapi/di.go:13` - Builder.GetStateProvider() signature
- `core/diapi/di.go:125` - SandwichDeps.StateProvider instance field
- `pipeline/sandwich/handler.go:63-82` - Handler resolution logic
- `pipeline/sandwich/factory.go:30-42` - Factory construction logic

**Handler Code:**
```go
// Handler RESOLVES the dependency
var stateProvider core.StateProvider
if opts.StateProvider != "" {
    stateProvider, err = b.GetStateProvider(opts.StateProvider)
    if err != nil {
        return nil, fmt.Errorf("failed to get state provider: %w", err)
    }
}

// Handler POPULATES deps with resolved instance
deps := diapi.SandwichDeps{
    CoreDeps:      b.GetCoreDeps(),
    Retriever:     retriever,
    LLM:           llm,
    Reranker:      reranker,
    RuleSet:       ruleSet,
    StateProvider: stateProvider,  // ← Instance set here
}

// Handler calls factory
built, err := factory.(core.Factory).Build(ctx, deps, cfg)
```

**Factory Code:**
```go
// Factory just constructs
s := &Orchestrator{
    retriever:     d.Retriever,
    reranker:      d.Reranker,
    ruleset:       d.RuleSet,
    llm:           d.LLM,
    stateProvider: d.StateProvider,  // ← No resolution, just assign
    obs:           d.Obs,
    // ... other fields ...
}
```

**Benefits:**
- ✅ Immutable after construction
- ✅ Type-safe (no duck typing)
- ✅ Handler owns resolution logic
- ✅ Factory is simple (just constructs)
- ✅ One clear path for initialization
- ✅ ADR 7 compliant

---

## After Removal: Single Correct Path

Once we delete `builder.go:216-225`:

```
builder.Build()
    ↓
Handler resolves ALL dependencies
    ↓
Handler populates typed Deps struct
    ↓
Handler calls Factory with deps
    ↓
Factory constructs Orchestrator
    ↓
Orchestrator fully initialized ✅
    ↓
No post-construction mutation ✅
    ↓
Clean, predictable DI flow ✅
```

---

## Redundancy Proof

The hack is setting the state provider that was already set by the handler:

| Step | Who Does It | Component | Instance Set? |
|------|-------------|-----------|---------------|
| 1 | Handler | SandwichDeps | YES ✓ |
| 2 | Factory | Orchestrator field | YES ✓ |
| 3 | Hack | Orchestrator.SetStateProvider | YES ✓ (REDUNDANT) |

**Proof:** By step 2, the work is done. Step 3 is unnecessary.

---

## State Provider Lifecycle Comparison

### ❌ Old Way (with hack)

```
Config YAML:
    state_provider: my-provider

builder.BuildOrchestrator()
    ↓
stateProviderName extracted from opts
    ↓
Handler builds orchestrator with state provider ✓
    ↓
Orchestrator is fully initialized ✓
    ↓
builder checks if stateProviderName != "" (redundant check)
    ↓
builder looks up state provider AGAIN from its own map
    ↓
builder calls orchestrator.SetStateProvider() (mutation)
    ↓
Final state: stateProvider is set ✓ (but via mutation)
```

### ✅ New Way (no hack)

```
Config YAML:
    state_provider: my-provider

builder.BuildOrchestrator()
    ↓
Handler reads opts.StateProvider
    ↓
Handler calls builder.GetStateProvider(opts.StateProvider)
    ↓
Handler receives resolved state provider
    ↓
Handler puts it in SandwichDeps
    ↓
Handler calls Factory with deps
    ↓
Factory constructs with deps.StateProvider
    ↓
Orchestrator is fully initialized ✓
    ↓
Final state: stateProvider is set ✓ (via DI, no mutation)
```

---

## Consistency Across Orchestrators

Both Sandwich and Declarative already follow the correct pattern:

### Sandwich Handler (pipeline/sandwich/handler.go:63-82)
```go
var stateProvider core.StateProvider
if opts.StateProvider != "" {
    stateProvider, err = b.GetStateProvider(opts.StateProvider)
    // ...
}
deps := diapi.SandwichDeps{
    // ...
    StateProvider: stateProvider,
}
```
✅ Correct pattern

### Declarative Handler (pipeline/declarative/handler.go:38-54)
```go
var stateProvider core.StateProvider
if opts.StateProvider != "" {
    sp, err := builder.GetStateProvider(opts.StateProvider)
    // ...
    stateProvider = sp
}
deps := diapi.DeclarativeOrchestratorDeps{
    // ...
    StateProvider: stateProvider,
}
```
✅ Correct pattern

**Both follow the same "Handler resolves, Factory constructs" rule.**

---

## Why The Hack Exists (Historical Context)

The comment in builder.go explains it:
```go
// This is a bit of a hack, but it's the only way to get the state provider to the orchestrator for now.
// A better solution would be to have the orchestrator handler resolve its own dependencies.
```

**At the time it was written:**
- Handlers might not have had GetStateProvider method
- Handler pattern was less mature
- It was a temporary solution

**Now:**
- Handlers have GetStateProvider (line 13 of di.go)
- Handler pattern is established
- The "better solution" is already implemented
- The hack is obsolete

---

## Removal Workflow

### Step 1: Delete the Hack
```go
// DELETE lines 216-225 from builder.go

// BEFORE:
if stateProviderName != "" {
    sp, ok := b.stateProviders[stateProviderName]
    if !ok {
        return nil, nil, fmt.Errorf("state provider %q not found", stateProviderName)
    }
    if orchWithState, ok := orchestrator.(interface{ SetStateProvider(core.StateProvider) }); ok {
        orchWithState.SetStateProvider(sp)
    }
}

// AFTER:
// (empty - code removed)
```

### Step 2: Verify Tests Pass
```bash
go test ./pipeline/sandwich/... -v
go test ./pipeline/declarative/... -v
go test ./. -v
```

All tests should pass because:
1. Handler already resolved the state provider
2. Factory already set it in the constructor
3. The hack is just redundant mutation

### Step 3: Commit
```
chore(refactor): remove SetStateProvider hack from builder

The handler now resolves state provider and populates SandwichDeps
correctly. The post-construction mutation in the builder is now
redundant and violates immutability principles.

This completes ADR 7 compliance for orchestrator DI patterns.

Files changed:
- builder.go: remove lines 216-225
```

---

## Impact Analysis

### Code Changes
- 1 file changed: `builder.go`
- 10 lines deleted
- 0 lines added
- Risk level: **MINIMAL** (removing redundant code)

### Test Coverage
- All existing tests should pass (hack is redundant)
- May want to add explicit unit tests for handler resolution
- May want to add explicit unit tests for factory construction

### Documentation Impact
- Update CONTEXT.md to reflect new pattern
- Update LLD.md if it documents the old pattern
- Add note to HLD.md about ADR 7 compliance

### Backward Compatibility
- ✅ **No breaking changes** - no public API changes
- ✅ **No functionality changes** - handler already does the work
- ✅ **Behavior identical** - just cleaner DI flow

---

## Verification: Pattern Correctness

### ✅ Matches ADR 7
- Handler encapsulates build logic ✓
- Factory accepts typed deps ✓
- Factory does not accept builder ✓
- Builder is not leaked into factory ✓

### ✅ Matches ADR 4
- Generic, type-safe registry ✓
- Provider self-identification ✓
- Typed Resolved/Deps structs ✓
- No runtime type erasure ✓

### ✅ Matches ADR 5
- Stage-based orchestrators ✓
- Typed dependencies ✓
- Clear initialization flow ✓

---

## Summary

| Aspect | Old Way (Hack) | New Way (DI) | Winner |
|--------|---|---|---|
| Immutability | ❌ Violates | ✅ Maintains | DI ✓ |
| Type Safety | ❌ Duck typing | ✅ Typed | DI ✓ |
| Clarity | ❌ Hidden flow | ✅ Explicit | DI ✓ |
| Maintainability | ❌ Two paths | ✅ One path | DI ✓ |
| ADR Compliance | ❌ Violates | ✅ Compliant | DI ✓ |
| Redundancy | ❌ Redundant | ✅ Necessary | DI ✓ |
| Performance | ✅ Same | ✅ Same | Tie |
| Testing | ❌ Harder | ✅ Easier | DI ✓ |

**Result:** DI pattern wins on all important metrics.

---

**Document Created:** 2025-11-11  
**Ready to:** Delete the hack and commit the fix  
**Estimated Time:** 5 minutes
