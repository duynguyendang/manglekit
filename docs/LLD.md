# Manglekit — Low Level Design (LLD)

**Version:** 1.0 (2025-09-24)  
**Audience:** Backend engineers implementing / extending the current Go codebase  
**Status:** Draft derived from the repository's checked-in implementation

---

## 1. Runtime Overview

The binary is started via `cmd/agent/main.go`. It bootstraps a Genkit instance, loads YAML configuration, wires the orchestrator with rule-engine and LLM settings, instantiates the RAG layer, and exposes a Genkit HTTP flow at `POST /answer` on `127.0.0.1:8082`.【F:cmd/agent/main.go†L26-L57】 Environment variables embedded in the YAML are expanded prior to unmarshalling so that API credentials can be injected at runtime.【F:cmd/agent/main.go†L59-L72】

At runtime the orchestrator executes the "Sandwich" pipeline:

1. **Mangle PreProcess** normalizes and expands queries using Datalog rules and fact stores.【F:internal/mangle/processor.go†L70-L108】
2. **RAG Retrieval** executes a Genkit `localvec` retriever seeded with markdown documents from the configured path and returns matching text spans.【F:internal/orchestrator/orchestrator.go†L61-L76】【F:internal/rag/rag.go†L37-L88】
3. **Mangle PostProcess** currently passes chunks through unchanged but is the hook for output policy enforcement.【F:internal/orchestrator/orchestrator.go†L78-L82】【F:internal/mangle/processor.go†L111-L115】
4. **LLM Gateway** renders a prompt that embeds retrieved context and streams the final answer while aggregating citations.【F:internal/orchestrator/orchestrator.go†L78-L93】【F:internal/llm/gateway.go†L49-L102】

The orchestrator also preserves an explanation for pre-processing actions so downstream consumers can trace rule firings.【F:internal/orchestrator/orchestrator.go†L84-L91】

---

## 2. Detailed Component Design

### 2.1 Configuration Loader

* **Location:** `cmd/agent/main.go`
* **Responsibilities:**
  * Initialize the Genkit runtime and HTTP mux.【F:cmd/agent/main.go†L26-L56】
  * Load `config.yaml`, expand environment variables, and hydrate `AppConfig` with orchestrator, LLM, RAG, and Mangle sub-structures.【F:cmd/agent/main.go†L19-L72】
  * Copy the global LLM and Mangle configuration into the orchestrator block before instantiation so the orchestrator receives a single `Config` payload.【F:cmd/agent/main.go†L35-L49】
  * Instantiate the Genkit flow `answer` wrapping `Orchestrator.RunFlow` and serve it through `github.com/firebase/genkit/go/plugins/server`.

### 2.2 Orchestrator Core

* **Location:** `internal/orchestrator/orchestrator.go`
* **Structs / Interfaces:**
  * `Config` exposes `MaxContextTokens`, `FallbackThreshold`, nested `LLMConfig`, and nested Mangle config for dependency injection.【F:internal/orchestrator/orchestrator.go†L14-L20】
  * `orchestrator` struct holds a rule processor, the LLM gateway, a `ragRetriever`, and (currently unused) `types.Retriever` for hybrid search.【F:internal/orchestrator/orchestrator.go†L22-L51】
* **Creation Path:** `New` builds the LLM gateway via `internal/llm`, the Mangle processor via `internal/mangle`, and stores the injected retriever handles. Failure to load either dependency aborts startup with contextualized errors.【F:internal/orchestrator/orchestrator.go†L34-L52】
* **Execution (`RunFlow`):**
  1. Call `Processor.PreProcess` to normalize the query and derive expansions / filters.【F:internal/orchestrator/orchestrator.go†L55-L59】
  2. Merge the normalized text and expansion terms to form the retrieval query string.【F:internal/orchestrator/orchestrator.go†L61-L64】
  3. Delegate to the injected `ragRetriever` (from `internal/rag`) to fetch raw text passages.【F:internal/orchestrator/orchestrator.go†L66-L76】
  4. Wrap each passage in a `types.Chunk` to preserve a uniform interface for the downstream gateway.【F:internal/orchestrator/orchestrator.go†L71-L76】
  5. Pass chunks through `Processor.PostProcess`; the current implementation returns them unchanged but keeps the contract for future policy enforcement.【F:internal/orchestrator/orchestrator.go†L78-L82】【F:internal/mangle/processor.go†L111-L115】
  6. Invoke the LLM gateway with the original user prompt and post-processed chunks, propagating any error back to the caller.【F:internal/orchestrator/orchestrator.go†L78-L83】
  7. Append a `mangle-pre` explanation to the response when the pre-processor produced a non-empty explanation string.【F:internal/orchestrator/orchestrator.go†L84-L91】

> **Note:** Although the orchestrator stores an injected `types.Retriever`, the current `RunFlow` implementation does not call it. The mock retriever remains available for future hybrid retrieval steps and is exercised via unit tests to validate the pre-processing output.【F:internal/orchestrator/orchestrator.go†L22-L51】【F:internal/retrieval/mock.go†L10-L31】

### 2.3 Rule Processor (Mangle)

* **Location:** `internal/mangle/processor.go`
* **Initialization:**
  * Validates that both rules and facts paths are configured.【F:internal/mangle/processor.go†L36-L44】
  * Parses the Datalog program, stratifies predicates, and loads all fact files (including directory walks and glob patterns).【F:internal/mangle/processor.go†L46-L198】
  * Evaluates the program once at startup so derived facts are cached in `baseFactStore`.【F:internal/mangle/processor.go†L56-L67】
* **PreProcess:**
  * Lowercases and trims the query, merges static facts into a working store, tokenizes the normalized query into unique alphanumeric terms, and asserts each token as `query_token` facts.【F:internal/mangle/processor.go†L72-L104】【F:internal/mangle/processor.go†L182-L199】
  * Adds `raw_query` and `normalized_query` atoms before re-evaluating the program to derive expansions (`expanded_query`) and policy filters (`query_filter`).【F:internal/mangle/processor.go†L82-L107】
  * Returns a `types.ExpandedQuery` with the normalized string, sorted expansion terms, collected filters, and an explanation reflecting whether any expansions fired.【F:internal/mangle/processor.go†L89-L108】
* **PostProcess:** presently a passthrough, but returns both chunks and a placeholder for explanation metadata to satisfy the interface.【F:internal/mangle/processor.go†L111-L115】

The rules (`rules.dlog`) declare alias relationships and default filters, while fact files (e.g., `data/aliases.facts`, `data/policies.facts`) seed the knowledge base for expansions and metadata defaults such as `visibility=public`.【F:rules.dlog†L1-L13】【F:data/aliases.facts†L1-L8】【F:data/policies.facts†L1-L8】

### 2.4 Retrieval Layers

#### 2.4.1 Genkit `localvec` RAG
* **Location:** `internal/rag/rag.go`
* **Behavior:**
  * Initializes Google Vertex-compatible embedder via `googlegenai.GoogleAIEmbedder` and ensures the `localvec` plugin is ready.【F:internal/rag/rag.go†L37-L46】
  * Defines an in-process vector retriever and loads markdown documents from the configured directory, indexing them into the vector store during startup.【F:internal/rag/rag.go†L48-L69】【F:internal/rag/rag.go†L90-L110】
  * `Retrieve` wraps the user query as a document request and calls `genkit.Retrieve` with `K=2`, returning the raw text for each retrieved chunk, in the order provided by the plugin.【F:internal/rag/rag.go†L72-L88】

#### 2.4.2 Mock Retriever
* **Location:** `internal/retrieval/mock.go`
* **Purpose:** Supplies deterministic chunks for tests or offline usage. `Search` ignores inputs and returns two hardcoded chunks describing Manglekit, each pre-populated with IDs and doc identifiers.【F:internal/retrieval/mock.go†L18-L31】

### 2.5 LLM Gateway

* **Location:** `internal/llm/gateway.go`
* **Initialization (`New`):** Selects a provider based on `types.LLMConfig`. For OpenAI it configures the compatibility plugin with API key options; for Ollama it initializes a new Genkit runtime with the Ollama plugin. Unsupported providers return explicit errors and a nil gateway.【F:internal/llm/gateway.go†L22-L47】
* **Generation Path:** Builds a templated prompt consisting of bulletized context followed by the original question, invokes `model.Generate` with a streaming callback to accumulate the final answer, and derives citations by unique document IDs from the chunk metadata.【F:internal/llm/gateway.go†L49-L102】 Error cases include failing to create the provider model or runtime generation failures, both returned to the orchestrator.

### 2.6 Shared Types

`internal/types` centralizes data contracts used between layers: query payloads, expanded query outputs, retrieval chunks, contextual metadata, explanations, responses, and interface definitions for processors, retrievers, gateways, and orchestrators.【F:internal/types/types.go†L10-L88】 These types are consumed across the orchestrator, rule processor, retriever, and gateway modules to keep dependencies loosely coupled.

---

## 3. Sequence Diagram (Textual)

```
Client → HTTP Handler : POST /answer
HTTP Handler → Orchestrator.RunFlow : *types.QueryInput
Orchestrator → Mangle.PreProcess : input
Mangle → Orchestrator : *types.ExpandedQuery
Orchestrator → RAG.Retrieve : normalized+expanded query string
RAG → Orchestrator : []string (context passages)
Orchestrator → Mangle.PostProcess : []*types.Chunk
Mangle → Orchestrator : []*types.Chunk (policy-adjusted)
Orchestrator → LLM.Generate : (user prompt, chunks)
LLM → Orchestrator : *types.Response
Orchestrator → Client : JSON response with answer/citations/explanations
```

---

## 4. Configuration Surface

Primary runtime knobs live in `config.yaml`:

* `orchestrator.maxContextTokens` and `orchestrator.fallbackThreshold` gate prompt assembly and fallback policy evaluation downstream (placeholders for future use).【F:config.yaml†L1-L3】
* `llm.provider`, `llm.model`, and `llm.apiKey` select and authenticate the LLM gateway (API key expanded from environment).【F:config.yaml†L5-L8】
* `rag.embedder.model` and `rag.retriever.path` configure the local vector store and document corpus; the loader walks the directory to ingest `.md` files.【F:config.yaml†L18-L22】【F:internal/rag/rag.go†L57-L110】
* `mangle.rulesFile` and `mangle.factsFile` point to the Datalog program and fact base consumed during processor initialization.【F:config.yaml†L24-L26】【F:internal/mangle/processor.go†L36-L67】

A top-level `retrieval` block also exists for the future hybrid retriever implementation (currently unused by production code).【F:config.yaml†L10-L16】

---

## 5. Error Handling & Observability Hooks

* Startup failures in configuration, RAG initialization, or orchestrator wiring terminate the process via `log.Fatalf`, ensuring misconfiguration is surfaced immediately.【F:cmd/agent/main.go†L30-L56】
* Mangle processor wraps errors with contextual messages (`load mangle program`, `evaluate query program`) making troubleshooting around facts/rules explicit.【F:internal/mangle/processor.go†L46-L108】
* Orchestrator propagates retrieval and generation errors directly to the caller, preserving the original context.【F:internal/orchestrator/orchestrator.go†L55-L83】
* The response explanation channel appends metadata about pre-processing which downstream services can log or expose for traceability.【F:internal/orchestrator/orchestrator.go†L84-L91】

Telemetry, metrics, and structured logging are not yet implemented; integration points include the orchestrator boundaries and the Genkit flow instrumentation.

---

## 6. Testing

`internal/orchestrator/orchestrator_test.go` covers the `RunFlow` happy path using a real Mangle processor (backed by repo facts/rules), a mock RAG retriever, and a mock LLM gateway. The test asserts:

* Token expansions include aliases such as "approximate nearest neighbor" for "ann" and "best matching 25" for "bm25".【F:internal/orchestrator/orchestrator_test.go†L16-L45】
* Default filters (e.g., `visibility=public`) are emitted by Mangle.【F:internal/orchestrator/orchestrator_test.go†L41-L49】
* The orchestrator forwards the composed RAG query string and retrieved chunks to the gateway, and the final response carries the mock answer plus explanations.【F:internal/orchestrator/orchestrator_test.go†L51-L88】

Additional unit coverage is recommended for error scenarios (retrieval failure, LLM failure), PostProcess logic once implemented, and the RAG indexing helper.

---

## 7. Extension Points & Open Items

* **Hybrid Retrieval Integration:** `types.Retriever` and the mock implementation are wired but unused; future work should merge Mangle-derived filters into a true hybrid search call within `RunFlow`.
* **PostProcess Policies:** Implement rule-based redaction and metadata filtering via Mangle facts, returning explanation records rather than `nil`.
* **Observability:** Add structured logging and tracing around Genkit flow execution, retrieval timings, and LLM calls.
* **Configuration Validation:** Enforce limits for `MaxContextTokens` and guard against empty document corpora before indexing.
* **LLM Prompt Safety:** Incorporate token counting and fallback text generation triggered by `FallbackThreshold` once scoring is available.

This LLD reflects the repository state as of commit time and should be updated alongside functional changes to keep the implementation and design artifacts aligned.
