# Manglekit Code Review

**Version:** 2.0
**Status:** Completed (2025-10-15)

---

## Introduction

This document provides a comprehensive analysis of the Manglekit codebase following a major architectural refactoring. It identifies code smells, design problems, and areas for improvement. This review supersedes all previous versions and is based on a fresh, holistic audit of the current source code.

The review is grounded in the architectural principles of SOLID, extensibility, and maintainability. Each identified smell includes a location, impact analysis, and a concrete refactoring suggestion.

## Part 1: Status of Previously Identified Smells

### 1. Violation of Open/Closed Principle in Builder

**Status: Resolved**

**Analysis:** The `Build` method in `builder.go` no longer uses a `switch` statement. It now correctly uses a dynamic factory lookup (`b.registry.OrchestratorFactories`) to construct the orchestrator, adhering to the Open/Closed Principle.

---

### 2. Inconsistent Dependency Injection

**Status: Partially Resolved**

**Smell:** The dependency injection mechanism has been standardized around a `FactoryDeps` map (`map[string]any`), which is a significant improvement. However, two inconsistencies remain.
1.  The `ClientFactory` in `registry.go` uses a completely different signature than all other factories.
2.  The use of `map[string]any` sacrifices compile-time type safety.

**Location(s):**
- `registry.go` (ClientFactory signature)
- `builder.go` (Use of `FactoryDeps`)

**Impact Analysis:** The lack of compile-time safety means that dependency errors (e.g., passing a `Retriever` where an `LLM` is expected) are only caught at runtime. The inconsistent `ClientFactory` signature adds a cognitive burden for developers.

**Refactoring Suggestion:**
- Unify the `ClientFactory` signature to be consistent with other factories.
- Future consideration: Evolve the `FactoryDeps` map into a strongly-typed struct to re-introduce compile-time safety.

---

### 3. Tight Coupling in Dependency Resolution

**Status: Resolved**

**Analysis:** The brittle `resolveDependencies` method in `builder.go` has been completely removed. The builder is no longer responsible for "guessing" component dependencies. Instead, dependencies are explicitly passed to factories via the `FactoryDeps` map, and it is the factory's responsibility to validate them. This is a major architectural improvement.

---

### 4. Global State in Registry and Typemap

**Status: Resolved**

**Analysis:** This was a critical flaw that has been fully addressed. The `typemap.go` file has been removed, and its maps (`nameToOptionsType`, `optionsTypeToName`) have been moved into the `Registry` struct in `registry.go`. Each `Registry` instance is now fully self-contained and encapsulated, eliminating the risk of interference from global state.

---

### 5. Use of `any` for Factory Functions

**Status: Partially Resolved**

**Smell:** While most component factories have been converted to type-safe function signatures (e.g., `RetrieverFactory`), the `ClientFactories` map in `registry.go` remains `map[string]any`.

**Location(s):**
- `registry.go`: `ClientFactories` map definition.

**Impact Analysis:** This forces a runtime type assertion in the builder when creating clients, re-introducing the risk of runtime panics that the rest of the refactoring successfully eliminated.

**Refactoring Suggestion:**
- Define a strong type for `ClientFactory` (e.g., `type ClientFactory func(...) (...)`) and use it in the `ClientFactories` map to achieve full type safety.

---

### 6. Tight Coupling Between `config.go` and `builder.go`

**Status: Unresolved**

**Smell:** The `NewBuilderFromYAML` and `NewBuilderFromEnv` functions in `config.go` are tightly coupled to the `Builder`. They are responsible for both loading/parsing configuration and calling the builder's `With...` methods.

**Location(s):**
- `config.go`: `NewBuilderFromYAML`, `NewBuilderFromEnv`

**Impact Analysis:** This coupling makes it difficult to test configuration loading in isolation from the builder. It also violates the Single Responsibility Principle, as `config.go` is doing more than just configuration management.

**Refactoring Suggestion:**
- Decouple configuration loading from builder instantiation. The config functions should parse the environment/file and return a pure `Config` struct. A separate `NewBuilderFromConfig(cfg *Config)` function should then be responsible for populating the builder from that struct.

---

### 7. Violation of SRP in `pipeline/sandwich.go`

**Status: Unresolved**

**Smell:** The `Execute` method in `pipeline/sandwich.go` is a large function that orchestrates the entire RAG pipeline. While helper methods exist, the main function is still too large and has too many responsibilities.

**Location(s):**
- `pipeline/sandwich.go`

**Impact Analysis:** The method is difficult to understand, test, and maintain due to its size and complexity.

**Refactoring Suggestion:**
- Continue to break down the `Execute` method into smaller, more focused private methods, each responsible for a single, well-defined stage of the pipeline (e.g., `runRetrieve`, `runRerank`, `runLlm`).

---

### 8. Magic Strings for `Meta` Map in `pipeline/sandwich.go`

**Status: Unresolved**

**Smell:** The code uses raw string literals ("reranked_docs", "history") as keys for the `Answer.Meta` map.

**Location(s):**
- `pipeline/sandwich.go`

**Impact Analysis:** This is error-prone, as typos in string literals are not caught by the compiler and lead to runtime bugs.

**Refactoring Suggestion:**
- Define constants for the keys of the `Meta` map (e.g., `const RerankedDocsKey = "reranked_docs"`) to provide compile-time checking and improve readability.

---

### 9. Magic Number in Hybrid Retriever

**Status: Unresolved**

**Smell:** The hybrid retriever in `internal/providers/hybrid/hybrid.go` uses a hard-coded constant `k = 60.0` in its Reciprocal Rank Fusion (RRF) algorithm.

**Location(s):**
- `internal/providers/hybrid/hybrid.go`

**Impact Analysis:** This magic number is a critical configuration value hidden in the implementation, making it impossible for users to tune the RRF algorithm for their specific use case.

**Refactoring Suggestion:**
- Expose the `k` constant as a configurable parameter in the `retrieve.HybridOptions` struct, with a sensible default value.

---

### 10. Use of `any` for Retriever in Orchestrator Interface

**Status: Unresolved**

**Smell:** The `Orchestrator` interface in `core/types.go` defines the `Retriever()` method as returning `any`.

**Location(s):**
- `core/types.go`

**Impact Analysis:** This sacrifices type safety at a critical boundary, forcing the caller to perform a type assertion. It was likely done to avoid a circular dependency between the `core` and `retrieve` packages, but it remains a design smell.

**Refactoring Suggestion:**
- Define a new, minimal `Retriever` interface within the `core` package itself (e.g., `type CoreRetriever interface { Retrieve(...) }`). The full `retrieve.Retriever` can embed this new interface. The `Orchestrator.Retriever()` method can then return the type-safe `core.CoreRetriever`.

---

## Part 2: New Code Smells Identified

### 11. Leaky Abstraction via Builder Callback (Status: Resolved)

**Analysis:** The original refactoring introduced a flaw where composite components (like the hybrid retriever) were given a dependency on the entire `BuilderAPI`. This has been **resolved** by the introduction of the `SubRetrieverBuilder` interface in `builder.go`. This is an excellent application of the Interface Segregation Principle, ensuring the factory only has access to the functionality it needs.

---

### 12. Dead Code in `builder.go`

**Smell:** The `embedderAlias` map at the end of `builder.go` is a remnant of the old dependency resolution logic and is now completely unused.

**Location(s):**
- `builder.go`

**Impact Analysis:** This is a minor issue, but dead code adds clutter and can confuse future developers who may waste time trying to understand its purpose.

**Refactoring Suggestion:**
- Remove the `embedderAlias` variable declaration.