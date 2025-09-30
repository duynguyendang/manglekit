# Manglekit — Low Level Design (LLD)

**Version:** 2.0 (2025-09-29)  
**Audience:** Backend engineers implementing / extending the current Go codebase  
**Status:** Updated draft reflecting the current repository implementation, including hybrid retrieval, intent parsing, reranking, and enhanced post-processing.

---

## 1. Runtime Overview

The binary is started via [`cmd/agent/main.go`](cmd/agent/main.go:26-56). It initializes a Genkit instance, loads and expands `config/config.yaml` (injecting environment variables like API keys), wires the orchestrator with LLM, Mangle, retrieval, and reranker configurations, and exposes a Genkit HTTP flow at `POST /answer` on `127.0.0.1:8082` using the server plugin.

At runtime, the orchestrator executes the enhanced "Sandwich" pipeline:

1. **Intent Parsing** uses a Genkit flow to detect user intent (e.g., "question", "troubleshoot") and extract entities (e.g., versions, platforms) via regex-based NER. [`internal/genintent/parser.go`](internal/genintent/parser.go:33-71) [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:76-80)

2. **Mangle PreProcess** normalizes the query, tokenizes it, asserts intent entities as facts, and derives expansions (aliases like "bm25" → "best matching 25"), normalized terms, stopwords, term constraints (must/should), entities, and metadata constraints using Datalog rules and facts. Explanations are generated for expansions. [`internal/mangle/processor.go`](internal/mangle/processor.go:74-164)

3. **Hybrid Retrieval** performs lexical (BM25 with must/should terms) and dense (localvec + Google AI embeddings) search over a chunked Markdown corpus, applying metadata filters (visibility, tenant) from Mangle. Returns scored candidates up to `topK`. [`internal/retrieval/hybrid.go`](internal/retrieval/hybrid.go:85-218) [`internal/retrieval/config.go`](internal/retrieval/config.go)

4. **Reranking** applies multi-dimensional reranking (MRL) using cosine similarity on multiple embedding dimensions, re-scoring and limiting candidates to `topK`, with explanations for retained/dropped chunks. [`internal/retrieval/reranker.go`](internal/retrieval/reranker.go:31-127)

5. **Mangle PostProcess** filters reranked chunks by visibility, tenant, metadata constraints; redacts sensitive content if flagged. Returns filtered chunks and explanations for drops/modifications. [`internal/mangle/processor.go`](internal/mangle/processor.go:167-291)

6. **Fallback Check** triggers if no chunks remain or top chunk score < `fallbackThreshold`, returning a policy message without LLM invocation. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:110-123)

7. **LLM Gateway** assembles a prompt with cited context chunks (limited by `maxContextTokens` via word count), generates a streaming response via OpenAI-compatible or Ollama, and extracts unique citations by DocID. [`internal/llm/gateway.go`](internal/llm/gateway.go:50-71) [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:125-129)

The orchestrator aggregates explanations from all stages (intent, Mangle pre/post, rerank) and attaches metadata (intent, expanded query) to the response. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:132-137)

---

## 2. Detailed Component Design

### 2.1 Configuration Loader

* **Location:** [`cmd/agent/main.go`](cmd/agent/main.go)
* **Responsibilities:**
  * Initialize Genkit runtime and HTTP mux. [`cmd/agent/main.go`](cmd/agent/main.go:26-28)
  * Load `config/config.yaml`, expand env vars (e.g., `${OPENAI_API_KEY}`), unmarshal into `AppConfig` with orchestrator, LLM, retrieval (corpus/hybrid/rerank), and Mangle sub-structs. [`cmd/agent/main.go`](cmd/agent/main.go:29-32) [`cmd/agent/main.go`](cmd/agent/main.go:58-70)
  * Wire global LLM/Mangle configs into orchestrator config for unified injection. [`cmd/agent/main.go`](cmd/agent/main.go:35-37)
  * Instantiate hybrid retriever (BM25+dense) and MRL reranker from retrieval config. [`cmd/agent/main.go`](cmd/agent/main.go:39-45)
  * Create orchestrator with Genkit, config, retriever, and reranker; define and serve the `answer` flow. [`cmd/agent/main.go`](cmd/agent/main.go:46-56)

### 2.2 Orchestrator Core

* **Location:** [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go)
* **Structs / Interfaces:**
  * `Config` includes `MaxContextTokens`, `FallbackThreshold`, nested `LLMConfig`, `MangleConfig`, and `IntentParserConfig`. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:19-25)
  * `orchestrator` struct holds retriever, reranker, LLM gateway, Mangle processor, intent parser, and config. Implements `types.Orchestrator`. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:28-35)
* **Creation Path:** `New` validates dependencies, initializes LLM gateway (OpenAI/Ollama), Mangle processor, and intent parser (Genkit flow); fails startup on errors. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:38-72)
* **Execution (`RunFlow`):**
  1. Parse intent/entities via Genkit flow; attach to input. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:76-81)
  2. PreProcess via Mangle for expansions/filters/constraints. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:82-85)
  3. Hybrid search with Mangle filters, yielding scored chunks. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:87-90)
  4. Rerank candidates, collecting explanations. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:92-95)
  5. PostProcess reranked chunks with user context/constraints, collecting explanations. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:97-100)
  6. Check fallback; if triggered, return policy message with explanations. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:110-123)
  7. Limit chunks by token estimate (word count), generate via LLM. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:125-129) [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:179-206)
  8. Aggregate explanations (rerank, post, intent, Mangle pre); attach metadata. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:131-137) [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:139-177)

> **Note:** The hybrid retriever and MRL reranker are fully integrated; mock retriever is for tests only. [`internal/retrieval/mock.go`](internal/retrieval/mock.go)

### 2.3 Rule Processor (Mangle)

* **Location:** [`internal/mangle/processor.go`](internal/mangle/processor.go)
* **Initialization:**
  * Validates rules/facts paths; parses Datalog, stratifies, loads facts (globbing/directories, supporting .facts/.dlog/etc.), evaluates base program for derived facts. [`internal/mangle/processor.go`](internal/mangle/processor.go:39-70) [`internal/mangle/processor.go`](internal/mangle/processor.go:293-462)
* **PreProcess:**
  * Normalizes/lowercases query, merges base facts, tokenizes/asserts query tokens/entities, evaluates program. [`internal/mangle/processor.go`](internal/mangle/processor.go:75-95)
  * Collects expansions (`expanded_query`), filters (`query_filter`), normalized terms, stopwords, term buckets (`term_constraint`), entities (`query_entity`), constraints (`query_constraint`). Filters stopwords, dedupes/terms. [`internal/mangle/processor.go`](internal/mangle/processor.go:99-139) [`internal/mangle/processor.go`](internal/mangle/processor.go:463-601) (continued in full file)
  * Returns `ExpandedQuery` with explanation. [`internal/mangle/processor.go`](internal/mangle/processor.go:144-164)
* **PostProcess:**
  * Filters chunks by visibility/tenant/metadata; redacts text if flagged (simple placeholder replacement). Generates explanations for drops/modifications. Defaults visibility to "public", tenant to "*". [`internal/mangle/processor.go`](internal/mangle/processor.go:167-291) [`internal/mangle/processor.go`](internal/mangle/processor.go:774-787)

Rules in `config/mangle/main.dlog` define aliases, stopwords, constraints; facts in `config/mangle/*/*.dlog` seed aliases, policies, pipelines, and retrieval metadata. [`config/mangle/main.dlog`](config/mangle/main.dlog) [`config/mangle/`](config/mangle)

### 2.4 Retrieval Layers

#### 2.4.1 Custom Hybrid Retrieval
* **Location:** [`internal/retrieval/hybrid.go`](internal/retrieval/hybrid.go)
* **Behavior:**
  * Loads/chunks Markdown corpus (sliding window with overlap), tokenizes, computes term freq/IDF, and indexes Genkit localvec documents using the Google AI embedder. [`internal/retrieval/hybrid.go`](internal/retrieval/hybrid.go:341-390) [`internal/retrieval/embedding.go`](internal/retrieval/embedding.go)
  * `Search`: Projects must/should terms from Mangle, embeds query via Google AI, scores BM25 (k1=1.6, b=0.75) + cosine dense from localvec store; normalizes/combines (0.6 lexical + 0.4 dense); filters metadata early. Limits candidates via topK union. [`internal/retrieval/hybrid.go`](internal/retrieval/hybrid.go:85-218)
  * Config: Corpus path/chunking, BM25 must/should, dense topK/model/storeDir. [`internal/retrieval/config.go`](internal/retrieval/config.go)

#### 2.4.2 Mock Retriever
* **Location:** [`internal/retrieval/mock.go`](internal/retrieval/mock.go)
* **Purpose:** Fixed chunks for tests; ignores query/filters. [`internal/retrieval/mock.go`](internal/retrieval/mock.go:18-32)

#### 2.4.3 Legacy LocalVec RAG (Unused)
* **Location:** [`internal/rag/rag.go`](internal/rag/rag.go)
* **Note:** Older demo showing direct Genkit localvec retriever wiring remains for reference; primary hybrid path now shares the same embedder/store primitives. [`internal/rag/rag.go`](internal/rag/rag.go:37-88)

### 2.5 Reranker (MRL)

* **Location:** [`internal/retrieval/reranker.go`](internal/retrieval/reranker.go)
* **Behavior:** Embeds query/chunks with the Google AI embedder, computes cosine similarities (reported with configured `dims` labels), averages scores (weighted with hybrid score), limits to topK. Adds rerank metadata/explanations. Config: dims, topK. [`internal/retrieval/reranker.go`](internal/retrieval/reranker.go:19-143) [`internal/retrieval/config.go`](internal/retrieval/config.go)

### 2.6 LLM Gateway

* **Location:** [`internal/llm/gateway.go`](internal/llm/gateway.go)
* **Initialization (`New`):** Supports "openai" (compat plugin with API key) or "ollama" (Genkit plugin); errors on unsupported providers. [`internal/llm/gateway.go`](internal/llm/gateway.go:24-47)
* **Generation Path:** Builds bulletized prompt with [citations], streams via `model.Generate`, aggregates answer, extracts unique DocID citations (sorted, with titles/scores). [`internal/llm/gateway.go`](internal/llm/gateway.go:50-125)

### 2.7 Intent Parser

* **Location:** [`internal/genintent/parser.go`](internal/genintent/parser.go)
* **Behavior:** Genkit flow detects intent (regex/keyword: troubleshoot/question/etc.) and extracts entities (versions, tickets, products, platforms, artifacts) via regex. Dedupes/sorts. Config: flowName. [`internal/genintent/parser.go`](internal/genintent/parser.go:33-187)

### 2.8 Shared Types

`internal/types` defines contracts: `QueryInput`, `ExpandedQuery`, `ConstraintSet`/`TermConstraints`/`MetadataConstraint`, `Chunk`, `Context`, `Explanation`, `Response`, `IntentResult`, interfaces (`Processor`, `Retriever`, `Reranker`, `Gateway`, `Orchestrator`, `IntentParser`), `LLMConfig`. [`internal/types/types.go`](internal/types/types.go:10-135)

---

## 3. Sequence Diagram (Textual)

```
Client → HTTP Handler : POST /answer {query, user_context}
HTTP Handler → Orchestrator.RunFlow : *types.QueryInput
Orchestrator → IntentParser.Parse : input
IntentParser → Orchestrator : *types.IntentResult
Orchestrator → Mangle.PreProcess : input + intent
Mangle → Orchestrator : *types.ExpandedQuery
Orchestrator → HybridRetriever.Search : expanded + filters
HybridRetriever → Orchestrator : []*types.Chunk (hybrid scored)
Orchestrator → MRLReranker.Rerank : expanded + candidates
MRLReranker → Orchestrator : []*types.Chunk (reranked) + explanations
Orchestrator → Mangle.PostProcess : reranked + context
Mangle → Orchestrator : filtered chunks + explanations
Orchestrator → Fallback Check : postContext
(if fallback) → Client : policy response + explanations
else
Orchestrator → LLMGateway.Generate : query + limited chunks
LLMGateway → Orchestrator : *types.Response (answer + citations)
Orchestrator → Client : response + aggregated explanations + metadata
```

---

## 4. Configuration Surface

`config/config.yaml`:

* `orchestrator.maxContextTokens` / `fallbackThreshold`: Prompt limiting, fallback trigger. [`config/config.yaml`](config/config.yaml:1-3)
* `llm.provider` / `model` / `apiKey`: OpenAI/Ollama selection (key expanded). [`config/config.yaml`](config/config.yaml:5-8)
* `retrieval.corpus.path` / `chunkSize` / `chunkOverlap`: Markdown ingestion/chunking. [`config/config.yaml`](config/config.yaml:10-12)
* `retrieval.hybrid.bm25.must` / `should`: Lexical term buckets. [`config/config.yaml`](config/config.yaml:14-15)
* `retrieval.hybrid.dense.topK` / `model`: Dense candidate pool + Google AI embedder model. [`config/config.yaml`](config/config.yaml:17-19)
* `retrieval.rerank.mrl.topK` / `dimensions`: Rerank limit/multi-dim (default [512,768]). [`config/config.yaml`](config/config.yaml:21-23)
* `mangle.rulesFile` / `factsFile`: Datalog program/facts (globbing supported). [`config/config.yaml`](config/config.yaml:25-26)
* `intentParser.flowName`: Genkit flow ID (default "parse_intent_ner"). [`config/config.yaml`](config/config.yaml)

---

## 5. Error Handling & Observability Hooks

* Startup: `log.Fatalf` on config/retriever/reranker/orchestrator failures with context. [`cmd/agent/main.go`](cmd/agent/main.go:30-49)
* Mangle: Wrapped errors (e.g., "load mangle program", "evaluate query program"). [`internal/mangle/processor.go`](internal/mangle/processor.go:46-108)
* Flow: Propagates intent/retrieval/rerank/post/LLM errors. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:77-128)
* Explanations: Attached for traceability (intent, Mangle, rerank, post); metadata includes expanded query/intent. [`internal/orchestrator/orchestrator.go`](internal/orchestrator/orchestrator.go:139-177)

No full telemetry/logging yet; hooks at boundaries for future spans/metrics.

---

## 6. Testing

* `internal/orchestrator/orchestrator_test.go`: End-to-end `RunFlow` with real Mangle (rules/facts), mock retriever/reranker/LLM/intent. Asserts expansions (e.g., "bm25" → aliases), filters (visibility=public), forwarding, explanations/metadata. [`internal/orchestrator/orchestrator_test.go`](internal/orchestrator/orchestrator_test.go:13-167)
* `internal/mangle/processor_test.go`: Fact loading from dirs/globs, parsing/comments. [`internal/mangle/processor_test.go`](internal/mangle/processor_test.go:13-118)

Recommend: Error paths (empty corpus, low scores), full integration (corpus indexing), PostProcess edge cases (redaction).

---

## 7. Extension Points & Open Items

* **Real Embeddings:** Replace hash-based dense with Vertex AI/GoogleAI or external (e.g., Qdrant). Integrate config for provider.
* **Advanced Rerank/Post:** LLM-as-judge for rerank; dynamic redaction via Mangle rules.
* **Observability:** Structured logs (Genkit tracers), metrics (latency, hit rates), error rates.
* **Config Validation:** Schema checks (e.g., positive topK, valid paths); runtime corpus reload.
* **Intent Enhancement:** LLM-powered NER over regex; multi-turn context.
* **Fallback Polish:** Configurable messages; partial generation if low-confidence chunks.
* **Corpus Management:** Auto-indexing watcher; support non-Markdown formats.

This LLD aligns with the current implementation as of 2025-09-29 and should be updated with major changes.
