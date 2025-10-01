# Manglekit — Low Level Design (LLD)

**Version:** 3.0 (2025-09-30)  
**Audience:** Backend engineers implementing / extending the current Go codebase  
**Status:** Updated draft reflecting recent refactoring: Builder pattern for dependency injection, split retrieval (BM25 lexical + Dense vector + Hybrid combiner), embedding-based reranker, Zap logging, simplified Mangle (rules-only, no separate facts), configurable LLM prompts/token limits, and rule-based post-processing via 'deny' facts.

---

## 1. Runtime Overview

The binary starts via [`cmd/agent/main.go`](cmd/agent/main.go), loading environment variables from `.env` via `godotenv`. It conditionally initializes Genkit, loading the `googlegenai` plugin only if the LLM or Embedder provider is set to `google`. It then loads `config/config.yaml` (with environment variable expansion), creates a `Builder` for dependency injection, builds the orchestrator, defines the `answer` flow, and serves it at `POST /answer` on `127.0.0.1:8082`. [`cmd/agent/main.go`](cmd/agent/main.go) [`cmd/agent/builder.go`](cmd/agent/builder.go)

The orchestrator executes the "Sandwich" pipeline with logging:

1. **Intent Parsing:** Genkit flow detects intent (e.g., "troubleshoot", "question") and extracts entities (versions, tickets, products, platforms, artifacts) via regex/NER. [`internal/genintent/parser.go`](internal/genintent/parser.go) [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go)

2. **Mangle PreProcess:** Normalizes the query, tokenizes and asserts facts, and evaluates Datalog rules (from `config/mangle/*.dlog`) to generate expansions (`expanded_query`) and filters (`query_filter`). No separate facts file is used; base facts are derived directly from rules. Generates an explanation for the expansion. [`internal/mangle/processor.go`](internal/mangle/processor.go)

3. **Hybrid Retrieval:** In parallel, retrieves documents using BM25 (lexical search) and Dense (vector search with GoogleAI embeddings via LocalVec). The results are combined and deduplicated up to a `topK` limit, applying any filters from the pre-processing stage. [`internal/retrieval/bm25.go`](internal/retrieval/bm25.go) [`internal/retrieval/dense.go`](internal/retrieval/dense.go) [`internal/retrieval/hybrid.go`](internal/retrieval/hybrid.go)

4. **Reranking:** Embeds the query and the retrieved documents using GoogleAI, computes cosine similarities, and sorts the documents to retain the `topK` most relevant ones. [`internal/reranker/reranker.go`](internal/reranker/reranker.go)

5. **Mangle PostProcess:** Converts the reranked documents into `Chunk` objects. It then asserts facts about the user context and each chunk, evaluating rules to identify any chunks that should be denied (`deny`). It filters out denied chunks and generates explanations for the denials. [`internal/mangle/processor.go`](internal/mangle/processor.go)

6. **Fallback:** If no chunks remain after post-processing, the orchestrator returns a fixed policy message. The score-based threshold check is not yet implemented. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go)

7. **LLM Gateway:** Constructs a prompt using a configurable template and the remaining chunks, respecting token limits. It streams the response from the configured LLM (OpenAI or Ollama) and extracts document ID citations from the answer. [`internal/llm/gateway.go`](internal/llm/gateway.go)

Finally, it aggregates explanations from the intent, Mangle pre-processing, and Mangle post-processing stages. Each stage is logged via Zap. The RAG query is constructed from the normalized query, expansion terms, and extracted entities. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go) [`internal/logger/logger.go`](internal/logger/logger.go)

---

## 2. Detailed Component Design

### 2.1 Builder & Configuration Loader

* **Location:** [`cmd/agent/builder.go`](cmd/agent/builder.go), [`cmd/agent/main.go`](cmd/agent/main.go)
* **Responsibilities:**
  * `NewBuilder`: Creates and returns a new `Builder`, initializing the Zap logger. [`cmd/agent/builder.go`](cmd/agent/builder.go)
  * `Build`: Constructs the application components in the correct order:
    1. Manually wires the `LLM` and `Mangle` configs into the main `Orchestrator` config.
    2. Creates the `embedder`.
    3. Initializes `localvec`.
    4. Loads documents from the specified corpus path.
    5. Creates the `bm25Retriever`, `denseRetriever`, and `hybridRetriever`.
    6. Creates the `reranker`.
    7. Finally, creates and returns the `orchestrator` with all dependencies injected. [`cmd/agent/builder.go`](cmd/agent/builder.go)
  * `loadDocuments`: A helper function that walks the document path, reads `.md` files, parses YAML front-matter for metadata, and returns a slice of `ai.Document` objects. [`cmd/agent/builder.go`](cmd/agent/builder.go)

Configs: `orchestrator` (tokens/threshold), `llm` (provider/model/key/prompt/maxTokens), `embedder.model`, `mangle.rulesFile` (globbing `config/mangle/*.dlog`), `retrieval.path` (Markdown corpus), `retrieval.hybrid.bm25/dense.topK`. [`config/config.yaml`](config/config.yaml)

### 2.2 Orchestrator Core

* **Location:** [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go)
* **Structs / Interfaces:**
  * `Config`: Holds configuration for `MaxContextTokens`, `FallbackThreshold`, and nested configs for `LLM`, `Mangle`, `IntentParser`, `Retrieval`, and `Reranker`. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go)
  * `orchestrator`: The main struct holding the injected dependencies: a hybrid retriever, a document reranker, an LLM gateway, a Mangle processor, an intent parser, the application config, and a Zap logger.
  * Internal Interfaces: The orchestrator defines its own internal (un-exported) interfaces for the `hybridRetriever` and `docReranker`, specifying the exact methods it requires. This decouples it from the concrete implementations. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go)
* **Creation Path:** The `New` function initializes and returns an orchestrator. It requires a Genkit runtime, configuration, and instances of a retriever, reranker, and logger. It is responsible for creating the `LLMGateway`, `Mangle Processor`, and `IntentParser`. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go)
* **Execution (`RunFlow`):**
  1. Log start; parse intent/entities. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:91-98)
  2. PreProcess for expansions/filters. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:100-105)
  3. Construct RAG query (normalized + expansions + entities). [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:107-109) [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:161-177)
  4. Parallel hybrid retrieve (BM25 + Dense) with filters/configs. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:111-117)
  5. Rerank docs. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:119-124)
  6. Convert to chunks; post-process with user context. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:126-135)
  7. Fallback if empty; else LLM generate. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:137-154)
  8. Add explanations (post, intent, Mangle pre); log end. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:156-158) [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:179-214)

### 2.3 Rule Processor (Mangle)

* **Location:** [`internal/mangle/processor.go`](internal/mangle/processor.go)
* **Initialization:** Validates rules path (globbing/directories for `.dlog`); parses/stratifies units, evaluates base for derived facts (no separate facts). [`internal/mangle/processor.go`](internal/mangle/processor.go:35-60) [`internal/mangle/processor.go`](internal/mangle/processor.go:159-257)
* **PreProcess:** Normalizes/tokenizes query, asserts facts, evaluates for expansions/filters; collects via `collectStrings`/`collectKeyValue`. Explanation for expansions. [`internal/mangle/processor.go`](internal/mangle/processor.go:62-101) [`internal/mangle/processor.go`](internal/mangle/processor.go:268-368)
* **PostProcess:** Asserts user/chunk facts (doc_id/content/metadata), evaluates for `deny` (doc_id → reason); filters/explains denies. [`internal/mangle/processor.go`](internal/mangle/processor.go:103-157)

Rules in `config/mangle/` (main/knowledge/aliases/stopwords, pipelines/retrieval, policies/access); stratified Datalog. [`config/mangle/`](config/mangle/)

### 2.4 Retrieval Layers

#### 2.4.1 BM25 (Lexical)
* **Location:** [`internal/retrieval/bm25.go`](internal/retrieval/bm25.go)
* **Behavior:** Loads/chunks Markdown (front-matter), builds vocab/tf-idf (go-nlp/tfidf), BM25 scores (k1=2.0, b=0.75) query tokens; filters metadata, topK. [`internal/retrieval/bm25.go`](internal/retrieval/bm25.go:39-115) [`internal/retrieval/bm25.go`](internal/retrieval/bm25.go:156-177)

#### 2.4.2 Dense (Vector)
* **Location:** [`internal/retrieval/dense.go`](internal/retrieval/dense.go)
* **Behavior:** GoogleAI embedder + LocalVec; indexes docs, retrieves topK via ANN, filters metadata. [`internal/retrieval/dense.go`](internal/retrieval/dense.go:25-70)

#### 2.4.3 Hybrid Combiner
* **Location:** [`internal/retrieval/hybrid.go`](internal/retrieval/hybrid.go)
* **Behavior:** Parallel BM25 + Dense (errgroup), dedupes/combines results. Interfaces: BM25Retriever/DenseRetriever. [`internal/retrieval/hybrid.go`](internal/retrieval/hybrid.go:19-65) [`internal/retrieval/types.go`](internal/retrieval/types.go)

### 2.5 Reranker

* **Location:** [`internal/reranker/reranker.go`](internal/reranker/reranker.go)
* **Behavior:** Embeds query/docs (GoogleAI), cosine similarities, sorts topK. [`internal/reranker/reranker.go`](internal/reranker/reranker.go:20-107)

### 2.6 LLM Gateway

* **Location:** [`internal/llm/gateway.go`](internal/llm/gateway.go)
* **Initialization:** OpenAI (compat) or Ollama; supports promptTemplate/maxContextTokens. [`internal/llm/gateway.go`](internal/llm/gateway.go:32-56)
* **Generation:** Word-estimated tokens for context limit; configurable template; streams, extracts unique DocIDs. [`internal/llm/gateway.go`](internal/llm/gateway.go:58-133)

### 2.7 Intent Parser

* **Location:** [`internal/genintent/parser.go`](internal/genintent/parser.go)
* **Behavior:** Genkit flow: keyword/regex intent (troubleshoot/question/etc.), entities (version/ticket/product/platform/artifact). Dedupes/sorts. [`internal/genintent/parser.go`](internal/genintent/parser.go:33-187)

### 2.8 Logging

* **Location:** [`internal/logger/logger.go`](internal/logger/logger.go)
* **Behavior:** Zap production logger; injected via Builder. Used in orchestrator for stages/errors. [`internal/logger/logger.go`](internal/logger/logger.go:8-12) [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:91-158)

### 2.9 Shared Types

* **Location:** [`internal/types/types.go`](internal/types/types.go)
* **Core Structs:**
  * `QueryInput`: Represents the initial user query and user context.
  * `ExpandedQuery`: Captures the output of the Mangle pre-processing stage, including the normalized query, expansion terms, and filters.
  * `Chunk`: Represents a document chunk. While the struct defines fields like `ID`, `DocID`, `Snippet`, `Embedding`, and `Score`, the current orchestrator implementation only populates the `Text` field when converting documents to chunks.
  * `Context`: Holds user context for Mangle post-processing.
  * `Explanation`: A structured record explaining why a decision was made (e.g., a document was denied).
  * `Response`: The final response object containing the answer, citations, explanations, and metadata.
  * `IntentResult`: The output from the intent parsing stage.
* **Interfaces:**
  * `Processor`: Defines the Mangle pre- and post-processing steps.
  * `Retriever`: A generic interface for retrieval that is **currently unused**. The orchestrator uses its own internal interface.
  * `Gateway`: Defines the LLM generation step.
  * `Orchestrator`: Defines the main `RunFlow` method.
  * `IntentParser`: Defines the intent parsing step.
* **Configuration Structs:** Defines various `...Config` structs (`BM25Config`, `DenseConfig`, `RerankConfig`, `RetrievalConfig`, `EmbedderConfig`, `LLMConfig`) used for loading settings from `config.yaml`. [`internal/types/types.go`](internal/types/types.go)

---

## 3. Sequence Diagram (Textual)

```
(Note: The Orchestrator and its dependencies are built and wired once at application startup.)

Client → HTTP Handler : POST /answer {query, user_context}
Handler → Orchestrator.RunFlow : input
Orchestrator.RunFlow → IntentParser.Parse : input
IntentParser → RunFlow : IntentResult (intent/entities)
RunFlow → Mangle.PreProcess : input + intent
Mangle → RunFlow : ExpandedQuery (normalized/expansions/filters)
RunFlow → constructRAGQuery : normalized + expansions + entities
RunFlow → HybridRetriever.Retrieve (parallel) : ragQuery + filters + configs
  BM25 → Hybrid : lexical topK docs
  Dense → Hybrid : vector topK docs
Hybrid → RunFlow : combined/deduped docs ([]string)
RunFlow → Reranker.Rerank : ragQuery + docs + topK
Reranker → RunFlow : cosine-sorted topK docs ([]string)
RunFlow → Convert docs to Chunks
RunFlow → Mangle.PostProcess : chunks + userContext
Mangle → RunFlow : allowed chunks + deny explanations
(if empty) → Client : fallback response + explanations
else
  RunFlow → LLMGateway.Generate : query + chunks (token-limited prompt)
  LLM → RunFlow : Response (answer + citations)
RunFlow → addExplanations : post/intent/mangle-pre
RunFlow → Client : final response + metadata
```

---

## 4. Configuration Surface

`config/config.yaml`:

* `orchestrator.maxContextTokens` / `fallbackThreshold`: Prompt/fallback (threshold unused).
* `llm.provider` / `model` / `apiKey` / `promptTemplate` / `maxContextTokens`: LLM provider (`google`, `ollama`, `openai`), model name, API key (env-expanded), and prompt settings.
* `embedder.provider` / `model`: Embedder provider (`google`) and model name.
* `mangle.rulesFile`: Glob pattern for Datalog rule files (e.g., `config/mangle/*.dlog`).
* `retrieval.path`: Markdown corpus (front-matter metadata).
* `retrieval.hybrid.bm25.topK` / `dense.topK`: Per-retriever limits.
* `reranker.topK`: Rerank limit.

Datalog in `config/mangle/` organizes rules (knowledge/policies/pipelines). 

---

## 5. Error Handling & Observability Hooks

* Startup: Wrapped errors via Builder (config/load/index/retrievers/reranker/orchestrator). [`cmd/agent/builder.go`](cmd/agent/builder.go:49-96)
* Flow: Logs (Zap Info/Warn/Error) at stages; propagates intent/retrieval/rerank/post/LLM errors. Fallback on empty post-chunks. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:91-158)
* Mangle: Errors on parse/eval (deny all on post error). [`internal/mangle/processor.go`](internal/mangle/processor.go:124-131)
* Explanations: Post (deny reasons), intent, Mangle pre; metadata (intent). No rerank explanations yet.

Zap production logging; extend for traces/metrics.

---

## 6. Testing

* `internal/orchestrator/orchestrator_test.go`: End-to-end with mocks (not updated for new retrieval/reranker).
* `internal/mangle/processor_test.go`: Rules loading/parsing (globbing/units).
* `internal/reranker/reranker_test.go`: Cosine rerank (embeddings/scores).
* `internal/retrieval/retrieval_test.go`: BM25/Dense/Hybrid (testdata Markdown).

Recommend: Integration (full flow with corpus), error resilience (empty results), logging assertions.

---

## 7. Extension Points & Open Items

* **Real Embeddings Integration:** Already GoogleAI/LocalVec; add Qdrant for persistent vectors.
* **Advanced Mangle:** Reintroduce facts; dynamic rules reload; more predicates (e.g., redaction).
* **Rerank Enhancements:** Add explanations; cross-encoder (LLM judge); config weights.
* **Fallback/Threshold:** Implement score-based fallback; configurable messages.
* **Observability:** Zap levels/config; Genkit tracers; metrics (retrieval latency/hit rate).
* **Corpus:** Watcher for reloads; non-Markdown support; chunking strategies.
* **Intent:** LLM NER over regex; confidence calibration.
* **Prompts:** Dynamic templates; token counting (tiktoken).

This LLD aligns with the refactored implementation as of 2025-09-30.