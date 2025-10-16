---
context_type: architecture_standard
project: manglekit
language: go
version: "0.5.0"
last_updated: "2025-10-16"
stability: stable
audience: humans_and_agents
---

### 0. Implementation Snapshot (Current State)

The Manglekit SDK is a Go framework for building rule-augmented RAG pipelines. Its current implementation is centered around a fluent builder, a factory-based component model, and a central registry.

-   **Configuration (`config.go`)**: Initialization is handled by `NewBuilderFromYAML` and `NewBuilderFromEnv`. These functions are tightly coupled to the builder, responsible for both parsing configuration (from YAML/env vars) and directly invoking builder methods. This violates SRP and involves significant duplicated logic.
-   **Builder (`builder.go`)**: A fluent `Builder` provides `With...` methods for programmatic configuration. On `Build()`, it resolves components by looking up factories in the registry and injecting dependencies (e.g., `Embedder`, `VectorStore`) via strongly-typed `diapi` structs. It correctly uses the Interface Segregation Principle for sub-builders (`SubRetrieverBuilder`) but suffers from an inconsistent public API (`WithEmbedder` behaves differently) and contains dead code.
-   **Registry (`registry.go`)**: The `Registry` is an instance-based catalog of component factories. All component types (LLMs, Retrievers, etc.) have their own typed factory map. It is now free of global state. A notable flaw is the `ClientFactories` map, which uses `map[string]any` and an inconsistent factory signature, creating a type-safety hole.
-   **Pipelines**: The primary orchestrator is the `Sandwich` pipeline (`pipeline/sandwich.go`). Its `Execute` method is a "god method" that hard-codes the sequence of operations (pre-rules, retrieve, rerank, LLM, post-rules) and uses brittle "magic strings" for passing metadata. A `Declarative` orchestrator (`pipeline/declarative/orchestrator.go`) exists but is currently a disabled stub.
-   **Providers (`internal/providers`)**: Component implementations (e.g., `bm25`, `google`, `openai`) reside in `internal/providers`. Their factories receive dependencies and options, create an instance, and return it along with an optional `ResourceCloser`.

### 1. Architectural Overview

Manglekit is designed as a modular, extensible framework for building neuro-symbolic AI applications, with a focus on verifiable, rule-based RAG. The architecture enables developers to compose pipelines from a set of pluggable components (Retrievers, LLMs, etc.) configured programmatically, via YAML, or environment variables.

```mermaid
graph TD
    subgraph "Configuration Phase"
        A[config.yaml / Env Vars] -->|Parsed by| B(NewBuilderFromYAML/Env);
        B --> C{Builder};
        P[Programmatic Code] -->|Calls With...()| C;
    end

    subgraph "Build Phase"
        C -- Build() calls --> D[Registry];
        D -- Returns Factory --> E(Component Factory);
        C -- Provides Deps (diapi) --> E;
        E -- Creates --> F[Provider Instance];
        C -- Collects --> G[ResourceClosers];
    end

    subgraph "Runtime Phase"
        H[Orchestrator] -- Contains --> F;
        H -- Contains --> G;
        I[Application] -- Calls Execute() --> H;
        J[Application] -- Calls Close() --> H;
    end

    F --> H;
    G --> H;
```

### 2. Dependency Rules (Non-Negotiable)

| Package                       | Allowed Dependencies                                     | Forbidden Dependencies                               | Rationale                                                                |
| ----------------------------- | -------------------------------------------------------- | ---------------------------------------------------- | ------------------------------------------------------------------------ |
| `core`                        | Go standard library                                      | All other project packages                           | Must be the foundational, dependency-free base.                          |
| `retrieve`, `llm`, `rerank`... | `core`                                                   | `builder`, `config`, `pipeline`, `internal/providers` | Defines component contracts (interfaces, options); must not know about implementations. |
| `internal/providers/*`        | `core`, its corresponding contract package (e.g., `llm`) | `builder`, `config`, other `internal/providers`      | Implementations depend on contracts, not the builder or other concrete providers. |
| `pipeline`                    | `core`, contract packages (`retrieve`, `llm`, etc.)      | `builder`, `config`, `internal/providers`            | Orchestrates contracts, but does not build them.                         |
| `builder`                     | `core`, contract packages, `internal/providers`, `diapi` | `config`                                             | The assembler; it is allowed to know about concrete types for building.  |
| `config`                      | `core`, `builder`, `registry`                            | N/A (but its dependency on `builder` is a code smell) | Currently coupled to the builder, which needs to be refactored.          |

**Key Rule**: There must be **no import cycles**. Using `any` in `core` to break cycles is a major anti-pattern.

### 3. Core Contracts

-   **Builder (`BuilderAPI`)**: The primary responsibility of the builder is to collect configuration for components and orchestrate their construction. It must not contain business logic. Its `Build(ctx)` method is the terminal operation that uses the registry and factories to assemble the final `Orchestrator`.
-   **Registry (`Registry`)**: The registry acts as a service locator for component factories. It maps a string name (e.g., `"openai"`) to a strongly-typed factory function. It is responsible for providing the correct factory but not for invoking it.
-   **Factory (e.g., `retrieve.Factory`)**: A factory is a function that creates a component instance. Its signature is `func(ctx context.Context, deps DEPS_STRUCT, opts any) (INTERFACE, core.ResourceCloser, error)`. It is responsible for checking for its required dependencies in the `DEPS_STRUCT` and returning a clear error if they are missing or of the wrong type.

### 4. Provider Composition

Composition of providers (e.g., a complex retriever that uses other retrievers) must occur inside the parent provider's factory. The parent's factory receives a `SubRetrieverBuilder` in its dependency struct, which it can use to build its children.

**Example**: The `hybrid` retriever factory receives `diapi.RetrieverDeps{ BuildSubRetriever: b.BuildRetriever }`. It then calls `deps.BuildSubRetriever(ctx, "bm25", ...)` to build its BM25 child. This prevents the factory from needing a dependency on the full `BuilderAPI`.

### 5. Configuration Flow

Configuration is mapped to factory calls. A YAML snippet is translated into a series of builder and factory invocations.

**YAML Example**:
```yaml
retriever:
  name: "hybrid"
  params:
    k: 60.0
    bm25:
      path: "corpus/"
    dense:
      model: "text-embedding-3-small"
```
1.  `NewBuilderFromYAML` reads the `retriever` section.
2.  It calls `builder.WithRetriever()` with a populated `retrieve.HybridOptions` struct.
3.  During `Build()`, the builder gets the `hybrid` factory from the registry.
4.  The factory receives the `HybridOptions` and uses the `bm25` and `dense` sub-structs to configure and build its child retrievers via the `SubRetrieverBuilder`.

### 6. Observability & Resource Lifecycle

-   **Observability**: An `core.Observability` struct containing a `Logger`, `Tracer`, and `Meter` can be passed to the builder via `WithObservability()`. This is then passed to components via `core.Options`.
-   **Resource Lifecycle**: Components that manage external resources (like API clients) must have a `Close(ctx) error` method. Their factory must return this method as a `core.ResourceCloser`. The builder collects all closers, and the `Orchestrator.Close()` method invokes them in reverse order of creation.

### 7. Error & Metric Surfaces

-   **Sentinel Errors**:
    -   `core.ErrInvalidOptions`: Invalid configuration.
    -   `core.ErrNoEvidence`: Retriever found no documents.
    -   `core.ErrDenied`: A rule explicitly blocked the request.
-   **Metrics (emitted by Sandwich pipeline)**:
    -   `manglekit.rules_pre_ms`
    -   `manglekit.retrieve_ms`
    -   `manglekit.rerank_ms`
    -   `manglekit.llm_ms`
    -   `manglekit.rules_post_ms`

### 8. Testing & Replaceability

Components should be tested in isolation using mock dependencies. The builder and registry facilitate this.

**Example Unit Test**:
```go
func TestMyOrchestrator(t *testing.T) {
    // 1. Use a real registry but register mock providers
    reg := manglekit.NewRegistry()
    err := providers.NewSet(providers.WithMockLLM(&mock.LLMOptions{})).Register(reg)
    require.NoError(t, err)

    // 2. Use the fluent builder with mock options
    builder := manglekit.NewBuilder(reg)
    builder.WithLLM(&mock.LLMOptions{ExpectedResponse: "mocked"})

    // 3. Build and test
    orch, err := builder.Build(context.Background())
    require.NoError(t, err)

    answer, err := orch.Execute(context.Background(), "s1", core.Query{Text: "test"})
    assert.Equal(t, "mocked", answer.Text)
}
```

### 9. Anti-Patterns (Red Lines)

-   **Dependency on Builder**: A component factory must **never** take a dependency on the `BuilderAPI`. Use a narrowly-scoped interface like `SubRetrieverBuilder` instead.
-   **Type Erasure**: Using `any` in core interfaces (`core.Orchestrator`) or options structs (`core.Options`) to break import cycles is forbidden. This is a symptom of incorrect package boundaries that must be fixed.
-   **Provider Branching**: Logic like `if provider.Name == "google"` inside the framework is forbidden. The factory pattern should be used to encapsulate provider-specific logic.
-   **Global State**: The registry and all components must be fully encapsulated in instances. No `init()`-based registration or global variables.
-   **Modifying `config.go`**: Adding a new provider must not require changes to `config.go`. The configuration loading should be generic.

### 10. Known Gaps

This table summarizes open architectural issues identified in the latest code review.

| Severity | Issue                                  | File(s)                                 | Description                                                                                             |
| :------- | :------------------------------------- | :-------------------------------------- | :------------------------------------------------------------------------------------------------------ |
| High     | **Tight Coupling in Configuration**    | `config.go`                             | `NewBuilderFromYAML` and `NewBuilderFromEnv` are coupled to the builder, violating SRP and duplicating logic. |
| High     | **Interface Pollution & Type Safety**  | `core/types.go`                         | The `Orchestrator` interface and `Options` struct use `any` to break import cycles, sacrificing compile-time safety. |
| High     | **God Method & Magic Strings**         | `pipeline/sandwich.go`                  | The `Execute` method has too many responsibilities (SRP violation), and the code uses brittle string literals for metadata keys. |
| Medium   | **Inconsistent Factory Signatures**    | `registry.go`                           | The `ClientFactories` map uses `any` and a custom signature, creating a type-safety hole in the registry. |
| Medium   | **Inconsistent Builder API**           | `builder.go`                            | The `With...` methods are inconsistent (some accept instances, others don't), and the client injection mechanism is unusable. |
| Medium   | **Hybrid Retriever Configuration Flaw** | `internal/providers/hybrid/hybrid.go`   | A key algorithm parameter is hard-coded, and it's impossible to configure the sub-retrievers.             |
| Low      | **Dead Code**                          | `builder.go`                            | The codebase contains unused variables (`embedderAlias`).                                               |

### 11. Provider Families

| Type            | Registered Providers        |
| :-------------- | :-------------------------- |
| **LLM**         | `google`, `openai`          |
| **Embedder**    | `google`, `openai`          |
| **Retriever**   | `bm25`, `dense`, `hybrid`, `in-memory` |
| **Reranker**    | `cosine`                    |
| **VectorStore** | `localvec`                  |
| **StateProvider**| `in-memory`, `redis`       |
| **RuleSet**     | `mangle`                    |

### 12. Versioning & Compatibility Policy

The project adheres to Semantic Versioning 2.0.0.
-   **MAJOR** version bump for incompatible API changes to public interfaces (e.g., `BuilderAPI`, `core.Orchestrator`).
-   **MINOR** version bump for adding functionality in a backward-compatible manner or changing provider option structs.
-   **PATCH** version bump for backward-compatible bug fixes.

This `context.md` document must be updated to reflect any MINOR or MAJOR changes.

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
    "orchestrator"
  ],
  "factories": {
    "retriever": {
      "bm25": {
        "options_type": "retrieve.BM25Options",
        "deps_type": "diapi.RetrieverDeps"
      },
      "dense": {
        "options_type": "retrieve.DenseOptions",
        "deps_type": "diapi.RetrieverDeps"
      },
      "hybrid": {
        "options_type": "retrieve.HybridOptions",
        "deps_type": "diapi.RetrieverDeps"
      }
    },
    "llm": {
      "google": {
        "options_type": "llm.GoogleOptions",
        "deps_type": "diapi.LLMDeps"
      },
      "openai": {
        "options_type": "llm.OpenAIOptions",
        "deps_type": "diapi.LLMDeps"
      }
    }
  },
  "registry_keys": [
    "google",
    "openai",
    "bm25",
    "dense",
    "hybrid",
    "in-memory",
    "cosine",
    "localvec",
    "redis",
    "mangle",
    "sandwich",
    "declarative"
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

-   **2025-10-16**: Performed a full, deep-dive code review. Regenerated `context.md` to the canonical "Live Standard" format. Updated the implementation snapshot, dependency rules, and all other sections to match the current codebase reality. Synchronized the "Known Gaps" section with the findings in `docs/code-review.md`. Added a machine-readable JSON appendix.
