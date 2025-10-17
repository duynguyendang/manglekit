# Manglekit SDK Code Review

**Author:** Jules, Senior Go Software Architect
**Date:** 2025-10-16
**Status:** Completed

---

## Executive Summary

This code review provides a deep-dive analysis of the Manglekit SDK's internal architecture, focusing on its core construction and orchestration logic. The review confirms that while previous refactorings have improved the system's structure (e.g., separating configuration from the builder, introducing pipeline stages), several significant architectural smells persist.

This report documents three primary issues that violate the SDK's stated goals of modularity, consistency, and extensibility. All three issues are currently marked with **Status: Open**. No code changes were made as part of this review; the goal is to provide a clear and actionable record of the current architectural state.

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

### Smell: Repetitive, Non-Generic Builder Logic
**Location:** `builder.go` (multiple `With...` and `build...` methods)
**Impact Analysis:** The builder contains significant code duplication. The `With<Component>` methods (e.g., `WithRetriever`, `WithLLM`) are nearly identical, each performing the same boilerplate logic of looking up an options type in the registry. Likewise, the `build<Component>` methods (`buildRetriever`, `buildLLM`, etc.) are also highly repetitive, each following the same pattern: get a factory, assemble dependencies, call the factory, and store the component. This violates the DRY (Don't Repeat Yourself) principle, increases the maintenance burden, and makes adding new component types a tedious, error-prone process.
**Refactoring Suggestion:** Refactor the builder and registry to use a single, generic registration and build mechanism. This could be achieved using reflection or generics (if Go version allows) to handle all component types through a unified `With(name, opts)` and `build(name)` flow, eliminating the per-type boilerplate.
**Status:** **Open**

### Smell: Type Erasure via `core.Options`
**Location:** `core/types.go` (definition of `Options`), `pipeline/sandwich.go` (`NewSandwich` constructor)
**Impact Analysis:** The `core.Options` struct uses `any` for its component fields (e.g., `Retriever any`, `LLM any`). The builder populates these fields, and the `NewSandwich` orchestrator constructor is then forced to use runtime type assertions to cast them back to their concrete interface types. This pattern, known as type erasure, moves type checking from compile-time to runtime, making the system more fragile. A mismatch between the type provided by the builder and the type expected by the orchestrator will only be caught at runtime, potentially leading to panics.
**Refactoring Suggestion:** The orchestrator factory (`OrchestratorFactory`) signature should be changed to accept a struct of fully-resolved, typed component interfaces instead of the `core.Options` struct. The builder would be responsible for populating this new struct, ensuring all components are of the correct type before the orchestrator is ever created.
**Status:** **Open**

### Smell: Rigid, Type-Specific Registries
**Location:** `registry.go`
**Impact Analysis:** The `Registry` uses separate, strongly-typed maps for each component factory type (e.g., `Retrievers map[string]RetrieverFactory`, `LLMs map[string]LLMFactory`). This rigidity is the root cause of the repetitive builder logic. To add a new type of component to the framework, one must modify the `Registry` struct, add a new `Register...` method, and add a new `build...` method to the builder. This makes the framework difficult to extend.
**Refactoring Suggestion:** Consolidate the disparate factory maps into a single, generic registry, likely a `map[string]any` where the `any` is a generic factory function type. This would allow new component types to be registered without modifying the core registry or builder code, dramatically improving extensibility.
**Status:** **Open**

---

## 3. Resolved Issues (Historical Context)

The following issues were identified and fixed in previous refactoring cycles. They are preserved here for historical context.

### Smell: Inconsistent Builder API
**Location:** `builder.go` (`WithEmbedder` method)
**Impact Analysis:** The `WithEmbedder` method deviated from the builder's established API pattern by accepting pre-built instances.
**Refactoring Action:** The `WithEmbedder` method was refactored to only accept typed options structs, consistent with the rest of the builder API.
**Status:** **Resolved**

### Smell: Hard-coded Orchestrator Selection
**Location:** `builder.go` (`Build` method)
**Impact Analysis:** The builder previously hard-coded the `"sandwich"` orchestrator, preventing programmatic selection of other pipelines.
**Refactoring Action:** A new `WithOrchestrator(name string)` method was added to the builder, and the `Build` method now uses a dynamic factory lookup.
**Status:** **Resolved**

### Smell: God Method & Magic Strings
**Location:** `pipeline/sandwich.go` (legacy)
**Impact Analysis:** The `Sandwich.Execute` method was a classic "God Method" with excessive responsibilities and used hardcoded "magic strings" for passing data.
**Refactoring Action:** The orchestrator was refactored into a sequence of discrete `pipeline.Stage` components executed by a `Runner`, using a typed `pipeline.PipelineContext` for data flow.
**Status:** **Resolved**

### Smell: SRP Violation in Configuration
**Location:** `config.go` (legacy), `builder.go` (legacy)
**Impact Analysis:** The builder was previously responsible for both loading configuration and wiring components.
**Refactoring Action:** A dedicated `config` package now handles loading, and the `NewBuilderFromConfig` function acts as the bridge to the builder.
**Status:** **Resolved**