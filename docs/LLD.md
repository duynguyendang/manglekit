# Manglekit — Low Level Design (LLD)

**Version:** 3.0 (2025-09-30)  
**Audience:** Backend engineers implementing / extending the current Go codebase  
**Status:** Updated draft reflecting recent refactoring: Builder pattern for dependency injection, split retrieval (BM25 lexical + Dense vector + Hybrid combiner), embedding-based reranker, Zap logging, simplified Mangle (rules-only, no separate facts), configurable LLM prompts/token limits, and rule-based post-processing via 'deny' facts.

---

## 1. Runtime Overview

The binary starts via [`cmd/agent/main.go`](cmd/agent/main.go:30-49), initializing Genkit, loading `config/config.yaml` (env-expanded), creating a `Builder` for wiring, building the orchestrator, defining the `answer` flow, and serving at `POST /answer` on `127.0.0.1:8082`. Front-matter parsing extracts metadata from Markdown docs during indexing. [`cmd/agent/main.go`](cmd/agent/main.go:1-83) [`cmd/agent/builder.go`](cmd/agent/builder.go:1-118)

The orchestrator executes the "Sandwich" pipeline with logging:

1. **Intent Parsing:** Genkit flow detects intent (e.g., "troubleshoot", "question") and extracts entities (versions, tickets, products, platforms, artifacts) via regex/NER. [`internal/genintent/parser.go`](internal/genintent/parser.go:33-71) [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:92-98)

2. **Mangle PreProcess:** Normalizes query, tokenizes/asserts facts, evaluates Datalog rules (from `config/mangle/*.dlog`) for expansions (`expanded_query`), filters (`query_filter`). No separate facts file; base facts derived from rules. Generates explanation. [`internal/mangle/processor.go`](internal/mangle/processor.go:62-101) [`internal/mangle/processor.go`](internal/mangle/processor.go:159-257)

3. **Hybrid Retrieval:** Parallel BM25 (lexical, tf-idf via go-nlp) + Dense (LocalVec with GoogleAI embeddings); combines/deduplicates results up to topK, applying filters. Loads/chunks Markdown corpus with front-matter metadata. [`internal/retrieval/bm25.go`](internal/retrieval/bm25.go:39-115) [`internal/retrieval/dense.go`](internal/retrieval/dense.go:25-70) [`internal/retrieval/hybrid.go`](internal/retrieval/hybrid.go:33-65) [`cmd/agent/builder.go`](cmd/agent/builder.go:65-118)

4. **Reranking:** Embeds query/docs via GoogleAI, computes cosine similarities, sorts/retains topK. [`internal/reranker/reranker.go`](internal/reranker/reranker.go:36-89)

5. **Mangle PostProcess:** Asserts user/chunk facts, evaluates rules for `deny` (doc_id → reason); filters out denied chunks, generates explanations. [`internal/mangle/processor.go`](internal/mangle/processor.go:103-157)

6. **Fallback:** If no chunks post-process, returns fixed policy message. No threshold check yet. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:137-146)

7. **LLM Gateway:** Builds configurable prompt (default template, word-based token estimate), streams via OpenAI/Ollama, extracts DocID citations. Limits context by estimated tokens. [`internal/llm/gateway.go`](internal/llm/gateway.go:58-114)

Aggregates explanations (intent, Mangle pre/post); logs via Zap at stages. Constructs RAG query from normalized + expansions + entities. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:89-158) [`internal/logger/logger.go`](internal/logger/logger.go)

---

## 2. Detailed Component Design

### 2.1 Builder & Configuration Loader

* **Location:** [`cmd/agent/builder.go`](cmd/agent/builder.go) [`cmd/agent/main.go`](cmd/agent/main.go)
* **Responsibilities:**
  * `NewBuilder`: Loads `config/config.yaml` (env-expanded), initializes Zap logger. [`cmd/agent/builder.go`](cmd/agent/builder.go:31-47)
  * `Build`: Wires configs (LLM/Mangle into orchestrator), creates GoogleAI embedder, LocalVec retriever/indexes Markdown (front-matter YAML metadata), BM25 (tf-idf corpus), Dense (embedded index), Hybrid combiner, reranker; builds orchestrator. [`cmd/agent/builder.go`](cmd/agent/builder.go:49-96)
  * Front-matter: Parses `---\nYAML\n---\ncontent` for doc metadata during indexing. [`cmd/agent/builder.go`](cmd/agent/builder.go:98-118) [`cmd/agent/main.go`](cmd/agent/main.go:65-83) [`internal/retrieval/bm25.go`](internal/retrieval/bm25.go:156-177)

Configs: `orchestrator` (tokens/threshold), `llm` (provider/model/key/prompt/maxTokens), `embedder.model`, `mangle.rulesFile` (globbing `config/mangle/*.dlog`), `retrieval.path` (Markdown corpus), `retrieval.hybrid.bm25/dense.topK`. [`config/config.yaml`](config/config.yaml)

### 2.2 Orchestrator Core

* **Location:** [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go)
* **Structs / Interfaces:**
  * `Config`: Tokens/threshold, nested LLM/Mangle/Intent/Retrieval (path/hybrid bm25/dense topK)/Rerank (topK). [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:21-29)
  * `orchestrator`: Holds hybrid retriever, doc reranker, LLM gateway, processor, intent parser, config, Zap logger. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:31-40)
* **Creation Path:** Validates Genkit; inits LLM (OpenAI/Ollama), Mangle (rules only), intent (Genkit flow); via Builder. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:51-87) [`cmd/agent/builder.go`](cmd/agent/builder.go:90)
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

`internal/types`: `QueryInput`, `ExpandedQuery` (simplified: normalized/expansions/filters/explanation), `Chunk` (no constraints), `Context` (user only), `Explanation` (added DocID), `Response`, `IntentResult`, interfaces (`Processor`, `Retriever` unused, `BM25Retriever`/`DenseRetriever`, `Gateway`, `Orchestrator`, `IntentParser`), configs (`BM25Config`/`DenseConfig`/`RerankConfig`/`RetrievalConfig`/`LLMConfig` with prompt/tokens). [`internal/types/types.go`](internal/types/types.go:1-131)

---

## 3. Sequence Diagram (Textual)

```
Client → HTTP Handler : POST /answer {query, user_context}
Handler → Builder.Build : config/genkit
Builder → Orchestrator.New : wired deps (retrievers/reranker/llm/processor/intent/logger)
Orchestrator.RunFlow → IntentParser.Parse : input
IntentParser → RunFlow : IntentResult (intent/entities)
RunFlow → Mangle.PreProcess : input + intent
Mangle → RunFlow : ExpandedQuery (normalized/expansions/filters)
RunFlow → constructRAGQuery : normalized + expansions + entities
RunFlow → HybridRetriever.Retrieve (parallel) : ragQuery + filters + configs
BM25 → Hybrid : lexical topK (tf-idf/BM25)
Dense → Hybrid : vector topK (LocalVec/ANN)
Hybrid → RunFlow : combined/deduped docs
RunFlow → Reranker.Rerank : ragQuery + docs + topK
Reranker → RunFlow : cosine-sorted topK docs
RunFlow → Chunk conversion : docs → Chunks
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
* `llm.provider` / `model` / `apiKey` / `promptTemplate` / `maxContextTokens`: LLM + prompt/tokens (env-expanded key).
* `embedder.model`: GoogleAI embedder.
* `mangle.rulesFile`: Globbing for `config/mangle/*.dlog` (main/knowledge/aliases/stopwords/pipelines/retrieval/policies/access).
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