# Manglekit Code Review

**Version:** 4.0
**Status:** Completed
**Date:** 2025-10-16

---

## Introduction

This document provides a comprehensive analysis of the Manglekit SDK codebase following a major architectural refactoring. It identifies code smells, design problems, and areas for improvement. This review supersedes all previous versions and is based on a fresh, holistic audit of the current source code, confirming previous findings and identifying new issues introduced during the refactor.

The review is grounded in the architectural principles of SOLID, extensibility, and maintainability. Each identified smell includes a location, impact analysis, and a concrete refactoring suggestion.

---

## Part 1: Architectural & Design Smells

### Smell: Tight Coupling Between `config.go` and `builder.go`
**Location:** `config.go`: `NewBuilderFromYAML`, `NewBuilderFromEnv`
**Impact Analysis:**
- **SRP Violation:** This coupling violates the Single Responsibility Principle. `config.go` should only be responsible for loading configuration into a pure `Config` struct, not for constructing a builder.
- **Difficult to Test:** It's impossible to test configuration loading in isolation from the builder logic.
- **Maintenance Overhead:** The duplicated logic for component configuration (`configureComponent` and `configureComponentFromEnv`) means that any change to how components are configured from a file/env must be implemented in two separate places.
**Refactoring Suggestion:**
1.  Decouple configuration loading from builder instantiation. The `config.go` functions should parse the environment/file and return a pure `Config` struct.
2.  Create a new function, perhaps `NewBuilderFromConfig(cfg *Config, r *Registry)`, which is responsible for populating the builder from that struct.
3.  Refactor the duplicated component-building logic into a single, private helper function that can be used by the new `NewBuilderFromConfig` method.
**Status:** Open

### Smell: SRP Violation and Magic Strings in `pipeline/sandwich.go`
**Location:** `pipeline/sandwich.go`: `Execute` method and its helpers.
**Impact Analysis:**
- **Low Cohesion:** The `Execute` method is difficult to understand, test, and maintain due to its size and mixed responsibilities (e.g., logging, tracing, state management, and pipeline orchestration). The control flow is complex, especially around error and state handling.
- **Error-Prone:** The use of magic strings (e.g., "reranked_docs", "history", "retrieve_ms") is brittle. A typo in a string literal will not be caught by the compiler and will lead to silent runtime bugs where metadata is not found or stored correctly.
**Refactoring Suggestion:**
1.  Break down the `Execute` method into smaller, more focused private methods. The main `Execute` method should be a clean, high-level summary of the pipeline, delegating all complex logic.
2.  Define constants for all `Meta` map keys (e.g., `const RerankedDocsKey = "reranked_docs"`) in a relevant package (e.g., `core`) to provide compile-time checking and improve readability.
3.  Clarify the error handling and state management flow to be more linear and predictable.
**Status:** Open

### Smell: Inconsistent and Opaque Builder API
**Location:** `builder.go`: `WithEmbedder` method, `buildLLM`/`buildEmbedder` methods.
**Impact Analysis:**
- **Poor Developer Experience (DevEx):** The API is unpredictable. A developer would reasonably expect to be able to provide a pre-built instance for any component, not just the embedder.
- **Untestable Code:** The logic that uses the internal `clients` map for dependency injection cannot be exercised through the public API, making it effectively dead or untestable code.
**Refactoring Suggestion:**
1.  Make the `With...` methods consistent. Either allow all of them to accept pre-built instances or none of them. A consistent approach is preferable.
2.  If providing shared clients is a desired feature, add a public `WithClient(name string, client any)` method to the `BuilderAPI` so that the internal `clients` map can be populated correctly.
**Status:** Open

### Smell: Interface Pollution and Type Safety Issues in `core`
**Location:** `core/types.go`
**Impact Analysis:**
- **Sacrifices Type Safety:** The `Orchestrator` interface and `Options` struct use `any` to avoid import cycles. This pushes type checking from compile-time to runtime, forcing callers to perform risky type assertions. A wrong type will cause a runtime panic.
- **Symptom of Poor Package Design:** Using `any` to break import cycles is a strong indicator that the package boundaries are incorrect. The `core` package should not have dependencies on higher-level packages like `retrieve` or `llm`.
**Refactoring Suggestion:**
1.  **For `Orchestrator.Retriever()`:** Define a new, minimal `Retriever` interface within the `core` package itself (e.g., `type CoreRetriever interface { Retrieve(...) }`). The full `retrieve.Retriever` can embed this new interface. The `Orchestrator.Retriever()` method can then return the type-safe `core.CoreRetriever`.
2.  **For `Options`:** This is harder to solve but could be addressed by moving the `Sandwich` orchestrator's constructor logic into its own package and defining a sandwich-specific options struct there, which *can* have dependencies on `retrieve`, `llm`, etc.
**Status:** Open

### Smell: Magic Number and Flawed Configuration in Hybrid Retriever
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:**
- **Prevents Tuning:** A hard-coded constant `k = 60.0` in its Reciprocal Rank Fusion (RRF) algorithm prevents users from tuning it for their specific use case, severely limiting the retriever's effectiveness.
- **Inflexible Composition:** Its factory builds the child retrievers ("bm25", "dense") with `nil` options, making it impossible to configure them.
**Refactoring Suggestion:**
1.  Expose the `k` constant as a configurable parameter in the `retrieve.HybridOptions` struct, with a sensible default value.
2.  Add fields to `retrieve.HybridOptions` to hold the options for the child retrievers (e.g., `BM25Options *retrieve.BM25Options`, `DenseOptions *retrieve.DenseOptions`).
3.  Update the hybrid retriever factory to use these options when calling `subBuilder.BuildRetriever`.
**Status:** Open

### Smell: Inconsistent Factory Signatures and `any` Usage in Registry
**Location:** `registry.go`
**Impact Analysis:**
- **Reduces Type Safety:** The `ClientFactories` map is `map[string]any`, and the `ClientFactory` type has a completely different signature from all other factories, forcing a runtime type assertion in the builder. This re-introduces the risk of runtime panics that the rest of the refactoring successfully eliminated.
- **Inconsistent API:** The different signature for `ClientFactory` (taking `*Config`) adds cognitive load for developers and couples it to the config structure, unlike other factories which use the more generic `FactoryDeps`.
**Refactoring Suggestion:**
1.  Define a strong type for `ClientFactory` that is consistent with the other factories (e.g., `type ClientFactory func(ctx context.Context, opts any, deps FactoryDeps) (any, core.ResourceCloser, error)`).
2.  Use this strong type in the `ClientFactories` map to achieve full type safety.
3.  The builder can then pass the relevant provider-family config (e.g., `GoogleConfig`) via the `opts any` parameter.
**Status:** Open

### Smell: Dead and Unused Code
**Location:** `builder.go`, `registry.go`
**Impact Analysis:**
- **Code Clutter:** Dead code adds noise and can confuse future developers, who may waste time trying to understand its purpose or be afraid to remove it. The `embedderAlias` map in `builder.go` and the `Options` map in `registry.go` are unused.
**Refactoring Suggestion:**
- Remove the `embedderAlias` variable declaration from `builder.go`.
- Remove the unused `Options` field from the `Registry` struct in `registry.go`.
**Status:** Open

---

## Part 2: Resolved or Mitigated Issues

### Smell: Violation of Open/Closed Principle in Builder
**Location:** `builder.go`
**Analysis:** The `Build` method in `builder.go` no longer uses a `switch` statement. It now correctly uses a dynamic factory lookup (`b.registry.OrchestratorFactories`) to construct the orchestrator, adhering to the Open/Closed Principle.
**Status:** Resolved

### Smell: Tight Coupling in Dependency Resolution
**Location:** `builder.go`
**Analysis:** The brittle `resolveDependencies` method has been completely removed. The builder is no longer responsible for "guessing" component dependencies. This responsibility now correctly lies within the component factories.
**Status:** Resolved

### Smell: Global State in Registry and Typemap
**Location:** `registry.go`
**Analysis:** This critical flaw has been fully addressed. The `typemap.go` file was removed, and its maps were moved into the `Registry` struct. Each `Registry` instance is now fully self-contained and encapsulated, eliminating the risk of interference from global state.
**Status:** Resolved

### Smell: Leaky Abstraction via Builder Callback
**Location:** `builder.go`
**Analysis:** The original refactoring introduced a flaw where composite components were given a dependency on the entire `BuilderAPI`. This has been resolved by the introduction of the `SubRetrieverBuilder` interface in `builder.go`, an excellent application of the Interface Segregation Principle.
**Status:** Resolved