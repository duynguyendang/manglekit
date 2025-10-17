# Manglekit SDK Code Review

**Author:** Jules, Senior Go Software Architect
**Date:** 2025-10-16
**Status:** Completed

---

## Executive Summary

This code review provides a deep-dive analysis of the Manglekit SDK's internal architecture, focusing on its core construction and orchestration logic. The review confirms that while previous refactorings have improved the system's structure (e.g., separating configuration from the builder, introducing pipeline stages), several significant architectural smells persist.

This report documents two primary issues that violate the SDK's stated goals of modularity, consistency, and extensibility. Both issues are located within the fluent builder (`builder.go`) and are currently marked with **Status: Open**. No code changes were made as part of this review; the goal is to provide a clear and actionable record of the current architectural state.

The accompanying `docs/CONTEXT.md` file has also been updated to reflect this reality, ensuring our architectural standard is synchronized with the actual implementation.

---

## 1. Orchestration Checks

The following principles were used to evaluate the orchestration logic and should be enforced for all future development.

-   **No god method**: Orchestration logic must be composed of discrete, testable components. Monolithic functions are forbidden.
-   **No magic strings**: Data passed between internal components must be via typed structs, not `map[string]any`.
-   **Strict Ctx Propagation**: The `context.Context` must be passed explicitly through all calls.
-   **Consistent Metrics**: Components are responsible for recording their own performance metrics.
-   **Leverage Genkit**: Providers interacting with external services must wrap standard Genkit plugins.

---

## 2. Open Architectural Issues

The following issues are present in the current codebase and are documented here as the official findings of this review.

### Smell: Inconsistent Builder API
**Location:** `builder.go` (`WithEmbedder` method)
**Impact Analysis:** The `WithEmbedder` method deviates from the builder's established API pattern. Unlike other `With...` methods that exclusively accept typed options structs (e.g., `WithLLM(opts *llm.OpenAIOptions)`), it contains a special case (`if emb, ok := opts.(ai.Embedder); ok`) that allows passing a pre-built instance. This inconsistency makes the builder's API less predictable, complicates the configuration-to-builder mapping logic (`NewBuilderFromConfig`), and creates a maintenance burden.
**Refactoring Suggestion:**
1.  **Remove the Special Case:** Eliminate the `if emb, ok := ...` block from the `WithEmbedder` method.
2.  **Enforce Uniformity:** Mandate that all `With...` methods operate consistently by only accepting typed options pointers. This simplifies the builder's internal logic and provides a cleaner, more predictable public API. Pre-built instances for testing should be handled via mock providers with corresponding mock options structs.
**Status:** Open

### Smell: Hard-coded Orchestrator Selection
**Location:** `builder.go` (`Build` method)
**Impact Analysis:** The `Builder.Build` method currently hard-codes the orchestrator type to `"sandwich"`, preventing users from programmatically selecting a different orchestrator (such as the planned `declarative` orchestrator). The choice of pipeline is a fundamental architectural decision that should be exposed in the builder's API, not hidden as an implementation detail. This limitation severely restricts the SDK's extensibility.
**Refactoring Suggestion:**
1.  **Add `WithOrchestrator` Method:** Introduce a new method, `WithOrchestrator(name string) BuilderAPI`, to the `BuilderAPI` interface and implement it on the `Builder`.
2.  **Use the Selected Orchestrator:** Modify the `Build` method to use the name provided via `WithOrchestrator` to look up the appropriate factory in the `OrchestratorFactories` map. It should default to "sandwich" only if no explicit selection has been made to maintain backward compatibility.
**Status:** Open

---

## 3. Resolved Issues (Historical Context)

The following issues were identified and fixed in previous refactoring cycles. They are preserved here for historical context.

### Smell: God Method & Magic Strings
**Location:** `pipeline/sandwich.go` (legacy)
**Impact Analysis:** The `Sandwich.Execute` method was a classic "God Method" with excessive responsibilities. It used hardcoded "magic strings" for passing data, which was brittle and error-prone.
**Refactoring Action:** The `Sandwich` orchestrator was refactored into a sequence of discrete `pipeline.Stage` components. A `pipeline.Runner` executes the stages, and a typed `pipeline.PipelineContext` is used to pass data, eliminating magic strings and making the data flow explicit and type-safe.
**Status:** **Resolved**

### Smell: SRP Violation in Configuration
**Location:** `config.go` (legacy), `builder.go` (legacy)
**Impact Analysis:** The builder was previously responsible for both loading configuration and wiring components, violating the Single Responsibility Principle.
**Refactoring Action:** A dedicated `config` package now handles all loading and parsing. The `NewBuilderFromConfig` function acts as the sole bridge to the builder, which now focuses exclusively on component wiring.
**Status:** **Resolved**