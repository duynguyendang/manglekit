---
context_type: low_level_design
project: manglekit
language: go
version: 1.0
last_updated: 2025-12-05T13:00:00Z
stability: stable
audience: developers
---

# 1. Purpose & Scope

This document provides the Low-Level Design (LLD) for Manglekit v1.0. It details the implementation of the **Universal AI Governance Kernel**, focusing on the `Client`, `Guard`, `Engine`, and `Adapter` subsystems. This design replaces the previous "Builder/Registry" architecture with a streamlined, composition-based approach.

# 2. Component Diagram

```mermaid
graph TD
    subgraph "User Space"
        UserCode[User Application]
        Config[config.yaml]
    end

    subgraph "Manglekit Kernel (sdk/)"
        Client[Client]
        Protect[Protect() API]
        Client --> Protect
    end

    subgraph "The Guard (guard/)"
        GuardedAction[GuardedAction]
        Lifecycle[Execute Lifecycle]
        
        Protect --> GuardedAction
        GuardedAction --> Lifecycle
    end

    subgraph "Logic Engine (engine/)"
        PolicyEngine[PolicyEngine]
        Runtime[MangleRuntime]
        Reflector[Reflector]
        
        Lifecycle -- Authorize/Validate --> PolicyEngine
        PolicyEngine --> Reflector
        PolicyEngine --> Runtime
    end

    subgraph "Adapters (adapters/)"
        AI[AI Adapter (Genkit)]
        Func[Func Adapter]
        Vector[Vector Adapter]
        
        Lifecycle -- Delegate --> AI
        Lifecycle -- Delegate --> Func
        Lifecycle -- Delegate --> Vector
    end

    UserCode -->|NewClient| Client
    UserCode -->|Protect(Action)| Protect
```

# 3. Core Kernel (`sdk`)

The `sdk` package provides the main entry point and public API.

### 3.1 Client
The `Client` struct is the central coordinator. It holds references to the `PolicyEngine`, `Registry`, and `MemoryStore`.

*   **Initialization**:
    *   `NewClient(ctx, policyFile, opts...)`: Basic initialization.
    *   `NewClientFromConfig(ctx, configPath, opts...)`: Loads settings from YAML.
*   **Responsibility**: Manages the lifecycle of the governance engine and observability providers.

### 3.2 Protect API
The `Protect(action core.Action) core.Action` method is the primary value proposition.
*   **Mechanism**: It wraps any implementation of `core.Action` with a `guard.GuardedAction`.
*   **Tracing**: If a tracer is configured, it wraps the action with `guard.NewWithTracer`; otherwise, it uses `guard.New`.

### 3.3 Helpers
*   `Call[Out]`: Generics-based helper to execute an action, handling `core.Envelope` packing/unpacking automatically.
*   `ExecuteByName`: Entry point for the Semantic State Machine (RunLoop).

# 4. The Guard (`guard`)

The `guard` package implements the **Guarded Action** pattern, enforcing the governance lifecycle.

### 4.1 GuardedAction
A decorator struct that implements `core.Action`.

### 4.2 Execution Lifecycle
The `Execute` method enforces the following strict sequence:

1.  **Trace Start**: Opens a new OpenTelemetry span (`Action.{Name}`).
2.  **Context Injection**: Injects the `core.Logger` into the context.
3.  **Pre-Computation (Authorize)**:
    *   Calls `engine.Authorize(ctx, meta, input)`.
    *   Reflects input payload to facts.
    *   Queries `deny(Input)?`.
    *   **Halt**: If denied, returns error immediately.
4.  **Execution**:
    *   Calls `inner.Execute(ctx, input)`.
    *   Captures result or error.
5.  **Post-Computation (Validate)**:
    *   Calls `engine.Validate(ctx, meta, result)`.
    *   Reflects output payload to facts.
    *   Queries `violation(Output)?`.
    *   **Halt**: If violated, returns error.
6.  **Steering**:
    *   Calls `engine.EvaluateSteering(ctx, input)`.
    *   Determines next step (`RETRY`, `ROUTE`, `ALLOW`).
7.  **Trace End**: Closes span and records outcome.

# 5. Logic Engine (`engine`)

The `engine` package encapsulates the Google Mangle Datalog engine and the reflection system.

### 5.1 PolicyEngine
High-level coordinator that bridges the Guard and the Runtime.
*   **Authorize**: Converts input to facts -> Checks `deny`.
*   **Validate**: Converts output to facts -> Checks `violation`.

### 5.2 MangleRuntime
Wrapper around `google/mangle`.
*   **Loading**: Supports loading `.dlog` (rules) and `.facts` (data) from files or directories.
*   **Stratification**: Performs analysis and stratification of the Datalog program for safe evaluation.
*   **Querying**: `ExecuteQuery` (boolean) and `QueryWithSolutions` (results).

### 5.3 Reflection
Converts Go structs to Datalog facts.
*   **Recursion**: Traverses nested structs and maps.
*   **Type Hooks**: Allows custom handling for types like `time.Time` or `net.IP`.
*   **Naming**: Converts `Struct.Field` to `struct.field` predicates.
*   **Reflector 2.0**: Supports deep traversal of Maps (key as argument) and JSON tags.

# 6. Universal Adapters (`adapters`)

Adapters convert external systems into the `core.Action` interface.

### 6.1 AI Adapter (`adapters/ai`)
Wraps Google Genkit models.
*   **GenkitLLMAdapter**: Implements `core.Action`.
*   **Mapping**: Maps `core.Envelope` payload to Genkit prompts and Genkit responses back to `core.Envelope`.

### 6.2 Func Adapter (`adapters/func`)
Wraps native Go functions.
*   **Mechanism**: Uses Go reflection to call a `func(ctx, In) (Out, error)`.
*   **Safety**: Catches panics and converts them to errors.

### 6.3 Vector Adapter (`adapters/vector`)
Wraps Genkit Retrievers.
*   **GenkitRetrieverAdapter**: Exposes `Retrieve`, `Index`, etc., as Actions.

### 6.4 MCP Adapter (`adapters/mcp`)
Wraps Model Context Protocol tools.
*   **MCPAction**: Wraps an MCP Tool.
*   **MCPLoader**: Discovers tools from MCP servers (Stdio/SSE) and registers them as Actions.

### 6.5 Extractor Adapter (`adapters/extractor`)
Uses an LLM to extract structured JSON from unstructured text.
*   **ExtractorAction**: Implements `core.Action`.

# 7. Configuration (`config`)

The `config` package handles YAML loading.

*   **Structure**:
    *   `Policy`: Path to `.dl` files.
    *   `Observability`: Service name, OTel endpoint, logging level.
*   **Loader**: Supports environment variable expansion (e.g., `${API_KEY}`).

# 8. Project Layout

```
manglekit/
├── adapters/          # Universal Adapters (AI, Func, Vector, MCP, Extractor)
├── cmd/               # CLI tools (mkit)
├── config/            # Configuration loading
├── core/              # Core interfaces (Action, Envelope, Logger)
├── docs/              # Documentation (CSD, HLD, LLD)
├── engine/            # Logic Engine (Mangle wrapper, Policy)
├── guard/             # GuardedAction implementation
├── internal/          # Internal utilities (Logger, Telemetry, Tools)
├── sdk/               # SDK Tooling (Client, Loop, Policy Copilot)
└── examples/          # Reference implementations
```

# 9. Changelog

*   **2025-12-05**: **LLD Sync**. Updated to reflect `sdk/` package, `Steering` lifecycle step, `Reflector 2.0`, and new adapters (MCP, Extractor).
*   **2025-11-29**: Architecture Cleanup. Logic moved from `policy/` to `engine/` and `sdk/`.
-   **2025-11-27**: Major rewrite for v1.0. Replaced Builder/Registry architecture with Client/Guard/Engine composition.
*   **2025-11-25**: Implemented Smart Router in Sandwich (now legacy).
*   **2025-11-20**: Added `core/reflection`.
