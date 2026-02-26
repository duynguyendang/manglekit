# ADR — Manglekit Architecture Masterplan

**Project:** Manglekit Core
**Status:** Live Document
**Version:** v1.2 (Cognitive Systems)
**Last Updated:** 2025-12-17

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
| **12** | [Unified Governance Gates](https://www.google.com/search?q=%23adr-12-unified-governance-gates) | Integration (v1.2) | Accepted | *Legacy distinct Assess/Validate* |
| **13** | [Semantic State Machine (The Loop)](https://www.google.com/search?q=%23adr-13-semantic-state-machine-the-loop) | Integration (v1.2) | Accepted | *Legacy linear execution* |
| **14** | [Native Structured Envelopes](https://www.google.com/search?q=%23adr-14-native-structured-envelopes) | Integration (v1.2) | Accepted | *Legacy JSON text parsing* |
| **15** | [Unified Persistent Silo (BadgerDB)](https://www.google.com/search?q=%23adr-15-unified-persistent-silo) | Convergence (v2.0) | Accepted | *In-memory volatile state* |
| **16** | [OODA Loop Cognitive Architecture](https://www.google.com/search?q=%23adr-16-ooda-loop-cognitive-architecture) | Convergence (v2.0) | Accepted | *Basic Semantic Control Loop* |
| **17** | [Tiered GenePool (Memoex)](https://www.google.com/search?q=%23adr-17-tiered-genepool-memoex) | Convergence (v2.0) | Accepted | *Flat Datalog blueprints* |
| **18** | [Zero-Trust Supervisor & Logic Contracts](https://www.google.com/search?q=%23adr-18-zero-trust-supervisor) | Convergence (v2.0) | Accepted | *Basic Guard wrapper* |
| **19** | [Source-to-Knowledge Pipeline](https://www.google.com/search?q=%23adr-19-source-to-knowledge-pipeline) | Convergence (v2.0) | Accepted | *External ingestion dependency* |

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

## V. The Integration (v1.2 - Cognitive Systems)

*This era focuses on the deep convergence of the Deterministic Engine and the Probabilistic LLM, enabling "Self-Correction" and "Structured Intelligence".*

### ADR 12: Unified Governance Gates

**(Standardizes Policy Engine Interaction)**

**Context:**
The internal logic for "Pre-Check" (`Assess`) and "Post-Check" (`Reflect`) was diverging. Pre-checks used `deny` predicates while Post-checks were ad-hoc. There was no consistent way to block an action based on Datalog results across different stages.
**Decision:**
Implement a **Unified Gate Mechanism** (`evaluateGate`):

1.  **Protocol:** Both input and output checks use the same underlying Datalog query logic.
2.  **Predicates:** Standardize on `halt(Entity, Reason)` (replacing `deny` and `infeasible`).
3.  **Flow:** Check `halt` first. If derived, return `AlignmentError`.
    **Rationale:**
    Reduces cognitive load by having a single "mental model" for how the Engine blocks actions. "Everything is a Gate."

### ADR 13: Semantic State Machine (The Loop)

**(The "Self-Correcting" Architecture)**

**Context:**
Simple linear execution (Input -> Action -> Output) is brittle for AI. Agents need to retry failed steps, route based on intent, or remember feedback.
**Decision:**
Transform the `sdk.Client` execution model into a **Semantic State Machine**:

1.  **State:** The `Envelope` + `FeedbackHistory`.
2.  **Transitions:** Driven by Engine "Steering" signals (`retry(Hint)`, `route(Target)`).
3.  **Loop:** A robust `runLoopInternal` that handles Context Injection, Execution, Evaluation, and Correction.
    **Rationale:**
    Moves Manglekit from a "Pipeline Runner" to an "Agentic Runtime". It enables the **Stochastic Runtime Paradox** by wrapping probabilistic actions in deterministic control loops.

### ADR 14: Native Structured Envelopes

**(Schema-First Generation)**

**Context:**
Relying on LLMs to output raw JSON text and then parsing it with Regex is flaky and slow. Modern providers (Genkit, OpenAI) support native schema enforcement.
**Decision:**
Enforce **Structured Generation** contracts:

1.  `ContentType: STRUCT` is the privileged mode.
2.  Adapters MUST support `OutputType` in `GenerationConfig`.
3.  The `Envelope` carries typed Go structs, not just `map[string]any`.
    **Rationale:**
    Type safety shouldn't stop at the LLM API. The Policy Engine requires structured facts; getting them directly from the LLM reduces translation errors.

-----

## VI. Appendix: Superseded Concepts (The Graveyard)

*The following concepts from the legacy architecture (v0.x) are deprecated and removed:*

  * **Generic Builder (`diapi`):** Removed. Replaced by `sdk.Client` composition.
  * **Component Handlers:** Removed. Replaced by `providers/` packages.
  * **DependencyResolver:** Removed. Replaced by FOP Adapters.
  * **Typed Resolved Structs:** Removed. The SDK now relies on dynamic `core.Envelope` passing rather than rigid compile-time structs for pipeline stages.

-----

## VII. The Convergence (v2.0 - Sovereign Kernel)

*This era marks the integration of the `meblo-wip` persistent storage and the `kronos-v1` OODA loop, transforming Manglekit from an embedded engine into a Sovereign Logic Kernel.*

### ADR 15: Unified Persistent Silo (BadgerDB)

**(Persistent Knowledge and Vectors)**

**Context:**
Manglekit primarily reasoned over transient context or externally managed databases via basic adapters. For true autonomy, the kernel needs to "remember" across sessions.
**Decision:**
Integrate `meb` (BadgerDB-backed SPOg quad datastore) as **The Silo**.
1. **Fact Storage:** Persist domain quad facts permanently.
2. **Vector Space:** Integrated SQ8-compressed vector storage.
3. **Dual Mode:** Support high-throughput ingestion and resource-constrained serving (`--readonly`, `--lowmem`).
**Rationale:**
Allows Manglekit to build and maintain its own contextual history (Long-term Memory) without relying entirely on the host application for state injection.

### ADR 16: OODA Loop Cognitive Architecture

**(Formalizing the Agent Lifecycle)**

**Context:**
The "Semantic State Machine" (ADR 13) handled local retries, but lacked a formal cognitive structure for complex agency.
**Decision:**
Adopt the **Observe -> Orient -> Decide -> Verify -> Act** loop from `kronos-v1`.
1. **Shadow Audit:** Explicitly verify proposed AI plans against Datalog rules *before* execution.
2. **Trace Rendering:** Output detailed markdown trace artifacts (`traces/plan_[id].md`) for every loop iteration.
**Rationale:**
Provides a strict, verifiable operational model that mathematically proves the safety of an agent's intended actions.

### ADR 17: Tiered GenePool (Memoex)

**(Trust-Based Policy Management)**

**Context:**
Blueprints were loaded as a flat set of Datalog rules, making it hard to manage kernel axioms vs. learned heuristics.
**Decision:**
Implement a tiered GenePool system:
1. **Tier 0 (Kernel):** Immutable axioms (system physics).
2. **Tier 1 (Admin):** Governance policies (user constraints).
3. **Tier 2 (AI):** Knowledge Induction (learned strategies parsed from Markdown).
**Rationale:**
Categorizes logic by trust level, ensuring AI-induced rules (Tier 2) can never override Kernel safety axioms (Tier 0).

### ADR 18: Zero-Trust Supervisor & Logic Contracts

**(The Final Mechanical Gate)**

**Context:**
The Guard layer (ADR 4) intercepted actions, but needed stronger declarative enforcement decoupled from Go code.
**Decision:**
Implement a **Zero-Trust Supervisor** at the ports layer using declarative **Logic Contracts**.
1. **Evaluator Enhancement:** Integrate stratified semi-naive evaluation and built-in math predicates.
2. **Failsafe:** The Supervisor acts as the final gate, blocking catastrophic actions even if the OODA generated plan seems sound, based purely on Datalog proof failures.
**Rationale:**
Moves ultimate security responsibility from procedural Go checks into mathematically provable declarative contracts.

### ADR 19: Source-to-Knowledge Pipeline

**(Autonomous Data Ingestion)**

**Context:**
Getting data into the system required developers to manually write reflection/adapter code for various document types.
**Decision:**
Integrate an end-to-end ingestion pipeline directly into the toolkit.
1. **Extractors:** Native support for parsing Markdown, code, and text into logical SPOg quad facts and embeddings simultaneously.
2. **Induction Loop:** Convert unstructured policy documents directly into Tier 2 Datalog rules.
**Rationale:**
Closes the loop on autonomy by allowing the system to read its own environment and learn from it without bespoke adapter development.