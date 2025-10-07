# Manglekit SDK — Low Level Design (LLD, Comprehensive Review & Final Update)

> Module path: `github.com/duynguyendang/manglekit`  
> Go version: 1.21+  
> Status: Core Sandwich and Declarative orchestrators implemented; Providers implemented (BM25, Dense, Hybrid, Cosine, OpenAI, Google); Config YAML loader implemented; Observability hooks present; Gaps: FromEnv, HTTP middleware, ingestion, advanced rule explanations.  
> Last Updated: 2025-10-07  
> Review Notes: Verified against current codebase: imports, registry removal of Try/Must, PromptBuilder usage, localvec vector store dependencies, Mangle FlowController. Divergences and risks documented; roadmap updated.

This document is code-facing LLD for Manglekit. It specifies public packages/files, responsibilities, interfaces, data contracts, flows, error handling, observability, and extension points. It aligns with docs/CSD.md and docs/HLD.md.

---

## 0) Design Tenets [VERIFIED ✓]

- Single-import UX: users import only `github.com/duynguyendang/manglekit` [CONFIRMED: [sdk.go](sdk.go)].
- Providers self-register via registry init() [CONFIRMED: e.g., [internal/providers/bm25/bm25.go](internal/providers/bm25/bm25.go), [internal/providers/llm/openai.go](internal/providers/llm/openai.go)].
- Composable pipelines: Sandwich and Declarative orchestrators [CONFIRMED: [pipeline/sandwich.go](pipeline/sandwich.go), [pipeline/declarative/orchestrator.go](pipeline/declarative/orchestrator.go)].
- Safe evolution: public contracts in root/subpackages; implementations in internal/ [CONFIRMED: e.g., [retrieve/retrieve.go](retrieve/retrieve.go), [internal/providers/hybrid/hybrid.go](internal/providers/hybrid/hybrid.go)].
- Observability hooks optional/no-op [CONFIRMED: [core/types.go](core/types.go)].

---

## 1) Repository / Package Layout (authoritative) [VERIFIED ✓]
```
github.com/duynguyendang/manglekit
├── go.mod
├── [sdk.go](sdk.go)                  # New() validates options, defaults TopK/MaxTokens, calls Sandwich
├── [builder.go](builder.go)         # Fluent Builder; resolves providers; declarative tool builder
├── [config.go](config.go)           # NewBuilderFromYAML(); env expansion; path resolution tags
├── [registry.go](registry.go)       # Global maps; Register*; Get() helper; Try/Must removed
├── core/
│   ├── [types.go](core/types.go)    # Query, Answer, Citation, Options, Observability, Orchestrator
│   └── [rules.go](core/rules.go)    # RuleSet, FlowController, FactConverter, Stage
├── retrieve/
│   ├── [retrieve.go](retrieve/retrieve.go)  # Retriever interface, Request/Result, Updatable
│   ├── [options.go](retrieve/options.go)    # Typed options (BM25, Dense, Hybrid, InMemory)
│   ├── [bm25.go](retrieve/bm25.go)          # Placeholder package header (impl in internal)
│   ├── [inmemory.go](retrieve/inmemory.go)  # Placeholder package header
│   └── [hybrid.go](retrieve/hybrid.go)      # Placeholder package header
├── rerank/
│   ├── [rerank.go](rerank/rerank.go)        # Reranker interface, Request, ScoredDoc
│   └── [options.go](rerank/options.go)      # CosineOptions, ColbertOptions (future)
├── llm/
│   ├── [llm.go](llm/llm.go)                 # Client interface, Request/Response
│   ├── [prompt.go](llm/prompt.go)           # PromptBuilder, DefaultRAGTemplate
│   └── [options.go](llm/options.go)         # OpenAIOptions, GoogleOptions
├── pipeline/
│   ├── [sandwich.go](pipeline/sandwich.go)  # Sandwich orchestrator
│   └── [declarative/orchestrator.go](pipeline/declarative/orchestrator.go) # Declarative orchestrator
├── internal/providers/
│   ├── bm25/[bm25.go](internal/providers/bm25/bm25.go)      # Keyword retriever using tfidf+BM25
│   ├── dense/[dense.go](internal/providers/dense/dense.go)  # Dense retriever using ai.Embedder + core.VectorStore
│   ├── hybrid/[hybrid.go](internal/providers/hybrid/hybrid.go) # Hybrid RRF fusion
│   ├── rerank/cosine/[cosine.go](internal/providers/rerank/cosine/cosine.go) # Cosine reranker
│   ├── llm/[openai.go](internal/providers/llm/openai.go)    # OpenAI/Groq LLM client
│   └── llm/[google.go](internal/providers/llm/google.go)    # Google LLM via Genkit
├── internal/vectorstores/localvec/[localvec.go](internal/vectorstores/localvec/localvec.go) # VectorStore
├── internal/embedders/google/[google.go](internal/embedders/google/google.go) # ai.Embedder
├── internal/embedders/openai/[openai.go](internal/embedders/openai/openai.go) # ai.Embedder
├── [typemap.go](typemap.go)             # Options type↔name mapping for builder
├── cmd/agent/[main.go](cmd/agent/main.go) # Basic HTTP demo
└── examples/...                          # Multiple runnable examples (02 logic-layer, 04 chat-w-data, etc.)
```

Implementation Notes: Verified layout and registrations; Registry.Try*/Must* removed; Builder infers provider names from typed options via [typemap.go](typemap.go). Dense retriever requires context key 'query_text' for localvec Search.

---

## 2) End-to-End Flow — Sandwich [VERIFIED ✓]
- Pre rules (optional): Evaluate core.Pre against Query; may mutate Query.Meta with filters or expansion terms. See [core/rules.go](core/rules.go).
- Retrieve: [internal/providers/hybrid/hybrid.go](internal/providers/hybrid/hybrid.go) concurrently calls BM25 and Dense, merges via RRF (k=60). Alternatively singular retriever if configured.
- Rerank (optional): [internal/providers/rerank/cosine/cosine.go](internal/providers/rerank/cosine/cosine.go) embeds query and docs, computes cosine, trims TopK.
- Fallback threshold: best_score compared to Options.FallbackThreshold; on low confidence returns ErrNoEvidence.
- LLM: [internal/providers/llm/openai.go](internal/providers/llm/openai.go) or [internal/providers/llm/google.go](internal/providers/llm/google.go) build prompts via [llm/prompt.go](llm/prompt.go) and generate text; records token_usage.
- Post rules (optional): Evaluate core.Post, may filter citations or deny; attaches reasons to answer.Meta.
- Observability: If hooks provided, record stage durations and structured logs.

Critical invariants:
- context.Context honored in orchestrator and providers where applicable.
- Internal provider types do not leak through public API.
- Answer.Meta includes retrieve_ms, rerank_ms when present, llm_ms, best_score, token_usage.

---

## 3) End-to-End Flow — Declarative [VERIFIED ✓: CORE]
- Flow definition: stages declared as Mangle facts (flow_stage/3, stage_tool/2), queried by [pipeline/declarative/orchestrator.go](pipeline/declarative/orchestrator.go).
- Pre rules: core.Pre evaluated to populate SkippedStages and mutate Query.Meta.
- Dispatch: tools map contains constructed instances (Retriever, Reranker, LLM); orchestrator sequentially executes and passes shared context.
- Assembly: final Answer built from accumulated context and token_usage.
- Post rules: Not currently applied in declarative orchestrator; recommended enhancement.

---

## 4) Public DTOs and Interfaces [VERIFIED ✓]
- [core/types.go](core/types.go): Query, Answer, Citation, Options, Observability, Orchestrator.
- [retrieve/retrieve.go](retrieve/retrieve.go): Retriever, Request, Result, Updatable.
- [rerank/rerank.go](rerank/rerank.go): Reranker, Request, ScoredDoc.
- [llm/llm.go](llm/llm.go): Client, Request, Response.
- [core/rules.go](core/rules.go): RuleSet, FlowController, FactConverter, Stage, MangleOptions.

Notes:
- Citation.Score JSON tag has a typo ('json_' prefix). Consider fixing to 'json:"score,omitempty"'.
- Options.* fields use 'any' for builder/type-assertion friendliness; Sandwich performs type assertions.

---

## 5) Observability [VERIFIED ✓]
- Hooks: Logger.Info/Error, Tracer.StartSpan, Meter.Record available via [core/types.go](core/types.go).
- Sandwich: emits manglekit.rules_pre_ms, manglekit.retrieve_ms, manglekit.rerank_ms, manglekit.llm_ms; logs start/finish and denials.
- Declarative: minimal logging; metrics can be added similarly.

---

## 6) Providers (Implemented) [VERIFIED ✓]
- BM25 retriever: [internal/providers/bm25/bm25.go](internal/providers/bm25/bm25.go)
  - Indexes markdown directory; parses optional front matter; uses go-nlp/tfidf + BM25; returns core.Doc with metadata including doc_id, source, score.
- Dense retriever: [internal/providers/dense/dense.go](internal/providers/dense/dense.go)
  - Uses ai.Embedder to embed query; searches core.VectorStore (localvec); filter via Query.Meta['filters']; requires ctx Value 'query_text'.
- Hybrid retriever: [internal/providers/hybrid/hybrid.go](internal/providers/hybrid/hybrid.go)
  - Parallel BM25 and Dense; RRF fusion with k=60; trims TopK; returns fused docs.
- Cosine reranker: [internal/providers/rerank/cosine/cosine.go](internal/providers/rerank/cosine/cosine.go)
  - Embeds query and docs; computes cosine; sorts desc; optional TopK override.
- LLM OpenAI/Groq: [internal/providers/llm/openai.go](internal/providers/llm/openai.go)
  - Uses PromptBuilder; chat completions; usage prompt/completion tokens.
- LLM Google: [internal/providers/llm/google.go](internal/providers/llm/google.go)
  - Genkit model abstraction; PromptBuilder; usage input/output tokens.
- Vector Store (localvec): [internal/vectorstores/localvec/localvec.go](internal/vectorstores/localvec/localvec.go)
  - Defines retriever, indexes docs at startup, supports AddDocuments and Search; filters via metadata; requires query_text.

---

## 7) Builder and Config [VERIFIED ✓]
- Builder: [builder.go](builder.go)
  - Fluent With* methods; WithEmbedder accepts pre-built ai.Embedder; resolves provider configs (google/openai/groq) via env; builds sandwich or declarative orchestrators.
  - Tool graph: buildTools() iteratively respects dependencies; buildSingleTool() dispatches by provider name.
  - Component builders for vector store, retrievers, rerankers, rules.
- Config loader: [config.go](config.go)
  - NewBuilderFromYAML() expands env vars; resolves path tags with baseDir; constructs options structs via JSON marshal/unmarshal trick; configures builder (TopK, MaxTokens, FallbackThreshold).
- Type mapping: [typemap.go](typemap.go)
  - optionsTypeToName ↔ nameToOptionsType used to infer provider names from typed options.

---

## 8) Rules Engine Integration [VERIFIED ✓]
- Provider: [internal/providers/mangle/rules.go](internal/providers/mangle/rules.go)
  - New(ctx, core.MangleOptions) loads rules (.dlog), parses schemas via registered schema parsers, wires default and custom FactConverters, evaluates base program.
  - Pre stage: collects skip_stage, deny reasons, expansion_terms, query_filter; mutates Query.Meta; can deny with reasons attached to Answer.Meta.
  - Post stage: converts cited docs to facts; filters citations by deny; attaches reasons.
- FlowController: Exposes Query(ctx, atom, onSolution) for declarative orchestrator to fetch flow definitions.

---

## 9) HTTP Demo [VERIFIED ✓: BASIC]
- [cmd/agent/main.go](cmd/agent/main.go): POST /answer decodes Query, runs orchestrator, encodes Answer; GET /health returns ok.
- Gaps: middleware (JWT, sanitize, timeout), structured error codes, ingestion endpoint.

---

## 10) Error Handling & Conventions [VERIFIED ✓]
- Errors wrapped with %w across orchestrators and providers.
- No global mutable state; dependencies injected via builder and registry.
- Small, focused packages.

---

## 11) Detailed Design for Missing Features (Gaps)
- Ingestion Flow
  - Ingestor interface; chunking; embedding; indexing via VectorStore.AddDocuments; async job queue; HTTP /v1/ingest.
- HTTP Service Mode
  - Add middleware (JWT auth, sanitize, timeout); structured logging and metrics; error contracts.
- Advanced Mangle Rules
  - RuleResult with explanations; redact PII using regex rules; persistent facts via BoltDB; attach drop reasons to Answer.Meta.
- Config FromEnv
  - Parse prefixed env vars (e.g., MKT_*) into provider configs; construct builder accordingly.
- Declarative Post Stage
  - Apply core.Post after LLM in declarative orchestrator; mirror Sandwich behavior.
- Prompt Templates
  - Expose Options.PromptTemplate per LLM; already supported by providers via PromptBuilder.

---

## 12) Risks & Mitigations
- Fallback score dependence on reranker
  - In Sandwich, best_score is populated only if reranker is present; without rerank, fallback threshold comparison may be ineffective. Mitigation: derive confidence from retrieval ranking or set default best_score from retrieval metadata.
- Citation.Score JSON tag typo
  - Fix serialization to ensure downstream consumers receive Score.
- localvec query_text requirement
  - Ensure Dense retriever always supplies ctx with 'query_text'; documented and enforced.
- Registry type assertions
  - Builder asserts types and returns clear errors; maintain typemap accuracy.

---

## 13) Roadmap
- Phase 1 (current): Core pipelines, providers, YAML config, PromptBuilder. Maintain tests.
- Phase 2: Ingestion + HTTP middleware + FromEnv + declarative Post stage.
- Phase 3: Advanced rules explanations/redaction + persistent facts + prompt template options at SDK level.
- Phase 4: Optional Genkit flows integration and multi-modal extensions.

Verification Badges:
- Sandwich orchestrator implemented: [pipeline/sandwich.go](pipeline/sandwich.go).
- Declarative orchestrator implemented: [pipeline/declarative/orchestrator.go](pipeline/declarative/orchestrator.go).
- Providers registered via init(): BM25/Dense/Hybrid/Rerank/LLM—see files linked above.
- Config YAML loader: [config.go](config.go).
- Builder: [builder.go](builder.go).

Appendix: Configuration Keys
- config.yaml fields: orchestrator.type, orchestrator.flowName, tools map, providers.google/openai/groq, embedder/retriever/vectorStore/reranker/rules/llm, topK, maxTokens, fallbackThreshold. See [config.go](config.go).
