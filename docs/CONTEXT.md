---
context_type: full_source_dump
project: manglekit
language: go
last_updated: 2025-11-27
scan_mode: critical_path
---

# Manglekit: Live Context Snapshot

> **STATUS**: ✅ ACTIVE | **VERSION**: 1.0.0 (Genesis) | **REF**: `UGA-ARCH`

## 1. Executive Summary

Manglekit is a **Neuro-Symbolic AI Governance Kernel** for Go.
It enforces policy-as-code (Datalog) on every AI interaction *before* it happens.
The architecture has shifted from a "sandwich" middleware to a **Universal Guarded Action (UGA)** model.

**Key Principles:**
1.  **Everything is an Action**: LLM calls, Tools, Retrievers, and API hits are all `core.Action`.
2.  **Every Action is Guarded**: The `Client` wraps every registered Action in a `Supervisor`.
3.  **Policy First**: The `Engine` evaluates Datalog rules to Authorize, Modify, or Deny execution.
4.  **Traceability**: Every decision is traced (OpenTelemetry) with reasoning metadata.

---

## 2. High-Level Architecture (UGA)

```mermaid
graph TD
    User[User Code] --> Client[sdk.Client]
    Client -->|ExecuteByName| Supervisor[Supervisor (Guard)]

    subgraph Governance Kernel
        Supervisor -->|1. Trace| Tracer[OTel Tracer]
        Supervisor -->|2. Assess| Engine[Policy Engine]
        Engine -->|Query| Rules[Datalog Rules]
        Supervisor -->|3. Execute| Action[Core Action]
        Action -->|Call| Adapter[Adapter (LLM/Tool)]
        Supervisor -->|4. Reflect| Engine
    end

    Adapter --> External[External API]
```

---

## 3. Critical Path Contracts (`core/`)

### 3.1 The Action Interface (`core/logic.go`)

This is the atomic unit of work.

```go
type Action interface {
    // Execute performs the operation.
    // Input/Output are wrapped in Envelopes for metadata propagation.
    Execute(ctx context.Context, input Envelope) (Envelope, error)

    // Metadata returns static definition (Name, Type).
    Metadata() ActionMetadata
}
```

### 3.2 The Envelope (`core/types.go`)

Data carrier for the system.

```go
type Envelope struct {
    ID             string            `json:"id"`
    Payload        any               `json:"payload"`
    Metadata       map[string]string `json:"metadata"` // Context, UserID, RiskScore
    SecurityLabels []string          `json:"security_labels"` // Taint tracking
    ContentType    string            `json:"content_type"`
}
```

### 3.3 The Policy Engine (`core/governance.go`)

```go
type Evaluator interface {
    // Assess checks if an action is allowed before execution.
    Assess(ctx context.Context, input Envelope, meta ActionMetadata) (Decision, error)

    // LoadPolicy updates the rule set.
    LoadPolicy(ctx context.Context, source string) error
}
```

---

## 4. Implementation Snapshot

### 4.1 SDK Client (`sdk/client.go`)

The `Client` is the factory and registry.

*   **Responsibility**: Wiring dependencies (Engine, Logger, Tracer).
*   **Method**: `Supervise(action)` -> returns `GuardedAction`.
*   **API**: `Memory()` -> exposes `core.AgentMemory` for RAG seeding.
*   **State**: Holds the `map[string]Action` registry.

### 4.2 The Supervisor (`internal/supervisor/supervisor.go`)

The "Guard" implementation.

1.  **Start Span**: `trace.Start(ctx, "action.execute")`.
2.  **Reflect Input**: Convert `Envelope.Payload` to Datalog facts.
3.  **Assess**: Query `allow(Action, Input)?`.
    *   If `deny(Msg)` -> Return `AlignmentError`.
4.  **Execute**: Call `Inner.Execute()`.
5.  **Reflect Output**: Convert Result to Facts.
6.  **Validate**: Query `valid_output(Action, Result)?`.

### 4.3 The Engine (`internal/engine/solver.go`)

Uses `google/mangle` (Datalog).

*   **Facts**: `input(req)`, `user(id)`, `risk_score(80)`.
*   **Rules**:
    ```prolog
    deny("Too risky") :- input(Req), risk_score(S), :gt(S, 90).
    ```

---

## 5. Adapter Landscape (`adapters/`)

| Adapter | Package | Status | Description |
|:---|:---|:---|:---|
| **Genkit** | `adapters/ai` | ✅ Stable | Wraps `genkit.Model` as Action. |
| **Function** | `adapters/func` | ✅ Stable | Wraps arbitrary Go func. |
| **MCP** | `adapters/mcp` | ⚠️ Beta | Model Context Protocol support. |
| **CircuitBreaker** | `adapters/resilience` | ✅ Stable | Failure protection. |

---

## 6. Directory Map (Critical)

```
/
├── sdk/                 # Public API (Client, Options)
│   ├── client.go        # Main entry point
│   ├── loop.go          # Steering Loop (ReAct)
│   └── ...
├── core/                # Interfaces (No dependencies!)
├── internal/            # Private Implementation
│   ├── engine/          # Mangle Datalog Runtime
│   ├── supervisor/      # Governance Logic
│   └── ...
├── adapters/            # Integrations
└── cmd/mkit/            # CLI Tool
```

---

## 10. Known Gaps

1.  **Lineage Tracking**: `SecurityLabels` are propagated but not fully integrated with a graph database yet.
2.  **State Management**: `ConversationHistory` is in-memory only (volatile).
3.  **Distributed Policies**: No centralized policy server support (local files only).

---

## 13. Machine Appendix (JSON Snapshot v1)

```json
{
  "architecture": "UGA",
  "version": "1.0.0",
  "components": {
    "client": "sdk/client.go",
    "engine": "internal/engine",
    "supervisor": "internal/supervisor"
  },
  "open_issues": [
    "feature: persistent_state",
    "refactor: consolidate_reflection"
  ]
}
```

---

## 14. Changelog

*   **2025-11-27**: Refactored getters to be idiomatic (`Memory()`, `Provider()`, `StdLib()`) and fixed observability gaps.
*   **2025-11-27**: Added `GetMemory()` to `sdk.Client` and `policy_bot` example.
*   **2025-11-27**: Complete rewrite of `CONTEXT.md` to match UGA architecture. Removed "Sandwich" references.
*   **2025-11-20**: Added `adapters/mcp`.
*   **2025-11-15**: v1.0 Release.
