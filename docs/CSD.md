# Conceptual Solution Design (CSD) — Manglekit Framework

**Version:** 1.0.0
**Status:** Approved
**Scope:** Core Framework Architecture & Integration Strategy
**Context:** Neuro-Symbolic Governance for Generative AI

## 1. Introduction

### 1.1 Purpose
Manglekit is a governance and orchestration framework designed to operate as a control layer above Generative AI execution engines (specifically Genkit). Its primary purpose is to enforce deterministic policy, ensure data safety, and manage complex logic flows using symbolic reasoning (Datalog), while delegating low-level execution tasks (LLM inference, Vector search) to the underlying Genkit ecosystem.

### 1.2 Problem Space
Native integration of LLMs into enterprise systems introduces non-deterministic behaviors that are difficult to audit and control. Direct dependency on specific model providers creates vendor lock-in. Manglekit addresses these issues by decoupling **Business Logic (Governance)** from **AI Execution (Infrastructure)**.

---

## 2. High-Level Architecture

The system follows a layered architecture pattern, enforcing a strict separation of concerns between decision-making ("Brain") and task execution ("Muscle").

### 2.1 Architectural Layers

| Layer | Component | Responsibility |
| :--- | :--- | :--- |
| **L4: Application** | `sdk.Builder`, YAML Config | Entry point for configuration and pipeline assembly. |
| **L3: Governance (Brain)** | **Manglekit Core** | Orchestration, State Management, Rule Evaluation (Policy), Data-to-Fact Conversion. |
| **L2: Adaptation** | **Universal Adapters** | Protocol translation between Manglekit Interfaces and Genkit Plugins. |
| **L1: Execution (Muscle)** | **Genkit Ecosystem** | Drivers for LLMs, Vector DBs, Embedders, and External Tools. |

### 2.2 Integration Model: The Universal Adapter Pattern

Manglekit minimizes code maintenance by utilizing generic adapters that wrap Genkit interfaces.

* **LLM Integration:** The `GenkitLLMAdapter` wraps `ai.Model` from Genkit, exposing it as `core.LLMClient`. It handles prompt marshaling, parameter injection, and response standardization.
* **Retriever Integration:** The `GenkitRetrieverAdapter` wraps `ai.Retriever`, exposing it as `core.Retriever`. It manages query formatting and metadata filtering.
* **Tool Integration:** The `HTTP` tool adapter allows Declarative Orchestrators to invoke external microservices or Genkit Flows via standard REST protocols.

---

## 3. Core Capabilities & Design Patterns

### 3.1 The "Safety Sandwich" Orchestration Pattern
All execution flows within Manglekit enforce a strict "intercept-execute-intercept" lifecycle:

1.  **Pre-Execution Phase (Policy Check):**
    * Input data is converted to Logical Facts.
    * **Pre-Rules** evaluate authorization (AuthZ), input validation, and routing logic.
    * *Outcome:* Proceed, Reject, or Mutate Input.
2.  **Execution Phase (Neural Processing):**
    * The request is passed to the selected Provider (via Universal Adapter).
    * Operations include Retrieval (RAG), Generation (LLM), or Tool Invocation.
3.  **Post-Execution Phase (Compliance Check):**
    * Output data is analyzed against **Post-Rules**.
    * Operations include PII Redaction, Format Validation (JSON/Schema), and Logic Consistency checks.
    * *Outcome:* Return Data or Reject.

### 3.2 Neuro-Symbolic Data Processing
Manglekit bridges the gap between unstructured data (Text/Vector) and structured logic (Rules) via the **Struct-to-Fact Converter**.

* **Mechanism:** Uses Golang Reflection to traverse application structures (Config, AST, User Profile).
* **Transformation:** Maps struct fields and tags to Datalog predicates (e.g., `User{Role: "admin"}` $\to$ `role(user_id, "admin")`).
* **Usage:** Enables the Rule Engine to reason directly about runtime application state without manual boilerplate code.

### 3.3 Declarative Microservices Orchestration
For distributed systems, Manglekit functions as an intelligent Gateway:

* **Symbolic Routing:** Uses rules to determine the optimal downstream service based on query intent and complexity (e.g., routing complex queries to GPT-4, simple ones to local Llama).
* **Tool Abstraction:** External services are modeled as `core.Tool` instances. The orchestrator invokes them dynamically based on the logical plan defined in Datalog.

---

## 4. Data Flow Specification

### 4.1 Request Lifecycle (RAG Flow Example)

1.  **Ingestion:** `Orchestrator.Execute(ctx, query)` is called.
2.  **Fact Generation:** Query metadata and User Context are converted to Facts.
3.  **Pre-Rule Evaluation:** Mangle Engine evaluates `allow_query(User, Query)`.
    * *If Deny:* Return Error immediately.
4.  **Retrieval:** `GenkitRetrieverAdapter` calls the underlying Vector Store (e.g., LocalVec, Pinecone).
5.  **Reranking (Optional):** Documents are re-scored based on semantic relevance.
6.  **Generation:** `GenkitLLMAdapter` constructs the prompt and invokes the LLM.
7.  **Post-Rule Evaluation:** Generated text is scanned for sensitive patterns defined in the RuleSet.
8.  **Response:** Final sanitized answer is returned with a `DecisionTrace` (Audit Log).

---

## 5. Component Specifications

### 5.1 Governance Components
* **RuleSet (Interface):** Abstract interface for policy engines.
    * *Default Implementation:* `mangle` (Datalog).
    * *Future Extensions:* `opa` (Rego), `casbin` (ACL).
* **StateProvider:** Manages session consistency.
    * *Implementations:* `inmemory`, `redis`.
* **SchemaParser:** Validates and parses structured data (JSON, RDF, AST) into Facts.

### 5.2 Execution Components (via Genkit)
* **Retriever:** Provides semantic search capabilities.
    * *Supported:* Any Genkit-compliant plugin (Pinecone, Weaviate, Qdrant, LocalVec).
* **LLM:** Provides generative capabilities.
    * *Supported:* Any Genkit-compliant model (Google Gemini, OpenAI GPT, Anthropic Claude, Ollama).

---

## 6. Non-Functional Requirements (NFRs)

### 6.1 Extensibility
* **Zero-Code Provider Adoption:** New Genkit plugins must be usable via configuration (`config.yaml`) without requiring code changes in the Manglekit core.
* **Open Registry:** The component registry allows external developers to register custom providers (implementing `core` interfaces) at runtime.

### 6.2 Observability
* **Auditability:** All rule decisions (Allow/Deny) must be logged with the specific rule ID and reason.
* **Tracing:** Integration with OpenTelemetry to trace requests across the Logic Layer (Mangle) and Execution Layer (Genkit).

### 6.3 Deployment
* **Single Binary:** The framework compiles into a standalone Go binary, suitable for containerized environments (Kubernetes) or edge devices.
* **Configuration:** All behavior is controlled via declarative files (YAML for structure, Datalog for logic).

---

## 7. Conclusion

Manglekit defines a standardized architecture for **Governed AI**. By decoupling policy from execution and leveraging the Universal Adapter pattern, it provides a robust foundation for building enterprise-grade, compliant, and explainable AI systems.