# Refactor Suggestions for Manglekit

## 1. Orchestrator flow composition
- `RunFlow` coordinates intent parsing, preprocessing, retrieval, reranking, post-processing, fallback handling, and LLM generation inside a single method of ~60 lines. Extract dedicated helpers (e.g. `parseIntent`, `retrieveContext`, `handleFallback`) to isolate concerns, simplify error handling, and improve observability hooks.【F:internal/orchestrator/orchestrator.go†L37-L137】
- `limitChunks` approximates token limits with `len(strings.Fields)` and mutates the original slice. Consider accepting a token counter strategy (e.g. TikToken adapter) and returning copies to avoid side effects on downstream components.【F:internal/orchestrator/orchestrator.go†L179-L205】
- Fallback handling only inspects the first chunk's score. Instead, compute an aggregate (mean or max) across surviving chunks or allow the reranker to supply a confidence value to better reflect retrieval quality.【F:internal/orchestrator/orchestrator.go†L110-L128】

## 2. Mangle processor structure
- `processor.go` couples lifecycle (rule/fact loading), preprocessing, post-processing, and a large set of helper utilities in one file of ~800 lines. Split into focused files/packages (e.g. `loader.go`, `preprocess.go`, `postprocess.go`, `facts_io.go`) to simplify maintenance and testing.【F:internal/mangle/processor.go†L1-L820】
- `PreProcess` dereferences `input.Query` before confirming that `input` is non-nil; add early validation to avoid panics and allow clearer error reporting.【F:internal/mangle/processor.go†L74-L95】
- `PostProcess` mutates chunk metadata and text in place, which leaks modifications to upstream slices reused elsewhere. Return cloned chunks (similar to `cloneChunk`) to keep the function side-effect free.【F:internal/mangle/processor.go†L166-L291】
- Helpers such as `dedupeStrings`, `differenceStrings`, and `tokenize` duplicate logic found in other packages; consolidate them in a shared utility module to reduce drift and make behaviours consistent across stages.【F:internal/mangle/processor.go†L695-L820】【F:internal/genintent/parser.go†L147-L166】

## 3. Retrieval layer modularity
- `hybridRetriever.Search` performs BM25 term prep, metadata merging, candidate scoring, normalisation, ranking, and response construction inline. Break this into reusable helpers (e.g. `buildQueryTerms`, `scoreCandidates`, `selectTopK`) to make policy adjustments easier and to enable independent testing for each sub-step.【F:internal/retrieval/hybrid.go†L42-L188】
- `loadCorpus` and `chunkDocument` interleave filesystem traversal, chunking, embedding, and metadata decoration. Extract an ingestion pipeline that can be reused by both hybrid and Genkit/localvec retrievers, enabling streaming updates and alternative storage backends.【F:internal/retrieval/hybrid.go†L341-L456】
- Tokenisation, metadata matching, and deduplication helpers repeat logic from other packages. Moving them into shared utilities would prevent inconsistencies (e.g. case handling) between pre-processing and retrieval stages.【F:internal/retrieval/hybrid.go†L231-L338】

## 4. Reranking transparency and safety
- The MRL reranker clones chunks but still shares metadata references if `chunk.Metadata` is nil, and applies a fixed `0.7/0.3` blend of average similarity and original score. Allow configurable weighting and always deep-copy metadata to avoid mutating original retriever results.【F:internal/retrieval/reranker.go†L13-L72】
- Capture and return structured per-dimension scores separately from human-readable explanations so downstream consumers can perform analytics without parsing strings.【F:internal/retrieval/reranker.go†L33-L71】

## 5. LLM gateway robustness
- `llm.New` ignores errors from `plugin.Init` and assumes models are immediately available. Wrap plugin initialisation with error handling and allow dependency injection for custom prompts or streaming handlers to facilitate testing.【F:internal/llm/gateway.go†L23-L71】
- `buildPrompt` embeds the prompt template directly in code. Move templates to configuration or provide strategy interfaces so different conversation styles (QA, summaries, comparisons) can share the gateway without forks.【F:internal/llm/gateway.go†L73-L95】

## 6. Intent and retrieval utilities reuse
- Intent parsing reimplements `dedupeStrings` and `flattenEntityValues`, which exist in other packages. Centralising these helpers would ensure consistent behaviour (ordering, casing) across the pipeline.【F:internal/genintent/parser.go†L104-L174】
- Provide an interface for intent detection so rule-based, ML-based, or remote parsers can be swapped without editing the orchestrator wiring. Currently the Genkit flow is baked into the constructor, reducing testability.【F:internal/orchestrator/orchestrator.go†L37-L71】【F:internal/genintent/parser.go†L33-L63】

## 7. RAG service alignment
- The RAG package bootstraps its own localvec retriever and document ingestion, overlapping with the hybrid retriever logic. Consider factoring corpus loading, chunking, and embedding configuration into shared services so both orchestrated and standalone RAG flows stay in sync.【F:internal/rag/rag.go†L21-L110】【F:internal/retrieval/hybrid.go†L341-L456】
- `RAG.New` indexes the entire corpus on construction, which can block startup and makes incremental updates difficult. Introduce background indexing or dependency injection for the document store to support streaming ingestion.【F:internal/rag/rag.go†L37-L88】

## 8. Configuration and typing consistency
- Multiple components accept raw maps for metadata (e.g. filters, user context). Define typed structs for common metadata and filter expressions to avoid repeated `map[string]any` casts and to improve schema validation at the boundaries.【F:internal/orchestrator/orchestrator.go†L97-L135】【F:internal/types/types.go†L9-L77】
- Centralise configuration defaults (chunk sizes, BM25 caps, reranker dimensions) so they live in one place instead of being re-applied in constructors, reducing the risk of diverging defaults between services.【F:internal/retrieval/hybrid.go†L42-L82】【F:internal/retrieval/reranker.go†L13-L31】
