# Manglekit SDK Code Review

**Author:** Jules, Senior Go Software Architect
**Date:** 2025-10-16
**Status:** In Progress

---

## Executive Summary

This code review provides a deep-dive analysis of the Manglekit SDK's internal architecture as of October 2025. The last major refactoring successfully decoupled the configuration loading mechanism from the fluent builder, which was a significant improvement.

However, this review has identified several new and persistent architectural smells that undermine the framework's goals of type safety, modularity, and extensibility. The most critical issues are the "God Method" in the `Sandwich` pipeline, which violates the Single Responsibility Principle, and several type-safety "holes" in the registry and core interfaces that rely on `any` and force unsafe type assertions.

This report outlines these issues, analyzes their impact, and provides actionable refactoring suggestions. The accompanying `docs/CONTEXT.md` file has been regenerated to reflect this new reality.

---

## 1. Open Architectural Issues

The following issues are currently present in the codebase and require attention.

### Smell: God Method & Magic Strings
**Location:** `pipeline/sandwich.go` (specifically the `Execute` method and its helpers)
**Impact Analysis:** The `Sandwich.Execute` method is a classic "God Method." It has far too many responsibilities: state management, pre-rule evaluation, retrieval, reranking, LLM prompt construction, LLM execution, and post-rule evaluation. This makes the pipeline rigid, difficult to test in isolation, and hard to modify or extend. Furthermore, it uses hardcoded "magic strings" (e.g., `"reranked_docs"`, `"retrieve_ms"`) to pass critical data between stages via a `map[string]any`. This is brittle, error-prone, and hides the data flow contract.
**Refactoring Suggestion:**
1.  **Decompose the Pipeline:** Refactor the `Sandwich` orchestrator into a sequence of discrete, composable pipeline stages (e.g., `StateLoader`, `PreRuleEvaluator`, `Retriever`, `Reranker`, `LLMGenerator`). Each stage should be an interface with a single `Execute` or `Process` method.
2.  **Introduce a Pipeline Runner:** Create a simple runner that takes a list of these stage components and executes them in order.
3.  **Create a Typed Data Context:** Define a `PipelineContext` struct to pass data between stages. This struct would have explicit, strongly-typed fields like `History`, `OriginalDocs`, `RerankedDocs`, and `Metrics`. This eliminates magic strings and makes the data flow explicit and type-safe.
**Status:** Open

### Smell: Interface Pollution & Type Safety Violation
**Location:** `core/types.go`, `pipeline/sandwich.go`
**Impact Analysis:** The `core.Orchestrator` interface defines methods like `Retriever() any`. This use of `any` forces consumers to perform unsafe type assertions (e.g., `orch.Retriever().(retrieve.Retriever)`) to access the underlying component. This completely bypasses compile-time type safety and moves type checking to runtime, where it can cause panics. This is a major architectural smell that indicates incorrect package boundaries or a failure to define appropriate, narrow interfaces.
**Refactoring Suggestion:**
1.  **Eliminate `any`-based accessors:** Remove methods like `Retriever() any` from the `Orchestrator` interface. An orchestrator's job is to `Execute` a pipeline, not to be a generic container for its components.
2.  **Provide Components at Build Time:** If specific components (like an `Updatable` retriever) need to be accessed after the build, the `Build()` method should return them as explicit, type-safe return values alongside the orchestrator itself. For example: `Build() (core.Orchestrator, retrieve.Updatable, error)`.
**Status:** Open

### Smell: Inconsistent Factory Signature & Type Safety Hole
**Location:** `registry.go`
**Impact Analysis:** The `Registry.ClientFactories` map is defined as `map[string]any`, and the `ClientFactory` signature returns `(client any, ...)`. This creates a type-safety hole in the framework's dependency injection system. The builder must fetch a factory from this map and then immediately perform a type assertion on the resulting client. This is unsafe and inconsistent with the other strongly-typed factory maps (e.g., `Retrievers`, `LLMs`).
**Refactoring Suggestion:**
1.  **Introduce a Generic Client Factory:** Define a generic `ClientFactory[T any]` type using Go generics.
2.  **Create Typed Client Registries:** Instead of one `any`-based map, have separate, type-safe maps for different client types if needed, or use a generic registration method that captures the type information safely.
3.  **Update the Builder:** The builder would then request a client of a specific type, eliminating the need for runtime assertions.
**Status:** Open

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

## 2. Resolved Issues

### Smell: SRP Violation in Configuration
**Location:** `config.go` (legacy), `builder.go` (legacy)
**Impact Analysis:** The builder was previously responsible for both loading configuration from files/env and wiring components. This violated the Single Responsibility Principle, making the builder bloated and tightly coupled to configuration sources.
**Refactoring Suggestion:** This has been addressed. A dedicated `config` package now handles loading/parsing, and the `NewBuilderFromConfig` function acts as the sole bridge to the builder, which now only handles component wiring.
**Status:** Resolved