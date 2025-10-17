---
context_type: architecture_standard
project: manglekit
language: go
version: "0.8.0"
last_updated: "2025-10-16"
stability: stable
audience: humans_and_agents
---

### 0. Implementation Snapshot (Current State)

The Manglekit SDK's architecture is based on a decoupled configuration system, a fluent builder, and a central registry for dependency injection. However, the implementation currently suffers from significant code duplication and a reliance on runtime type checking.

-   **Configuration (`config` package)**: All configuration loading and schema definition is encapsulated within the dedicated `config` package. It has no knowledge of the builder.
-   **Bridge (`from_config.go`)**: The `NewBuilderFromConfig` function translates a validated `config.Config` struct into a series of fluent builder calls.
-   **Builder (`builder.go`)**: A fluent `Builder` provides a repetitive `With...` API for programmatic component configuration. Each component type has its own `With<Component>` and `build<Component>` method, leading to significant boilerplate. The builder populates a `core.Options` struct where components are stored as `any`.
-   **Registry (`registry.go`)**: An instance-based catalog that uses separate, strongly-typed maps for each component factory type (e.g., `map[string]RetrieverFactory`, `map[string]LLMFactory`). This rigid structure forces the builder's repetitive design.
-   **Pipelines**:
    -   **`Sandwich` (`pipeline/sandwich.go`)**: The default orchestrator. Its constructor receives the `core.Options` struct and uses runtime type assertions to extract the components it needs, making it vulnerable to runtime panics if types do not match.
    -   **`Declarative` (`pipeline/declarative/orchestrator.go`)**: Exists but is currently a disabled stub.

### 1. Architectural Overview

Manglekit is a modular Go framework for building verifiable, rule-based RAG applications. The architecture separates configuration from construction, allowing developers to compose pipelines from pluggable components. The current implementation relies heavily on runtime checks and contains significant boilerplate that hinders extensibility.

```mermaid
graph TD
    subgraph "Configuration Phase"
        A[config.yaml / Env Vars] -->|Loaded by| B(config Package);
        B --> C[config.Config Struct];
        C -->|Translated by| D(NewBuilderFromConfig);
    end

    subgraph "Build Phase"
        P[Programmatic Code] -->|Calls With...()| E{Builder};
        D -->|Calls With...()| E;
        E -- build() calls --> F[Registry];
        F -- Returns Factory --> G(Component Factory);
        G -- Creates --> H[Provider Instance as any];
        E -- Populates --> I[core.Options with any];
    end

    subgraph "Runtime Phase"
        I -->|Passed to| J[Orchestrator Factory];
        J -- Performs runtime type assertions --> K[Orchestrator];
        L[Application] -- Calls Execute() --> K;
    end
```

### 2. Dependency Rules (Non-Negotiable)

| Package                       | Allowed Dependencies                                     | Forbidden Dependencies                               | Rationale                                                                |
| ----------------------------- | -------------------------------------------------------- | ---------------------------------------------------- | ------------------------------------------------------------------------ |
| `core`                        | Go standard library                                      | All other project packages                           | Must be the foundational, dependency-free base.                          |
| `config`                      | Go standard library                                      | `builder`, `pipeline`, `internal/providers`          | Configuration must be independent of the application logic.              |
| `retrieve`, `llm`, `rerank`... | `core`                                                   | `builder`, `config`, `pipeline`, `internal/providers` | Defines component contracts; must not know about implementations. |
| `internal/providers/*`        | `core`, its corresponding contract package (e.g., `llm`) | `builder`, `config`, other `internal/providers`      | Implementations depend on contracts, not the builder. |
| `pipeline`                    | `core`, contract packages (`retrieve`, `llm`, etc.)      | `builder`, `config`, `internal/providers`            | Orchestrates contracts, but does not build them.                         |
| `manglekit` (root)            | All other packages                                       | N/A                                                  | The root package acts as the assembler.                                  |

**Key Rule**: There must be **no import cycles**.

### 3. Core Contracts

-   **Builder (`BuilderAPI`)**: The builder's responsibility is to collect configuration for components and orchestrate their construction via factories.
-   **Orchestrator (`core.Orchestrator`)**: A pure behavioral interface (`Execute`, `Close`).
-   **Registry (`Registry`)**: A service locator for component factories, mapping string names to factory functions.
-   **Factory (e.g., `retrieve.Factory`)**: A function that creates a component instance, receiving dependencies via a typed `diapi` struct.

### 4. Provider Composition

Composition of providers (e.g., a hybrid retriever) occurs inside the parent provider's factory, which receives a `SubRetrieverBuilder` to build its children.

### 5. Configuration Flow

`config.Load()` parses YAML into a `config.Config` struct. `NewBuilderFromConfig` receives this struct, looks up provider option types in the registry, unmarshals the options, and calls the corresponding `builder.With...` method.

### 6. Observability & Resource Lifecycle

-   **Observability**: An `core.Observability` struct can be passed to the builder via `WithObservability()`.
-   **Resource Lifecycle**: Component factories can return a `core.ResourceCloser`. The builder collects all closers, and `Orchestrator.Close()` invokes them in reverse order.

### 7. Error & Metric Surfaces

| Type    | Name                        | Description                                          |
| :------ | :-------------------------- | :--------------------------------------------------- |
| Error   | `core.ErrInvalidOptions`    | Invalid or missing options during initialization.    |
| Error   | `core.ErrNoEvidence`        | Retriever found no documents for the query.          |
| Error   | `core.ErrDenied`            | A Mangle rule explicitly denied the request.         |
| Metric  | `manglekit.rules_pre_ms`    | Latency of the pre-retrieval rule evaluation stage.  |
| Metric  | `manglekit.retrieve_ms`     | Latency of the document retrieval stage.             |
| Metric  | `manglekit.rerank_ms`       | Latency of the document reranking stage.             |
| Metric  | `manglekit.llm_ms`          | Latency of the final LLM generation stage.           |
| Metric  | `manglekit.rules_post_ms`   | Latency of the post-generation rule evaluation stage.|

### 8. Testing & Replaceability

Components should be tested in isolation using mock dependencies registered with a test-specific registry.

### 9. Anti-Patterns (Red Lines)

-   **Dependency on Builder**: A component factory must **never** take a dependency on the `BuilderAPI`.
-   **Type Erasure**: Using `any` in core interfaces or for factory registries is forbidden. (VIOLATION)
-   **Provider Branching**: Logic like `if provider.Name == "google"` inside the framework is forbidden.
-   **Global State**: The registry and all components must be fully encapsulated in instances.
-   **Code Duplication**: The DRY (Don't Repeat Yourself) principle should be followed. (VIOLATION)

### 10. Known Gaps

This table summarizes open architectural issues identified in the latest code review. These items represent deviations from the architectural standard.

| Severity | Issue                                  | File(s)                                                              | Description                                                                                                                              | Status |
| :------- | :------------------------------------- | :------------------------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------- | :----- |
| High     | **Repetitive, Non-Generic Builder Logic** | `builder.go`                                                         | The builder contains significant code duplication in its `With...` and `build...` methods, making it hard to maintain and extend.          | Open   |
| High     | **Type Erasure via `core.Options`**       | `core/types.go`, `pipeline/sandwich.go`                                | Components are stored as `any` and extracted with runtime type assertions, moving type checking from compile-time to runtime.             | Open   |
| Medium   | **Rigid, Type-Specific Registries**       | `registry.go`                                                        | The registry uses separate, hard-coded maps for each component factory type, preventing easy extension of the framework with new types. | Open   |

### 11. Provider Families

| Type            | Registered Providers        |
| :-------------- | :-------------------------- |
| **LLM**         | `google`, `openai`, `mock-llm` |
| **Embedder**    | `google`, `openai`, `mock-embedder` |
| **Retriever**   | `bm25`, `dense`, `hybrid`, `in-memory` |
| **Reranker**    | `cosine`                    |
| **VectorStore** | `localvec`                  |
| **StateProvider**| `in-memory`, `redis`       |
| **RuleSet**     | `mangle`                    |
| **Orchestrator**| `sandwich`, `declarative`   |
| **SchemaParser**| `jsonschema`, `rdf`         |
| **FactConverter**| `mangle`                    |

### 12. Versioning & Compatibility Policy

The project adheres to Semantic Versioning 2.0.0. This `CONTEXT.md` document must be updated to reflect any MINOR or MAJOR changes.

### 13. Machine Appendix (JSON Snapshot v1)

```json
{
  "version": "1",
  "capabilities": [
    "llm",
    "embedder",
    "retriever",
    "reranker",
    "vectorstore",
    "stateprovider",
    "ruleset",
    "orchestrator",
    "schemaparser",
    "factconverter"
  ],
  "factories": {
    "retriever": {
      "bm25": { "options_type": "retrieve.BM25Options", "deps_type": "diapi.RetrieverDeps" },
      "dense": { "options_type": "retrieve.DenseOptions", "deps_type": "diapi.RetrieverDeps" },
      "hybrid": { "options_type": "retrieve.HybridOptions", "deps_type": "diapi.RetrieverDeps" },
      "in-memory": { "options_type": "retrieve.InMemoryRetrieverOptions", "deps_type": "diapi.RetrieverDeps" }
    },
    "llm": {
      "google": { "options_type": "llm.GoogleOptions", "deps_type": "diapi.LLMDeps" },
      "openai": { "options_type": "llm.OpenAIOptions", "deps_type": "diapi.LLMDeps" }
    },
    "orchestrator": {
      "sandwich": { "deps_type": "core.Options" },
      "declarative": { "deps_type": "core.Options" }
    }
  },
  "registry_keys": [
    "google", "openai", "mock-llm", "mock-embedder",
    "bm25", "dense", "hybrid", "in-memory",
    "cosine", "localvec", "redis", "mangle",
    "sandwich", "declarative", "jsonschema", "rdf"
  ],
  "metrics": [
    "manglekit.rules_pre_ms",
    "manglekit.retrieve_ms",
    "manglekit.rerank_ms",
    "manglekit.llm_ms",
    "manglekit.rules_post_ms"
  ],
  "errors": [
    "core.ErrInvalidOptions",
    "core.ErrNoEvidence",
    "core.ErrDenied"
  ]
}
```

### 14. Changelog

-   **2025-10-16**: Performed a deep-dive code review and updated documentation to reflect the current architectural state. Identified three major architectural smells: repetitive builder logic, type erasure in the core options struct, and rigid, type-specific registries. No code was modified. Updated `docs/code-review.md` and regenerated `docs/CONTEXT.md` to align with these findings. Updated version to 0.8.0.
-   **2025-10-16 (previous)**: Regenerated `CONTEXT.md` to the canonical "Live Standard" format. Updated the implementation snapshot, dependency rules, and all other sections to match the current codebase reality, reflecting the new decoupled configuration. Synchronized the "Known Gaps" section with the findings in the new `docs/code-review.md`. Added a machine-readable JSON appendix.
-   **2025-10-13 (initial)**: Initial version of the context document.