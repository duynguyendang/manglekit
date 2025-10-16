# Manglekit Code Review

**Version:** 3.0
**Status:** In Progress (2025-10-15)

---

## Introduction

This document provides a comprehensive analysis of the Manglekit SDK codebase following a major architectural refactoring. It identifies code smells, design problems, and areas for improvement. This review supersedes all previous versions and is based on a fresh, holistic audit of the current source code, confirming previous findings and identifying new issues introduced during the refactor.

The review is grounded in the architectural principles of SOLID, extensibility, and maintainability. Each identified smell includes a location, impact analysis, and a concrete refactoring suggestion.

---

## Part 1: Architectural & Design Smells

### 1. Tight Coupling Between `config.go` and `builder.go`

**Status: Open**

**Smell:** The `NewBuilderFromYAML` and `NewBuilderFromEnv` functions in `config.go` are tightly coupled to the `Builder`. They are responsible for both loading/parsing configuration *and* calling the builder's `With...` methods. Furthermore, the logic for processing components is duplicated across both functions.

**Location(s):**
- `config.go`: `NewBuilderFromYAML`, `NewBuilderFromEnv`

**Impact Analysis:**
- **SRP Violation:** This coupling violates the Single Responsibility Principle. `config.go` should only be responsible for loading configuration into a pure `Config` struct, not for constructing a builder.
- **Difficult to Test:** It's impossible to test configuration loading in isolation from the builder logic.
- **Maintenance Overhead:** The duplicated logic for component configuration means that any change to how components are configured from a file/env must be implemented in two separate places.

**Refactoring Suggestion:**
1.  Decouple configuration loading from builder instantiation. The `config.go` functions should parse the environment/file and return a pure `Config` struct.
2.  Create a new function, perhaps `NewBuilderFromConfig(cfg *Config, r *Registry)`, which is responsible for populating the builder from that struct.
3.  Refactor the duplicated component-building logic into a single, private helper function that can be used by the new `NewBuilderFromConfig` method.

---

### 2. SRP Violation and Magic Strings in `pipeline/sandwich.go`

**Status: Open**

**Smell:** The `Execute` method in `pipeline/sandwich.go` is a "god method" that orchestrates the entire RAG pipeline, violating the Single Responsibility Principle. Additionally, the code uses numerous raw string literals ("reranked_docs", "history", "retrieve_ms") as keys for the `Answer.Meta` map.

**Location(s):**
- `pipeline/sandwich.go`: `Execute` method and its helpers.

**Impact Analysis:**
- **Low Cohesion:** The `Execute` method is difficult to understand, test, and maintain due to its size and mixed responsibilities (e.g., logging, tracing, state management, and pipeline orchestration). The control flow is complex, especially around error and state handling.
- **Error-Prone:** The use of magic strings is brittle. A typo in a string literal will not be caught by the compiler and will lead to silent runtime bugs where metadata is not found or stored correctly.

**Refactoring Suggestion:**
1.  Continue to break down the `Execute` method into smaller, more focused private methods. The main `Execute` method should be a clean, high-level summary of the pipeline, delegating all complex logic.
2.  Define constants for all `Meta` map keys (e.g., `const RerankedDocsKey = "reranked_docs"`) in a relevant package (e.g., `core`) to provide compile-time checking and improve readability.
3.  Clarify the error handling and state management flow to be more linear and predictable.

---

### 3. Inconsistent and Opaque Builder API

**Status: Open**

**Smell:** The builder's fluent API is inconsistent, and its dependency management mechanism is not fully exposed.
1.  The `WithEmbedder` method can accept a pre-built `ai.Embedder` instance, but other `With...` methods (like `WithRetriever`) only accept options structs.
2.  The builder has an internal `clients` map used for dependency injection, but there is no public `WithClient` method to populate it, making this feature unusable from the outside.

**Location(s):**
- `builder.go`: `WithEmbedder` method, `buildLLM`/`buildEmbedder` methods.

**Impact Analysis:**
- **Poor Developer Experience (DevEx):** The API is unpredictable. A developer would reasonably expect to be able to provide a pre-built instance for any component, not just the embedder.
- **Untestable Code:** The logic that uses the internal `clients` map cannot be exercised through the public API, making it effectively dead or untestable code.

**Refactoring Suggestion:**
1.  Make the `With...` methods consistent. Either allow all of them to accept pre-built instances or none of them.
2.  If providing shared clients is a desired feature, add a public `WithClient(name string, client any)` method to the `BuilderAPI` so that the internal `clients` map can be populated correctly.

---

### 4. Interface Pollution and Type Safety Issues in `core`

**Status: Open**

**Smell:** The core interfaces suffer from type-safety issues caused by attempts to avoid circular dependencies.
1.  The `Orchestrator` interface in `core/types.go` defines the `Retriever()` method as returning `any`.
2.  The `Options` struct in `core/types.go` uses `any` for all of its component fields (`Retriever`, `Reranker`, `LLM`).

**Location(s):**
- `core/types.go`: `Orchestrator` interface, `Options` struct.

**Impact Analysis:**
- **Sacrifices Type Safety:** This pushes type checking from compile-time to runtime, forcing callers to perform risky type assertions. A wrong type will cause a runtime panic.
- **Symptom of Poor Package Design:** Using `any` to break import cycles is a strong indicator that the package boundaries are incorrect. The `core` package should not have dependencies on higher-level packages like `retrieve` or `llm`.

**Refactoring Suggestion:**
1.  **For `Orchestrator.Retriever()`:** Define a new, minimal `Retriever` interface within the `core` package itself (e.g., `type CoreRetriever interface { Retrieve(...) }`). The full `retrieve.Retriever` can embed this new interface. The `Orchestrator.Retriever()` method can then return the type-safe `core.CoreRetriever`.
2.  **For `Options`:** This is harder to solve but could be addressed by moving the `Sandwich` orchestrator's constructor logic into its own package and defining a sandwich-specific options struct there, which *can* have dependencies on `retrieve`, `llm`, etc.

---

### 5. Magic Number and Flawed Configuration in Hybrid Retriever

**Status: Open**

**Smell:** The hybrid retriever has two major configuration flaws.
1.  It uses a hard-coded constant `k = 60.0` in its Reciprocal Rank Fusion (RRF) algorithm.
2.  Its factory builds the child retrievers ("bm25", "dense") with `nil` options, making it impossible to configure them.

**Location(s):**
- `internal/providers/hybrid/hybrid.go`: `Retrieve` method and factory function.

**Impact Analysis:**
- **Prevents Tuning:** The hard-coded `k` value prevents users from tuning the RRF algorithm for their specific use case, severely limiting the retriever's effectiveness.
- **Inflexible Composition:** The inability to configure child retrievers makes the hybrid retriever inflexible. Users are stuck with the default behavior of the BM25 and Dense retrievers.

**Refactoring Suggestion:**
1.  Expose the `k` constant as a configurable parameter in the `retrieve.HybridOptions` struct, with a sensible default value.
2.  Add fields to `retrieve.HybridOptions` to hold the options for the child retrievers (e.g., `BM25Options *retrieve.BM25Options`, `DenseOptions *retrieve.DenseOptions`).
3.  Update the hybrid retriever factory to use these options when calling `subBuilder.BuildRetriever`.

---

### 6. Inconsistent Factory Signatures and `any` Usage in Registry

**Status: Open**

**Smell:** The component factory system, while mostly type-safe, has a glaring inconsistency. The `ClientFactories` map in `registry.go` remains `map[string]any`, and its `ClientFactory` type has a completely different signature from all other factories.

**Location(s):**
- `registry.go`: `ClientFactories` map definition, `ClientFactory` type definition, `RegisterClientFactory` method.

**Impact Analysis:**
- **Reduces Type Safety:** The use of `any` forces a runtime type assertion in the builder when creating clients, re-introducing the risk of runtime panics that the rest of the refactoring successfully eliminated.
- **Inconsistent API:** The different signature for `ClientFactory` (taking `*Config`) adds cognitive load for developers and couples it to the config structure, unlike other factories which use the more generic `FactoryDeps`.

**Refactoring Suggestion:**
1.  Define a strong type for `ClientFactory` that is consistent with the other factories (e.g., `type ClientFactory func(ctx context.Context, opts any, deps FactoryDeps) (any, core.ResourceCloser, error)`).
2.  Use this strong type in the `ClientFactories` map to achieve full type safety.
3.  The builder can then pass the relevant provider-family config (e.g., `GoogleConfig`) via the `opts any` parameter.

---

### 7. Dead and Unused Code

**Status: Open**

**Smell:** The codebase contains unused variables and struct fields left over from previous refactoring efforts.
1.  The `embedderAlias` map in `builder.go` is completely unused.
2.  The exported `Options` map in `registry.go` is never populated or read.

**Location(s):**
- `builder.go`
- `registry.go`

**Impact Analysis:**
- **Code Clutter:** Dead code adds noise and can confuse future developers, who may waste time trying to understand its purpose or be afraid to remove it.

**Refactoring Suggestion:**
- Remove the `embedderAlias` variable declaration from `builder.go`.
- Remove the unused `Options` field from the `Registry` struct in `registry.go`.

---

## Part 2: Resolved or Mitigated Issues

### 8. Violation of Open/Closed Principle in Builder

**Status: Resolved**

**Analysis:** The `Build` method in `builder.go` no longer uses a `switch` statement. It now correctly uses a dynamic factory lookup (`b.registry.OrchestratorFactories`) to construct the orchestrator, adhering to the Open/Closed Principle. This is a significant improvement.

---

### 9. Tight Coupling in Dependency Resolution

**Status: Resolved**

**Analysis:** The brittle `resolveDependencies` method in `builder.go` has been completely removed. The builder is no longer responsible for "guessing" component dependencies. This responsibility now correctly lies within the component factories.

---

### 10. Global State in Registry and Typemap

**Status: Resolved**

**Analysis:** This critical flaw has been fully addressed. The `typemap.go` file was removed, and its maps were moved into the `Registry` struct. Each `Registry` instance is now fully self-contained and encapsulated, eliminating the risk of interference from global state.

---

### 11. Leaky Abstraction via Builder Callback

**Status: Resolved**

**Analysis:** The original refactoring introduced a flaw where composite components were given a dependency on the entire `BuilderAPI`. This has been resolved by the introduction of the `SubRetrieverBuilder` interface in `builder.go`. This is an excellent application of the Interface Segregation Principle.