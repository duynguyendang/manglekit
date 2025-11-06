---
context_type: low_level_design
project: manglekit
language: go
version: 0.5.0
last_updated: 2025-11-06
stability: stable
audience: developers
---

# 1. Purpose & Scope

This document provides a detailed low-level design for the Manglekit SDK's core framework components. It covers the decentralized, handler-based Builder, the Registry, provider factories, the dependency injection mechanism, and the stage-based pipeline architecture. It is intended to be a technical reference for developers extending the framework or diagnosing its behavior.

# 2. Component Diagram

```mermaid
graph TD
    subgraph "Configuration Layer"
        A[YAML Config] --> B{sdk.FromConfig}
    end

    subgraph "Construction Layer"
        B --> C[builder.go]
        C --> D[registry.go]
        D -- Contains --> E[Component Handlers]
        D -- Contains --> F[Provider Factories]
    end

    subgraph "Core Contracts"
        G[core/interfaces.go]
        H[core/handler.go]
        I[core/diapi]
    end

    subgraph "Implementation Layer"
        E -- Implements --> H
        F -- Consumes --> I
        C -- Provides --> I
        J[internal/providers/*] --> E
        J -- Also provides --> F
    end

    subgraph "Execution Layer"
        K["pipeline/sandwich.go"]
        L["pipeline/declarative/orchestrator.go"]
        C --> K
        C --> L
    end

    G --> K
    G --> L
```

# 3. Builder Subsystem

The `Builder` is the central component for constructing an orchestrator. It follows a handler-based process that is decentralized and respects the Open/Closed Principle.

**Process Flow:**
1.  **Configuration:** The builder is configured programmatically via `With(opts)` calls. The `sdk.FromConfig` function translates YAML into these calls.
2.  **Component Grouping:** All configured components are grouped by their `core.Kind`.
3.  **Ordered Build:** The builder iterates through a hard-coded build order (`Embedder` -> `VectorStore` -> `Retriever`, etc.).
4.  **Handler Invocation:** For each component, it looks up the corresponding `core.ComponentHandler` in the `Registry`.
5.  **Delegated Build:** The builder calls the handler's `BuildComponent` method, passing itself as a dependency provider (`builderDI`), the component's factory, its configuration, and the map of already resolved components.
6.  **Component Construction:** The handler is responsible for creating the dependency struct, calling the factory, and placing the resulting component instance into the resolved map.
7.  **Orchestrator Creation:** After all components are built, the `Resolved` struct is assembled and passed to the selected orchestrator's factory.

```mermaid
sequenceDiagram
    participant User/Config
    participant Builder
    participant Registry
    participant ComponentHandler
    participant ProviderFactory

    User/Config->>+Builder: With(opts)
    Builder->>+Builder: Stores opts in cfgs list
    User/Config->>+Builder: Build(ctx)
    Builder->>+Builder: Groups cfgs by kind
    Builder->>+Builder: Iterates through build order...
    Builder->>+Registry: GetHandler(kind)
    Registry-->>-Builder: Returns handler
    Builder->>+Registry: Get(kind, name)
    Registry-->>-Builder: Returns factory
    Builder->>+ComponentHandler: BuildComponent(ctx, builder, factory, resolved, cfg, name)
    ComponentHandler->>+ProviderFactory: Build(ctx, deps, cfg)
    ProviderFactory-->>-ComponentHandler: Returns component instance
    ComponentHandler->>+Builder: Places instance in resolved map
    ComponentHandler-->>-Builder: Returns closer
    Builder->>-Builder: Repeats for all components...
    Builder-->>-User/Config: Returns final orchestrator
```

# 4. Factory Interface Layer

All component factories must adhere to the `core.Factory` interface.

```go
// core/factory.go
type Factory interface {
	Build(ctx context.Context, deps any, cfg any) (any, error)
}
```

This generic interface is made type-safe by the `ComponentHandler`, which is responsible for creating the specific, typed dependency (`diapi.*`) and configuration structs required by the factory.

```go
// internal/providers/retrievers/handler.go
func (h *Handler) BuildComponent(...) (core.ResourceCloser, error) {
    // The handler knows the specific types needed.
    b, _ := builderDI.(diapi.Builder)
    f, _ := factory.(core.Factory)

    // It constructs the typed dependency struct.
    deps := diapi.RetrieverDeps{
        Embedder:     b.GetEmbedder(...),
        VectorStore:  b.GetVectorStore(...),
    }

    // It calls the factory with the typed structs.
    built, err := f.Build(ctx, deps, cfg)
    // ...
}
```

# 5. Dependency Injection Layer

The builder implements the `diapi.Builder` interface, which exposes methods like `GetEmbedder(name)` and `GetVectorStore(name)`. This allows component handlers and factories to request specific, named dependencies.

*   `diapi.Builder`: The core DI interface, implemented by `manglekit.Builder`.
*   The handler for a given component is responsible for using the `diapi.Builder` to construct the correct dependency struct for its factory.

Circular dependencies are prevented by the hard-coded linear build order defined in `builder.go`.

# 6. Provider Family Details

### LLM: `openai`
*   **Handler:** `internal/providers/llm/handler.go`
*   **Factory Entrypoint:** `openai.New`
*   **Registered Key:** `openai`
*   **Config Struct:** `openai.Options`
*   **Dependencies:** `diapi.LLMDeps` (constructed by the handler).

### Retriever: `hybrid`
*   **Handler:** `internal/providers/retrievers/handler.go`
*   **Factory Entrypoint:** `hybrid.New`
*   **Registered Key:** `hybrid`
*   **Config Struct:** `hybrid.HybridOptions`
*   **Dependencies:** `diapi.RetrieverDeps` (constructed by the handler). The factory uses `deps.SubRetrievers` to access its dependencies.

# 7. Configuration Binding

Configuration from YAML is mapped to provider-specific `Options` structs using `mapstructure`. The `sdk.FromConfig` function looks up the `reflect.Type` of a provider's `Options` struct in the registry and uses it to decode the raw `map[string]any` from the YAML.

**YAML Example (`config.yaml`):**
```yaml
retrievers:
  - name: my_hybrid
    provider: hybrid
    options:
      retrievers: ["bm25", "dense"]
      rrf_k: 60.0
```

**Go Mapping:**
The loader finds the `hybrid.HybridOptions` type associated with the `hybrid` retriever, creates an instance of it, and `mapstructure` decodes the `options` map into the struct. This typed options object is then passed to `builder.With()`.

# 8. Lifecycle & Resource Management

Resource cleanup is handled via the `core.ResourceCloser` function type.

1.  A `ComponentHandler` is responsible for checking if a newly built component has a `Close(ctx) error` method.
2.  If it does, the handler returns the method as a `core.ResourceCloser`.
3.  The `Builder` collects all returned `ResourceCloser` functions.
4.  This list is passed to the final orchestrator inside the `core.Resolved` struct.
5.  The orchestrator's `Close` method iterates through these functions and executes them, ensuring graceful shutdown.

# 9. Logging & Observability Hooks

The `core.Observability` struct (logger, tracer, meter) is the central point for instrumentation. It is configured on the `Builder` and passed to the final orchestrator via the `core.Resolved` struct. The `Sandwich` orchestrator then passes the logger and meter to each of its pipeline stages.

# 10. Example Construction Path

Tracing the `hybrid` retriever:
1.  **Config:** YAML defines a retriever named `my_hybrid` with provider `hybrid`.
2.  **SDK Loader:** `sdk.FromConfig` finds the `hybrid.HybridOptions` type, decodes the YAML into it, and calls `builder.With("my_hybrid", hybrid.HybridOptions{...})`.
3.  **Build Process:**
    *   The `buildAll` method reaches `core.KindRetriever`.
    *   It gets the retriever `ComponentHandler` from the registry.
    *   It calls `handler.BuildComponent` for the `my_hybrid` component.
4.  **Handler Execution (Multiplexer):**
    *   The `retrievers.Handler` acts as a multiplexer. It performs a type switch on the provider's `Options` struct (`cfg`) to determine which dependency struct to build.
    *   For `hybrid.HybridOptions`, it constructs `diapi.RetrieverDeps`, resolving the sub-retrievers named in the config (e.g., `bm25`, `dense`) from the `resolved` map.
    *   For `dense.DenseOptions`, it would construct `diapi.DenseRetrieverDeps` instead.
5.  **Factory Execution:**
    *   The handler gets the `hybrid` factory from the registry.
    *   It calls `factory.Build(ctx, diapi.RetrieverDeps{...}, cfg)`.
    *   The factory correctly consumes the `diapi.RetrieverDeps` struct to access its sub-retrievers.
6.  **Instance:** The fully constructed `hybrid` retriever is returned to the handler, which places it in the `resolved.Retrievers` map.

# 11. Design Constraints & Guardrails

*   **No Global Singletons:** All component instances are managed by the builder and contained within the orchestrator.
*   **Stateless Factories & Handlers:** Provider factories and handlers should be stateless.
*   **Type-Safe DI:** The combination of `ComponentHandler` and `diapi` structs ensures that dependency injection is type-safe without runtime reflection.

# 12. Deviations & Blockers

The codebase is **stable** and has no open deviations from the LLD.

# 13. Changelog
*   **2025-11-06**: Verified code compliance with ADR-7 (R14). Reverted 'unstable' status. The system is stable and compliant with the LLD.
*   **2025-11-05**: Reverted stability status to **unstable**. Updated "Deviations & Blockers" to reflect that the "Builder Leaking into Handler" violation (ADR 7 / R14) is present in the codebase, which is a direct contradiction of the design specified in this document.
*   **2025-11-05**: Final baseline of all architectural documents to stable.
*   **2025-11-04**: Enforced ADR-7 (R14) compliance by refactoring the `declarative` and `sandwich` orchestrator providers to use typed `diapi.*Deps` structs, removing the final `diapi.Builder` violations. Rewrote orchestrator tests to use the modern YAML-based `sdk.LoadWithRegistry` pattern.
*   **2025-11-03**: Completed final architectural cleanup and document synchronization. Aligned this LLD with the stable, 100% complete state of the codebase.
*   **2025-11-03**: Reconciled LLD with the actual (in-progress) state of the DI refactor. The previous audit claim that all GAPs were resolved has been reverted, and the document is marked as unstable pending completion of the factory signature migration.
*   **2025-10-23**: Updated deviations to reflect current gaps (orchestrator handler coverage, hybrid factory signature, declarative state selection). Clarified hybrid construction path note.
*   **2025-10-20**: Regenerated LLD to reflect the decentralized, handler-based builder architecture. Updated diagrams and construction path to show the new flow. Synchronized deviations with the latest code review.
*   **2025-10-19**: Initial draft of the LLD.
