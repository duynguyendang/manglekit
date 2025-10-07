# Manglekit Project Context

## Overview
Manglekit is a lightweight Go framework for controlled Retrieval-Augmented Generation (RAG) workflows, integrating a rules engine (Mangle) with semantic retrieval and LLM orchestration via the Sandwich Pattern: Mangle-Pre → Retrieval → Mangle-Post → LLM. It addresses business needs for explainable, policy-compliant AI in knowledge management, such as internal support systems and compliant chatbots (from CSD.md). The architecture emphasizes modularity, performance (<500ms responses), and scalability for 100+ users, using Go 1.21+ with pluggable components (HLD.md).

**Core Goals (CSD Alignment):**
- Functional: Ontology-aware search with policy enforcement for accurate, scoped responses.
- Non-Functional: High uptime, explainability for audits, cost efficiency via hybrid retrieval.
- Use Cases: Knowledge bases, compliance-checked bots, exploratory analytics, edge deployment.

**Tech Stack (HLD Alignment):**
- Rules: github.com/google/mangle (Datalog for pre/post processing).
- Retrieval: Hybrid (vector + BM25) with pluggable DBs (e.g., FAISS, Pinecone).
- Orchestration: Genkit-inspired flows (experimental integration pending).
- LLM: Pluggable (OpenAI, Google).
- Deployment: Embeddable library or HTTP service.

## Current Implementation Status
As of 2025-10-03, the core SDK is operational, implementing the Sandwich pipeline with:
- **Entrypoints:** New(), Builder(), Registry (auto-registers providers), FromYAML (env-expanded).
- **Core Components:**
  - Types/Interfaces: Query/Answer/Citation, Orchestrator, RuleSet, Retriever, Reranker, LLM Client (core/types.go, etc.).
  - Pipeline: Full Sandwich flow in pipeline/sandwich.go (Pre/Post rules, retrieve, optional rerank, LLM, fallback threshold).
  - Providers (internal/providers/): BM25/Dense/Hybrid retrieval (RRF fusion), Cosine rerank, OpenAI/Google LLM, Mangle rules.
  - Observability: Logger/Tracer/Meter interfaces with timings/metrics (e.g., retrieve_ms, best_score).
- **Examples:** simple/main.go and chat-w-data/main.go demonstrate SDK usage with mock data.
- **Demo Service:** cmd/agent/main.go provides basic /answer endpoint.
- **Testing:** Unit tests for providers (>80% coverage); E2E via examples.

**Key Features Implemented:**
- Rule enforcement (pre: normalize/expand; post: validate/redact) via Mangle integration.
- Hybrid retrieval with metadata filtering from rules.
- Citations and Meta (scores, latencies, token usage) for traceability.
- Error handling: ErrNoEvidence, ErrDenied, wrapped errors.

**Alignments with CSD/HLD:**
- **Sandwich Pattern:** Ensures compliance/explainability (CSD: guardrails; HLD: linear pipeline).
- **Modularity:** Interfaces/pluggable providers support integration/scalability (CSD: no vendor lock-in; HLD: backends).
- **Performance:** Parallel retrieval, low overhead; timings tracked (HLD: <300ms E2E).
- **Observability:** Structured logging/metrics for audits/ROI (CSD: non-functional).

## Gaps and Pending Work
- **Ingestion:** No async document upload/chunking/embedding/indexing (HLD: ingestion flow; CSD: dynamic updates). Planned: ingest/ package with /v1/ingest endpoint.
- **HTTP Service:** Basic /answer; missing middleware (JWT, sanitization), /ingest, health probes (HLD: service mode).
- **Advanced Rules:** Basic Mangle; pending annotations/explanations, PII redaction, persistent facts store (BoltDB) (HLD: post-validation; CSD: compliance).
- **Config:** FromYAML done; FromEnv pending (env var parsing).
- **Enhancements:** LLM templates/retries/caching, Genkit integration, security hooks (JWT/encryption), multi-modal support (phase 2).
- **Testing:** Expand to ingestion/HTTP; full edge cases (policy violations, empty retrievals).

**Risks/Mitigations:**
- Recall/Perf: Tune TopK/thresholds; add caching.
- Security: Enforce in rules; no PII logging.
- Dependencies: Vendored; audits for providers.

## Key Decisions and Rationale
- **Registry + Builder:** Enables zero-config via YAML/env (HLD: modularity; rationale: developer productivity).
- **Hybrid Retrieval Default:** Balances precision/recall/cost (CSD: high-recall discovery; HLD: vector + keyword).
- **No Global State:** Interfaces passed explicitly (Go best practices; HLD: stateless scalability).
- **Fallback Threshold:** Prevents hallucination on low-evidence (CSD: trustworthy AI).
- **Observability Optional:** No-op defaults for lightweight embedding (HLD: hooks without overhead).

## Data Flows (High-Level, Current)
- **Query:** User → Orchestrator → Pre-Rules → Retrieve (hybrid, filtered) → Rerank (cosine) → LLM (prompt + passages) → Post-Rules → Answer (with citations/Meta).
- **Ingestion:** [PENDING] Doc → Chunk/Embed → Index (vector/keyword).
- **Diagram:** See HLD.md for Mermaid; current impl matches linear flow.

## Next Steps (Roadmap)
- Phase 2: Implement ingestion, FromEnv, security middleware (2-3 weeks).
- Phase 3: Advanced rules (annotations/redaction), Genkit orchestrator (3-4 weeks).
- Validate: Run E2E tests on examples; benchmark latency.
- Review: Align with stakeholders on gaps (CSD next steps).

This CONTEXT.md serves as the living source of truth for the project's state, updated post-implementation/analysis. For code details, see LLD.md; for business mapping, CSD.md; for architecture, HLD.md.