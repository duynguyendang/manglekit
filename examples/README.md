## Manglekit Examples

This directory contains canonical examples demonstrating the major capabilities of the **Manglekit SDK**.

Manglekit enables **Operational Neuro-Symbolic** application development by combining the flexibility of LLMs with the safety and determinism of Datalog policies.

-----

## 🧠 Strategic Example Catalog

The examples are categorized by the architectural layer they demonstrate, moving from raw data handling to complex system governance.

### 1. Tier I: The Data Bridge & Logic Foundation

*(Focus: Translating unstructured data into structured facts for the Logic Engine.)*

| Folder | Concept | Description |
| :--- | :--- | :--- |
| **`extractor_demo`** | **Semantic Fact Generation** | Uses the Extractor Adapter to enforce **strongly-typed Go Structs** as output from LLMs, providing clean, structured facts for policy evaluation. |
| **`kernel_knowledge_demo`** | **Static Knowledge Integration** | Demonstrates loading and querying static, external knowledge bases (e.g., **RDF/Turtle files**) directly within Datalog policies to ground AI decisions. |
| **`dynamic_pricing`** | **Runtime Context Injection** | Shows how to inject dynamic application state (user tier, inventory, time) as **Datalog Facts** to govern complex, real-time business logic. |
| **`context_aware_rag`** | **Knowledge Graph Filtering** | Demonstrates **Context-Aware Knowledge** by filtering N-Quads data using Datalog rules based on user roles (Multi-tenancy). |

-----

### 2. Tier II: Autonomous Intelligence & Flow Control

*(Focus: Managing complex, multi-step execution and ensuring agents self-correct.)*

| Folder | Concept | Description |
| :--- | :--- | :--- |
| **`planner`** | **Logic-Driven Planning** | Utilizes the **Planner** to decompose high-level goals into executable sequences by reasoning over available actions and policy subgoals. |
| **`steering`** | **Dynamic Routing & Dispatch** | Demonstrates how **Policy Rules** determine control flow (e.g., **Route** to a different action, **Retry** the current action) based on context and results. |
| **`semantic_feedback`** | **Teacher-Student Protocol** | The core **Self-Correction** mechanism. Shows an agent receiving a `PolicyViolationError` and using the feedback message to generate a correct output on retry. |
| **`rag_flow`** | **Governed RAG Pipeline** | A complete RAG pipeline demonstrating retrieval, context formatting, and LLM generation within the unified `Action Sandwich` structure. |

-----

### 3. Tier III: Enterprise Governance & Determinism

*(Focus: Applying mathematical certainty (Determinism) to high-risk business and security processes.)*

| Folder | Concept | Description |
| :--- | :--- | :--- |
| **`sre_guardrail`** | **Operational Guardrails** | Prevents high-risk operations (e.g., deleting a critical K8s resource) by checking contextual facts against deterministic security policies. |
| **`taint_demo`** | **Information Flow Control** | Demonstrates Taint Analysis: tracking the propagation of security labels (`"secret"`, `"pii"`) through the system and blocking sensitive data leakage via policy. |
| **`fintech_approval`** | **Complex Recursive Logic** | Handles intricate compliance scenarios (e.g., loan eligibility, nested approvals) using the power of Datalog's **Recursive Rules**. |
| **`policy-copilot`** | **Neuro-Symbolic Policy Gen** | A "Text-to-Policy" demonstration where an LLM generates Datalog rules from natural language, which are then validated and executed by the Engine. |

-----

### 4. Tier IV: Architecture & Observability

*(Focus: Demonstrating the robustness and system integrity of the Go-native framework.)*

| Folder | Concept | Description |
| :--- | :--- | :--- |
| **`lineage_demo`** | **Tracing & Data Lineage** | Tracks the full execution path and data flow across multiple chained Actions using built-in **OpenTelemetry** spans. |

-----

## 🧩 Architectural Highlights

Manglekit allows you to build systems that are:

  * **Safe**: Policies are deterministic and guaranteed to execute.
  * **Observable**: Every action is traced, and data flow is tracked.
  * **Flexible**: Mix and match "Neural" (LLM) and "Symbolic" (Code/Rules) components.

### Key Patterns

  * **Facade Pattern**: Use `manglekit.NewClient` and `manglekit.Define` for a clean, idiomatic Go API.
  * **Guardrails**: Wrap any function or struct with `client.Protect` to enforce policies.
  * **Adapters**: Integrate with external ecosystems (like Genkit) via `adapters/`.

-----

## 🧠 Running the Examples

### Prerequisites

  * **Go 1.24+**
  * **Environment:** Some examples may require API keys (e.g., `GOOGLE_API_KEY`) if they connect to real LLMs, though many use mocks.

### Instructions

Navigate to any example directory and run `main.go`. For example:

```bash
cd examples/sre_guardrail
go run .
```

-----

### License

All examples are MIT-licensed and intended for educational and experimental use within the Manglekit ecosystem.