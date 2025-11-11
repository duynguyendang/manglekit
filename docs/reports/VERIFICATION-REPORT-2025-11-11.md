# Architectural Pattern Verification Report
**Date:** 2025-11-11  
**Scope:** Code Smell Fix Validation Against ADR & Actual Implementation  
**Status:** ⚠️ CRITICAL FINDINGS

---

## Executive Summary

✅ **Good News:** The proposed architectural fix in the documents follows Manglekit's patterns correctly.  
⚠️ **Critical Finding:** The actual code has NOT been updated yet. The `SetStateProvider` hack (lines 216-225 in `builder.go`) is still present and functional.  
✅ **Verification:** The handler and factory implementations in `sandwich/handler.go` and `sandwich/factory.go` are **already correctly implemented** and ready to replace the builder hack.

---

## Architectural Pattern Compliance

### ADR 7: Per-Kind Handlers and Typed DI Enforcement

**Key Decision from ADR 7:**

> Adopt a strict separation of responsibilities:
> * Per-kind `core.ComponentHandler` encapsulates build logic for that kind
> * Provider factories **MUST** accept typed deps (`diapi.*Deps`) and **MUST NOT** accept the builder
> * Every orchestrator must have a matching handler to be buildable via the builder

**Status:** ✅ **FULLY COMPLIANT**

The proposed fix follows ADR 7 exactly:

1. ✅ Handler encapsulates build logic (resolves dependencies)
2. ✅ Factory accepts typed deps (`diapi.SandwichDeps`)
3. ✅ No builder leaking into factory
4. ✅ Orchestrator has a matching handler

---

## Code Review: Implementation Status

### ✅ Part 1: Handler (Already Correct!)

**File:** `pipeline/sandwich/handler.go` (Lines 1-103)

The handler is **already implemented with the correct pattern:**

```go
// Lines 63-73 (Handler RESOLVES)
var stateProvider core.StateProvider
if opts.StateProvider != "" {
    stateProvider, err = b.GetStateProvider(opts.StateProvider)
    if err != nil {
        return nil, fmt.Errorf("sandwich orchestrator: failed to get state provider %q: %w", opts.StateProvider, err)
    }
}

// Lines 75-82 (Handler POPULATES deps)
deps := diapi.SandwichDeps{
    CoreDeps:      b.GetCoreDeps(),
    Retriever:     retriever,
    LLM:           llm,
    Reranker:      reranker,
    RuleSet:       ruleSet,
    StateProvider: stateProvider,  // ← Resolved instance
}

// Lines 84-87 (Handler calls Factory)
f, ok := factory.(core.Factory)
built, err := f.Build(ctx, deps, cfg)
```

**Verification:** ✅ **CORRECT**
- Handler gets builder via DI
- Handler resolves state provider using `builder.GetStateProvider()`
- Handler populates `SandwichDeps.StateProvider` with the resolved instance
- No factory involvement in resolution

---

### ✅ Part 2: Factory (Already Correct!)

**File:** `pipeline/sandwich/factory.go` (Lines 1-52)

The factory is **already implemented with the correct pattern:**

```go
// Lines 20-25 (Factory just asserts deps)
func (f *Factory) Build(ctx context.Context, deps any, cfg any) (any, error) {
    sandwichDeps, ok := deps.(diapi.SandwichDeps)
    if !ok {
        return nil, fmt.Errorf("invalid deps type for sandwich orchestrator: got %T", deps)
    }

// Lines 30-42 (Factory just constructs)
s := &Orchestrator{
    retriever:           sandwichDeps.Retriever,
    reranker:            sandwichDeps.Reranker,
    ruleset:             sandwichDeps.RuleSet,
    llm:                 sandwichDeps.LLM,
    stateProvider:       sandwichDeps.StateProvider,  // ← No resolution, just assign
    conversationManager: statehelper.NewConversationManager(),
    obs:                 sandwichDeps.Obs,
    topK:                opts.TopK,
    maxTokens:           opts.MaxTokens,
    fallbackThreshold:   opts.FallbackThreshold,
}
```

**Verification:** ✅ **CORRECT**
- Factory receives fully-populated `SandwichDeps`
- Factory does NOT call `builder.GetStateProvider()` (correct!)
- Factory just assigns the already-resolved instance to the orchestrator
- Factory has NO resolution logic (correct!)

---

### ✅ Part 3: DI Interfaces (Already Correct!)

**File:** `core/diapi/di.go` (Lines 1-151)

**Verification:**
- ✅ Line 13: `Builder` interface includes `GetStateProvider(name string) (core.StateProvider, error)`
- ✅ Lines 121-128: `SandwichDeps` struct has instance field `StateProvider core.StateProvider` (NOT a function)
- ✅ Lines 130-135: `DeclarativeOrchestratorDeps` struct has instance field `StateProvider core.StateProvider` (consistent pattern)

---

### ⚠️ Part 4: Builder Hack (STILL PRESENT!)

**File:** `builder.go` (Lines 216-225) ⚠️ **NEEDS REMOVAL**

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

**Current Status:** ❌ **STILL IN CODE**

**Why This Must Be Removed:**
1. Post-construction mutation violates immutability
2. Runtime duck typing (`interface{}`) violates type safety
3. Builder is doing orchestrator-specific logic (violates ADR 7)
4. The handler already resolved and set the state provider (this code is redundant!)

---

### ✅ Part 5: Declarative Handler (Already Correct!)

**File:** `pipeline/declarative/handler.go` (Lines 1-67)

The Declarative handler is **also correctly implemented:**

```go
// Lines 38-44 (Handler resolves state provider)
var stateProvider core.StateProvider
if opts.StateProvider != "" {
    sp, err := builder.GetStateProvider(opts.StateProvider)
    if err != nil {
        return nil, fmt.Errorf("declarative orchestrator: failed to get state provider %q: %w", opts.StateProvider, err)
    }
    stateProvider = sp
}

// Lines 50-54 (Handler populates deps)
deps := diapi.DeclarativeOrchestratorDeps{
    CoreDeps:      builder.GetCoreDeps(),
    StateProvider: stateProvider,
    Tools:         tools,
}
```

**Verification:** ✅ **CORRECT**
- Follows same pattern as Sandwich handler
- Consistent DI approach across both orchestrators

---

## ADR Compliance Summary

### ADR 4: Generic, Type-Safe Registry & Builder

| Requirement | Status | Notes |
|-------------|--------|-------|
| One generic registry | ✅ | Uses typed factories |
| Provider self-identification via Options | ✅ | Implemented in registry |
| Type-safe Resolved struct | ✅ | Contains all resolved components |
| Typed orchestrator inputs | ✅ | SandwichDeps and DeclarativeOrchestratorDeps are typed |

---

### ADR 5: Orchestrator Modernization

| Requirement | Status | Notes |
|-------------|--------|-------|
| Stage-based pipeline | ✅ | Sandwich has retrieve, rerank, rules, LLM stages |
| Typed Resolved dependencies | ✅ | Handlers populate typed Deps structs |
| Orchestrator factories accept Resolved | ⚠️ | Factories accept typed Deps (even better!), but builder hack is still present |

---

### ADR 7: Per-Kind Handlers and Typed DI Enforcement

| Requirement | Status | Notes |
|-------------|--------|-------|
| Per-kind ComponentHandler | ✅ | Sandwich and Declarative handlers exist |
| Handlers encapsulate build logic | ✅ | Handlers resolve all dependencies |
| Factories accept typed deps | ✅ | Both factories accept typed Deps structs |
| Factories MUST NOT accept builder | ✅ | Neither factory accesses builder |
| No builder leaking into providers | ⚠️ | Except for the builder hack in builder.go (lines 216-225) |

---

## Critical Issue: The Builder Hack is Redundant

### Evidence

The hack in `builder.go:216-225` is **now completely redundant** because:

1. **Handler already resolved it:** The handler in `sandwich/handler.go:63-73` already called `b.GetStateProvider()` and got the state provider
2. **Handler already set it:** The handler in `sandwich/handler.go:75-82` already put it in `SandwichDeps.StateProvider`
3. **Factory already received it:** The factory in `sandwich/factory.go:30-42` already received the populated deps and assigned it
4. **Orchestrator is fully initialized:** By the time the builder hack runs, the orchestrator already has the state provider set correctly

### Current Flow (with hack still in place)

```
1. Handler resolves state provider:
   stateProvider, err := b.GetStateProvider(opts.StateProvider)
   
2. Handler populates deps:
   deps.StateProvider = stateProvider
   
3. Factory receives deps:
   sandwichDeps.StateProvider = stateProvider
   
4. Factory constructs with state provider already set:
   return &Orchestrator{stateProvider: sandwichDeps.StateProvider}
   
5. Builder hack tries to SET AGAIN (REDUNDANT):
   if orchWithState, ok := orchestrator.(interface{ SetStateProvider(...) }); ok {
       orchWithState.SetStateProvider(sp)  // Already set in step 4!
   }
```

---

## Documentation Accuracy Check

### action-items-tracking-2025-11-11.md

**Checked Sections:**
- ✅ Task 1.1: Add `GetStateProvider` to Builder - Already exists in code
- ✅ Task 1.2: Add `StateProvider` instance field to SandwichDeps - Already exists (line 125 in `di.go`)
- ✅ Task 1.3: Handler resolves - Already correctly implemented (lines 63-73 in `sandwich/handler.go`)
- ✅ Task 1.4: Factory constructs - Already correctly implemented (lines 30-42 in `sandwich/factory.go`)
- ⚠️ Task 1.5: Remove builder hack - **NOT YET DONE** (still at lines 216-225 in `builder.go`)

**Status:** Documentation describes current state correctly. Tasks 1.1-1.4 are already done in code. Only Task 1.5 (removing the hack) remains.

---

### code-smell-deep-dive-2025-11-11.md

**Checked Sections:**
- ✅ Step 1: Add GetStateProvider to Builder - Already done
- ✅ Step 2: Add StateProvider instance field - Already done
- ✅ Step 3: Handler RESOLVES with code example - Code matches actual implementation
- ✅ Step 4: Factory CONSTRUCTS with code example - Code matches actual implementation
- ⚠️ Step 5: Remove builder hack - Still needed

**Status:** Documentation is accurate. Implementation is 80% complete (only hack removal remains).

---

## Handler vs. Factory Pattern Verification

### Pattern Check: Sandwich Orchestrator

```
❌ WRONG PATTERN (what the hack is doing):
   Builder → Resolve dependency → Call SetStateProvider()
   (Post-construction mutation)

✅ CORRECT PATTERN (what handlers + factories do):
   Builder → Handler resolves → Populates Deps → Factory constructs
   (Immutable after construction)
```

**Current Implementation:** ✅ **CORRECT** (Handlers + factories)  
**Builder Hack:** ❌ **ANTIPATTERN** (Post-construction mutation)  
**Status:** Both patterns are in the codebase simultaneously. The hack is now unnecessary.

---

## Recommendation: Remove the Hack

### What to Do

In `builder.go`, lines 216-225, remove this entire block:

```go
// DELETE THIS:
if stateProviderName != "" {
    sp, ok := b.stateProviders[stateProviderName]
    if !ok {
        return nil, nil, fmt.Errorf("state provider %q not found", stateProviderName)
    }
    if orchWithState, ok := orchestrator.(interface{ SetStateProvider(core.StateProvider) }); ok {
        orchWithState.SetStateProvider(sp)
    }
}
```

### Why It's Safe

1. **Handler already handled it:** `sandwich/handler.go` resolved the state provider
2. **Factory already used it:** `sandwich/factory.go` constructed the orchestrator with it
3. **Orchestrator is correct:** The state provider is already set via the normal DI flow
4. **No SetStateProvider needed:** The orchestrator receives the state provider in the constructor

### Verification After Removal

Run the sandwich orchestrator tests:
```bash
go test ./pipeline/sandwich/... -v
```

Expected result: All tests pass, state provider is correctly initialized.

---

## Summary Table

| Component | Expected | Actual | Status |
|-----------|----------|--------|--------|
| Handler resolves state provider | ✅ Yes | ✅ Yes (lines 63-73) | ✅ **DONE** |
| Handler populates SandwichDeps | ✅ Yes | ✅ Yes (lines 75-82) | ✅ **DONE** |
| Factory accepts typed deps | ✅ Yes | ✅ Yes (SandwichDeps) | ✅ **DONE** |
| Factory has no resolution logic | ✅ Yes | ✅ Yes (just constructs) | ✅ **DONE** |
| Builder hack removed | ✅ Yes | ❌ No (still present) | ⚠️ **PENDING** |
| ADR 7 compliance | ✅ Yes | ✅ Yes (handlers + factories) | ✅ **DONE** |
| ADR 4 compliance | ✅ Yes | ✅ Yes (typed deps) | ✅ **DONE** |

---

## Conclusion

✅ **Architecture Pattern:** CORRECT and following all ADRs  
✅ **Handler Implementation:** CORRECT and complete  
✅ **Factory Implementation:** CORRECT and complete  
⚠️ **Builder Hack:** Still present and now REDUNDANT  

**Action:** Remove the SetStateProvider hack from `builder.go:216-225` to complete the fix.

**Effort to Complete:** 5 minutes (delete 10 lines of code)

---

## Cross-Reference

- **ADR 7 Reference:** `docs/ADR.md` lines 309-347
- **Handler Implementation:** `pipeline/sandwich/handler.go:63-82`
- **Factory Implementation:** `pipeline/sandwich/factory.go:30-42`
- **DI Interfaces:** `core/diapi/di.go:121-135`
- **Builder Hack:** `builder.go:216-225` ← **TO BE REMOVED**

---

**Report Generated:** 2025-11-11  
**Verified Against:** ADR.md, sandwich handler, sandwich factory, diapi interfaces, builder.go  
**Next Action:** Remove builder hack to complete ADR 7 compliance
