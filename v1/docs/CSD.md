# Conceptual Solution Design (CSD) — Manglekit

**Version:** 2.0
**Status:** Stable
**Context:** Operational Neuro-Symbolic AI Framework

## 1. Problem Statement

The current AI engineering landscape faces a fundamental **Impedance Mismatch**:

1.  **The Probabilistic Gap:** Business logic requires **100% determinism** (RBAC, Quotas, Regulatory Compliance), but LLMs operate on **stochastic probabilities**. Controlling behavior via Prompt Engineering is fragile and non-testable.
2.  **The Runtime Mismatch:** AI logic is predominantly Python-based (heavy, slow start, GIL-bound), while modern Cloud-Native infrastructure is Go-based (lightweight, concurrent, static binary).
3.  **The Unstructured Data Trap:** Software logic operates on structured data (JSON/Structs), but AI operates on unstructured text/vectors, creating a disconnect between application state and model reasoning.

## 2. Design Philosophy

Manglekit is an **Embeddable Governance Framework** written in Go. It aims to solve the impedance mismatch by treating AI not as a magic box, but as a **Governed Subsystem**.

### 2.1 The Core Concept: "Prefrontal Cortex" Architecture

The system enforces a strict separation of concerns based on the Neuro-Symbolic principle:

  * **The Neuro Layer (Genkit):** Handles fuzzy perception, semantic extraction, and natural language synthesis. It acts as the **I/O mechanism**.
  * **The Symbolic Layer (Mangle):** Handles decision-making, routing, and policy enforcement. It acts as the **Control Unit**.

### 2.2 Engineering Principles

  * **Safety by Design:** Default-deny architecture. No action executes without passing a logic gate.
  * **Zero-Cost Abstraction:** Leveraging Go's compile-time safety and reflection to map data without heavy runtime overhead.
  * **Infrastructure Agnostic:** The core logic is decoupled from the underlying model provider (OpenAI/Gemini) via Universal Adapters.

-----

## 3. Architectural Patterns

### 3.1 The "Action Sandwich" (Intercept-Execute-Intercept)

Manglekit replaces ad-hoc chains with a formalized execution pipeline. Every operation (RAG, RPC Call, Computation) is wrapped in a standard governance loop.

```text
[ Request Context (Go Struct) ]
        │
        ▼
[ REFLECTION LAYER ] ───► Projects State into Datalog Facts
        │
        ▼
[ 1. PRE-COMPUTATION GATE ] 🛡️
    ├── AuthZ Check: "Can User U perform Action A?"
    ├── Validation: "Is the input schema valid?"
    └── Symbolic Routing: "Based on intent, select Provider P."
        │
        ├── [ DENY ] ──► Return Error (Zero Cost)
        │
        ▼
[ 2. EXECUTION UNIT (Generic Action) ] ⚡
    ├── RAG: Vector Search + Re-ranking
    ├── Tool: gRPC / REST API Call
    └── Local: Native Go Function
        │
        ▼
[ 3. POST-COMPUTATION GATE ] 🛡️
    ├── Data Masking: Redact PII/Sensitive fields based on User Role.
    ├── Consistency Check: Logic validation of the output.
    └── Audit Logging: Record the decision trace.
        │
        ▼
[ Response Generation (LLM Synthesis) ]
```

### 3.2 Data Bridge: Struct-to-Fact Reflection

To enable reasoning over application state without glue code, Manglekit implements a zero-config reflection engine.

  * **Input:** Standard Go Structs (`User`, `Config`, `AST`).
  * **Process:** The engine traverses the object graph at runtime, converting fields and tags into Datalog predicates.
  * **Output:** A queryable knowledge base representing the exact runtime state.

-----

## 4. Orchestration Modes

The framework provides two orchestration strategies depending on control flow requirements:

### 4.1 Pipeline Mode (Imperative)

  * **Pattern:** Linear Chain.
  * **Use Case:** High-throughput, low-latency tasks (e.g., Search, Form Processing).
  * **Mechanism:** A compiled sequence of `Pre-Check -> Action -> Post-Check`. Zero reasoning latency.

### 4.2 Agent Mode (Declarative)

  * **Pattern:** Dynamic Dispatch.
  * **Use Case:** Complex problem solving requiring multi-step execution.
  * **Mechanism:** The **Action Router** evaluates Symbolic Rules to dynamically select the next tool.
      * *Example:* `next_step(tool_B) :- result(tool_A, "incomplete").`

-----

## 5. Capabilities & Modules

### 5.1 Logic Engine (Mangle)

  * **Engine:** Google Mangle (Datalog).
  * **Storage:** In-memory execution (Microsecond latency).
  * **Policy Loading:** Supports Hot-Reload via file watchers (`.dl`, YAML) or GitOps pipelines.

### 5.2 Universal Adapters

  * **LLM/Embedder:** Wraps Genkit `ai.Model` and `ai.Embedder`.
  * **Action:** Generic interface for wrapping `func(ctx, input) output`.
  * **Semantic Extraction:** Dedicated agents for converting unstructured text (Docs, Logs) into structured Facts for the logic engine.

-----

## 6. Engineering Use Cases

### Pattern A: The Deterministic RAG Gateway

**Problem:** Vector DBs lack Row-Level Security (RLS). Metadata filtering is insufficient for complex ACLs.
**Solution:**

1.  **Retrieve:** Fetch documents with metadata (tags, owner).
2.  **Project:** Convert metadata + User Context to Facts.
3.  **Filter:** Execute Datalog rules to filter the document list *before* context injection.
    **Outcome:** Mathematically guaranteed data isolation.

### Pattern B: Architectural Linter (Code-as-Data)

**Problem:** LLMs generate code that violates architectural constraints (e.g., Layer violations).
**Solution:**

1.  **Parse:** Convert generated code to AST facts (using Tree-sitter integration).
2.  **Validate:** Check against dependency graph rules (e.g., `deny :- import(domain, infrastructure)`).
3.  **Reject:** Block code committing or return feedback to LLM for self-correction.

### Pattern C: Edge AI Controller (IoT)

**Problem:** Cloud latency and connectivity issues make cloud-only AI unsafe for physical control.
**Solution:**

1.  **Local Runtime:** Run Manglekit as a static binary on Edge Gateway (ARM64).
2.  **Local Logic:** Enforce safety rules locally (e.g., "Do not open door if time \> 22:00").
3.  **Hybrid AI:** Offload complex understanding to Cloud LLM, but keep execution control local.

-----

## 7. Summary

Manglekit is **Infrastructure for AI**.
It moves the complexity of AI governance from "Prompt Engineering" (Fragile) to "Systems Engineering" (Robust).

  * **Language:** Go.
  * **Paradigm:** Neuro-Symbolic.
  * **Deliverable:** Single Static Binary / Library.