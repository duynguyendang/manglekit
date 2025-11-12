# AGENTS.md — Manglekit Coding Agent Configuration (2025.11)

*Last updated: 2025-11-12*

---

## Overview

This document defines how coding agents interact with the **Manglekit** codebase.  
It specifies conventions, automated tasks, and safety rules to ensure that all AI-driven modifications stay consistent with the system design and context documentation.

Agents use this file as an **operational manual** to perform reasoning, refactoring, documentation updates, and observability instrumentation within the Go SDK.

Agents must treat the documentation set (`CONTEXT.md`, `HLD.md`, `LLD.md`, `ADR.md`) as the synchronized **architecture source of truth**.  
`CONTEXT.md` is the **live baseline** — always validate and update it whenever architecture or dependency injection logic changes.

---

## 1. Agent Goals

Coding agents working in this repository should be capable of:

* Understanding the full architecture from `docs/HLD.md`.
* Updating context and documentation automatically when code changes in `docs/CONTEXT.md`.
* Maintaining observability and performance best practices.
* Identifying gaps and inconsistencies in provider registration, orchestrator wiring, or options mapping.

---

## 2. File Structure Awareness

Agents must recognize the following key locations and their purposes (aligned with repo layout):

* `core/`: Contracts, types, diapi, handler/factory interfaces, observability.
* Root: `builder.go`, `registry.go` (builder + registry), `go.mod`.
* `sdk/`: Config→builder bridge (`sdk.FromConfig`).
* `config/`: YAML parsing, normalization, validation.
* `pipeline/`: Orchestrators and stages (Sandwich, Declarative).
* `internal/providers/`: Families of retrievers, rerankers, LLMs, rules, state, schema parsers.
* `internal/embedders/`, `internal/vectorstores/`: Embedder and vector store providers and handlers.
* `providers/all/`: Convenience registrar for standard providers.
* `docs/`: Technical docs and architecture rules (`docs/rules/manglekit-arch.yml`).

---

## 3. Context-Aware Reasoning

Agents must treat `docs/CONTEXT.md` as the **live architectural memory** of the repository.  
It is both the *source of truth* for reasoning and the *target* that must stay synchronized with source changes.

### 3.1 Context Loading and Validation

Before performing any modification, the agent must:

1. **Load the full content** of `docs/CONTEXT.md`, including YAML front-matter, diagrams, tables, and changelog.  
2. **Verify freshness:** compare the `last_updated:` field to the most recent code commit.  
   - If the context is **older than 3 days**, treat it as *potentially stale* and rely on the code diff as the authoritative source.  
   - If up-to-date, use it directly as the architectural baseline.

### 3.2 How to Use the Context for Reasoning

Use the document’s internal structure as a guide:

| Section | Purpose |
|----------|----------|
| **Implementation Snapshot** | Understand module relationships (builder, registry, handlers, providers, orchestrators). |
| **Dependency Rules** | Validate layering — ensure no illegal imports or coupling. |
| **Core Contracts** | Retrieve interface and type definitions to generate compliant code. |
| **Provider Composition** | Infer where new providers or handlers should register. |
| **Known Gaps** | Identify open architectural issues and avoid reintroducing resolved gaps. |
| **Machine Appendix (JSON)** | Track gap statuses and last synchronization date. |
| **Changelog** | Understand recent context or architectural updates. |

The agent should **avoid scanning the entire repository** unless the context is stale or incomplete.
Instead, reason from the structured data within `CONTEXT.md` and cross-references to other docs (HLD, LLD, ADR).

### 3.3 Reflecting Code Changes

When code changes affect architecture, interfaces, or runtime behavior, the agent must **edit `docs/CONTEXT.md` directly** —  
updating sections, known gaps, and timestamps as described in section **6 (Self-Managed Context Synchronization)**.  
This guarantees that the documentation always mirrors the true, current state of the system.

---

## 4. Editing Policies

* Never remove sections or metadata blocks from `CONTEXT.md`.
* Maintain YAML front-matter integrity.
* Keep the Machine Appendix JSON snapshot and Known Gaps statuses in sync.
* Use status icons (✅ / ⚠️ / ❌) consistently.
* Preserve Markdown headings, tables, and code formatting.
* Apply minimal, targeted updates with precise summaries.

---

## 5. Commit Conventions

Agents must follow semantic commit conventions:

```
feat: introduce new feature or provider
fix: resolve bug or panic
refactor: improve structure without changing behavior
docs: update documentation (includes CONTEXT.md)
chore(context): auto-sync CONTEXT.md after code change
test: add or fix unit tests
```

---

## 6. Self-Managed Context Synchronization (No External Commands)

> Goal: Whenever source changes affect architecture or runtime behavior, the **coding agent must directly update** `docs/CONTEXT.md` (and related docs) itself — by editing the file — instead of calling any CLI tools or Make targets.

### 6.1 When to Trigger an Update

Trigger the **auto-sync process** whenever any of the following occur (including test logic changes):

- Any Go file changes under `builder.go`, `registry.go`, `core/**`, `pipeline/**`, or `internal/providers/**`
- Addition or modification of a **provider** (retriever, reranker, LLM, vector store, rules, state, tool, etc.)
- Changes to **Options** or **Contracts** in `core/**` or `internal/providers/**`
- Addition of new **configuration keys** or **environment variables**
- Modifications to an **orchestrator** (Sandwich / Declarative) or build-order spec
- Changes in **observability**, **lifecycle**, or **Known Gaps**

### 6.2 Self-Update Algorithm

1. **Collect diff context**
   - Obtain the list of modified files from the current patch or PR.
2. **Infer impact scope**
   - Identify affected kinds and changed contracts or wiring.
3. **Edit `docs/CONTEXT.md` directly**
   - Preserve formatting and metadata.
   - Bump `last_updated`.
   - Update sections: Snapshot, Dependency Rules, Core Contracts, Configuration Flow, Known Gaps, JSON appendix, and Changelog.
4. **Edit `docs/LLD.md` and `docs/HLD.md` if needed**
   - Update diagrams or sequences if DI, handlers, or factories changed.
   - Add changelog entries.
5. **Safety**
   - Never hallucinate or extrapolate beyond diff evidence.
6. **Commit format**
   - Docs-only update:  
     `chore(context): auto-sync CONTEXT.md (+LLD/HLD) after code changes`
   - Combined with code change:  
     `feat|fix: <summary>` + `chore(context): auto-sync …`

### 6.3 Example Commit

```
feat(retriever/hybrid): accept diapi.RetrieverDeps and expose rrf_k

chore(context): auto-sync CONTEXT.md (Resolved GAP-006; update JSON snapshot; changelog)
chore(lld): refresh sequence for retriever handler deps
```

---

## 7. Hooks & Optional Automation

> No CLI commands are required. The agent must self-update documentation as described in §6.  
> This section only defines optional automation policies for CI enforcement.

- **Pre-commit (recommended):**  
  If a patch modifies files in §6.1 but does not include doc updates (`docs/CONTEXT.md`, `LLD.md`, `HLD.md`), the agent should **automatically append** a commit:
```
chore(context): auto-sync CONTEXT.md (+LLD/HLD)
```

- **CI lint (recommended):**
- Ensure `docs/CONTEXT.md`'s `last_updated` is within ±3 days of the latest commit touching core/builder/providers/pipeline.
- Validate that the Machine Appendix JSON block parses successfully.

---

## 8. Observability and Logging Rules

* Never print directly to stdout in production paths.  
* Use the shared `core.Observability` struct.  
* Always use `core.Logger`, `core.Tracer`, and `core.Meter` interfaces.  
* Ensure each pipeline stage emits metrics via `core.Observability.Meter`.  
* Record latencies and token usage in metrics when calling LLMs or retrievers.

---

## 9. Testing Enforcement

Agents must maintain or extend test coverage when altering code in these areas:

* Pipeline orchestrator and stages (`pipeline/sandwich_test.go`, `pipeline/declarative/*_test.go`)
* Provider families under `internal/providers/*`
* Core builder/registry logic

Use naming pattern `Test<Component>_<Behavior>`.

---

## 10. Summary

This file defines how coding agents maintain, reason about, and update Manglekit.  
The automation described here ensures that `docs/CONTEXT.md` always mirrors the true state of the codebase.

> *“An agent is only as smart as its context — keep it fresh, structured, and faithful.”*

---

## 11. Architecture Rules (Enforced)

- Layered dependencies:  
`core` must not import providers, pipeline, or root.  
`pipeline` must not import concrete providers.  
Providers import only `core`.  
- Provider registration:  
All providers register via typed `manglekit.Register[T, D, O]`.  
- Handlers per kind:  
Each kind must have a dedicated `core.ComponentHandler`.  
- Configuration binding via `sdk.FromConfig`.  
- Observability & lifecycle unified through `core.Observability`.

---

## 12. Implementation Checklists

### New provider (retriever, reranker, embedder, LLM, vector store, rules, state)

- Define `Options` implementing `core.ProviderOptions`.  
- Declare dependencies via `diapi` dep marker interfaces.  
- Register via `manglekit.Register[T, D, O]`.  
- **(CRITICAL) Modify `providers/all/all.go`:** Add a blank import (e.g., `_ "path/to/your/provider/package"`) to this file. This ensures your provider's `init()` function is executed by the Go runtime, adding it to the central registry.
- Ensure a handler exists for that kind.  
- Add deterministic tests under the provider folder.  
- **Docs:** Ensure `docs/CONTEXT.md` (and LLD/HLD) are auto-synced after implementation.

### New orchestrator

- Create `Options` implementing `core.ProviderOptions` (kind = `core.KindOrchestrator`).  
- Register a typed factory and handler; avoid implicit selection.  
- Add tests for explicit component selection.  
- **Docs:** Ensure context documents are updated and known gaps refreshed.

### Core or DI changes

- Keep layering rules intact.  
- Update handlers/factories for signature consistency.  
- **Docs:** Bump `last_updated` across `CONTEXT.md`, `LLD.md`, and `HLD.md`.

---

## 13. Post-Change Actions

- Run all tests and static checks.  
- Verify architecture rules in `docs/rules/manglekit-arch.yml` pass.  
- Confirm context auto-sync commit exists.

---

## 14. Agent Safety Summary

* Modify only files with verifiable evidence from diffs.  
* Never hallucinate architecture changes.  
* Preserve YAML headers and Markdown formatting in all docs.  
* Always bump `last_updated` and maintain JSON appendix validity.  
* Never remove sections or metadata blocks from context documents.  
* Keep generated commits atomic and syntactically valid.  

---

## 15. Architectural Patterns & Known Issues (Reference)

This section documents key architectural patterns that have been established through resolution of code smells and design improvements. Agents should be aware of these patterns when implementing new features or providers.

**Overall Status:** ✅ **STABLE (9/10)**

### 15.1 Resolved Patterns (Reference for Future Implementation)

#### Pattern: Type-Safe Dependency Injection via diapi Structs

**Context:** Early versions used generic `diapi.Builder` in factories, causing runtime type assertions and brittleness.

**Current Implementation:**
- Handlers resolve typed `diapi.*Deps` structs (e.g., `RetrieverDeps`, `SandwichDeps`)
- Factories receive fully-typed dependencies at construction time
- No post-construction mutation of dependencies
- All factory signatures follow pattern: `func(ctx context.Context, deps diapi.XyzDeps, opts any) (core.Xyz, error)`

**Example (Sandwich Orchestrator):**
```go
// Handler resolves StateProvider from options
var stateProvider core.StateProvider
if opts.StateProvider != "" {
    sp, err := b.GetStateProvider(opts.StateProvider)
    if err != nil {
        return nil, fmt.Errorf("failed to get state provider: %w", err)
    }
    stateProvider = sp
}

// Typed deps struct populated by handler
deps := diapi.SandwichDeps{
    CoreDeps:      b.GetCoreDeps(),
    Retriever:     retriever,
    StateProvider: stateProvider,  // ← Instance set by handler
}

// Factory receives fully-populated deps
orchestrator, closer, err := factory.Build(ctx, deps, cfg)
```

**When Implementing:** Always use this pattern for new handlers/factories. Never use post-construction mutation or runtime type assertions.

#### Pattern: Explicit Component Selection (No Map Iteration)

**Context:** Early versions selected singleton components (StateProvider, RuleSet) by iterating over maps, causing non-deterministic behavior.

**Current Implementation:**
- All Options structs include explicit string fields for singleton dependencies (e.g., `stateProvider string`, `ruleSet string`)
- Handlers resolve these by name: `builder.GetStateProvider(opts.StateProvider)`
- No map iteration for component selection

**Example (Declarative Orchestrator Options):**
```go
type Options struct {
    Orchestrator    string  // Required
    StateProvider   string  // Explicit, not first-from-map
    Retriever       string
    LLM             string
    Reranker        string  // Optional
    // ... other config
}
```

**When Implementing:** Always add explicit string fields to Options for any dependencies that could have multiple implementations. Avoid implicit "first available" selection.

#### Pattern: Deterministic Map Iteration (Where Map Iteration Is Necessary)

**Context:** Some internal operations (debug output, state dumps) iterate over maps, requiring deterministic ordering.

**Current Implementation:**
- Extract keys/values into a slice
- Sort slice by appropriate criterion
- Iterate over sorted slice

**Example (Rules Engine Debug Output):**
```go
// Sort predicates for deterministic output
predicates := make([]ast.PredicateSym, 0, len(edbDecls))
for p := range edbDecls {
    predicates = append(predicates, p)
}
sort.Slice(predicates, func(i, j int) bool {
    if predicates[i].Symbol != predicates[j].Symbol {
        return predicates[i].Symbol < predicates[j].Symbol
    }
    return predicates[i].Arity < predicates[j].Arity
})
for _, p := range predicates {
    // Process in deterministic order
}
```

**When Implementing:** If you iterate over maps, always use this pattern to ensure deterministic output for debugging and testing.

#### Pattern: Deterministic Tie-Breaking in Reranking

**Context:** Hybrid retriever could return documents in non-deterministic order when scores tied.

**Current Implementation:**
- Two-pass sorting: ID-based sort first, then stable sort by score
- Preserves determinism while maintaining score priority

**Example (Hybrid Retriever):**
```go
sort.Slice(finalDocs, func(i, j int) bool {
    return finalDocs[i].ID < finalDocs[j].ID  // 1. Sort by ID first
})

sort.SliceStable(finalDocs, func(i, j int) bool {
    scoreI := scores[finalDocs[i].ID]
    scoreJ := scores[finalDocs[j].ID]
    return scoreI > scoreJ  // 2. Stable sort by score (preserves ID order for ties)
})
```

**When Implementing:** Use this two-pass pattern for any reranking/scoring logic to ensure deterministic results.

#### Pattern: Extensible Handler Dispatch via DependencyResolver

**Context:** Handler type-switches were brittle and required modification when adding new provider subtypes.

**Current Implementation:**
- `core/diapi/resolvers.go` defines `DependencyResolver` interface
- Handlers delegate dependency resolution to registered resolvers
- Resolvers match provider options by type and resolve dependencies
- Extensible without modifying handler code

**Example (Retriever Handler):**
```go
// Clean delegation, no switch statements
deps, err := h.resolver.Resolve(ctx, core.KindRetriever, builderDI, opts)

// To add new retriever type, simply register resolver:
type BranchingResolver struct{}
func (r *BranchingResolver) Matches(opts any) bool {
    _, ok := opts.(diapi.BranchingRetrieverDep)
    return ok
}
func (r *BranchingResolver) Resolve(ctx, builderDI, cfg any) (any, error) {
    // Resolution logic
}
registry.Register(core.KindRetriever, &BranchingResolver{})
```

**When Implementing:** Use this pattern when adding new provider subtypes. Implement a `DependencyResolver`, not a type-switch in the handler.

### 15.2 Eliminated Patterns (Anti-Patterns to Avoid)

#### Anti-Pattern: Post-Construction State Mutation

**Status:** ✅ ELIMINATED

**What It Was:**
```go
// WRONG - Do NOT do this
orchestrator := factory.Build(...)
if stateProvider != nil {
    // Duck-type: check if orchestrator has SetStateProvider method
    if orchWithState, ok := orchestrator.(interface{ SetStateProvider(core.StateProvider) }); ok {
        orchWithState.SetStateProvider(sp)
    }
}
```

**Why It's Bad:**
- Runtime type assertions to unnamed interfaces
- Component state changes after construction (violates immutability)
- Hard to test and debug
- Hides dependencies from static analysis

**Correct Pattern:**
- Resolve dependencies in handler
- Pass fully-populated typed `diapi.*Deps` to factory
- Factory receives all dependencies during construction
- Component is immutable post-construction

#### Anti-Pattern: Generic Builder Leaking into Core Logic

**Status:** ✅ RESOLVED (verified safe by code review)

**What Agents Should Avoid:**
- Using generic `any` type for dependencies
- Passing `diapi.Builder` directly to factories
- Runtime type assertions in critical paths

**Correct Pattern:**
- Define specific `diapi.*Deps` structs for each component
- Handlers resolve all dependencies and populate deps struct
- Factories receive typed deps struct
- No generic types in factory signatures

#### Anti-Pattern: Implicit Component Selection

**Status:** ✅ RESOLVED

**What It Was:**
```go
// WRONG - implicit map iteration
var sp core.StateProvider
for _, provider := range stateProviders {  // Non-deterministic!
    sp = provider
    break
}
```

**Why It's Bad:**
- Non-deterministic behavior across runs
- Makes configurations ambiguous
- Hard to debug when behavior differs between runs

**Correct Pattern:**
- Explicit string field in Options: `StateProvider string`
- Named lookup: `builder.GetStateProvider(opts.StateProvider)`
- Clear dependency specification in configuration

### 15.3 Current Known Limitations & Acceptable Trade-offs

#### 1. Configurable Resource Cleanup Timeout

**Location:** `builder.go:222`  
**Current:** Hard-coded 5-second timeout for cleanup  
**Impact:** Low (current timeout is reasonable for most use cases)  
**Acceptable Trade-off:** Yes — can be enhanced in future if needed  

```go
closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)  // Hard-coded
defer cancel()
```

**Enhancement Path:** Make timeout configurable via `BuilderOptions`.

#### 2. Cleanup Failure Logging

**Location:** `builder.go:229-239`  
**Current:** Errors aggregated via `errors.Join()` but not individually logged  
**Impact:** Low (errors still reported at end, but intermediate failures not visible)  
**Acceptable Trade-off:** Yes — logs could be added for better debugging  

```go
for i := len(b.opts.ResourceClosers) - 1; i >= 0; i-- {
    if err := b.opts.ResourceClosers[i](closeCtx); err != nil {
        combined = errors.Join(combined, err)  // Aggregated, not logged
    }
}
```

**Enhancement Path:** Add debug logging per closer for troubleshooting.

### 15.4 Architecture Rules Quick Reference

**Dependency Layering:**
```
┌─────────────────────────────────────────────────┐
│ User Code & Tests                               │
├─────────────────────────────────────────────────┤
│ sdk.FromConfig() or sdk.NewBuilder()            │
│ (Public Entrypoints)                            │
├─────────────────────────────────────────────────┤
│ Builder & Registry (Wiring Logic)               │
├─────────────────────────────────────────────────┤
│ Handlers (Build Instructions)                   │
├─────────────────────────────────────────────────┤
│ Providers & Orchestrators                       │
│ (Concrete Implementations)                      │
├─────────────────────────────────────────────────┤
│ core/ (Types, Interfaces, DI Contracts)         │
├─────────────────────────────────────────────────┤
│ External Go Modules & Standard Library          │
└─────────────────────────────────────────────────┘

Rules:
✅ Providers import only core
✅ Handlers import core, diapi, and their providers
✅ Builder imports handlers, registry, providers
✅ core must NOT import providers, handlers, or builder
```

**Component Registration Checklist:**
1. ✅ Define typed `Options` struct implementing `core.ProviderOptions`
2. ✅ Register via `manglekit.Register[Concrete, DiDeps, Options]`
3. ✅ Implement `core.ComponentHandler` for the kind
4. ✅ Register handler in `internal/providers/{kind}/handler.go`
5. ✅ Add blank import to `providers/all/all.go`
6. ✅ Create integration tests (Config-First strategy for internal deps)
7. ✅ Update `docs/CONTEXT.md`, `LLD.md`, `HLD.md`

---

## 16. Provider Test Architecture (Enforced)

This section defines the mandatory patterns for testing providers. You must identify if the provider's dependencies are **Internal (Manglekit)** or **External (Go Modules)** and apply the correct strategy.

---

### 16.1 Strategy 1: Internal DI / Config-First Test

* **Use Case:** Testing a provider's integration with the Manglekit framework (e.g., `dense`, `hybrid`).
* **Dependency Type:** Internal Manglekit providers (e.g., `VectorStore`, `Embedder`, `Retriever`).
* **Test Method:** Use `sdk.LoadWithRegistry` and a test YAML.
* **Error Solved:** Fixes the "could not find options type" error.

**The 3-Part Registration Rule (Mandatory):**
To fix the "could not find options type" parsing error, your test setup (e.g., `registerTestDeps`) **must** register **all three** components for *every provider* defined in your test YAML (the provider-under-test *and* its mock dependencies):

1. **The `ComponentHandler`:** (e.g., `retrievers.NewHandler()`). Tells the Builder *how* to build the `Kind`.
2. **The `Factory`:** (e.g., `dense.NewFactory()`, `mock_vs.NewFactory()`). Tells the Handler *what* to build for the `provider:` string.
3. **The `Options` Sample:** (e.g., `dense.Options{}`, `mock_vs.Options{}`). Tells the Config Loader *what Go type* to parse the YAML into.

---

### 16.2 Strategy 2: External Dependency / Unit Test

* **Use Case:** Testing a provider's core business logic in isolation (e.g., `llm`, `google-embedder`).
* **Dependency Type:** External Go modules or I/O (e.g., `genkit`, `http.Client`, `os.Getenv`).
* **Test Method:** Standard Go unit test. **DO NOT** use `sdk.LoadWithRegistry` or YAML.

**Rules for External Dependency Tests:**

1. **Do not use `sdk.LoadWithRegistry`.** This is not a DI integration test.
2. **Instantiate the provider directly:** Call the provider's Go constructor in your test (e.g., `provider, err := google.NewProvider(opts)` — this is pseudocode; use the actual constructor name for your provider).
3. **Mock the External I/O:** Do not mock the Manglekit `Registry`. Instead, mock the *external* dependency. For example, if the provider uses an `http.Client`, use `httptest.NewServer` and pass the mock server's URL into the provider's `Options`.
4. Call the provider's methods (e.g., `provider.Execute(...)`) and assert the results.

---

## 17. Documentation Cross-References

For deeper understanding of Manglekit architecture and patterns, refer to:

| Document | Purpose | When to Use |
|-----------|---------|------------|
| `docs/CONTEXT.md` | Live architecture snapshot, state, and known gaps | Reasoning about current system state before modifying code |
| `docs/HLD.md` | High-level design, layering, and system interaction flows | Understanding overall system design and user journeys |
| `docs/LLD.md` | Low-level design, handler dispatch, type resolution, lifecycle | Implementing new providers/handlers or debugging DI issues |
| `docs/ADR.md` | Architecture decisions and rationale | Understanding *why* patterns exist and trade-offs |
| `docs/LOGGING.md` | Observability and logging conventions | Implementing logging and metrics in new code |
| `docs/CSD.md` | Conceptual System Design | Understanding user-facing abstractions |

**Pattern:** When implementing or reviewing code, consult these docs in order:
1. **CONTEXT.md** — Understand current state
2. **ADR.md** — Understand decisions behind current state
3. **LLD.md** — Understand implementation details
4. **LOGGING.md** — Implement observability correctly

---

## 18. Agent Diagnostic Checklist

Before committing any code changes, verify:

### Code Quality Checks
- [ ] All new functions/types have clear documentation
- [ ] Type assertions are minimal; use typed structs instead
- [ ] No post-construction mutation of dependencies
- [ ] Map iterations sorted deterministically (or use typed slices)
- [ ] Resource closers properly registered if component manages I/O

### Architecture Compliance Checks
- [ ] No illegal cross-layer imports (verify layering rules)
- [ ] Handlers use typed `diapi.*Deps` structs
- [ ] Factories receive fully-populated deps at construction time
- [ ] Explicit component selection (no implicit map iteration for singletons)
- [ ] Provider registered in `providers/all/all.go` if applicable

### Testing Checks
- [ ] Tests follow appropriate strategy (Config-First or Unit Test)
- [ ] Test names follow `Test<Component>_<Behavior>` pattern
- [ ] Internal provider tests register all 3 parts (Handler, Factory, Options)
- [ ] External provider tests mock external I/O, not Registry
- [ ] Error paths tested (not just happy path)

### Documentation Checks
- [ ] `docs/CONTEXT.md` updated with new components/patterns
- [ ] `docs/LLD.md` updated if handler/factory logic changed
- [ ] `docs/ADR.md` updated if new architectural decisions made
- [ ] Commit message follows semantic conventions
- [ ] `last_updated` timestamp bumped in all modified docs

### Safety Checks
- [ ] No hallucinated code — all changes based on diff evidence
- [ ] YAML metadata preserved in all docs
- [ ] JSON appendix in CONTEXT.md still valid
- [ ] No removal of documented sections or warnings
- [ ] Commit is atomic and logically coherent

---

## 19. Summary & Core Principles

**The Manglekit SDK embodies these core principles:**

1. **Config-First:** YAML/ENV are first-class inputs; configuration is reproducible and versionable
2. **Type-Safe DI:** Dependencies injected via typed structs, never generic `any`
3. **Deterministic:** Operations are reproducible across runs; map iteration sorted; explicit component selection
4. **Extensible:** New providers/handlers added without modifying existing code (via DependencyResolver pattern)
5. **Observable:** Unified logging, metrics, and tracing throughout
6. **Well-Documented:** Architecture decisions captured in ADRs; current state in CONTEXT.md; implementation details in LLD.md

**For Agents:** Your role is to maintain these principles as you modify and extend the codebase. Use this document and the referenced docs as your guide.

> *"Architecture is about making the right things easy and the wrong things hard."* — Make the codebase robust, explicit, and self-documenting for future maintainers and agents.