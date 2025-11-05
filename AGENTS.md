# AGENTS.md — Manglekit Coding Agent Configuration (2025.10)

*Last updated: 2025-11-03*

---

## Overview

This document defines how coding agents interact with the **Manglekit** codebase.  
It specifies conventions, automated tasks, and safety rules to ensure that all AI-driven modifications stay consistent with the system design and context documentation.

Agents use this file as an **operational manual** to perform reasoning, refactoring, documentation updates, and observability instrumentation within the Go SDK.

Agents must treat the documentation set (`CONTEXT.md`, `HLD.md`, `LLD.md`, `ADR.md`) as the synchronized **architecture source of truth**.  
`CONTEXT.md` is the **live baseline** — always validate and update it whenever architecture or dependency injection logic changes.

---

## 1. Agent Goals

Coding agents working in this repository should be capable of:

* Understanding the full architecture from `docs/HLD.md`.
* Updating context and documentation automatically when code changes in `docs/CONTEXT.md`.
* Maintaining observability and performance best practices.
* Identifying gaps and inconsistencies in provider registration, orchestrator wiring, or options mapping.

---

## 2. File Structure Awareness

Agents must recognize the following key locations and their purposes (aligned with repo layout):

* `core/`: Contracts, types, diapi, handler/factory interfaces, observability.
* Root: `builder.go`, `registry.go` (builder + registry), `go.mod`.
* `sdk/`: Config→builder bridge (`sdk.FromConfig`).
* `config/`: YAML parsing, normalization, validation.
* `pipeline/`: Orchestrators and stages (Sandwich, Declarative).
* `internal/providers/`: Families of retrievers, rerankers, LLMs, rules, state, schema parsers.
* `internal/embedders/`, `internal/vectorstores/`: Embedder and vector store providers and handlers.
* `providers/all/`: Convenience registrar for standard providers.
* `docs/`: Technical docs and architecture rules (`docs/rules/manglekit-arch.yml`).

---

## 3. Context-Aware Reasoning

Agents must treat `docs/CONTEXT.md` as the **live architectural memory** of the repository.  
It is both the *source of truth* for reasoning and the *target* that must stay synchronized with source changes.

### 3.1 Context Loading and Validation

Before performing any modification, the agent must:

1. **Load the full content** of `docs/CONTEXT.md`, including YAML front-matter, diagrams, tables, and changelog.  
2. **Verify freshness:** compare the `last_updated:` field to the most recent code commit.  
   - If the context is **older than 3 days**, treat it as *potentially stale* and rely on the code diff as the authoritative source.  
   - If up-to-date, use it directly as the architectural baseline.

### 3.2 How to Use the Context for Reasoning

Use the document’s internal structure as a guide:

| Section | Purpose |
|----------|----------|
| **Implementation Snapshot** | Understand module relationships (builder, registry, handlers, providers, orchestrators). |
| **Dependency Rules** | Validate layering — ensure no illegal imports or coupling. |
| **Core Contracts** | Retrieve interface and type definitions to generate compliant code. |
| **Provider Composition** | Infer where new providers or handlers should register. |
| **Known Gaps** | Identify open architectural issues and avoid reintroducing resolved gaps. |
| **Machine Appendix (JSON)** | Track gap statuses and last synchronization date. |
| **Changelog** | Understand recent context or architectural updates. |

The agent should **avoid scanning the entire repository** unless the context is stale or incomplete.  
Instead, reason from the structured data within `CONTEXT.md` and cross-references to other docs (HLD, LLD, ADR, code-review).

### 3.3 Reflecting Code Changes

When code changes affect architecture, interfaces, or runtime behavior, the agent must **edit `docs/CONTEXT.md` directly** —  
updating sections, known gaps, and timestamps as described in section **6 (Self-Managed Context Synchronization)**.  
This guarantees that the documentation always mirrors the true, current state of the system.

---

## 4. Editing Policies

* Never remove sections or metadata blocks from `CONTEXT.md`.
* Maintain YAML front-matter integrity.
* Keep the Machine Appendix JSON snapshot and Known Gaps statuses in sync.
* Use status icons (✅ / ⚠️ / ❌) consistently.
* Preserve Markdown headings, tables, and code formatting.
* Apply minimal, targeted updates with precise summaries.

---

## 5. Commit Conventions

Agents must follow semantic commit conventions:

```

feat: introduce new feature or provider
fix: resolve bug or panic
refactor: improve structure without changing behavior
docs: update documentation (includes CONTEXT.md)
chore(context): auto-sync CONTEXT.md after code change
test: add or fix unit tests

```

---

## 6. Self-Managed Context Synchronization (No External Commands)

> Goal: Whenever source changes affect architecture or runtime behavior, the **coding agent must directly update** `docs/CONTEXT.md` (and related docs) itself — by editing the file — instead of calling any CLI tools or Make targets.

### 6.1 When to Trigger an Update

Trigger the **auto-sync process** whenever any of the following occur (including test logic changes):

- Any Go file changes under `builder.go`, `registry.go`, `core/**`, `pipeline/**`, or `internal/providers/**`
- Addition or modification of a **provider** (retriever, reranker, LLM, vector store, rules, state, tool, etc.)
- Changes to **Options** or **Contracts** in `core/**` or `internal/providers/**`
- Addition of new **configuration keys** or **environment variables**
- Modifications to an **orchestrator** (Sandwich / Declarative) or build-order spec
- Changes in **observability**, **lifecycle**, or **Known Gaps**

### 6.2 Self-Update Algorithm

1. **Collect diff context**
   - Obtain the list of modified files from the current patch or PR.
2. **Infer impact scope**
   - Identify affected kinds and changed contracts or wiring.
3. **Edit `docs/CONTEXT.md` directly**
   - Preserve formatting and metadata.
   - Bump `last_updated`.
   - Update sections: Snapshot, Dependency Rules, Core Contracts, Configuration Flow, Known Gaps, JSON appendix, and Changelog.
4. **Edit `docs/LLD.md` and `docs/HLD.md` if needed**
   - Update diagrams or sequences if DI, handlers, or factories changed.
   - Add changelog entries.
5. **Safety**
   - Never hallucinate or extrapolate beyond diff evidence.
6. **Commit format**
   - Docs-only update:  
     `chore(context): auto-sync CONTEXT.md (+LLD/HLD) after code changes`
   - Combined with code change:  
     `feat|fix: <summary>` + `chore(context): auto-sync …`

### 6.3 Example Commit

```

feat(retriever/hybrid): accept diapi.RetrieverDeps and expose rrf_k

chore(context): auto-sync CONTEXT.md (Resolved GAP-006; update JSON snapshot; changelog)
chore(lld): refresh sequence for retriever handler deps

```

---

## 7. Hooks & Optional Automation

> No CLI commands are required. The agent must self-update documentation as described in §6.  
> This section only defines optional automation policies for CI enforcement.

- **Pre-commit (recommended):**  
  If a patch modifies files in §6.1 but does not include doc updates (`docs/CONTEXT.md`, `LLD.md`, `HLD.md`), the agent should **automatically append** a commit:
```

chore(context): auto-sync CONTEXT.md (+LLD/HLD)

```

- **CI lint (recommended):**
- Ensure `docs/CONTEXT.md`’s `last_updated` is within ±3 days of the latest commit touching core/builder/providers/pipeline.
- Check that `Known Gaps` entries match `docs/code-review.md`.
- Validate that the Machine Appendix JSON block parses successfully.

---

## 8. Observability and Logging Rules

* Never print directly to stdout in production paths.  
* Use the shared `core.Observability` struct.  
* Always use `core.Logger`, `core.Tracer`, and `core.Meter` interfaces.  
* Ensure each pipeline stage emits metrics via `core.Observability.Meter`.  
* Record latencies and token usage in metrics when calling LLMs or retrievers.

---

## 9. Testing Enforcement

Agents must maintain or extend test coverage when altering code in these areas:

* Pipeline orchestrator and stages (`pipeline/sandwich_test.go`, `pipeline/declarative/*_test.go`)
* Provider families under `internal/providers/*`
* Core builder/registry logic

Use naming pattern `Test<Component>_<Behavior>`.

---

## 10. Summary

This file defines how coding agents maintain, reason about, and update Manglekit.  
The automation described here ensures that `docs/CONTEXT.md` always mirrors the true state of the codebase.

> *“An agent is only as smart as its context — keep it fresh, structured, and faithful.”*

---

## 11. Architecture Rules (Enforced)

- Layered dependencies:  
`core` must not import providers, pipeline, or root.  
`pipeline` must not import concrete providers.  
Providers import only `core`.  
- Provider registration:  
All providers register via typed `manglekit.Register[T, D, O]`.  
- Handlers per kind:  
Each kind must have a dedicated `core.ComponentHandler`.  
- Configuration binding via `sdk.FromConfig`.  
- Observability & lifecycle unified through `core.Observability`.

---

## 12. Implementation Checklists

### New provider (retriever, reranker, embedder, LLM, vector store, rules, state)

- Define `Options` implementing `core.ProviderOptions`.  
- Declare dependencies via `diapi` dep marker interfaces.  
- Register via `manglekit.Register[T, D, O]`.  
- Ensure a handler exists for that kind.  
- Add deterministic tests under the provider folder.  
- **Docs:** Ensure `docs/CONTEXT.md` (and LLD/HLD) are auto-synced after implementation.

### New orchestrator

- Create `Options` implementing `core.ProviderOptions` (kind = `core.KindOrchestrator`).  
- Register a typed factory and handler; avoid implicit selection.  
- Add tests for explicit component selection.  
- **Docs:** Ensure context documents are updated and known gaps refreshed.

### Core or DI changes

- Keep layering rules intact.  
- Update handlers/factories for signature consistency.  
- **Docs:** Bump `last_updated` across `CONTEXT.md`, `LLD.md`, and `HLD.md`.

---

## 13. Post-Change Actions

- Run all tests and static checks.  
- Verify architecture rules in `docs/rules/manglekit-arch.yml` pass.  
- Confirm context auto-sync commit exists.

---

## 14. Agent Safety Summary

* Modify only files with verifiable evidence from diffs.  
* Never hallucinate architecture changes.  
* Preserve YAML headers and Markdown formatting in all docs.  
* Always bump `last_updated` and maintain JSON appendix validity.  
* Never remove sections or metadata blocks from context documents.  
* Keep generated commits atomic and syntactically valid.  

---

## 15. Provider Test Architecture (Enforced)

This section defines the mandatory patterns for testing providers. You must identify if the provider's dependencies are **Internal (Manglekit)** or **External (Go Modules)** and apply the correct strategy.

---

### 15.1 Strategy 1: Internal DI / Config-First Test (Category 3)

* **Use Case:** Testing a provider's integration with the Manglekit framework (e.g., `dense`, `hybrid`).
* **Dependency Type:** Internal Manglekit providers (e.g., `VectorStore`, `Embedder`, `Retriever`).
* **Test Method:** Use `sdk.LoadWithRegistry` and a test YAML.
* **Error Solved:** Fixes the "could not find options type" error.

**The 3-Part Registration Rule (Mandatory):**
To fix the "could not find options type" parsing error, your test setup (e.g., `registerTestDeps`) **must** register **all three** components for *every provider* defined in your test YAML (the provider-under-test *and* its mock dependencies):

1. **The `ComponentHandler`:** (e.g., `retrievers.NewHandler()`). Tells the Builder *how* to build the `Kind`.
2. **The `Factory`:** (e.g., `dense.NewFactory()`, `mock_vs.NewFactory()`). Tells the Handler *what* to build for the `provider:` string.
3. **The `Options` Sample:** (e.g., `dense.Options{}`, `mock_vs.Options{}`). Tells the Config Loader *what Go type* to parse the YAML into.

---

### 15.2 Strategy 2: External Dependency / Unit Test (Category 1)

* **Use Case:** Testing a provider's core business logic in isolation (e.g., `llm`, `google-embedder`).
* **Dependency Type:** External Go modules or I/O (e.g., `genkit`, `http.Client`, `os.Getenv`).
* **Test Method:** Standard Go unit test. **DO NOT** use `sdk.LoadWithRegistry` or YAML.

**Rules for External Dependency Tests:**

1. **Do not use `sdk.LoadWithRegistry`.** This is not a DI integration test.
2. **Instantiate the provider directly:** Call the provider's Go constructor in your test (e.g., `provider, err := openai.NewProvider(opts)`).
3. **Mock the External I/O:** Do not mock the Manglekit `Registry`. Instead, mock the *external* dependency. For example, if the provider uses an `http.Client`, use `httptest.NewServer` and pass the mock server's URL into the provider's `Options`.
4. Call the provider's methods (e.g., `provider.Execute(...)`) and assert the results.