# ADR — Manglekit Architecture Masterplan

**Project:** Manglekit Core
**Status:** Live Document
**Version:** v1.1 (Consolidated)
**Last Updated:** 2025-12-15

## 1\. Architecture Roadmap (The Shift)

This table illustrates the evolution from the legacy "Builder/DI" pattern to the modern "Composition/Plugin" architecture.

| ID | Title | Era | Status | Replaces/Evolves |
|----|-------|-----|--------|------------------|
| **1** | [Config-First & Declarative Strategy](https://www.google.com/search?q=%23adr-1-config-first--declarative-strategy) | Foundation | Accepted | N/A |
| **2** | [Observability-Native Lifecycle](https://www.google.com/search?q=%23adr-2-observability-native-lifecycle) | Foundation | Accepted | N/A |
| **3** | [Universal Context Propagation](https://www.google.com/search?q=%23adr-3-universal-context-propagation) | Foundation | Accepted | N/A |
| **4** | [Composition-Based Governance Kernel](https://www.google.com/search?q=%23adr-4-composition-based-governance-kernel) | Renaissance (v1.0) | Accepted | *Old ADR 4 (Generic Builder)* |
| **5** | [Domain-Driven Core Abstractions](https://www.google.com/search?q=%23adr-5-domain-driven-core-abstractions) | Expansion (v1.1) | Accepted | *Legacy `core/*` structure* |
| **6** | [Hybrid "Low-Code" Gateway Pattern](https://www.google.com/search?q=%23adr-6-hybrid-low-code-gateway-pattern) | Expansion (v1.1) | Accepted | *Old ADR 10 (Dual-Path)* |
| **7** | [Decoupled Provider Registry (Plugins)](https://www.google.com/search?q=%23adr-7-decoupled-provider-registry-plugins) | Expansion (v1.1) | Accepted | *Old ADR 7 (Per-Kind Handlers)* |
| **8** | [Functional Options for Providers](https://www.google.com/search?q=%23adr-8-functional-options-for-providers) | Expansion (v1.1) | Accepted | *Old ADR 11 (DependencyResolver)* |
| **9** | [Strict Dependency Inversion in Governance](https://www.google.com/search?q=%23adr-9-strict-dependency-inversion-in-governance) | Expansion (v1.1) | Accepted | *N/A (Fix)* |
| **10** | [Architectural Boundaries Enforcement](https://www.google.com/search?q=%23adr-10-architectural-boundaries-enforcement) | Process | Accepted | *Old ADR 8 (Static Rules)* |
| **11** | [Dual-Tier Testing Strategy](https://www.google.com/search?q=%23adr-11-dual-tier-testing-strategy) | Process | Accepted | *Old ADR 6 (Testing)* |

-----

## I. The Foundation (Immutable Principles)

*These decisions established the bedrock of the system and remain active.*

### ADR 1: Config-First & Declarative Strategy

**Context:**
Early versions mixed runtime construction with ad-hoc configuration.
**Decision:**
Configuration (YAML/ENV) is a first-class citizen. System behavior must be definable declaratively.
**Rationale:**
Allows "Manglekit as a Service" deployments where artifacts are immutable.
**Consequences:**
The `config` package is a foundational dependency.

### ADR 2: Observability-Native Lifecycle

**Context:**
AI applications are non-deterministic; resource leaks (goroutines/connections) were common.
**Decision:**
Observability and Lifecycle are mandatory.

  * **Trace:** Every Action emits spans.
  * **Log:** Unified `core.Logger` via Context.
  * **Close:** Graceful shutdown via `ResourceCloser`.
    **Rationale:**
    "If it's not traced, it didn't happen." Essential for debugging AI policies.

### ADR 3: Universal Context Propagation

**Context:**
Request-scoped data (Trace ID, Cancellation) was lost across async boundaries.
**Decision:**
`context.Context` is mandatory as the first argument of every API.
**Rationale:**
Ensures distributed tracing continuity and proper timeout handling.

-----

## II. The Renaissance (v1.0 - The Kernel Shift)

*This era marks the move from a complex "Dependency Injection Framework" to a lightweight "Composition" model.*

### ADR 4: Composition-Based Governance Kernel

*(Supersedes Legacy ADR 4: Generic Type-Safe Registry & Builder)*

**Context:**
The v0.x architecture (Legacy ADR 4) used a massive "Generic Builder" and "DIAPI" to construct graphs. This created high cognitive load; adding a component required writing Handlers, Factories, and DI logic. The system was too rigid.
**Decision:**
**Abandon the monolithic Builder Pattern.** Adopt a **Composition Root** pattern.

1.  **Client:** A lightweight coordinator.
2.  **Guard:** A `Supervise(action)` method that wraps *any* implementation with governance (Trace -\> AuthZ -\> Exec -\> Validate).
3.  **Engine:** A dedicated Datalog runtime.
    **Rationale:**

<!-- end list -->

  * **Simplicity:** "Wrap, Don't Build." The framework governs objects instantiated by the user (or factory), rather than trying to be a "God Builder".
  * **Flexibility:** Users can bring any library (Genkit, LangChain) and simply wrap it.
    **Consequences:**
  * **Deleted:** `builder` package, `diapi` package.
  * **Introduced:** `sdk.Client`, `supervisor.SupervisedAction`.

-----

## III. The Expansion (v1.1 - Low Code & Modularity)

*This era introduces the "Gateway" capabilities to support YAML-driven bots without compromising the clean kernel.*

### ADR 5: Domain-Driven Core Abstractions

**Context:**
The `core` package became a "junk drawer" of mixed logic and interfaces (`action.go`, `memory.go`, `constants.go`), causing circular dependencies.
**Decision:**
Refactor `core` into 5 strict domain files containing **only Interfaces and Types**:

1.  `types.go`: Vocabulary (Constants, Envelope).
2.  `logic.go`: Execution Contracts (AI/Actions).
3.  `data.go`: Storage Contracts.
4.  `governance.go`: Policy Contracts.
5.  `infra.go`: Observability Contracts.
    **Rationale:**
    Creates a stable "Ubiquitous Language". Separates *Definitions* from *Implementations*.

### ADR 6: Hybrid "Low-Code" Gateway Pattern

*(Evolved from Legacy ADR 10: Dual-Path Build)*

**Context:**
ADR 4 optimized for "Pro-Code" (Go devs). However, Ops teams require defining Bot topology via `mangle.yaml` without writing code.
**Decision:**
Implement a **Dual-Mode Initialization Strategy**:

1.  **Pro-Code:** `sdk.NewClient()` + Manual `RegisterAction()`.
2.  **Low-Code:** `sdk.NewClientFromConfig()` which uses a **Factory Layer** (`sdk/config_loader.go`) to hydrate actions from YAML.
    **Rationale:**
    Balances flexibility for developers with ease-of-use for operators.
    **Consequences:**
    The SDK now contains logic to map YAML strings to Go structs via the Registry (ADR 7).

### ADR 7: Decoupled Provider Registry (Plugins)

*(Supersedes Legacy ADR 7: Per-Kind Handlers)*

**Context:**
To support ADR 6, the SDK initially hard-coded dependencies (e.g., `googleai`) in the loader. This violated OCP and bloated the SDK. Legacy ADR 7's "Handlers" were too complex and coupled to the Builder.
**Decision:**
Adopt a **Registry Pattern** with **External Providers**.

  * **SDK:** Exposes `sdk.RegisterProvider()`.
  * **Providers:** (`providers/google`, `providers/openai`) register themselves at runtime.
    **Rationale:**
  * **Zero-Bloat:** SDK core has no heavy AI dependencies.
  * **Extensibility:** Users can write custom providers without modifying SDK.
    **Consequences:**
    Users must explicitly import providers in `main.go` (e.g., `google.Register()`).

### ADR 8: Functional Options for Providers

*(Supersedes Legacy ADR 11: DependencyResolver)*

**Context:**
Legacy ADR 11 tried to resolve complex dependencies via a "Resolver" pattern. This was over-engineered. Providers just need flexible configuration.
**Decision:**
Use the **Functional Options Pattern (FOP)** for all Provider Constructors.

  * **Pattern:** `New(ctx, opts...)`
  * **Adapter:** Each provider implements a `NewGenerator` to map YAML to Options.
    **Rationale:**
  * **Type Safety:** Go developers get explicit options (`WithModel`).
  * **Cleanliness:** Separates the "YAML mapping" logic from the "Instance creation" logic.

### ADR 9: Strict Dependency Inversion in Governance

**Context:**
The `SupervisedAction` (App Layer) was casting `core.Evaluator` to `*engine.PolicyEngine` (Implementation) to access `GetActionConfig`, coupling the layers.
**Decision:**
Promote `GetActionConfig` to the `core.Evaluator` interface.
**Rationale:**
High-level modules must not depend on low-level modules. Enables mocking of config logic.

-----

## IV. The Guardrails (Process & Quality)

### ADR 10: Architectural Boundaries Enforcement

*(Replaces Legacy ADR 8)*
**Context:**
To maintain the integrity of ADR 5 & 7, we must prevent dependency leaking.
**Decision:**
Enforce strict import rules:

1.  **Core:** MUST NOT import `sdk`, `internal`, or `providers`.
2.  **SDK:** MUST NOT import `providers` or specific AI libraries.
3.  **Providers:** CAN import `sdk` (to register) and `core`.

### ADR 11: Dual-Tier Testing Strategy

*(Replaces Legacy ADR 6)*
**Context:**
With the Hybrid Gateway (ADR 6), unit tests are not enough.
**Decision:**
Adopt two tiers:

1.  **Unit Tests:** Test logic using Mocks (Pro-Code path).
2.  **Integration Tests:** Test the full pipeline by loading a real `mangle.yaml` (Low-Code path).

-----

## V. Appendix: Superseded Concepts (The Graveyard)

*The following concepts from the legacy architecture (v0.x) are deprecated and removed:*

  * **Generic Builder (`diapi`):** Removed. Replaced by `sdk.Client` composition.
  * **Component Handlers:** Removed. Replaced by `providers/` packages.
  * **DependencyResolver:** Removed. Replaced by FOP Adapters.
  * **Typed Resolved Structs:** Removed. The SDK now relies on dynamic `core.Envelope` passing rather than rigid compile-time structs for pipeline stages.