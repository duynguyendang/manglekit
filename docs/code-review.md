# Manglekit SDK Code Review

**Author:** Jules, Senior Go Software Architect
**Date:** 2025-10-16
**Status:** In Progress

---

## Executive Summary

This code review provides a deep-dive analysis of the Manglekit SDK's internal architecture as of October 2025. The last major refactoring successfully decoupled the configuration loading mechanism from the fluent builder, which was a significant improvement. A more recent refactoring has now resolved the "God Method" monolith in the `Sandwich` pipeline by decomposing it into a typed, stage-based architecture.

However, this review has identified several persistent architectural smells that undermine the framework's goals of type safety, modularity, and extensibility. The most critical remaining issues are the type-safety "holes" in the registry and core interfaces that rely on `any` and force unsafe type assertions.

This report outlines these issues, analyzes their impact, and provides actionable refactoring suggestions. The accompanying `docs/CONTEXT.md` file has been regenerated to reflect this new reality.

---

## 1. Orchestration Checks

The following checks must be enforced for all orchestration logic.

-   **No god method**: Orchestration logic must be composed of discrete components that implement the `pipeline.Stage` interface. Monolithic functions that handle multiple, distinct responsibilities (e.g., retrieval, reranking, and LLM calls) are forbidden.
-   **No magic strings**: All data passed between orchestration stages must be done via the typed `pipeline.PipelineContext` struct. Using `map[string]any` with string literals as keys for passing data is forbidden.
-   **Ctx propagation**: Every stage must receive and use the `p.Ctx` from the `PipelineContext`. Stages must not create their own background contexts or hidden timeouts.
-   **Metrics consistency**: Each stage is individually responsible for recording its own timing and performance metrics to the `PipelineContext`.
-   **Use Genkit Plugins**: Providers that interact with external services (e.g., LLMs, embedders) must be implemented as wrappers around Genkit plugins. Custom client implementations and factories are forbidden.

---

## 2. Open Architectural Issues

The following issues are currently present in the codebase and require attention.

### Smell: Interface Pollution & Type Safety Violation
**Location:** `core/types.go`, `pipeline/sandwich.go`
**Impact Analysis:** The `core.Orchestrator` interface previously defined methods like `Retriever() any`. This use of `any` forced consumers to perform unsafe type assertions, bypassing compile-time type safety.
**Refactoring Action:** The `Orchestrator` interface has been refactored to be a pure executor (`Execute`, `Close`). All `any`-based accessors have been removed. The `builder.Build()` method now returns typed components (e.g., `retrieve.Updatable`) alongside the orchestrator, providing a type-safe mechanism for accessing components that require runtime interaction. A new rule has been added: **No `any` accessors** — all typed components must be returned explicitly from builder factories.
**Status:** **Resolved**

### Smell: Inconsistent Builder API
**Location:** `builder.go`
**Impact Analysis:** The `WithEmbedder` method has a special case: `if emb, ok := opts.(ai.Embedder); ok`. It allows passing a pre-built embedder instance directly, bypassing the standard factory mechanism. While potentially convenient, this makes the builder's API inconsistent and less predictable compared to other methods like `WithLLM` or `WithRetriever`, which only accept options structs. This inconsistency complicates the configuration logic, especially in `NewBuilderFromConfig`.
**Refactoring Suggestion:**
1.  **Remove the Special Case:** Remove the `if emb, ok := ...` block from `WithEmbedder`.
2.  **Enforce Uniformity:** Require all `With...` methods to operate consistently by only accepting typed options pointers (e.g., `*embed.GoogleOptions`). This simplifies the builder's internal logic and makes the public API more predictable. If a pre-built instance is needed for testing, it can be handled via a mock provider with mock options.
**Status:** Open

### Smell: Hard-coded Orchestrator Selection
**Location:** `builder.go` (specifically the `Build` method)
**Impact Analysis:** The `Builder.Build` method currently hard-codes the orchestrator type to `"sandwich"`. This prevents users from programmatically selecting a different orchestrator (like the planned `declarative` orchestrator) when using the fluent builder. The choice of orchestrator is a fundamental architectural decision that should be exposed in the builder's API.
**Refactoring Suggestion:**
1.  **Add `WithOrchestrator` Method:** Introduce a new method `WithOrchestrator(name string)` to the `BuilderAPI`.
2.  **Use the Selected Orchestrator:** In the `Build` method, use the name provided via `WithOrchestrator` to look up the factory in the `OrchestratorFactories` map. Default to "sandwich" only if it hasn't been explicitly set.
**Status:** Open

---

## 3. Resolved Issues

### Smell: God Method & Magic Strings
**Location:** `pipeline/sandwich.go` (legacy)
**Impact Analysis:** The `Sandwich.Execute` method was a classic "God Method" with too many responsibilities. It used hardcoded "magic strings" to pass data, which was brittle and error-prone.
**Refactoring Action:** The `Sandwich` orchestrator has been refactored into a sequence of discrete, composable pipeline stages (`pipeline.Stage`). A `pipeline.Runner` executes the stages, and a typed `pipeline.PipelineContext` is used to pass data, eliminating magic strings and making the data flow explicit and type-safe.
**Status:** **Resolved**

### Smell: SRP Violation in Configuration
**Location:** `config.go` (legacy), `builder.go` (legacy)
**Impact Analysis:** The builder was previously responsible for both loading configuration from files/env and wiring components. This violated the Single Responsibility Principle, making the builder bloated and tightly coupled to configuration sources.
**Refactoring Action:** This has been addressed. A dedicated `config` package now handles loading/parsing, and the `NewBuilderFromConfig` function acts as the sole bridge to the builder, which now only handles component wiring.
**Status:** **Resolved**