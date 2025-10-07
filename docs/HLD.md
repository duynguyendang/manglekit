# Manglekit SDK — High-Level Architecture

## **1. Vision & Architectural Principles**

### **1.1. Vision**

Manglekit is an **SDK-first, embeddable Go framework** for building verifiable and controllable neuro-symbolic RAG applications. Its architecture prioritizes a superior developer experience by abstracting complexity, enabling flexibility through a pluggable provider model, and enforcing correctness via a declarative rules engine.

### **1.2. Architectural Principles**

  * **SDK-First, Service-Second:** The core product is a library. A runnable service is a reference implementation, not the primary artifact. This ensures maximum flexibility for integration.
  * **Zero-Wiring & Convention over Configuration:** The primary user interaction is through a fluent `Builder` API that manages all dependency injection and initialization internally. The user declares *what* they want, and the SDK handles *how* to wire it.
  * **Pluggable & Agnostic Core:** All major components (LLM, Retriever, Reranker, Embedder) are defined by Go interfaces. This decouples the core pipeline logic from specific implementations, making the system provider-agnostic.
  * **Separation of Concerns:** A strict separation exists between:
      * **Orchestration Logic** (the fixed "Sandwich Pipeline").
      * **Business Logic** (the pluggable `Providers`).
      * **Declarative Logic** (the external `Mangle` rules and facts).
  * **Fail Fast, Fail Clear:** The configuration and build process must validate the entire dependency tree at initialization. Missing configurations or incompatible components will result in an immediate, clear error, not a runtime failure deep within the pipeline.
  * **Stateless Pipeline:** The core RAG pipeline is stateless, making it inherently scalable. State (like indexes or caches) is managed by stateful providers that are injected into the pipeline.

-----

## **2. System Architecture & Boundaries (C4 Model - Level 2)**

The Manglekit ecosystem consists of the Application Layer, the SDK Core, Pluggable Providers, and External Systems. The SDK Core is the central component, providing the framework and orchestration.

```mermaid
graph TD
    subgraph "Application Layer (User's Domain)"
        UserApp[Go Application / Genkit Flow]
    end

    subgraph "Manglekit SDK (System Boundary)"
        Builder[Builder API & Config System]
        Pipeline[Orchestrator: The Sandwich Pipeline]

        subgraph "Core Interfaces"
            I_Retriever[retrieve.Retriever]
            I_LLM[llm.Client]
            I_RuleSet[core.RuleSet]
            I_Reranker[rerank.Reranker]
        end
    end

    subgraph "Provider Ecosystem (Implementations)"
        P_Mangle[Mangle RuleSet Provider]
        P_Hybrid[Hybrid Retriever]
        P_InMemory[In-Memory Retriever]
        P_OpenAI[OpenAI LLM/Embedder]
        P_Google[Google LLM/Embedder]
        P_Custom[...]
    end

    subgraph "External Systems"
        Ext_LLM["LLM APIs (OpenAI, Google)"]
        Ext_VDB["Vector DBs (Qdrant, Chroma)"]
        Ext_Data["Data Sources (Docs, SQL DBs, KGs)"]
        Ext_Rules["Rule Files (.dlog)"]
    end

    UserApp -- Uses --> Builder
    Builder -- Instantiates & Wires --> Pipeline
    UserApp -- Invokes `Run()` --> Pipeline

    Pipeline -- Uses --> I_RuleSet
    Pipeline -- Uses --> I_Retriever
    Pipeline -- Uses --> I_Reranker
    Pipeline -- Uses --> I_LLM

    P_Mangle -- Implements --> I_RuleSet
    P_Hybrid -- Implements --> I_Retriever
    P_InMemory -- Implements --> I_Retriever
    P_OpenAI -- Implements --> I_LLM
    P_Google -- Implements --> I_LLM

    P_OpenAI --> Ext_LLM
    P_Google --> Ext_LLM
    P_Hybrid --> Ext_VDB
    P_Mangle --> Ext_Rules
```

-----

## **3. Component Breakdown & Interaction Patterns**

### **3.1. The Builder & Configuration System**

  * **Responsibility**: To provide the sole, user-facing entry point for constructing a pipeline and to manage all configuration and dependency injection.
  * **Interaction Pattern**:
    1.  The user instantiates the `Builder`.
    2.  The user declaratively configures the pipeline using `With...` methods (e.g., `WithLLM("openai", ...)`).
    3.  Optionally, the user provides a type-safe `Config` struct via `WithConfig()` for explicit, code-based configuration.
    4.  The user calls `Build()`.
    5.  The `Builder` performs a **layered configuration lookup** (explicit config -\> environment variables) for each requested provider.
    6.  It **automatically initializes** any required clients (e.g., OpenAI Go client) using the resolved credentials.
    7.  It fetches the provider implementations from an internal **Registry**.
    8.  It **injects** the initialized clients and configurations into the provider constructors.
    9.  It instantiates the `Pipeline` orchestrator with the fully configured providers.
    10. It returns the ready-to-use `Orchestrator` instance or a detailed error.

### **3.2. The Orchestrator (Sandwich Pipeline)**

  * **Responsibility**: To execute the core RAG workflow in a fixed, predictable sequence. It is stateless and unopinionated about the specifics of the providers it orchestrates.
  * **Interaction Pattern (The "Sandwich")**:
    1.  Receives a `core.Query` via `Run()` or `RunWithContext()`.
    2.  **`Pre-Process`**: Invokes the `core.RuleSet` provider to validate and enrich the query. The result is a mutated `core.Query` object with strategic metadata.
    3.  **`Retrieve`**: Invokes the `retrieve.Retriever` provider with the enriched query.
    4.  **`Rerank`**: Invokes the `rerank.Reranker` provider to re-order the retrieved context.
    5.  **`Synthesize`**: Invokes the `llm.Client` provider, using a configurable prompt template to combine the query and the vetted context.
    6.  **`Post-Process`**: Invokes the `core.RuleSet` provider again to validate, redact, or filter the final `core.Answer` generated by the LLM.
    7.  Returns the final, policy-compliant `core.Answer`.

### **3.3. The Provider Ecosystem**

  * **Responsibility**: To provide concrete implementations for the SDK's core interfaces. Each provider is a self-contained unit of business logic.
  * **Interaction Pattern**:
      * Providers are discovered by the `Builder` via an internal `Registry`.
      * They are designed to be **stateless or to manage their own state** (e.g., a retriever managing a connection pool to a vector DB).
      * They are configured and instantiated *only* by the `Builder`, never directly by the end-user.

-----

## **4. Core Operational Modes**

The architecture is designed to support two distinct operational modes, providing maximum utility.

### **4.1. Integrated RAG Framework Mode**

  * **Architectural Pattern**: Manglekit acts as a complete, self-contained RAG system. The user configures a `Retriever` that connects to a persistent data source. The application calls `orch.Run(query)`.
  * **Use Case**: Building new, end-to-end RAG applications where Manglekit manages the entire data flow.

### **4.2. Pluggable Logic Layer Mode**

  * **Architectural Pattern**: Manglekit acts as a specialized processing engine. The application layer is responsible for retrieval. It passes a set of pre-retrieved documents to Manglekit via `orch.RunWithContext(query, retrievedDocs)`.
  * **Internal Flow**: The `Orchestrator` detects this mode, bypasses its internal `Retrieve` step, and injects the user-provided documents directly into the `Rerank` stage.
  * **Use Case**: Augmenting existing RAG pipelines with Manglekit's unique symbolic reasoning, guardrail capabilities, and robust LLM generation features.

-----

## **5. Non-Functional Requirements (High-Level)**

  * **Performance**: The stateless nature of the core pipeline allows for horizontal scaling. Performance targets (e.g., P99 \<300ms) are the responsibility of the injected providers. The SDK will provide OTel traces to measure and identify bottlenecks in each stage.
  * **Scalability**: The SDK itself is a library and scales with the application it's embedded in. For service mode, standard cloud-native scaling patterns (e.g., Kubernetes replicas) apply.
  * **Security**: Security is addressed at two levels:
      * **SDK Level**: The SDK provides the mechanism for security via `Mangle Post-Rules` (for redaction and policy enforcement). It does not handle transport-level security or user authentication.
      * **Application Level**: The consuming application is responsible for securing its endpoints, managing user identity, and passing user context into the Manglekit pipeline for policy evaluation.
