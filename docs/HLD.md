# Manglekit — High‑Level Design (HLD)

**Revision:** 3.0.0 (Genesis Edition)
**Scope:** Core Kernel Architecture, Universal Governance & Subsystems
**Audience:** Framework Architects, Enterprise Integrators, Platform Engineers
**Mission:** To provide a **Universal AI Governance Kernel** that transforms any atomic operation (LLM, Function, Tool) into a managed, observable, and policy-compliant unit of execution.

-----

## 1. Executive Summary

As AI systems evolve from simple chatbots to complex Agentic workflows, the primary challenge shifts from **connectivity** to **control**. Enterprises struggle to maintain visibility and safety when LLMs act autonomously or process sensitive data through multiple steps.

**Manglekit** addresses this by functioning as an **AI Operating System**.

Unlike traditional frameworks that act as simple wrappers, Manglekit separates **Management (Kernel)** from **Execution (Drivers)**. It treats the underlying execution engines—whether it's Google Genkit for AI, Vector DBs for memory, or native Go functions for business logic—as interchangeable **Drivers**.

Manglekit wraps these drivers in a strictly enforced **"Guarded Action"** lifecycle. This ensures that every operation, regardless of its nature, is automatically subject to authorization, deep observability, and deterministic policy enforcement before it ever executes.

-----

## 2. Goals & Non‑Goals

### 2.1 Goals

  * **Action-Centric Architecture:** Treat every operation (LLM generation, DB query, API call) as a uniform `core.Action` interface, eliminating the bias towards "RAG-only" pipelines.
  * **Neuro-Symbolic Native:** Implement a strict **"Perception Pipeline"** that converts unstructured AI outputs into structured Go types, and then into Datalog facts for deterministic reasoning.
  * **Zero-Config Reflection:** Provide a **Reflector 2.0** engine that automatically maps complex Go structs (nested maps, pointers, slices) to logic facts without manual boilerplate.
  * **Deep Observability:** Treat **Logic** as a first-class citizen in traces (Logical Spans) and automatically track data provenance (Lineage) across execution steps.
  * **Single Static Binary:** Compile the entire kernel, logic engine, and adapters into a single, dependency-free Go binary for easy deployment.

### 2.2 Non‑Goals

  * **Reinventing Drivers:** Manglekit does not build low-level clients for OpenAI or Anthropic. It relies on **Genkit** and other libraries as adapters to handle the raw I/O.
  * **UI Frameworks:** Manglekit is a backend kernel. It does not provide UI components or chat widgets.

-----

## 3. Architectural Principles

  * **Kernel & Driver Separation:** The **Kernel** (Manglekit) handles policy, routing, and state. The **Drivers** (Adapters) handle execution. The Kernel controls the Drivers; the Drivers never control the Kernel.
  * **The Guarded Action:** The atomic unit of the system is not a "Chain" or a "Flow", but a **Guarded Action**. This is a decorated operation that enforces the lifecycle of `Authorize -> Execute -> Validate`.
  * **Structure First:** Unstructured text is never fed directly into the Logic Engine. It must first be extracted into a **Go Struct** (Schema) to ensure type safety before reasoning.
  * **Compliance by Default:** The architecture is "Secure by Design". An Action cannot run unless it is explicitly wrapped and authorized by the Kernel.

-----

## 4. Component Taxonomy

Manglekit organizes the system into three distinct layers based on the Hexagonal Architecture.

### 4.1 The Kernel (The Brain)

  * **Engine:** The core runtime that manages the lifecycle of policies. It wraps the Google Mangle Datalog engine for high-performance in-memory reasoning.
  * **Reflector:** The translation layer that converts arbitrary Go memory states (Structs, Maps) into Datalog Atoms.
  * **LineageTracker:** A subsystem that automatically links Output IDs to Input IDs to construct a provenance graph.
  * **Router:** A logic-driven dispatcher that queries the Engine to determine the `NextAction` based on current facts.

### 4.2 The Guard (The Control Plane)

  * **GuardedAction:** The universal decorator. It intercepts the execution flow to inject Context, enforce Policy, and record Trace data.
  * **PolicyChecker:** The component responsible for executing `Authorize (Pre)` and `Validate (Post)` queries.

### 4.3 The Adapters (The Muscle)

  * **AI Adapter:** Wraps **Genkit** models (LLMs, Embedders) into the `core.Action` interface.
  * **Vector Adapter:** Wraps Vector Stores for retrieval operations.
  * **Func Adapter:** Wraps standard Go functions (Business Logic, Tools) using Go Generics for type safety.

-----

## 5. Architecture Overview

### 5.1 Hexagonal Design

Manglekit enforces a concentric design where the Logic Engine sits at the center, protected by the Guard, interacting with the world via Adapters.

```mermaid
graph TD
    subgraph "Application Layer"
        Config[YAML Config]
        UserCode[User Go Code]
    end

    subgraph "The Kernel (Center)"
        Logic[Logic Engine]
        Reflect[Reflector]
        Lineage[Lineage Tracker]
    end

    subgraph "The Guard (Middleware)"
        Guard[GuardedAction Decorator]
        Guard <--> Logic
    end

    subgraph "The Adapters (Ports)"
        AI[AI Adapter (Genkit)]
        Func[Func Adapter]
        Store[Vector Adapter]
        
        Guard --> AI
        Guard --> Func
        Guard --> Store
    end
    
    UserCode -->|Protect()| Guard
```

### 5.2 The Universal Action Interface

To achieve "Action-Centricity", Manglekit standardizes all operations on a single interface.

  * **The Contract:** `Execute(ctx context.Context, input core.Envelope) (core.Envelope, error)`
  * **The Envelope:** A standardized container holding the `UUID` (for lineage), `Payload` (the actual data), and `Metadata`.
  * **Benefit:** The Kernel does not need to know if it is governing an LLM or a Database. It simply governs the **Envelope flow**.

-----

## 6. Core Subsystems

### 6.1 The "Guarded Action" Lifecycle

This replaces the legacy "Sandwich" pattern. It describes the immutable sequence of events for every protected operation.

1.  **Trace Start:** A new OpenTelemetry Span is initialized.
2.  **Context Injection:** Trace IDs and Logger instances are injected into the Go Context.
3.  **Pre-Computation (Authorize):**
      * The Input Payload is **Reflected** into Datalog Facts.
      * The Engine queries `deny(Input)?`.
      * *Outcome:* If denied, execution halts immediately with a Policy Violation error.
4.  **Execution:** The inner Adapter (Genkit/Func) is executed.
5.  **Post-Computation (Validate):**
      * **Auto-Lineage:** The Kernel records `derived_from(OutputID, InputID)`.
      * The Output Payload is **Reflected** into Facts.
      * The Engine queries `violation(Output)?`.
      * *Outcome:* If violated, the result is discarded or redacted.
6.  **Trace End:** The Span is closed, and the Audit Log is written.

### 6.2 The Perception Pipeline (Neuro-to-Symbolic)

Manglekit solves the "Unstructured Data" problem via a rigorous 3-stage pipeline:

  * **Stage 1: Extraction (Neuro):** Use an LLM (via Genkit Adapter) to extract data from text into a JSON Schema.
  * **Stage 2: Structuring (Static):** Unmarshal the JSON into a **Go Struct**. This enforces strict Type Safety (e.g., ensuring an 'Age' field is an Integer, not a String).
  * **Stage 3: Reasoning (Symbolic):** The **Reflector** converts the Go Struct into Datalog Facts for the Logic Engine.

### 6.3 Logic-Driven Routing

Manglekit abandons hard-coded `if/else` orchestration.

  * **Mechanism:** The **Router** component evaluates the current state against the RuleSet.
  * **Query:** `next_step(ActionName) :- intent("buy"), user_tier("gold").`
  * **Result:** The logic engine dictates the next step dynamically, enabling **Hot-Reloadable Agent Behaviors**.

-----

## 7. Configuration & Construction

### 7.1 "Protect" API

The SDK creates a simplified Developer Experience (DX) focused on composition rather than complex builders.

  * **Mechanism:** Developers use a `kernel.Protect(name, rawAction)` factory method.
  * **Effect:** This method automatically wraps the raw action with the `GuardedAction` decorator and registers it with the Kernel.

### 7.2 Configuration-Driven Kernel

  * **Config Loading:** The Kernel initializes itself fully from a `config.yaml`, loading Policies, setting up Observability exporters, and configuring Adapters.
  * **Dynamic Registry:** Adapters are registered via a Factory pattern, allowing the Kernel to instantiate only the drivers required by the configuration (avoiding bloat).

-----

## 8. Observability & Audit

Manglekit provides **Deep Observability** that transcends simple logging.

  * **Logical Spans:** The Logic Engine's reasoning process (microsecond-level Datalog evaluation) is recorded as a distinct Span in the Trace timeline. This proves that governance does not introduce latency bottlenecks.
  * **Lineage Tracking:** By propagating IDs through the `context.Context`, Manglekit constructs a graph of data provenance, allowing audits to trace exactly which input document led to a specific LLM output.
  * **Evidence Logging:** When a request is blocked, the specific **Rule ID** and **Facts** that caused the denial are attached to the trace for debugging.

-----

## 9. Extensibility

### 9.1 Adapter Ecosystem

Developers can extend Manglekit by implementing the `core.Action` interface.

  * **Built-in:** `adapters/ai` (Genkit), `adapters/vector` (Stores), `adapters/func` (Go Native).
  * **Custom:** Any proprietary logic can be wrapped using `adapters.FromFunc[In, Out](fn)` to become a first-class citizen of the ecosystem.

### 9.2 Rule Extensions

Developers can define custom Datalog predicates and rules in `.dl` files, which are loaded by the Kernel at startup. This allows for domain-specific governance (e.g., Financial Rules, Medical Compliance) without modifying the binary.

-----

## 10. Deployment Model

  * **Single Static Binary:** Manglekit compiles into a highly efficient, dependency-free Go binary.
  * **Embeddable:** It can run as a library inside a larger Go application or as a standalone service.
  * **Air-Gap Ready:** With local Datalog reasoning and local Go functions, Manglekit requires no internet connection for its decision-making logic.

-----

## 11. Glossary

  * **Kernel:** The core logic and management layer of Manglekit.
  * **Driver:** An external execution engine (like Genkit) controlled by the Kernel.
  * **Guarded Action:** The atomic unit of execution; a decorator enforcing policy and observability.
  * **Reflector:** The engine component that maps Go memory states to Datalog facts.
  * **Envelope:** The standardized data container (ID + Payload + Metadata) used for internal communication.
  * **Lineage:** The historical graph of data provenance (Input -\> Output relationships).