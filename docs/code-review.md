# Manglekit Code Review

**Version:** 5.0
**Status:** Completed
**Date:** 2025-10-16

---

## Introduction

This document provides a comprehensive analysis of the Manglekit SDK codebase, superseding all previous versions. It is based on a fresh, holistic audit of the current source code, confirming previous findings and identifying areas for improvement.

The review is grounded in the architectural principles of SOLID, extensibility, and maintainability. Each identified smell includes a location, impact analysis, a concrete refactoring suggestion, and its current status.

---

## Part 1: Open Architectural & Design Smells

### Smell: Tight Coupling Between `config.go` and `builder.go`
**Location:** `config.go`: `NewBuilderFromYAML`, `NewBuilderFromEnv`
**Impact Analysis:**
- **SRP Violation:** These functions violate the Single Responsibility Principle. `config.go` should only be responsible for loading configuration into a pure `Config` struct, not for constructing a builder.
- **Difficult to Test:** It's impossible to test configuration loading in isolation from the builder logic.
- **Maintenance Overhead:** The logic for component configuration is duplicated across `NewBuilderFromYAML` and `NewBuilderFromEnv`, meaning changes must be made in two places.
**Refactoring Suggestion:**
1.  Decouple configuration loading from builder instantiation. The `config.go` functions should parse the environment/file and return a pure `Config` struct.
2.  Create a new function, `NewBuilderFromConfig(cfg *Config, r *Registry)`, which is responsible for populating the builder from that struct, centralizing the configuration logic.
**Status:** Open

### Smell: SRP Violation and Magic Strings in `pipeline/sandwich.go`
**Location:** `pipeline/sandwich.go`: `Execute` method and its helpers.
**Impact Analysis:**
- **Low Cohesion:** The `Execute` method is a "god method" that is difficult to understand, test, and maintain due to its size and mixed responsibilities (logging, tracing, state management, and pipeline orchestration).
- **Error-Prone:** The use of magic strings (e.g., `"reranked_docs"`, `"retrieve_ms"`) for `Meta` map keys is brittle. A typo will not be caught by the compiler and will lead to silent runtime bugs.
**Refactoring Suggestion:**
1.  Refactor the `Execute` method into smaller, more focused private methods, with the main method acting as a high-level summary of the pipeline.
2.  Define and export constants for all `Meta` map keys (e.g., `const RerankedDocsKey = "reranked_docs"`) in the `core` package to provide compile-time checking and improve readability.
**Status:** Open

### Smell: Interface Pollution and Type Safety Issues in `core`
**Location:** `core/types.go`
**Impact Analysis:**
- **Sacrifices Type Safety:** The `Orchestrator` interface and `Options` struct use `any` to avoid import cycles. This pushes type checking from compile-time to runtime, forcing callers to perform risky type assertions.
- **Symptom of Poor Package Design:** Using `any` to break import cycles is a strong indicator that the package boundaries are incorrect. The `core` package should not have dependencies on higher-level packages like `retrieve` or `llm`.
**Refactoring Suggestion:**
1.  Define minimal, core interfaces within the `core` package itself (e.g., `type CoreRetriever interface { Retrieve(...) }`).
2.  The full-featured interfaces in other packages (e.g., `retrieve.Retriever`) can embed these core interfaces.
3.  Update `Orchestrator` and `Options` to use these new, type-safe core interfaces, eliminating the need for `any`.
**Status:** Open

### Smell: Inconsistent and Opaque Builder API
**Location:** `builder.go`: `WithEmbedder` method.
**Impact Analysis:**
- **Poor Developer Experience (DevEx):** The API is unpredictable. The `WithEmbedder` method can accept a pre-built instance, but other `With...` methods cannot, creating an inconsistent and confusing API.
- **Untestable Code:** The internal `clients` map for dependency injection is populated from config files, but there is no public API method (e.g., `WithClient`) to inject these dependencies programmatically, making parts of the builder's logic untestable.
**Refactoring Suggestion:**
1.  Make the `With...` methods consistent: either all should accept pre-built instances or none should.
2.  Add a public `WithClient(name string, client any)` method to the `BuilderAPI` to allow programmatic injection of shared clients, making the dependency injection mechanism fully usable and testable.
**Status:** Open

### Smell: Magic Number and Flawed Configuration in Hybrid Retriever
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:**
- **Prevents Tuning:** A hard-coded constant `k = 60.0` in the Reciprocal Rank Fusion (RRF) algorithm prevents users from tuning it, severely limiting the retriever's effectiveness.
- **Inflexible Composition:** The factory builds its child retrievers ("bm25", "dense") with `nil` options, making it impossible for users to configure them.
**Refactoring Suggestion:**
1.  Expose the `k` constant as a configurable parameter in the `retrieve.HybridOptions` struct.
2.  Add fields to `retrieve.HybridOptions` to hold the options for the child retrievers (e.g., `BM25Options *retrieve.BM25Options`).
3.  Update the factory to use these options when calling `subBuilder.BuildRetriever`.
**Status:** Open

### Smell: Inconsistent Factory Signatures and `any` Usage in Registry
**Location:** `registry.go`
**Impact Analysis:**
- **Reduces Type Safety:** The `ClientFactories` map is `map[string]any`, and the `ClientFactory` type has a completely different signature from all other factories, forcing a runtime type assertion in the builder. This re-introduces the risk of runtime panics.
**Refactoring Suggestion:**
1.  Define a strong type for `ClientFactory` that is consistent with the other factories (e.g., `func(ctx context.Context, opts any, deps FactoryDeps) (any, core.ResourceCloser, error)`).
2.  Use this strong type in the `ClientFactories` map to achieve full type safety.
**Status:** Open

### Smell: Dead and Unused Code
**Location:** `builder.go`
**Impact Analysis:**
- **Code Clutter:** Dead code adds noise and can confuse future developers. The `embedderAlias` map in `builder.go` is unreferenced and serves no purpose.
**Refactoring Suggestion:**
- Remove the `embedderAlias` variable declaration from `builder.go`.
**Status:** Open

---

## Part 2: Resolved or Mitigated Issues

### Smell: Violation of Open/Closed Principle in Builder
**Location:** `builder.go`
**Analysis:** The `Build` method no longer uses a `switch` statement. It now correctly uses a dynamic factory lookup (`b.registry.OrchestratorFactories`) to construct the orchestrator, adhering to the Open/Closed Principle.
**Status:** Resolved

### Smell: Tight Coupling in Dependency Resolution
**Location:** `builder.go`
**Analysis:** The brittle `resolveDependencies` method has been completely removed. This responsibility now correctly lies within the component factories.
**Status:** Resolved

### Smell: Global State in Registry and Typemap
**Location:** `registry.go`
**Analysis:** This critical flaw has been fully addressed. The `Registry` instance is now fully self-contained and encapsulated, eliminating risks from global state.
**Status:** Resolved

### Smell: Leaky Abstraction via Builder Callback
**Location:** `builder.go`
**Analysis:** The introduction of the `SubRetrieverBuilder` interface, an application of the Interface Segregation Principle, prevents composite components from depending on the entire `BuilderAPI`.
**Status:** Resolved