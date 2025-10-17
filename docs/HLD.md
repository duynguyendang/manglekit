# Manglekit — High‑Level Design (HLD)

**Revision:** Oct 2025
**Scope:** Core SDK (registry, builder, orchestrators, providers, config bridge)
**Audience:** Framework maintainers, provider authors, application teams
**Mission:** A **neuro‑symbolic AI composition framework** for building explainable, policy‑aware systems that combine statistical models (LLMs, embedders) with symbolic reasoning (rules, planners, schema/graph tooling).

---

## 1. Executive Summary

Modern intelligent systems increasingly require **neuro‑symbolic integration** — combining data‑driven neural models with explicit symbolic reasoning and policy layers.  LLMs excel at open‑ended inference but lack verifiability, determinism, and policy control; symbolic systems (rules, planners, logic engines) provide these guarantees but lack contextual flexibility.

**Manglekit** exists to bridge that gap.

It provides a single, composable framework that lets developers declaratively assemble **neural** (LLM, embedder, retriever) and **symbolic** (rules, reasoner, planner, knowledge graph) engines into unified pipelines.  This yields AI applications that are **explainable, auditable, and policy‑compliant**, without giving up adaptability.

The framework supplies:

* A **config‑first**, declarative construction flow.
* A **generic, type‑safe DI system** unifying all component kinds.
* **Composable orchestrators** for hybrid reasoning and data retrieval.
* **Built‑in observability, lifecycle, and policy enforcement**.

**Manglekit’s key purpose**: enable neuro‑symbolic AI composition—where neural models reason under explicit symbolic rules, producing decisions that can be explained, traced, and verified.

---

## 2. Goals & Non‑Goals

### 2.1 Goals

* **Neuro‑symbolic composition:** Blend statistical components (LLM, embedder) with symbolic ones (rules, logic, planners, KGs) in one pipeline.
* **Deterministic control:** Enable explicit, auditable control flow and constraints via the **Declarative Orchestrator** and rule stages.
* **Strong typing:** Compile‑time guarantees for component wiring; no runtime type guessing in orchestrators.
* **Extensibility:** Add new component kinds without editing the core—**Open/Closed** by design.
* **Operational excellence:** Uniform metrics, tracing hooks, structured logs, and graceful shutdown.
* **Reproducibility:** Versionable configs; environment‑portable pipelines.

### 2.2 Non‑Goals

* Building a new LLM or solver: Manglekit **integrates** engines; it doesn’t replace them.
* Vendor lock‑in: contracts remain neutral; provider packages are pluggable.
* One‑size‑fits‑all safety: we provide hooks and contracts, not a prescriptive policy.

---

## 3. Design Tenets

* **Config‑First:** YAML/ENV → validated structs → builder calls (no parsing in the builder).
* **Type‑Safe DI:** Generic registry + unified factory signature.
* **Stages, not god‑methods:** Orchestrators assemble small, testable stages.
* **Context everywhere:** `context.Context` is mandatory for all factories and runtime calls.
* **Observability by default:** Metrics, structured logging, and closers are first‑class.
* **No magic strings:** Typed options/contracts over `map[string]any`.

---

## 4. Component Taxonomy (Neuro‑Symbolic)

Manglekit recognizes the following **kinds**. Each kind is implemented by **providers** registered in the registry.

* **LLM** — Text generation / reasoning engines.
* **Embedder** — Vectorization for dense retrieval/similarity.
* **Retriever** — Evidence discovery (BM25, dense, hybrid, KG search).
* **Reranker** — Re‑ordering / scoring (cosine or learned).
* **RuleSet** — Policy & logic evaluation for Pre/Post stages and mid‑flow guards.
* **Reasoner** — Symbolic/constraint solvers (Datalog, Prolog‑like, SMT wrappers) with structured I/O.
* **Planner** — Task/Tool planners (symbolic or LLM‑assisted) producing execution plans.
* **Tool** — Executable capabilities (functions, APIs) invoked by orchestrators or planners.
* **SchemaParser** — Validates/parses schemas (JSON Schema, RDF/OWL).
* **FactConverter** — Normalizes/derives facts for the logic layer.
* **KnowledgeStore** — Graph/relational stores and vector stores (KGs, SQL, vector DBs).
* **StateProvider** — Conversation/session state persistence.

> All kinds share the **same factory shape** and DI approach, so adding new kinds is non‑breaking.

### 4.1 First‑Class Integrations: Genkit & Mangle

**Genkit**

* **Role:** Provider family for embedders, vector stores (e.g., `localvec`), and tools; optionally a planning layer.
* **How it plugs in:** Ships as providers implementing the standard factory signature. Registered under `embed/`, `vectorstore/`, and `tool/` kinds.
* **Contracts:** Uses `diapi.Deps` for logger/metrics/state; honors ctx for timeouts; contributes `ResourceClosers`.
* **Examples:** `internal/vectorstores/localvec` for corpus indexing; Genkit tools callable from the Declarative Orchestrator via the **Tool** kind.
* **Why first‑class:** Enables local/offline experimentation, fast iteration, and unified observability with the rest of the stack.

**Mangle (Rules & Converters)**

* **Role:** The built‑in **RuleSet** and **FactConverter** family for policy, gating, redaction, and symbolic normalization.
* **How it plugs in:** Providers under `internal/providers/mangle/*` implement `RuleSet` and `FactConverter`; `SchemaParser` options allow structural validation before/after model calls.
* **Contracts:** Pre/Post rule stages receive typed inputs, may **deny** or **mutate** the flow; denials carry `denial_reason` and redaction metadata into `Answer.Meta`.
* **Declarative Flow:** Rules guard **Planner/Tool** execution; plans must be approved by Mangle rules before side‑effects occur.
* **Why first‑class:** Guarantees explainability and compliance; keeps neural components bounded by explicit symbolic policy.

---

## 5. Core Contracts

### 5.1 Factory Signature (Uniform)

```go
func(ctx context.Context, deps diapi.Deps, cfg any) (T, error)
```

* **ctx:** cancellation, deadlines, tracing.
* **deps:** typed sub‑dependencies supplied by the builder (e.g., an embedder a retriever needs, a vector store for dense search, a logger, etc.).
* **cfg:** provider options (typed by the caller; registry validates/decodes).

### 5.2 Provider Options Contract (Self‑Identifying)

```go
type ProviderOptions interface {
    ProviderName() string   // e.g., "openai-chat", "bm25", "datalog"
    ProviderKind() Kind     // e.g., KindLLM, KindReasoner
}
```

### 5.3 Orchestrator Contract

Orchestrators accept a **typed Resolved** bundle (no `any`) and drive execution through stages. They must:

* propagate ctx;
* record stage metrics;
* honor rule denials/guards;
* flush resource closers on `Close()`.

---

## 6. Architecture Overview

### 6.1 Build & Run (Happy Path)

1. **Config** is loaded & validated.
2. **Bridge** (`from_config`) converts config to typed options and calls the builder.
3. **Builder** uses the **spec table** and **registry** to construct components in dependency order, accumulating `ResourceClosers`.
4. **Orchestrator** (Sandwich or Declarative) receives a **Resolved** struct and executes stages.
5. **Metrics & Logs** are recorded uniformly; `Close()` drains closers LIFO.

### 6.2 Dependency Rules

* `core` is foundational (no project imports).
* Contracts (`llm`, `retrieve`, `reasoner`, etc.) depend only on `core`.
* Providers implement contracts; they **never** import the builder.
* Orchestrators depend on contracts, not provider implementations.
* Config package depends on nothing but stdlib and its own types.

---

## 7. Orchestrators

### 7.1 Sandwich (Deterministic RAG‑plus)

A strongly‑typed, fixed‑order pipeline suitable for classic RAG and many hybrid flows:

* **Pre‑Rules → Retrieve → Rerank → (Reasoner optional) → LLM → Post‑Rules**
* Captures timings in `Answer.Meta` and retains `original_docs` for audit.

### 7.2 Declarative (Flow‑Driven, Neuro‑Symbolic)

A first‑class orchestrator for **logic‑rich control flow**:

* **Flow Controller** determines stage order and tool binding.
* **Guards** via `RuleSet` decide skips/denials/mutations.
* **Tool/Planner** integration for agentic sequences with explicit safety gates.
* Keeps a shared, typed execution context (no map‑of‑any), enabling **symbolic constraints** to govern neural calls.

---

## 8. Configuration Model

* Config files define: orchestrator, component kinds & providers, and options.
* The **config→builder bridge** validates and converts into typed options; any failure occurs **before** runtime construction.
* Declarative flows may embed logical predicates/policies and tool plans.

Example sketch:

```yaml
orchestrator: declarative
components:
  - kind: retriever
    use: hybrid
    options: { top_k: 16, rrf_k: 60 }
  - kind: reasoner
    use: datalog
    options: { ruleset: "policies/records.dl" }
  - kind: tool
    use: http
    options: { endpoint: https://api.example.com, auth: env:API_TOKEN }
```

---

## 9. Observability & Lifecycle

* **Logger:** A structured logger is installed if none provided.
* **Metrics:** Stage timings (`retrieve_ms`, `rerank_ms`, `llm_ms`, `rules_pre_ms`, `rules_post_ms`, plus reasoner/planner/tool timings) are standardized.
* **Tracing Hooks:** Optional interfaces for spans at stage boundaries.
* **Closers:** All providers that hold resources register closers; orchestrators drain LIFO.

---

## 10. Security, Safety, and Policy

* **Pre/Post RuleSets** for content policy, PII redaction, safety blocks.
* **Schema Parsers** enforce structural constraints on inputs/outputs.
* **State Providers** can implement rate‑limits/quotas per session.
* **Tool Execution** requires explicit allow‑lists and typed inputs; planner outputs are validated by rules before execution.

---

## 11. Extension & Provider Authoring

### 11.1 Registering a Provider

* Implement the contract interface (e.g., `reasoner.Reasoner`).
* Provide `Options` implementing `ProviderOptions`.
* Register with the registry and (optionally) a helper set like `providers/all`.

### 11.2 Dependencies Between Providers

* Complex providers (e.g., **Hybrid Retriever**, **Tool‑using Reasoner**) receive **builder delegates** in `deps` to build subcomponents without knowing the builder itself.

---

## 12. Example Patterns (Neuro‑Symbolic)

1. **Policy‑Aware Data QA**
   Retrieve records → Reason over constraints (Datalog) → LLM explains discrepancies → Post‑rules redact.

2. **Tool‑Grounded Agent**
   Planner proposes API/tool calls → Rules vet plan → Tools execute → LLM synthesizes response with provenance.

3. **KG‑Augmented Answering**
   Dense retrieval → KG lookup (KnowledgeStore) → Reasoner derives canonical facts → LLM composes answer.

---

## 13. Compatibility & Migration

* Legacy per‑type `With…` builder calls are replaced by **generic `With(opts)`** + spec‑driven build.
* Orchestrators now receive typed **Resolved** deps; remove runtime type assertions.
* Provider factories must adopt the **uniform signature** and `ProviderOptions`.

---

## 14. Performance & Reliability

* **Budgeted stages:** enforce timeouts/token limits via ctx and options.
* **Back‑pressure:** downstream denials/short‑circuiting to preserve quotas.
* **Warm‑ups/caches:** future ADR will cover client warm‑up and connection pooling policies.

---

## 15. Roadmap

* **Centralize conversation/state handling** across orchestrators.
* **Token limit conformance** in LLM clients (honor `MaxTokens`).
* **Expose hybrid RRF params** as options.
* **Schema export** (JSON Schema) for all options.
* **Build‑graph introspection & DOT export**.
* **WASM/plugin sandbox** for untrusted providers.

---

## 16. Appendix — Package Layout (Authoritative)

```
github.com/duynguyendang/manglekit
├── builder.go
├── from_config.go
├── registry.go
├── sdk.go
├── config/
├── core/
├── retrieve/
├── rerank/
├── embed/
├── llm/
├── pipeline/
│   ├── sandwich.go
│   └── declarative/
├── internal/
│   ├── providers/... (bm25, dense, hybrid, llm, mangle, rerank/cosine, schemaparsers, state, tools)
│   ├── vectorstores/localvec/
│   └── logger/
├── providers/all/
├── examples/
└── docs/
```

---

## 17. Glossary

* **Neuro‑symbolic:** Systems that combine numerical/statistical methods (e.g., neural nets) with symbolic logic/constraints.
* **Resolved:** The fully constructed, typed set of runtime dependencies provided to an orchestrator.
* **Spec Table:** Data structure describing dependency order and required injections for builder construction.
