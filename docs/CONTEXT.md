# Manglekit Project Context

_Last updated: 2025-05-31_

---

## Overview
Manglekit is a Go 1.24+ SDK that wraps Google’s Mangle rules engine around Genkit-backed retrieval, reranking, and LLM calls. The default “Sandwich” path enforces **Mangle-Pre → Retrieval → optional Rerank → LLM → Mangle-Post** to keep answers grounded, compliant, and explainable while remaining provider-agnostic through the runtime registry in `registry.go`.

---

## Core Entry Points
- `sdk.New` guards the Sandwich constructor: it validates that `Retriever` and `LLM` were supplied, fills default `TopK` (8) and `MaxTokens` (512), type-asserts the concrete interfaces, and delegates to `pipeline.NewSandwich`.
- `NewBuilder`/`BuilderAPI` (`builder.go`) collect typed options via fluent `With*` calls, track provider names through `optionsTypeToName`, and accumulate `ResourceClosers` to be invoked LIFO on teardown.
- `NewBuilderFromYAML`/`NewBuilderFromEnv` (`config.go`) deserialize declarative configs, expand environment variables, resolve any `path:"resolve"` fields relative to the config source, and feed the builder with pre-populated typed option structures.

---

## Builder (`builder.go`)
- **Build**: chooses orchestrator type (default `sandwich`), aggregates any previous `errs`, resolves providers, and either:
  - For `sandwich`: calls `resolveDependencies` (infers missing embedder names, pre-warms provider clients), then `buildComponents` (Embedder → VectorStore → Retriever → Reranker → Rules → LLM), finally wrapping the populated `core.Options` with `sdk.New`.
  - For `declarative`: requires the rules engine (`buildRules`), builds the tool graph iteratively via `buildTools`/`buildSingleTool`, promotes the resulting `FlowController`, and instantiates `declarative.New` with all constructed tools and closers.
- **resolveProviderConfig**: lazily creates provider clients, validates API keys (GOOGLE/OpenAI/Groq), installs connection-closing callbacks, and caches both the config and the constructed client (including a shared Genkit+GenAI bundle wrapped in `googleClients`).
- **resolveDependencies**: backfills missing embedder names by inspecting configured components, then calls `resolveProviderConfig` for each required provider family to guarantee prerequisites (e.g., embedder or LLM clients) exist before construction.
- **buildComponents**: orchestrates construction order so shared dependencies (embedder/vector store) are ready before retrievers or rerankers. Each `build*` helper type-asserts the registry constructor, passes through typed options extracted from `With*` calls, and appends closers when components implement the local `closer` interface.
- **buildTools / buildSingleTool**: declarative mode support. Tool dependencies are inferred by scanning top-level string fields in tool params; the loop keeps building tools whose dependencies are already materialized, preventing circular references. Dispatch covers embedders, vector stores, retrievers (bm25/dense/hybrid), and LLMs (Google/OpenAI/Groq) by combining registry constructors with previously created tool instances.
- **closeResources**: on error paths (or explicit Close) unwinds accumulated `ResourceClosers` with a 5s timeout, aggregating any shutdown errors.
- **Known nuance**: the typed-to-name mapping maps `embed.GoogleEmbedderOptions` to `"google"` while the Google embedder registers as `"google-embedder"`; this mismatch currently prevents the builder from instantiating that embedder without manual registry shims.

---

## Configuration Helpers (`config.go` / `typemap.go`)
- `componentCfg`/`ToolConfig` capture provider names plus free-form params. When loading YAML, `configureComponent` marshals raw maps through JSON into typed structs, then calls `resolvePathsInStruct` to rewrite relative file paths.
- `NewBuilderFromEnv` mirrors this via `MKT_*` environment variables containing provider names and JSON blobs of params.
- `resolvePathsInStruct` recursively walks struct fields tagged `path:"resolve"` (including slices) and rewrites them relative to the YAML file or working directory.
- `optionsTypeToName` + `nameToOptionsType` map typed options pointers to registry names for both Sandwich and declarative flows. Aliases are provided for OpenAI-compatible providers and embedder variants.

---

## Orchestrators
- **Sandwich (`pipeline/sandwich.go`)**:
  - `NewSandwich` validates type assertions for retriever, reranker, and LLM before freezing the provided `core.Options`.
  - `Run` executes the golden flow:
    1. Logs/traces start, seeds `Answer.Meta`.
    2. If rules are configured, executes `Rules.Evaluate(core.Pre, q, nil)`, records rule latency, applies mutations (filters/expansions) into `Query.Meta`, or short-circuits on denial.
    3. Calls `Retriever.Retrieve` with query text, requested `TopK`, and mutated metadata; stores duration and the original doc set in `Answer.Meta["original_docs"]`.
    4. If no docs, returns `core.ErrNoEvidence`. If a reranker exists, reranks and constructs aligned `Answer.Citations`, saves `best_score`, and enforces `FallbackThreshold` only when reranker output exists.
    5. Invokes `LLM.Complete` with prompt, grounded passages, and query metadata; records latency and token usage.
    6. Runs post rules via `Rules.Evaluate(core.Post, q, &answer)`—which may mutate the answer/citations, deny with reason, or add audit metadata—and logs success before returning.
  - `Close` walks `ResourceClosers` in reverse order to release API clients and background resources.
  - Metadata emitted: `retrieve_ms`, `best_score`, `llm_ms`, `token_usage`, plus whatever the rules engine adds.
- **Declarative (`pipeline/declarative/orchestrator.go`)**:
  - `New` enforces non-nil `FlowController`, tool map, and flow name.
  - `Run` flow:
    1. `getFlowStages` queries `flow_stage/3` and `stage_tool/2` facts via the `FlowController`, sorting stage order numerically.
    2. Executes `pre` rules, allowing them to deny the request, mutate the query/answer stub, or mark `SkippedStages`.
    3. Seeds a shared execution context map (`query`, `answer`, `docs`, `meta`, denial flags).
    4. Iterates sorted stages, skips any flagged by pre-rules, and dispatches to tools based on type:
       - `retrieve.Retriever`: logs filters/expansions, executes retrieval, stores docs, and preserves originals in the answer meta.
       - `rerank.Reranker`: reranks docs, refreshes citations, tracks `best_score`.
       - `llm.Client`: only runs if not already denied, builds passages, calls LLM, and saves text/token usage.
       - `core.PostRuleEvaluator`: runs advanced post-rules, filters docs/citations, appends rule diagnostics, enforces denials, and writes audit metadata.
       - `core.RuleSet`: treated as no-op (rules engine already invoked).
    5. Returns the final `core.Answer`, propagating denial reasons through `Answer.Meta`.
  - `Close` mirrors Sandwich behavior with LIFO closers.

---

## Rules Engine (`internal/providers/mangle`)
- **New**: orchestrates engine construction by parsing optional schema sources (via registered `SchemaParser`s), instantiating default/custom converters, computing EDB declarations (code-first unless `FileFirst`), loading `.dlog` and `.facts` files, stratifying the program once, seeding a base fact store with schema/file facts, and performing an initial evaluation.
- **preProcess**: clones the base fact store, feeds query facts from configured converters, evaluates rules, and returns `RuleResult` with:
  - `Allowed=false` + reason when any `deny/1` fact produced, capturing reasons in `Answer.Meta`.
  - `Mutate` that injects `filters` (from `query_filter(Key, Value)`) and `expansion_terms` (from `expanded_query/2`) into the query metadata.
  - `SkippedStages` derived from `skip_stage(Stage)` atoms for declarative orchestrations.
- **postProcess**: targets Sandwich post rules. It converts cited documents into facts, runs the rules, scrubs denied citations based on `deny(DocID, Reason)` facts, and copies denial reasons into `Answer.Meta`.
- **Post** (declarative post hook): loads query/user/doc facts plus execution metadata, evaluates rules, and returns `core.PostRuleResult` containing:
  - Filtered evidence (drops docs referenced by `drop_doc` facts).
  - Redaction specs (`redact` atoms) applied via built-in regex matchers.
  - Denial reason (first `deny/1`) and audit info (`rule_results`, `dropped_docs`, `redactions`, timing info).
- **Query**: provides read-only access to base facts (used by declarative orchestrator) by manually unifying query atoms against stored facts.
- Supporting helpers (`collectStrings`, `collectKeyValue`, `applySingleRedaction`, etc.) translate fact stores into Go structures.

---

## Providers & Components
- **Retrieval**
  - `internal/providers/bm25.New`: loads Markdown files with YAML front matter, tokenizes content into a TF-IDF model, and builds a BM25 retriever with default `TopK=10`. `Retrieve` tokenizes the query, scores docs via Okapi BM25, attaches raw BM25 scores to doc metadata, and returns the top-K as `core.Doc`s.
  - `internal/providers/dense.New`: wires an `ai.Embedder` with a `core.VectorStore`. `Retrieve` embeds the query through Genkit, forwards the vector (and metadata filters under `Meta["filters"]`) to the vector store, and returns matching docs.
  - `internal/providers/hybrid.New`: accepts pre-built BM25 and (optional) dense retrievers. `Retrieve` executes both in parallel (`errgroup`), applies Reciprocal Rank Fusion (k=60) on IDs, and returns the fused top-K.
  - `internal/providers/retrievers/inmemory.New`: stores docs in a map, implements both `Retriever` and `Updatable` with thread-safe `Retrieve`, `Upsert`, and `Replace`.
- **Vector Store**
  - `internal/vectorstores/localvec.New`: initializes Genkit’s LocalVec plugin with a long-lived context, indexes Markdown documents upfront, and returns a `core.VectorStore`. `Search` requires the original query text via `ctx.Value("query_text")`, uses LocalVec’s retriever, and post-filters matches on metadata.
- **Rerank**
  - `internal/providers/rerank/cosine.New`: couples a shared embedder with default/requested `TopK`. `Rerank` embeds query and docs concurrently, calculates cosine similarity per doc, sorts descending, and truncates to the requested cut.
- **LLM**
  - `internal/providers/llm/google.NewGoogle`: receives a shared Genkit instance, wraps a `googlegenai` model, and uses `llm.PromptBuilder` (default template) in `Complete` to merge context snippets, call `Generate`, and return text plus prompt/completion token counts (if provided).
  - `internal/providers/llm/openai.NewOpenAI`: reuses an `openai-go` client (OpenAI or Groq base URL), builds prompts via `PromptBuilder`, and calls Chat Completions, returning the first choice and usage counts.
- **Embedders**
  - `internal/embedders/openai.New`: constructs an embedder around the OpenAI Embeddings API, optionally accepting custom dimensions, and casts float64 vectors to float32.
  - `internal/embedders/google.New`: uses Genkit’s `GoogleAIEmbedder`; currently registers under `"google-embedder"` (see builder mismatch note above).
- **Prompt Builder (`llm/prompt.go`)**: caches compiled templates, provides helper funcs (`toJSON`, `join`, `truncate`), and defaults to the built-in RAG prompt that insists on grounded answers.

---

## Observability & Metadata
- `core.Observability` surfaces optional `Logger`, `Tracer`, and `Meter`. Sandwich and declarative orchestrators guard logging with `if logger != nil`, fallback to `fmt.Printf` otherwise, and emit latency metrics (e.g., `manglekit.retrieve_ms`, `manglekit.rules_post_ms`).
- `Answer.Meta` accumulates operational insight: timings, reranker scores, token usage, original document lists, denial flags, and rule audit payloads produced by the Mangle provider.
- `ResourceClosers` are appended by the builder whenever a component exposes a `Close(context.Context)` method, ensuring idempotent shutdown when orchestrators are disposed.

---

## Known Gaps & Risks
- **Provider aliasing**: the builder maps `embed.GoogleEmbedderOptions` to `"google"` but the Google embedder registers as `"google-embedder"`, so the stock builder cannot instantiate that embedder without manual override.
- **Context propagation**: dense retriever (`Embed`), cosine reranker, LocalVec indexing, and LLM clients occasionally derive their own contexts or rely on background Genkit instances, limiting respect for caller deadlines/cancellation.
- **Fallback semantics**: Sandwich only enforces `FallbackThreshold` when a reranker exists; with no reranker configured, the pipeline proceeds to the LLM even if evidence is weak.
- **Declarative dependency heuristic**: `getToolDependencies` treats every string param as a potential dependency, so literal strings in tool configs can create false dependency edges or circularity errors.
- **LocalVec lifecycle**: `localvec.New` creates a Genkit instance with `context.WithCancel` but never registers the closer with the builder (it returns the `LocalVecStore`, which implements `Close`, yet the builder only appends closers for tool instances, not components created outside declarative mode). Verify lifecycle in Sandwich flows.

---

## Roadmap / Suggested Improvements
- Align registry names and builder aliases for the Google embedder (or update `optionsTypeToName`) to enable turnkey configuration.
- Thread caller contexts through Genkit embedding/retrieval and all external SDK calls to honor cancellations.
- Expand fallback handling to cover non-reranked paths (e.g., deterministic fallback copy when `FallbackThreshold` is set but no reranker).
- Introduce explicit dependency metadata for declarative tools (e.g., `dependsOn` list) to avoid heuristic misfires.
- Add integration tests that cover: YAML → Sandwich build, YAML → Declarative flow execution, LocalVec shutdown, and redaction/denial scenarios in both orchestrators.
