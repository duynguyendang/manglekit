# Manglekit SDK — High-Level Architecture

**Version:** 6.0 (Target Architecture)
**Status:** Proposed

---

## 1. Vision & Architectural Principles

### 1.1 Vision

Manglekit is an embeddable Go framework for building **verifiable, neuro-symbolic AI applications**. Its vision is to fuse the power of **Neural** components (via Genkit) for understanding unstructured data with the power of **Symbolic** components (the Mangle Datalog Engine) for imposing logic, controlling workflows, and ensuring reliability.

The goal is not merely to build safe RAG pipelines, but to provide a platform for building sophisticated AI applications where **correctness, explainability, and compliance** are non-negotiable requirements.

### 1.2 Architectural Principles

- **SDK-first, Service-optional:** The primary artifact is a Go library. Optional executables (CLI, services) are thin layers that consume the SDK.
- **Dual Orchestration Models:** Manglekit provides two first-class orchestration models to serve different needs:
    - **Declarative Orchestrator (Default for Flexibility):** Workflows are driven entirely by the **Rule Engine (Mangle)**. This is the most powerful mode, enabling complex, conditional, and concurrent workflows where the LLM is treated as just another tool.
    - **Sandwich Orchestrator (Available for Simplicity):** Provides a fixed, secure RAG flow (`Rules → Retrieve → Rules`) for rapid deployment of common use cases.
- **Orchestrators are Stateless, Workflows are Stateful:** The logical processing engines themselves hold no state between requests, enabling horizontal scaling. However, the framework provides a first-class **State Manager** component to persist the state of long-running workflows or conversational sessions externally.
- **Extensibility via Registry:** All major components—retrievers, embedders, LLMs, state providers—are pluggable and managed via a central registry, preventing vendor lock-in.
- **Fail Fast Construction:** The `Builder` validates configuration and initializes external resources upfront to catch errors before runtime.

---
## 2. System Architecture Diagram

The architecture places the **Declarative Orchestrator** and **Mangle Engine** at the center, acting as the control plane for executing complex workflows.

1.  **Input Processing:** An incoming request, containing a user query and a `Session ID`, is received by the Manglekit service.
2.  **State Loading:** The **Declarative Orchestrator** interacts with the **State Manager** component. Using the `Session ID`, it loads the current context of the workflow or conversation from an external **Persistent Store** (e.g., Redis, PostgreSQL, Firestore).
3.  **Logical Inference:** The loaded context is converted into facts. These facts, along with the user's query, are fed into the **Mangle Engine**. The engine evaluates its Datalog rule sets to determine the next logical step or sequence of steps in the workflow. This can involve conditional branching and identifying tasks that can run in parallel.
4.  **Tool Execution:** The **Orchestrator** executes the plan determined by the Mangle Engine. It invokes the necessary **Tools** from the **Registry**. These tools are not limited to AI components and can include:
    -   **Neural Tools (via Genkit):**
        -   **Retriever:** To efficiently fetch candidate documents from a large corpus.
        -   **LLM:** As a specialized tool for language tasks (summarization, entity extraction, natural language generation).
    -   **Custom Tools:** Any user-defined Go function, such as calling an internal API or performing a database lookup.
5.  **State Update & Response:** After the tools have been executed, the **Orchestrator** sends the updated context to the **State Manager**, which persists it back to the external store. The final result is formatted and returned to the user.

---
## 3. Core Components

This section details the primary components and their concrete implementations available within the Manglekit SDK.

- **Builder & SDK:** The entrypoint for configuring and instantiating a Manglekit pipeline, either programmatically (`builder.BuilderAPI`) or via YAML (`config.NewBuilderFromYAML`). It manages the lifecycle of all resources via `core.ResourceCloser`.

- **Registry:** A central service locator for registering and retrieving providers (LLMs, Retrievers, State Providers, etc.). It is the foundation of the plug-and-play architecture.

- **Orchestrators:**
    - **Declarative Orchestrator:** The primary control plane. It interprets Datalog rules (`flow_stage`, `stage_tool`, `concurrent_group`) to build and execute a dynamic execution graph. It is responsible for interacting with the State Manager and managing complex, stateful workflows.
    - **Sandwich Orchestrator:** A simpler, stateless implementation that executes a fixed sequence of stages (`Rules → Retrieve → Rules`). It is a specific use case of the more general declarative pattern.

- **Mangle Provider:** Integrates the Mangle Datalog engine, acting as the central component for both **Rule Execution** and **Fact Management**. Its responsibilities include:
    -   **Rule Execution:** Loading rule sets and performing high-speed logical inference to drive workflow decisions.
    -   **Fact Management:** Dynamically converting application context (user queries, retriever results, workflow state from the State Manager) into a transient set of facts for the engine to evaluate during a single request-response cycle.

- **Neural Providers (via Genkit):**
    -   **Retrievers:** Responsible for fetching candidate documents. The SDK provides several built-in retriever types to support diverse search strategies:
        -   `bm25`:** A sparse retriever based on the BM25 algorithm, excellent for keyword-based matching.
        -   `dense`:** A dense retriever that performs semantic search over vector embeddings. It requires an `Embedder` and a `Vector Store`.
        -   `hybrid`:** A sophisticated retriever that combines the results from both sparse (`bm25`) and dense retrievers, using a **Reciprocal Rank Fusion (RRF)** algorithm to merge the results for superior accuracy.
    -   **Rerankers:** Refine and re-order the candidate documents returned by a retriever to improve relevance before they are passed to the LLM.
        -   `cosine`:** A lightweight reranker that re-scores documents based on the cosine similarity between the query embedding and the document embeddings.
        -   **(Future) `llm`:** A more powerful (but slower and more expensive) reranker that uses an LLM to evaluate the relevance of each document to the query.
    -   **Embedders:** Generate vector embeddings for text. Providers for major services are included:
        -   `google` (e.g., Gemini embeddings)
        -   `openai` (e.g., text-embedding-ada-002)
    -   **LLMs:** Generate text and perform language tasks. Providers for major services are included:
        -   `google` (Gemini family)
        -   `openai` (GPT family)
        -   `groq` (for high-speed inference)
    -   **Vector Stores:** Store and search over vector embeddings for the `dense` retriever.
        -   `localvec` (in-memory/on-disk): A simple, file-based vector store for rapid prototyping and small-scale applications.
        -   **(Future) Enterprise Connectors:** Adapters for popular vector databases like Pinecone, Chroma, and PostgreSQL/pgvector.

- **State Manager (New Core Component):**
    - **Description:** A first-class component responsible for persisting the state of long-running workflows or conversational sessions. It is the key to enabling stateful applications on a stateless server architecture.
    - **Interface:** Defines a `StateProvider` interface (`LoadContext`, `SaveContext`) to abstract away the underlying storage mechanism.
    - **Implementations:** The framework will provide sample implementations, such as `InMemoryStateProvider` (for testing and simple cases) and `RedisStateProvider` (for production environments).

---
## 4. Observability

Observability is a first-class citizen in Manglekit, designed to provide deep insights into the performance and behavior of AI pipelines. The `core.Observability` struct is the central point for configuring these features.

- **Structured Logging:** The framework integrates with a structured logging library (e.g., `zap`). When configured, all components from orchestrators to providers will emit detailed logs with consistent fields (e.g., `trace_id`, `component`, `duration_ms`), allowing for easier filtering and analysis in production. If unconfigured, it defaults to a standard logger for ease of use in examples.
- **Distributed Tracing:** Manglekit supports OpenTelemetry. When a tracer is provided, the orchestrator will create a parent span for each request and propagate it through all stages. Each major operation (retrieval, reranking, rule evaluation, LLM call) creates its own child span, enabling operators to visualize the entire workflow, identify bottlenecks, and debug complex interactions.
- **Metrics:** Key performance indicators (KPIs) are exposed via a metrics interface compatible with systems like Prometheus. These include:
    -   **Latency:** Timings for each stage (`mangle.rules_pre.ms`, `retrieval.ms`, `llm.ms`).
    -   **Counters:** Total requests, number of errors by type (`err_no_evidence`, `err_denied`).
    -   **Gauges:** Token usage per model.
- **Explainability:** Beyond traditional observability, `Answer.Meta` is populated with rich metadata for auditing and explainability, including which source documents were retrieved, which rules were fired, and the final scores from rerankers.

---
## 5. Resource Lifecycle Management

Manglekit is designed for long-running services and guarantees graceful shutdown of all external resources. This is managed through a clear and consistent lifecycle pattern.

- **`core.ResourceCloser` Interface:** A simple interface (`Close() error`) that any component managing an external resource (like a network client or a database connection) must implement.
- **Builder's Role:** The `Builder` (`builder.BuilderAPI`) acts as the central owner of the application's lifecycle. During the construction phase, as it initializes various providers (e.g., Google client, `localvec` store), it checks if they implement `core.ResourceCloser`.
- **Accumulation and Shutdown:** If a provider is a `ResourceCloser`, the builder adds it to an internal list. The `Builder.Build()` method returns not only the orchestrator but also a final, aggregated `Close()` function. The application's `main` function is responsible for calling this single `Close()` function upon receiving a shutdown signal (e.g., `SIGINT`). This ensures that all registered resources are terminated cleanly and in a predictable order, preventing leaks.

---
## 6. Usage Patterns

- **Secure RAG Applications (via Sandwich Orchestrator):** For internal copilots or search tools requiring strict policy adherence on a single request-response cycle.
- **Complex Business Workflows (via Declarative Orchestrator):** Automating multi-step, conditional processes like claims processing, security incident response, or dynamic UI generation where the workflow logic is defined in Datalog, not code.
- **Stateful Conversational Agents (via Declarative Orchestrator + State Manager):** Building intelligent chatbots that can remember context, manage long conversations, and perform complex tasks over multiple user interactions.
- **Controlled Search & Filter (Rules → Retrieve → Rules):** Using Manglekit without an LLM to build a powerful, policy-aware search engine that returns filtered and verified source documents directly to the user.
- **Standalone Logic Engine (via Mangle Provider only):** Leveraging the high-performance Mangle engine for business logic tasks completely unrelated to AI, providing a unified technology stack.

---
## 7. Non-Functional Requirements

- **Performance:**
    - **Concurrency:** The Declarative Orchestrator must support the parallel execution of independent workflow stages (`errgroup`) to reduce I/O-bound latency.
    - **Latency:** The architecture encourages smaller, specialized models to improve overall response times.
- **Scalability:**
    - **Horizontal Scaling:** The stateless nature of the orchestrators allows for easy replication of service instances to handle high load. All shared state is externalized via the `State Manager`.
- **Security & Compliance:** Policies are explicitly encoded in auditable Datalog rules. Hallucination is prevented by grounding LLM responses in retrieved source data, a process that can be verified by post-retrieval rules.