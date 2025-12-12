---
context_type: low_level_design
project: manglekit
language: go
version: 1.0
last_updated: 2025-12-11
stability: implementation
audience: developers
---

# 1. Purpose & Scope

This document provides the Low-Level Design (LLD) for **Manglekit v2.0**. It details the implementation of the **Neuro-Symbolic Kernel**, focusing on the `Client`, `Supervisor`, `Engine`, and `Adapter` subsystems.

This version introduces the **"Headless Kernel"** architecture:
* **Explicit Initialization:** User wires dependencies (Genkit, Models) in `main.go`.
* **Clean Adapters:** Core SDK depends only on interfaces, not specific provider plugins.
* **Active Supervision:** Renaming "Guard" to "Supervisor" to reflect active orchestration (Steering/Correction).

# 2. Component Diagram

```mermaid
graph TD
    subgraph "User Space (main.go)"
        UserCode[User Application]
        GenkitInit[Genkit Init & Model]
    end

    subgraph "Manglekit Kernel (sdk/)"
        Client[Client]
        Define[Define() / Supervise()]
        Loop[Execution Loop]
    end

    subgraph "The Supervisor (internal/supervisor)"
        SupervisedAction[SupervisedAction]
        Lifecycle[Trace -> Align -> Run -> Steer]
        
        Client -->|Register| SupervisedAction
        SupervisedAction --> Lifecycle
    end

    subgraph "Logic Engine (internal/engine)"
        PolicyEngine[PolicyEngine]
        Runtime[MangleRuntime]
        StdLib[std.dl (Standard Lib)]
        
        Lifecycle -- Authorize (Pre) --> PolicyEngine
        Lifecycle -- Validate (Post) --> PolicyEngine
        PolicyEngine --> Runtime
        Runtime -.-> StdLib
    end

    subgraph "Adapters (adapters/)"
        AI[AI Adapter (Genkit Core)]
        Resilience[Circuit Breaker]
        MCP[MCP Adapter]
        
        Lifecycle -- Delegate --> Resilience
        Resilience -- Wrap --> AI
        AI -- ai.Generate --> GenkitInit
    end

    UserCode --> GenkitInit
    GenkitInit -->|Inject| AI
    UserCode -->|NewClient| Client
```

# 3. Core Kernel (`sdk`)

The `sdk` package is the user-facing facade. It enforces the **Explicit Initialization** pattern.

### 3.1 Client

The `Client` struct is the governance kernel.

  * **Initialization**:
      * `NewClient(ctx, opts...)`: The primary constructor.
      * `WithBlueprintPath(path)`: Loads the Datalog blueprint (formerly Policy).
      * `WithLLM(gen sdk.TextGenerator)`: Injects the default AI adapter.
  * **Responsibility**: Holds the `PolicyEngine`, manages the `Registry`, and provides the `ExecuteByName` entry point.

### 3.2 Supervise API

Replaces the old `Protect` API.

  * **Function**: `Supervise(action core.Action) core.Action`
  * **Mechanism**: Wraps a raw capability with the `internal/supervisor.SupervisedAction` decorator.

### 3.3 Semantic Loop (The Teacher-Student Protocol)

The `ExecuteByName` method implements the **Semantic State Machine**:

1.  **Execute**: Runs the action.
2.  **Catch**: If `AlignmentError` occurs (Blueprint violation).
3.  **Feedback**: Extracts feedback msg from the error.
4.  **Inject**: Puts feedback into `Envelope.Metadata["mangle_feedback"]`.
5.  **Retry**: Re-runs the action. The AI Adapter sees the feedback and self-corrects.

# 4. The Supervisor (`internal/supervisor`)

Formerly `guard`, this package implements the active governance lifecycle.

### 4.1 SupervisedAction

A decorator struct that implements `core.Action`. It binds the Logic Engine to the Execution Runtime.

### 4.2 Execution Lifecycle

The `Execute` method enforces the following strict sequence:

1.  **Trace Start**: Opens OTel span `Action.{Name}`.
2.  **Authorization (Pre-Check)**:
      * Converts Input --> Facts (`json_xxx`).
      * Queries: `deny("input")`.
      * **Halt**: If true, returns `AlignmentError`.
3.  **Execution**:
      * Calls `inner.Execute(ctx, input)`.
4.  **Validation (Post-Check)**:
      * Converts Output --> Facts.
      * Queries: `deny("output")`.
      * **Halt**: If true, queries `violation_msg(Msg)` and returns `AlignmentError`.
5.  **Steering**:
      * Queries: `correction(Hint)` OR `next_step(Target)`.
      * Updates Envelope Metadata for the SDK Loop to handle.

# 5. Logic Engine (`internal/engine`)

Hosts the Google Mangle runtime and Standard Library.

### 5.1 Standard Library (`std.dl`)

Auto-loaded on startup (`engine/resources/`). Defines the vocabulary for the outside world:

  * `json_str`, `json_num`, `json_bool`: Primitives for Reflection.
  * `quad`, `triple`: Primitives for Knowledge Graph.

### 5.2 Reflector

  * **Flatten**: Recursively walks Go structs to generate `json_xxx` facts.
  * **Zero-Config**: No tags required; uses Go reflection rules.

# 6. Adapters (`adapters`)

Implements the **Clean Adapter Pattern** with pragmatic concessions for framework stability.

### 6.1 AI Adapter (`adapters/ai`)

Wraps Google Genkit to provide Native Structured Output.

  * **Struct**: `genkitAdapter` holds `model ai.Model` AND `gk *genkit.Genkit`.
      * *Note:* The `gk` registry reference is retained to ensure `ai.Generate` functions correctly with advanced features.
  * **Utils**: `GenerateStruct[T]`
      * Uses `ai.Generate(ctx, adapter.gk, ai.WithModel(adapter.model), ai.WithOutput(&result))`.
      * Leverages Genkit's native schema generation and JSON parsing.

### 6.2 Resilience Adapter (`adapters/resilience`)

**New in v2.0**. Implements the Circuit Breaker pattern.

  * **Component**: `CircuitBreaker` struct wrapping `core.Action`.
  * **States**: `Closed` (Normal) --> `Open` (Fail Fast) --> `HalfOpen` (Probe).
  * **Concurrency**: Uses `sync.RWMutex` for thread safety.

### 6.3 MCP Adapter (`adapters/mcp`)

Wraps Model Context Protocol tools.

  * **Loader**: `NewLoader` supports `FailOnStartup` configuration.

# 7. Configuration (`config`)

Handles `mangle.yaml` loading.

  * **Changes**:
      * `Policy` field is semantically treated as `Blueprint`.
      * Validation logic uses strict schema checking (planned).

# 8. Project Layout

```text
manglekit/
├── adapters/          
│   ├── ai/            # Genkit Adapter (Clean + Native Utils)
│   ├── resilience/    # Circuit Breaker
│   ├── mcp/           # MCP Tools
│   └── ...
├── cmd/               # CLI (mkit)
├── config/            # YAML Loader
├── core/              # Interfaces (Action, Envelope)
├── docs/              # Architecture (LLD, CSD)
├── internal/          
│   ├── engine/        # Logic Engine (Mangle + std.dl)
│   └── supervisor/    # The Interceptor (formerly guard)
├── sdk/               # Public API (Client, Loop, Define)
└── examples/          # Demos (Semantic Feedback, RAG)
```

# 9. Changelog

* **2025-12-11**: **v2.0 Architecture**.
    * Renamed `Guard` --> `Supervisor`.
    * Renamed `Policy` --> `Blueprint`.
    * Added `Resilience` adapter (Circuit Breaker).
    * Refactored `AI` adapter to use pragmatic registry injection for native structured output.
    * Updated SDK to use Explicit Initialization.
* **2025-12-05**: **v1.0 Baseline**. Initial LLD release.
* **2025-11-29**: Architecture Cleanup. Logic moved from `policy/` to `engine/` and `sdk/`.
* **2025-11-27**: Major rewrite for v1.0. Replaced Builder/Registry architecture with Client/Guard/Engine composition.
* **2025-11-25**: Implemented Smart Router in Sandwich (now legacy).
* **2025-11-20**: Added `core/reflection`.