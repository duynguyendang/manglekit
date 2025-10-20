# Manglekit SDK - Code Review

## Smell: Hard-Coded Default Orchestrator
**Location:** `builder.go` (in `Build` method)
**Impact Analysis:** The builder defaults to the `"sandwich"` orchestrator if no explicit choice is made. This creates a tight coupling between the core builder and a specific implementation, making the framework less modular. If the "sandwich" provider were ever removed or renamed, this would cause a runtime failure that is not obvious from the configuration.
**Refactoring Suggestion:** Remove the hard-coded default. Require the orchestrator to be explicitly specified via `WithOrchestrator(name)` or in the configuration file. The `Build` method should return an error if no orchestrator is chosen, forcing a conscious decision by the user.
**Status:** Open

## Smell: Arbitrary Component Selection in Sandwich Orchestrator
**Location:** `pipeline/sandwich.go` (in `NewSandwich` factory)
**Impact Analysis:** The `NewSandwich` factory iterates through the maps of resolved components (e.g., `deps.Retrievers`, `deps.LLMs`) and picks the *first one* it finds. This behavior is non-deterministic, as map iteration order in Go is not guaranteed. A user configuring multiple retrievers or LLMs has no control over which one the primary orchestrator will use, leading to unpredictable behavior.
**Refactoring Suggestion:** The `Sandwich` orchestrator should be configurable. Introduce a `SandwichOptions` struct that allows the user to specify the names of the components it should use (e.g., `Retriever: "bm25"`, `LLM: "openai"`). The factory would then look up these specific components in the `deps` maps.
**Status:** Open

## Smell: Redundant `WithKind` Method in Builder API
**Location:** `builder.go`
**Impact Analysis:** The `BuilderAPI` has two methods for adding components: `With(opts any)` and `WithKind(kind, name, opts)`. The `With` method is type-safe and looks up the kind and name in the registry, which is the preferred, modern pattern. The `WithKind` method is a remnant of the config-file-driven approach and adds complexity. It forces the builder to support two parallel configuration paths, increasing the surface area for bugs.
**Refactoring Suggestion:** Deprecate and remove the `WithKind` method. The config loader (`from_config.go`) should be refactored to use the registry to look up the provider's options type from its name and kind. It can then instantiate that type and unmarshal the config options into it, before passing the typed options struct to the builder's `With` method. This makes `With` the single, canonical way to add a component.
**Status:** Open

## Smell: Monolithic `specTable` in Builder
**Location:** `builder.go`
**Impact Analysis:** The `specTable` function defines the entire build process for every component type in a single, large map. While data-driven, this centralizes all component-specific logic (dependency creation, assignment, resource closing) into the main builder. This violates the Open/Closed Principle; adding a new component kind requires modifying the builder itself.
**Refactoring Suggestion:** Abstract the `compSpec` into an interface, `ComponentHandler` or similar. Each provider package could optionally register a handler for its kind. The builder would iterate through registered handlers instead of a hard-coded table. This would decentralize the build logic and make the framework more extensible.
**Status:** Open

## Smell: Implicit Dependency on Last-Built Component
**Location:** `builder.go`
**Impact Analysis:** During the build process, the builder stores the "last-built" component of each kind (e.g., `b.embedder`, `b.vectorStore`) and injects it as a dependency for subsequent components. This is fragile. The correctness of the dependency injection relies entirely on the order of `With` calls or the order in the config file. It prevents the creation of parallel, independent pipelines that might use different embedders or vector stores.
**Refactoring Suggestion:** Components should declare their dependencies by name. For example, a `RetrieverOptions` struct should have a field like `Embedder: "openai-embedder"`. The builder would then be responsible for resolving this dependency by looking up the named embedder from its map of built components. This makes the dependency graph explicit and configuration-driven.
**Status:** Open

## Smell: Hard-Coded TopK in Tool Adapters
**Location:** `core/tool_adapters.go`
**Impact Analysis:** The `RetrieverTool` and `RerankerTool` adapters have a hard-coded `TopK` value of 10. This prevents the declarative orchestrator from being able to configure this crucial parameter, limiting its flexibility. The behavior of the pipeline is fixed at compile time.
**Refactoring Suggestion:** The `ToolStepConfig` in `pipeline/declarative/orchestrator.go` should be extended to include a generic `Params map[string]any` field. The tool adapters' `Execute` methods should look for relevant parameters (like `topK`) in this map within the `ExecutionContext` and use them if present, falling back to a default otherwise. This allows `TopK` to be configured per-step in the YAML file.
**Status:** Open
