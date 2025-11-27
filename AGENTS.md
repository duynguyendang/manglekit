# AGENTS.md — Manglekit Coding Agent Configuration (Genesis Edition)

*Last updated: 2025-11-27*

---

## Overview

This document defines how coding agents interact with the **Manglekit v3.0.0 (Genesis)** codebase.
It specifies conventions, automated tasks, and safety rules to ensure that all AI-driven modifications stay consistent with the system design and context documentation.

Agents must treat the documentation set (`CONTEXT.md`, `HLD.md`, `LLD.md`, `ADR.md`) as the synchronized **architecture source of truth**.
`CONTEXT.md` is the **live baseline** — always validate and update it whenever architecture or dependency injection logic changes.

---

## 1. Agent Goals

Coding agents working in this repository should be capable of:

* Understanding the **Universal AI Governance Kernel** architecture (`Client` -> `Guard` -> `Engine`).
* Updating context and documentation automatically when code changes in `docs/CONTEXT.md`.
* Maintaining observability and performance best practices.
* Identifying gaps and inconsistencies in adapter implementation or policy logic.

---

## 2. File Structure Awareness

Agents must recognize the following key locations and their purposes (aligned with repo layout):

* `manglekit.go`: The main entry point (`Client`, `Protect`).
* `core/`: Contracts (`Action`, `Envelope`, `Logger`, `Tracer`).
* `guard/`: The `GuardedAction` implementation (Trace -> AuthZ -> Exec -> Validate).
* `engine/`: The Policy Engine and Mangle Runtime.
* `adapters/`: Universal adapters (`ai`, `func`, `vector`).
* `config/`: Configuration loading.
* `docs/`: Technical docs and architecture rules.

---

## 3. Context-Aware Reasoning

Agents must treat `docs/CONTEXT.md` as the **live architectural memory** of the repository.

### 3.1 Context Loading and Validation

Before performing any modification, the agent must:

1. **Load the full content** of `docs/CONTEXT.md`.
2. **Verify freshness:** compare the `last_updated:` field to the most recent code commit.
   - If the context is **older than 3 days**, treat it as *potentially stale*.
   - If up-to-date, use it directly as the architectural baseline.

### 3.2 Reflecting Code Changes

When code changes affect architecture, interfaces, or runtime behavior, the agent must **edit `docs/CONTEXT.md` directly** —
updating sections, known gaps, and timestamps. This guarantees that the documentation always mirrors the true, current state of the system.

---

## 4. Editing Policies

* Never remove sections or metadata blocks from `CONTEXT.md`.
* Maintain YAML front-matter integrity.
* Use status icons (✅ / ⚠️ / ❌) consistently.
* Preserve Markdown headings, tables, and code formatting.
* Apply minimal, targeted updates with precise summaries.

---

## 5. Commit Conventions

Agents must follow semantic commit conventions:

```
feat: introduce new feature or adapter
fix: resolve bug or panic
refactor: improve structure without changing behavior
docs: update documentation (includes CONTEXT.md)
chore(context): auto-sync CONTEXT.md after code change
test: add or fix unit tests
```

---

## 6. Self-Managed Context Synchronization

> Goal: Whenever source changes affect architecture or runtime behavior, the **coding agent must directly update** `docs/CONTEXT.md` (and related docs) itself.

### 6.1 When to Trigger an Update

Trigger the **auto-sync process** whenever any of the following occur:

- Changes to `manglekit.go`, `guard/**`, `engine/**`, `core/**`, or `adapters/**`.
- Addition or modification of an **Adapter**.
- Changes to **Policy Logic** or **Reflection**.
- Changes in **observability** or **lifecycle**.

### 6.2 Self-Update Algorithm

1. **Collect diff context**: Identify modified files.
2. **Infer impact scope**: Identify affected components.
3. **Edit `docs/CONTEXT.md` directly**: Update Snapshot, Contracts, Gaps, Changelog.
4. **Edit `docs/LLD.md` and `docs/HLD.md` if needed**.
5. **Commit format**: `chore(context): auto-sync ...`

---

## 7. Observability and Logging Rules

* Never print directly to stdout in production paths.
* Use `core.Logger` injected via Context (`core.LoggerFromContext(ctx)`).
* Use `core.Tracer` via `Client` or `GuardedAction`.
* Ensure `GuardedAction` always starts a parent span.

---

## 8. Testing Enforcement

Agents must maintain or extend test coverage when altering code in these areas:

* `guard/`: Trace hierarchy and error handling.
* `engine/`: Policy evaluation and reflection.
* `adapters/`: Adapter correctness (mocking external drivers).

Use naming pattern `Test<Component>_<Behavior>`.

---

## 9. Architecture Rules (Enforced)

- **Wrap, Don't Build**: The framework does not construct objects; it wraps them.
- **Layered dependencies**:
    - `guard` depends on `engine` and `core`.
    - `engine` depends on `core` and `google/mangle`.
    - `adapters` depend on `core` and external drivers.
    - `core` has NO dependencies.
- **Context Propagation**: `ctx` must be passed through all layers.

---

## 10. Implementation Checklists

### New Adapter

- Implement `core.Action` interface.
- Define `Metadata()` returning name and type.
- Ensure `Execute(ctx, env)` passes context through.
- Do NOT start spans (leave that to Guard).

### Policy Logic Change

- Update `engine/policy.go` or `engine/runtime.go`.
- Ensure `Authorize` and `Validate` methods are updated.
- Update `docs/LLD.md` reflection section if needed.

---

## 11. Agent Diagnostic Checklist

Before committing any code changes, verify:

### Code Quality Checks
- [ ] All new functions/types have clear documentation
- [ ] No global state
- [ ] Context used for Logger/Tracer access

### Architecture Compliance Checks
- [ ] No illegal cross-layer imports
- [ ] Adapters do not depend on Guard or Engine
- [ ] Guard always creates a span

### Documentation Checks
- [ ] `docs/CONTEXT.md` updated
- [ ] `docs/LLD.md` updated if logic changed
- [ ] `last_updated` timestamp bumped

---

## 12. Summary & Core Principles

**The Manglekit SDK embodies these core principles:**

1.  **Universal Governance**: Everything is an Action; every Action is Guarded.
2.  **Composition**: Wrap existing objects; don't reinvent them.
3.  **Observability**: Tracing and Logging are first-class citizens.
4.  **Policy-Driven**: Logic Engine controls execution flow.

> *"An agent is only as smart as its context — keep it fresh, structured, and faithful."*