# Manglekit — High Level Design (HLD)

**Owner:** Duy Nguyen
**Version:** 1.2 (2025-09-24)  
**Audience:** Engineering, Solution Architects, Developers  
**Status:** Draft - Ready for Technical Review  

---

## 0. Executive Summary

Manglekit is a lightweight, embeddable Go framework for Retrieval-Augmented Generation (RAG) workflows with integrated declarative rules and semantic search. It comprises:

* **Mangle**: Datalog-style engine for rule-based pre/post-processing, handling normalization, constraints, and validations.
* **Vector DB Layer**: Pluggable hybrid retrieval (vector + keyword) for unstructured data.
* **Genkit**: Orchestration for LLM flows via the Sandwich Pattern (Mangle-Pre → RAG → Mangle-Post → LLM).

This HLD details the technical architecture, components, interfaces, data contracts, flows, and deployment for implementation. It provides a blueprint for building scalable, modular systems using Go stdlib and minimal dependencies, targeting high-performance AI retrieval (e.g., <300ms E2E latency).

---

## 1. Goals & Non-Goals (Technical Scope)

### 1.1 Goals
* **Performant Go Framework**: Embeddable library with low overhead (goroutines for concurrency, <10MB binary).
* **Rule Enforcement**: Pre/Post Datalog rules for query shaping and output validation, adding <50ms latency.
* **Hybrid Retrieval**: Vector (ANN) + BM25 for precision/recall, with metadata filtering.
* **Modularity**: Interfaces for pluggable backends (e.g., Vector DBs, LLMs) to support testing/migration.
* **Traceability**: Structured logging of decisions for debugging and optimization.
* **Basic Scalability**: Stateless design for 100+ QPS; concurrency via channels/mutexes.

### 1.2 Non-Goals
* **Complex Inference**: No full KG; simple fact/rule store only.
* **Advanced Auth**: JWT integration hooks; no built-in IAM.
* **Multi-Modal**: Text RAG only.
* **Hosted Platform**: Library/service mode; no PaaS.
* **Streaming Pipelines**: Batch ingestion; no Kafka-level async.

---

## 2. Architecture Overview

Linear, modular pipeline with hooks for extension. Core: Sandwich Pattern for rule-wrapped RAG.

```mermaid
flowchart TD
  U[Client SDK/HTTP] --> G[Genkit Orchestrator]
  G --> MP[Mangle-Pre]
  MP --> RE[Retrieval Engine]
  RE --> VDB[(Vector DB: FAISS/Pinecone)]
  RE --> KI[Keyword Index: Bleve/BM25]
  RE --> MF[Metadata Filter: SQL-like]
  RE --> R[Ranked Chunks (RRF Fusion)]
  R --> MPost[Mangle-Post]
  MPost --> LG[LLM Gateway]
  LG --> Prov[Provider: OpenAI/Ollama]
  Prov --> Synth[Response Generation]
  Synth --> G
  G --> U

  subgraph "Stores & Observability"
    C[Facts/Rules: In-memory/BoltDB]
    L[Logging/Metrics: Zap/Prometheus/OTel]
  end

  MP -.-> C
  MPost -.-> C
  G -.-> L
  LG -.-> L
```

- **Layers**: Orchestration → Logic → Retrieval → Synthesis.
- **Flows**: Sync queries; async ingestion via goroutines.
- **Tech Stack**: Go 1.21+; deps: gonum (vectors), github.com/blevesearch/bleve (index), github.com/google/mangle. net/http for service; no external web frameworks.

---

## 3. Component Responsibilities

### 3.1 Genkit Orchestrator
* **Interface**: Central entry; parses inputs, routes flows, assembles outputs.
* **Features**:
  - Intent classification (regex-based or lite NLP).
  - Workflow defs (Go structs or YAML: e.g., conditional expansion).
  - Context merging (token limits, truncation).
  - Fallbacks (e.g., direct LLM on retrieval fail).
* **Config** (YAML):
  ```yaml
  orchestrator:
    maxTokens: 4000
    fallbackScore: 0.5
  ```
* **Go Interface**:
  ```go
  type Orchestrator interface {
    RunFlow(ctx context.Context, input *QueryInput) (*Response, error)
  }
  ```

### 3.2 Mangle Engine
* **Logic Core**: Datalog parser for facts/rules.
* **Pre**:
  - Normalization: Entity extraction (regex/porter stemmer).
  - Constraints: Fact queries (e.g., role-based filters).
  - Expansion: Ontology inference (simple joins).
* **Post**:
  - Validation: Rule checks (e.g., relevance thresholds).
  - Redaction: Regex/pattern matching.
  - Annotations: JSON traces of firings.
* **Notes**: Eval sandboxed; store: map[string]Fact or BoltDB.
* **Example Rule**: `allowed(C) :- matches(C, Q), score(C) > 0.7, policy(C, User).`
* **Interface**:
  ```go
  type Processor interface {
    PreProcess(input *Query) (*ExpandedQuery, error)
    PostProcess(chunks []*Chunk, ctx *Context) ([]*Chunk, *Explanation)
  }
  ```

### 3.3 Retrieval Layer
* **Hybrid Engine**: Parallel vector/keyword, fused ranking.
* **Features**:
  - Vector: Embed (HuggingFace via API or local), k-NN search.
  - Keyword: BM25 on indexed text.
  - Fusion: RRF or weighted sum.
  - Filters: Key-value or query lang (e.g., tags AND date).
  - Chunking: Fixed-size with 20% overlap.
* **Backends**: FAISS (in-mem), Pinecone (cloud).
* **Config**:
  ```yaml
  retrieval:
    vector:
      embedModel: "all-MiniLM-L6-v2"
      k: 10
    keyword:
      analyzer: "porter"
    fusion: "rrf"
  ```
* **Interface**:
  ```go
  type Retriever interface {
    Search(ctx context.Context, q *Query, filters map[string]interface{}) ([]*Chunk, error)
  }
  ```

### 3.4 LLM Gateway
* **Provider Abstraction**: Prompt building, calling, parsing.
* **Features**:
  - Plugs: OpenAI (API), Ollama (local).
  - Templates: Go templates for context injection.
  - Limits: Token counting, rate (semaphore).
  - Citations: Function calls or post-parse.
* **Handling**: Retries (backoff), cache (LRU).
* **Config**:
  ```yaml
  llm:
    provider: "openai"
    model: "gpt-4o-mini"
    key: "${API_KEY}"
    maxTokens: 500
  ```
* **Interface**:
  ```go
  type Gateway interface {
    Generate(ctx context.Context, prompt string, chunks []*Chunk) (*Generation, error)
  }
  ```

---

## 4. Data Contracts

### 4.1 Document
```json
{
  "doc_id": "uuid",
  "title": "str",
  "tags": ["str"],
  "metadata": {"author": "str", "date": "ISO", "level": "public"},
  "text": "body",
  "chunks": [{"id": "str", "text": "str", "embedding": [float], "meta": {}}],
  "version": 1
}
```

### 4.2 Query/Expansion
Input:
```json
{"query": "str", "context": {"role": "str"}, "params": {"k": 5, "explore": true}}
```
Output:
```json
{"norm_query": "str", "terms": ["str"], "filters": {}, "expl": "str"}
```

### 4.3 Response
```json
{
  "answer": "str",
  "citations": [{"doc_id": "str", "score": float}],
  "explanations": [{"type": "str", "rule": "str", "ts": "ISO"}],
  "meta": {"latency": int, "score": float}
}
```

---

## 5. Interfaces (APIs)

### 5.1 HTTP (Service Mode)
- POST /v1/answer: Query endpoint (JSON in/out).
- POST /v1/ingest: Async doc upload (multipart).
- GET /v1/health: Probes.

### 5.2 SDK Example
```go
kit := New(Config{...})
resp, _ := kit.Answer(ctx, "query", ctx)
```

Package: orchestrator/, mangle/, retrieval/, llm/, types/.

---

## 6. Core Flow

1. Intake: Parse/validate.
2. Pre: Normalize/constrain/expand.
3. Retrieve: Parallel search/fuse/filter.
4. Post: Validate/redact/annotate.
5. Synthesize: Prompt/call/parse.
6. Assemble: Bundle/log.

Pseudo:
```go
pre := mangle.Pre(input)
chunks := retrieve.Search(pre)
post, expl := mangle.Post(chunks)
gen := llm.Generate(post)
return Response{gen, expl}
```

Errors: 403 (deny), fallback (empty chunks).

---

## 7. Security, Observability, Performance

### 7.1 Security
- JWT: net/http middleware.
- Sanitize: xss/escape utils.
- Sandbox: Limited Datalog scope.
- Encrypt: AES for stores.

### 7.2 Observability
- Log: Zap structured.
- Metrics: Prometheus (latency, hits).
- Trace: OTel spans.

### 7.3 Performance
- Targets: 100ms retrieve, 300ms E2E.
- Opts: Cache (sync.Map), parallel (wg.WaitGroup).
- Scale: Replicas, sharding.

---

## 8. Deployment

- Library: go.mod import.
- Service: Binary + Docker.
Dockerfile:
```dockerfile
FROM golang:1.21 AS build
COPY . /app
RUN go build -o bin ./cmd
FROM alpine
COPY bin /manglekit
CMD ["/manglekit", "serve"]
```
- Infra: K8s (sidecar DB), Helm.
- Ingest: CLI/API goroutines.

---

## 9. Risks/Mitigations
- Perf (rules): Depth limits, profiling.
- Lock-in (LLM): Mocks/interfaces.
- Recall: Tuning, re-rankers.
- Deps: Vendoring, audits.

---

## 10. Next Steps
- LLD: API specs, rule lang.
- MVP: Mocks/impl (2w).
- Tests: 80% unit, e2e.
- Review: Tech guild.

**End of Detailed HLD**
