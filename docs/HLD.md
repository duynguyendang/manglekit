# Manglekit SDK — High-Level Architecture

**Version:** 5.0 (Current Implementation)  
**Status:** Maintained

## 1. Vision & Architectural Principles

### 1.1 Vision

Manglekit is an embeddable Go framework for building verifiable, policy-aware retrieval-augmented experiences. It fuses Google’s Mangle Datalog engine with Genkit-powered neural components so that every answer is routed through programmable rules before and after retrieval and generation.

### 1.2 Architectural Principles

- **SDK-first, Service-optional:** The primary artifact is a Go library. Optional executables (CLI demos or custom hosts) are thin layers that consume the SDK.
- **Sandwich by Default:** The core orchestrator always runs Mangle-Pre → Retrieval/Rerank → Mangle-Post → LLM to keep guardrails wrapped around neural stages.
- **Provider-Agnostic Extensibility:** Every major capability—retrievers, rerankers, embedders, vector stores, LLM clients, schema parsers—is selected via a registry so teams can swap implementations without touching pipeline code.
- **Fail Fast Construction:** The fluent builder resolves configuration, validates dependencies, and creates external clients up front so misconfiguration never reaches runtime.
- **Stateless Pipelines, Explicit Resources:** Orchestrators keep no mutable state. External handles (Genkit clients, OpenAI clients, vector stores) are owned by providers and closed through `core.ResourceCloser` hooks.
- **Declarative Hooks Everywhere:** Rules, schema imports, orchestration flows, and provider wiring can all be described declaratively (YAML + Datalog) to enable runtime reconfiguration.

---

## 2. System Architecture & Boundaries (C4 Level 2)

Manglekit is split into an SDK core, a provider ecosystem, and the consuming application. Applications can load configuration from YAML, register custom providers, or instantiate components programmatically.

```mermaid
graph TD
    subgraph "Application Layer"
        App[Go App / Genkit Flow / CLI]
        Config[config.yaml | typed With* calls]
        RulesFiles[Rules & Facts (.dlog, .ttl, .md)]
    end

    subgraph "Manglekit SDK"
        Builder[Builder API & Config Loader]
        Registry[Provider Registry]
        Options[core.Options & Observability]
        subgraph "Orchestrators"
            Sandwich[Sandwich Pipeline]
            Declarative[Declarative Orchestrator]
        end
    end

    subgraph "Provider Implementations"
        Rules[Mangle RuleSet & FlowController]
        Retriever[Retrievers (bm25, dense, hybrid, in-memory)]
        Reranker[Cosine Reranker]
        Embedder[Embedders (Google, OpenAI)]
        VectorStore[Vector Store (localvec)]
        LLM[LLM Clients (Google, OpenAI, Groq)]
        Schema[Schema Parsers (JSON Schema, RDF)]
    end

    subgraph "External Systems"
        Genkit[Genkit runtime & transformers]
        LLMAPI["LLM APIs"]
        VectorDB["Vector DBs / Local Storage"]
        Data["Markdown, RDF, Custom Corpora"]
    end

    App --> Builder
    Builder -- reads --> Config
    Builder -- resolves --> RulesFiles
    Builder -- uses --> Registry
    Builder -- produces --> Options
    Options --> Sandwich
    Options --> Declarative
    Sandwich --> Rules
    Sandwich --> Retriever
    Sandwich --> Reranker
    Sandwich --> LLM
    Declarative --> Rules
    Declarative --> Retriever
    Declarative --> LLM
    Retriever --> Embedder
    Retriever --> VectorStore
    Reranker --> Embedder
    Rules --> Schema
    Embedder --> Genkit
    LLM --> Genkit
    LLM --> LLMAPI
    VectorStore --> VectorDB
    Retriever --> Data
```

---

## 3. Component Breakdown & Interaction Patterns

### 3.1 Builder, Configuration, and Registry

- `manglekit.NewBuilder()` exposes type-safe `With*` methods for programmatic assembly. Each call stores a provider name plus typed options.
- `NewBuilderFromYAML` parses `config.yaml`, resolves relative paths, expands environment variables, and seeds the builder’s state. Declarative flows are described through the `tools` map and `orchestrator` block.
- The builder consults `config.Providers` to initialize external clients (Google Genkit, OpenAI/Groq). Created clients register a `ResourceCloser` so orchestrators can cleanly shut them down.
- Providers register constructors via `manglekit.Register*` during their `init()` functions. The builder looks up constructors in the global registry, injects typed options, and performs strict signature assertions.
- Dependency ordering is handled automatically: the builder constructs embedders first, then vector stores, then retrievers, rerankers, rules, and finally the orchestrator.
- Successful builds produce a populated `core.Options` struct that captures the chosen components, numeric knobs (`TopK`, `MaxTokens`, `FallbackThreshold`), and observability hooks.

### 3.2 Standard Provider Set

A single blank import `github.com/duynguyendang/manglekit/providers/all` registers:

- **Retrievers:** in-memory, BM25 (markdown front matter aware), dense (Genkit embedder + vector store), and hybrid (BM25 + dense via reciprocal rank fusion).
- **Rerankers:** cosine similarity reranker backed by the configured embedder.
- **Embedders:** Google and OpenAI implementations that rely on Genkit or `openai-go` clients.
- **Vector Store:** `localvec`, a lightweight local filesystem-backed store.
- **LLM Clients:** Google (Genkit), OpenAI, and Groq (OpenAI-compatible) clients with prompt templating.
- **Rules:** the Mangle engine implementing both `core.RuleSet` and `core.FlowController`.
- **Schema Parsers:** JSON Schema and RDF parsers that emit Mangle facts.

Custom providers can register alternative constructors without editing core packages.

### 3.3 Orchestrators

- **Sandwich (`pipeline.Sandwich`):** The default orchestrator. It enforces the rule-wrapped flow:
  1. Evaluate pre-rules (`core.RuleSet.Evaluate(core.Pre)`) to normalize the query, apply filters, or deny the request.
  2. Call the retriever with `retrieve.Request{Query, TopK, Meta}`. Filters and expansion terms are passed via `Meta`.
  3. Optionally rerank results with `rerank.Reranker`. The cosine implementation embeds the query and documents in parallel (`errgroup`) and trims to the configured `TopK`.
  4. Enforce the fallback threshold before invoking the LLM to avoid low-confidence responses.
  5. Invoke the LLM client with a prompt constructed by `llm.PromptBuilder`, capturing token usage in `Answer.Meta`.
  6. Execute post-rules (`core.Post`) to filter citations or redact output before returning the final `core.Answer`.
  Observability spans/logs/metrics flow through `core.Observability`. Resource closers run in LIFO order when `Close` is called.
- **Declarative (`pipeline/declarative`):** Interprets orchestration steps from Datalog facts (`flow_stage/3`, `stage_tool/2`). The builder injects a map of named tools (retrievers, rerankers, LLMs, custom utilities). Pre-rules can flag stages to skip, mutate the query, or deny the request. Each stage operates on a shared context map (`query`, `docs`, `answer`, `meta`). This enables complex conditional flows without recompiling code.

### 3.4 Rules Engine & Fact Management

- The Mangle provider consumes `core.MangleOptions`: rule paths, schema sources, converter lists, and a `FileFirst` toggle deciding whether declarations come from `.dlog` files or converters.
- Built-in converters (query, user context, documents) are enabled via `DefaultConverters` and can be augmented with custom converters registered in the component registry.
- Schema sources such as JSON Schema or RDF are parsed into facts and declarations so that rules can reason about ontologies and typed relationships.
- Rules are stratified once at initialization using Mangle analysis APIs and evaluated against an in-memory fact store (`factstore.SimpleInMemoryStore`). Runtime evaluations clone the base store, add transient facts from converters, and execute the requested stage.
- Because the provider implements `core.FlowController`, the same engine powers both rule stages and declarative flow queries.

### 3.5 Retrieval, Embedding, Ranking, and Generation

- **BM25 Retriever:** Indexes Markdown documents (with YAML front matter metadata) and scores via Okapi BM25. Scores are attached to `Doc.Meta["score"]`.
- **Dense Retriever:** Uses a Genkit embedder to embed the query and delegates vector search to the configured vector store. Metadata filters emitted by pre-rules pass through to the store.
- **Hybrid Retriever:** Executes sparse and dense retrieval concurrently and fuses rankings using Reciprocal Rank Fusion.
- **Localvec Vector Store:** Stores embeddings on disk, supporting metadata filters and streaming large corpora without external dependencies.
- **Cosine Reranker:** Embeds query/documents in parallel, computes cosine similarity, and returns scored documents for downstream use.
- **LLM Clients:** Wrap provider SDKs, share a thread-safe `PromptBuilder`, and expose token usage via `llm.Response.Usage`. Google models run through Genkit’s plugin, OpenAI/Groq use `openai-go`.
- **Embedders:** Built atop Genkit (Google) or OpenAI APIs. The builder ensures the same embedder instance can be reused by retrievers and rerankers.

### 3.6 Observability & Lifecycle

- `core.Observability` carries optional logger, tracer, and meter interfaces. The Sandwich orchestrator emits lifecycle events and timing metrics if hooks are provided.
- `core.Options.ResourceClosers` ensures all external clients (Genkit, GenAI, OpenAI) are closed exactly once. The builder registers closers as it constructs providers.
- Metadata emitted by providers (retrieval scores, rule traces, token usage) is collected in `Answer.Meta` for downstream auditing.

---

## 4. Usage Patterns

- **Sandwich SDK Mode:** Import `providers/all`, instantiate the builder (`NewBuilder()` or `NewBuilderFromYAML`), call `Build()`, then run queries through `core.Orchestrator.Run`. This is the default path for HTTP services or CLI tools.
- **Declarative Flow Mode:** Provide a `config.yaml` with `tools` and select `orchestrator.type: declarative`. Author orchestration logic in Datalog (`flow_stage/3`, `stage_tool/2`, `stage_param/3`) to direct stage execution, including custom tools that invoke domain-specific code.
- **Rules-first Utilities:** For lightweight policy checks (e.g., `apps/rdf-knowledge-base`), applications can construct the Mangle rules provider directly via the registry and call `RuleSet.Evaluate` without bringing up the full orchestrator.

---

## 5. Non-Functional Requirements

- **Performance:** Retrieval and embedding workloads use concurrency (`errgroup`) to keep latency low. The stateless orchestrators allow horizontal scaling; caching lives in provider implementations such as vector stores.
- **Scalability:** Because orchestrators hold no mutable state, they can be cloned per request or shared across goroutines. External services (vector DBs, LLMs) should be scaled independently.
- **Security & Compliance:** Sensitive information is enforced through Mangle post-rules and converter redactions. Secrets (API keys) stay outside code, pulled from environment variables at build time.
- **Observability:** Providers and orchestrators expose metrics (e.g., `manglekit.rules_pre_ms`, retrieval timings) and structured logs so operators can audit decisions. Token usage and rule outcomes are surfaced in the response metadata for downstream analytics.

---

This document reflects the current implementation of Manglekit’s architecture, aligning the high-level design with the code in the repository.
