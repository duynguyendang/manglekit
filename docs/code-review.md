## Smell: Type Assertions in Core Component Factories
**Location:** `pipeline/sandwich.go` (function `NewSandwich`)
**Impact Analysis:** The `core.Resolved` struct, which is intended to be a type-safe container for fully built components, uses `any` for its fields (e.g., `Retriever any`). This forces the `NewSandwich` factory to perform type assertions (e.g., `deps.Retriever.(retrieve.Retriever)`). This undermines the goal of end-to-end type safety, re-introducing runtime risks that the builder pattern was designed to eliminate. A failure here would cause a panic during orchestrator construction.
**Refactoring Suggestion:** Modify the `core.Resolved` struct to use concrete interface types (e.g., `Retriever retrieve.Retriever`, `LLM llm.Client`). This will enforce type safety at compile time and remove the need for any type assertions in the factory, making the code more robust and self-documenting.
**Status:** Resolved

## Smell: Broken Resource Cleanup Lifecycle
**Location:** `pipeline/sandwich.go` (struct `Sandwich`, method `Close`)
**Impact Analysis:** The `Builder` in `builder.go` correctly collects `ResourceCloser` functions from components that require cleanup. However, it never passes this list of closers to the created `Sandwich` orchestrator. The `Sandwich` struct has its own `closers` slice that is never populated, so its `Close` method does nothing. This is a critical bug that will lead to resource leaks (e.g., open database connections, running background goroutines) for any provider that requires graceful shutdown.
**Refactoring Suggestion:** The `core.Resolved` struct passed to orchestrator factories should contain the slice of `core.ResourceCloser` functions collected by the builder. The `NewSandwich` factory must then assign this slice to the `closers` field in the `Sandwich` struct, ensuring that the `Close` method can correctly iterate and call them.
**Status:** Open

## Smell: Dead Code - Declarative Orchestrator is Unreachable
**Location:** `pipeline/declarative/orchestrator.go`, `builder.go`
**Impact Analysis:** The entire declarative orchestrator is dead code. The `builder.go` has no logic to construct the dependencies required by `declarative.New` (specifically the `core.FlowController` and the `map[string]any` of tools). The builder is hard-wired to always instantiate the `sandwich` orchestrator. This represents a significant amount of un-used, un-tested, and potentially bit-rotting code in the repository.
**Refactoring Suggestion:** Either complete the implementation by adding logic to `builder.go` to build and configure the declarative orchestrator from a config file, or remove the `pipeline/declarative` package entirely to reduce codebase size and maintenance overhead. If kept, the builder would need to be ableto construct a `FlowController` (e.g., a Mangle engine) and dynamically build a `tools` map based on the config.
**Status:** Resolved

## Smell: Magic Strings for Execution Context
**Location:** `pipeline/declarative/orchestrator.go`
**Impact Analysis:** The declarative orchestrator uses a `map[string]any` as a property bag to pass state between stages, using string constants like `contextKeyQuery`. This pattern is fragile, error-prone, and lacks type safety. A typo in a key would lead to a runtime bug that the compiler cannot catch. It also makes the data flow difficult to follow and debug.
**Refactoring Suggestion:** Replace the `map[string]any` with a dedicated, typed struct, similar to `pipeline.PipelineContext` used by the `sandwich` orchestrator. This struct would have explicit, typed fields for `Query`, `Docs`, `Answer`, etc., ensuring compile-time safety and dramatically improving code clarity.
**Status:** Open

## Smell: Violation of Open/Closed Principle via Type Switch
**Location:** `pipeline/declarative/orchestrator.go` (function `dispatchToTool`)
**Impact Analysis:** The `dispatchToTool` function uses a large `switch tool.(type)` block to execute different components. This is a classic code smell. To add a new type of tool or component to the declarative pipeline, this central function must be modified, violating the Open/Closed Principle. This makes the system rigid and harder to extend.
**Refactoring Suggestion:** Define a common interface, such as `DeclarativeStage`, with an `Execute(ctx, execContext)` method. Each tool wrapper would implement this interface. The `dispatchToTool` function would then simply become a single call: `tool.Execute(ctx, execContext)`. This removes the type switch and allows new tool types to be added without modifying the core orchestrator logic.
**Status:** Open

## Smell: Dependency Injection Bypass
**Location:** `internal/providers/llm/openai.go`
**Impact Analysis:** The OpenAI and Groq provider factories create their own `openai.OpenAI` client instances directly, ignoring the `diapi.OpenAIClient` interface defined in the dependency injection layer. This bypasses the intended DI mechanism, which is designed to create a single, shared client instance in the builder and inject it into all providers that need it. The current implementation is inefficient as it creates multiple client objects where one would suffice.
**Refactoring Suggestion:** The `Builder` should be responsible for creating a single `diapi.OpenAIClient`. This client should be added to a dependency struct (e.g., `diapi.LLMDeps`). The OpenAI and Groq factories should then *receive* this client via their `deps` struct instead of creating their own.
**Status:** Open

## Smell: Hard-coded Dependencies in Factory
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:** The hybrid retriever's factory is hard-coded to build its "bm25" and "dense" sub-retrievers. This makes the component inflexible. It's impossible to configure a different set of sub-retrievers (e.g., two different dense retrievers, or a dense retriever and a graph retriever) without changing the factory's code. The composition of a component should be defined in configuration, not in code.
**Refactoring Suggestion:** The `retrieve.HybridOptions` struct should contain the *names* of the sub-retrievers to use (e.g., `SparseRetrieverName: "bm25"`, `DenseRetrieverName: "dense"`). The factory would then use the `deps.BuildSubRetriever` function to build the retrievers specified in the configuration, making the component fully composable.
**Status:** Open

## Smell: Hard-coded Magic Number
**Location:** `internal/providers/hybrid/hybrid.go` (function `Retrieve`)
**Impact Analysis:** The Reciprocal Rank Fusion (RRF) algorithm contains a hard-coded constant `k = 60.0`. This "magic number" is a critical tuning parameter for the fusion algorithm. Having it hard-coded prevents users from tuning the retriever's behavior for their specific use case without modifying the source code.
**Refactoring Suggestion:** Add a `RRF_K` field to the `retrieve.HybridOptions` struct with a sensible default value. The `Retrieve` method should use the value from the options struct instead of the hard-coded constant. This exposes the parameter for configuration and tuning.
**Status:** Open

## Smell: Deprecated Code Present in Core
**Location:** `core/types.go`
**Impact Analysis:** The `core.LocalvecOptions` struct is marked as deprecated, stating it has been moved to the provider's own package. Leaving deprecated code, especially in a `core` package, increases maintenance overhead and can confuse developers about which component to use. Core APIs should be clean and reflect the current state of the architecture.
**Refactoring Suggestion:** Remove the `core.LocalvecOptions` struct and update any code that still references it to use the new options struct from the `internal/vectorstores/localvec` package. This enforces the architectural principle that providers define their own options.
**Status:** Resolved