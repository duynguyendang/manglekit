# Manglekit Code Review

**Version:** 1.0
**Status:** In Progress

---

## Introduction

This document provides a comprehensive analysis of the Manglekit codebase, identifying code smells, potential design problems, and areas for improvement. The review is grounded in the architectural principles outlined in the `HLD.md` and `LLD.md` documents, with a strong focus on SOLID principles, extensibility, and maintainability. Each identified smell includes a location, impact analysis, and a concrete refactoring suggestion.

---

## 1. Violation of Open/Closed Principle in Builder

**Smell:** The `Build` method in `builder.go` uses a `switch` statement to handle different orchestrator types ("sandwich" and "declarative").

**Location(s):**
- `builder.go`: L444

**Impact Analysis:** This design violates the Open/Closed Principle, which states that software entities should be open for extension but closed for modification. If a new orchestrator type is added, the `Build` method must be modified. This increases the risk of introducing bugs and makes the builder less maintainable and extensible, which contradicts the core architectural principle of extensibility via a registry.

**Refactoring Suggestion:**
- Introduce an `OrchestratorFactory` interface with a `Build` method.
- Create concrete factory implementations for each orchestrator type (e.g., `SandwichOrchestratorFactory`, `DeclarativeOrchestratorFactory`).
- Register these factories in the registry.
- The `Builder.Build` method would then look up the appropriate factory from the registry based on the configured orchestrator type and delegate the build process to it. This would eliminate the `switch` statement and allow new orchestrator types to be added without modifying the builder.

---

## 2. Inconsistent Dependency Injection

**Smell:** The dependency injection mechanism in `builder.go` is inconsistent. In `buildSingleTool`, dependencies are passed in a generic `FactoryDeps` map, while in other `build*` methods, dependencies are passed as direct arguments.

**Location(s):**
- `builder.go`: L692 (buildSingleTool)
- `builder.go`: L880 (buildVectorStore)
- `builder.go`: L911 (buildRetrieverComponent)

**Impact Analysis:** This inconsistency makes the code harder to understand and maintain. It also makes it difficult to reason about the dependencies of a component without inspecting the implementation of its factory and the specific `build*` method that calls it.

**Refactoring Suggestion:**
- Standardize on a single dependency injection mechanism.
- The `FactoryDeps` map used in `buildSingleTool` is a good candidate, as it is more flexible and allows for a more generic build process.
- All `build*` methods should be refactored to use this mechanism.
- The factory functions for all components should be updated to accept a `FactoryDeps` map.

---

## 3. Tight Coupling in Dependency Resolution

**Smell:** The `resolveDependencies` method in `builder.go` contains logic that is tightly coupled to the specific requirements of different retriever and reranker types.

**Location(s):**
- `builder.go`: L760

**Impact Analysis:** This tight coupling makes it difficult to add new retrievers or rerankers with different dependency requirements without modifying the `resolveDependencies` method. This violates the Open/Closed Principle and makes the builder less extensible.

**Refactoring Suggestion:**
- Introduce a mechanism for components to declare their dependencies.
- This could be done by adding a `Dependencies() []string` method to the component's `Options` struct.
- The `resolveDependencies` method could then iterate over the configured components, inspect their declared dependencies, and resolve them accordingly.

---

## 4. Stringly-Typed Dependencies

**Smell:** Tool dependencies in the declarative orchestrator are identified by inspecting the string values in the tool's parameters.

**Location(s):**
- `builder.go`: L646 (getToolDependencies)

**Impact Analysis:** This approach is brittle and error-prone. If a tool's parameter happens to have a string value that matches the name of another tool, it will be incorrectly identified as a dependency. This can lead to incorrect build ordering and difficult-to-debug errors.

**Refactoring Suggestion:**
- Introduce a more explicit way to declare tool dependencies.
- This could be done by adding a dedicated `dependencies` field to the `ToolConfig` struct.
- The `getToolDependencies` method could then read the dependencies from this field, which would be more reliable and less prone to errors.

---

## 5. Lack of `context.Context` Propagation

**Smell:** Several `build*` methods in `builder.go` do not accept a `context.Context` argument, which is then passed to the component factories.

**Location(s):**
- `builder.go`: L838 (buildEmbedder)
- `builder.go`: L907 (buildRetriever)
- `builder.go`: L954 (buildReranker)
- `builder.go`: L1022 (buildLLM)

**Impact Analysis:** This prevents proper propagation of request-scoped values and cancellation signals, which is a critical aspect of building robust and resilient distributed systems. It also makes it difficult to implement tracing and other observability features.

**Refactoring Suggestion:**
- All `build*` methods should accept a `context.Context` as their first argument.
- This context should be passed down to the component factories.
- The component factories should use this context when making external calls or creating long-lived resources.

---

## 6. Potential Nil Pointer Dereference

**Smell:** In `buildSingleTool`, the code that resolves dependencies for `embedder`, `vectorStore`, `bm25`, and `dense` does not check if `cfg.Params` is nil before accessing it.

**Location(s):**
- `builder.go`: L719-L732

**Impact Analysis:** If a tool is configured without any parameters, this will cause a panic due to a nil pointer dereference.

**Refactoring Suggestion:**
- Add a nil check for `cfg.Params` before accessing it.
- If `cfg.Params` is nil, skip the dependency resolution for that tool.

---

## 7. Overly Complex Retriever Build Logic

**Smell:** The `buildRetriever` method contains special-cased logic for the "hybrid" retriever.

**Location(s):**
- `builder.go`: L928

**Impact Analysis:** This makes the `buildRetriever` method more complex and harder to maintain. It also violates the Single Responsibility Principle, as the method is responsible for both building individual retrievers and composing the hybrid retriever.

**Refactoring Suggestion:**
- The logic for building the hybrid retriever should be moved into its own factory function.
- The `buildRetriever` method would then simply call this factory function when the retriever name is "hybrid".
- This would simplify the `buildRetriever` method and make the code more modular and easier to understand.

---

## 8. Global State in Registry and Typemap

**Smell:** The `Registry` in `registry.go` and the `nameToOptionsType`/`optionsTypeToName` maps in `typemap.go` are global variables.

**Location(s):**
- `registry.go`: L25
- `typemap.go`: L7

**Impact Analysis:** Global state makes the system harder to test, as tests can interfere with each other by modifying the same global variables. It also introduces the risk of concurrency issues if multiple goroutines access the global maps without proper synchronization. This design makes it difficult to have multiple, isolated instances of the Manglekit framework within the same application, each with its own set of providers.

**Refactoring Suggestion:**
- Encapsulate the registry and typemap within a `Framework` or `App` struct.
- The `Builder` would be created from this `Framework` instance and would have access to its encapsulated registry and typemap.
- This would eliminate the global state, improve testability, and allow for multiple, isolated Manglekit instances.

---

## 9. Use of `any` for Factory Functions

**Smell:** The `Registry` stores factory functions as `any`, requiring type assertions in the builder.

**Location(s):**
- `registry.go`: L32, L38, L40

**Impact Analysis:** The use of `any` for factory functions sacrifices type safety. If a provider is registered with a factory function that has an incorrect signature, the error will only be caught at runtime when the type assertion fails. This makes the code more brittle and harder to refactor.

**Refactoring Suggestion:**
- Use strongly-typed factory function signatures in the registry.
- For example, instead of `map[string]any`, the `Rules` registry could be `map[string]func(core.MangleOptions) (core.RuleSet, error)`.
- This would allow the compiler to catch errors at compile time, improving the overall robustness of the framework.

---

## 10. Unsafe Reflection in `typemap.go`

**Smell:** The `RegisterOptions` function in `typemap.go` relies on reflection to register provider options, which is not type-safe.

**Location(s):**
- `typemap.go`: L12

**Impact Analysis:** While the function does perform some checks, it is still possible to register an incorrect options type, which would lead to runtime errors. Reflection-based code is also generally harder to understand and maintain.

**Refactoring Suggestion:**
- Introduce a more type-safe way to register options.
- This could be done by using generics to create a strongly-typed `RegisterOptions` function.
- For example: `func RegisterOptions[T any](providerName string)`. This would ensure that only valid options types can be registered.

---

## 11. Tight Coupling Between `config.go` and `builder.go`

**Smell:** The `NewBuilderFromYAML` and `NewBuilderFromEnv` functions in `config.go` are tightly coupled to the `Builder`'s `With...` methods.

**Location(s):**
- `config.go`: L285, L400

**Impact Analysis:** This tight coupling makes it difficult to use a different builder implementation with the configuration loading functions. It also makes the code harder to test, as the configuration loading functions cannot be tested in isolation from the builder.

**Refactoring Suggestion:**
- Decouple the configuration loading from the builder instantiation.
- The configuration loading functions should return a `Config` struct.
- A new function, `NewBuilderFromConfig`, could then be created to take a `Config` struct and return a `Builder`.
- This would allow the configuration loading and builder instantiation to be tested independently.

---

## 12. Violation of Single Responsibility Principle in `pipeline/sandwich.go`

**Smell:** The `Execute` method in `pipeline/sandwich.go` is a large function that orchestrates the entire pipeline.

**Location(s):**
- `pipeline/sandwich.go`: L152

**Impact Analysis:** While the method delegates to several `run...` helper functions, the main `Execute` method is still responsible for the overall orchestration, state management, and error handling. This makes the method difficult to understand, test, and maintain.

**Refactoring Suggestion:**
- Break down the `Execute` method into smaller, more focused methods, each responsible for a single stage of the pipeline.
- Introduce a `Pipeline` struct that encapsulates the stages and their execution logic.
- The `Execute` method would then simply iterate over the stages and execute them.

---

## 13. Implicit Side Effects in `pipeline/sandwich.go`

**Smell:** The `prepareLlmRequest` method has a side effect of modifying the `Answer` struct's `Citations` field.

**Location(s):**
- `pipeline/sandwich.go`: L368

**Impact Analysis:** The method's name suggests that it only prepares the request for the LLM, but it also has the side effect of creating the citations. This makes the code harder to understand and reason about.

**Refactoring Suggestion:**
- Rename the method to more accurately reflect its behavior, for example, `prepareLlmRequestAndCreateCitations`.
- Alternatively, separate the citation creation logic into its own method.

---

## 14. Magic Strings for `Meta` Map in `pipeline/sandwich.go`

**Smell:** The code uses string literals as keys for the `Answer.Meta` map.

**Location(s):**
- `pipeline/sandwich.go`: L291, L301, L321, L332, L420

**Impact Analysis:** This is error-prone, as a typo in a string literal will not be caught by the compiler and will lead to runtime errors. It also makes it difficult to find all the places where a particular key is used.

**Refactoring Suggestion:**
- Define constants for the keys of the `Meta` map.
- This will provide compile-time checking and make the code more robust and easier to maintain.

---

## 15. Violation of Open/Closed Principle in `pipeline/declarative/orchestrator.go`

**Smell:** The `dispatchToTool` method is a large `switch` statement that handles different tool types.

**Location(s):**
- `pipeline/declarative/orchestrator.go`: L410

**Impact Analysis:** This design violates the Open/Closed Principle. If a new tool type is added, the `dispatchToTool` method must be modified. This increases the risk of introducing bugs and makes the orchestrator less extensible.

**Refactoring Suggestion:**
- Introduce a `Tool` interface with an `Execute` method.
- Each tool implementation would then implement this interface.
- The `dispatchToTool` method would simply call the `Execute` method on the `Tool` interface, eliminating the `switch` statement.

---

## 16. Use of `map[string]any` as Execution Context in `pipeline/declarative/orchestrator.go`

**Smell:** The use of a generic `map[string]any` for the execution context is not type-safe and relies on "magic strings" for keys.

**Location(s):**
- `pipeline/declarative/orchestrator.go`: L257

**Impact Analysis:** This makes the code brittle and error-prone. A typo in a key name or an incorrect type assertion will lead to runtime errors. It also makes it difficult to understand the structure of the execution context without inspecting the implementation of each tool.

**Refactoring Suggestion:**
- Introduce a strongly-typed `ExecutionContext` struct.
- This struct would have fields for the query, documents, answer, and other relevant data.
- This would provide compile-time checking and make the code more robust and easier to understand.

---

## 17. Magic Number in Hybrid Retriever

**Smell:** The hybrid retriever in `internal/providers/hybrid/hybrid.go` uses a hard-coded constant `k = 60.0` in its Reciprocal Rank Fusion (RRF) algorithm.

**Location(s):**
- `internal/providers/hybrid/hybrid.go`: L129

**Impact Analysis:** This "magic number" is a configuration value that is hidden in the implementation. This makes it difficult for users to tune the RRF algorithm for their specific use case. It also makes the code harder to understand, as the significance of the number 60.0 is not immediately obvious.

**Refactoring Suggestion:**
- Expose the `k` constant as a configurable parameter in the `retrieve.HybridOptions` struct.
- This would allow users to tune the RRF algorithm by setting the `k` value in their configuration.
- A default value of 60.0 should be used if the parameter is not provided.

---

## 18. Use of `any` for Retriever in Orchestrator Interface

**Smell:** The `Orchestrator` interface in `core/types.go` defines the `Retriever()` method as returning `any`.

**Location(s):**
- `core/types.go`: L111

**Impact Analysis:** This sacrifices type safety and requires the caller to perform a type assertion. This is done to avoid a circular dependency between the `core` and `retrieve` packages, but it pushes the burden of type safety onto the consumer of the interface.

**Refactoring Suggestion:**
- Define a new, minimal `Retriever` interface in the `core` package that includes only the methods needed by the orchestrator's consumers (e.g., `Updatable.AddDocuments`).
- The full `retrieve.Retriever` interface in the `retrieve` package can then embed this new `core.Retriever` interface.
- The `Orchestrator.Retriever()` method can then be updated to return `core.Retriever`, which would be type-safe and would not cause a circular dependency.

---

## 19. Deprecated `LocalvecOptions` in `core/types.go`

**Smell:** The `LocalvecOptions` struct is defined in `core/types.go` but is marked as deprecated.

**Location(s):**
- `core/types.go`: L38

**Impact Analysis:** Deprecated code should be removed to keep the codebase clean and avoid confusion. Its continued presence in the `core` package suggests that the refactoring to move it to its own provider package was not fully completed.

**Refactoring Suggestion:**
- Remove the `LocalvecOptions` struct from `core/types.go`.
- Ensure that all code that was using this struct has been updated to use the new options struct in the `localvec` provider package.
