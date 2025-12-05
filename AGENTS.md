# AGENTS.md — Manglekit Coding Agent Configuration

*Last updated: 2025-11-27*

---

## Overview

This document defines how coding agents interact with the **Manglekit v1.0** codebase.
It specifies conventions, automated tasks, and safety rules to ensure that all AI-driven modifications stay consistent with the system design and context documentation.

Agents use this file as an **operational manual** to perform reasoning, refactoring, documentation updates, and observability instrumentation within the Go SDK.

Agents must treat the documentation set (`CONTEXT.md`, `HLD.md`, `LLD.md`, `ADR.md`, `ARCHITECTURE_RULES.md`) as the synchronized **architecture source of truth**.
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
   - If the context is **older than 3 days**, treat it as *potentially stale* and rely on the code diff as the authoritative source.
   - If up-to-date, use it directly as the architectural baseline.

### 3.2 How to Use the Context for Reasoning

Use the document’s internal structure as a guide:

| Section | Purpose |
|----------|----------|
| **Implementation Snapshot** | Understand the Client/Guard/Engine relationship. |
| **Dependency Rules** | Validate layering — ensure `guard` doesn't depend on `adapters`. |
| **Core Contracts** | Retrieve `Action` and `Envelope` definitions. |
| **Guarded Action Lifecycle** | Understand the Trace -> AuthZ -> Exec -> Validate flow. |
| **Known Gaps** | Identify open architectural issues (e.g., Lineage). |
| **Changelog** | Understand recent architectural shifts (e.g., v1.0). |

### 3.3 Reflecting Code Changes

When code changes introduce **new features, functions, or modify workflows**, or affect architecture, interfaces, or runtime behavior, the agent must **edit `docs/CONTEXT.md` directly** —
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

- **New features, functions, or workflow changes.**
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

## 7. Hooks & Optional Automation

> No CLI commands are required. The agent must self-update documentation as described in §6.
> This section only defines optional automation policies for CI enforcement.

- **Pre-commit (recommended):**
  If a patch modifies files in §6.1 but does not include doc updates (`docs/CONTEXT.md`, `LLD.md`, `HLD.md`), the agent should **automatically append** a commit:
```
chore(context): auto-sync CONTEXT.md (+LLD/HLD)
```

- **CI lint (recommended):**
- Ensure `docs/CONTEXT.md`'s `last_updated` is within ±3 days of the latest commit touching core/guard/engine.

---

## 8. Observability and Logging Rules

* Never print directly to stdout in production paths.
* Use `core.Logger` injected via Context (`core.LoggerFromContext(ctx)`).
* Use `core.Tracer` via `Client` or `GuardedAction`.
* Ensure `GuardedAction` always starts a parent span.

---

## 9. Testing Enforcement

Agents must maintain or extend test coverage when altering code in these areas:

* `guard/`: Trace hierarchy and error handling.
* `engine/`: Policy evaluation and reflection.
* `adapters/`: Adapter correctness (mocking external drivers).

Use naming pattern `Test<Component>_<Behavior>`.

---

## 10. Architecture Rules (Enforced)

Agents must strictly adhere to the **Veto Rules** defined in `docs/ARCHITECTURE_RULES.md`.
Any code violating these rules must be rejected or refactored immediately.

**Key Constraints Summary:**
- **⛔ STRICT IMPORT BOUNDARIES**: User-space code must NEVER import `internal/`.
- **⛔ NO MANUAL LOGGING**: Use Guard Middleware; no `fmt.Println`.
- **⛔ TYPE SAFETY**: Use `manglekit.Define[In, Out]`.
- **⛔ NO 3RD-PARTY DEPS IN CORE**: Keep `core` lightweight.
- **⛔ CONTEXT PROPAGATION**: `ctx` is mandatory everywhere.
- **Layered dependencies**:
    - `guard` depends on `engine` and `core`.
    - `engine` depends on `core` and `google/mangle`.
    - `adapters` depend on `core` and external drivers.
    - `core` has NO dependencies.

---

## 11. Implementation Checklists

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

## 12. Architectural Patterns (Reference)

This section documents key architectural patterns for v1.0.

### 12.1 The "Guarded Action" Pattern

**Context:** We need to enforce policy and observability on *any* operation without modifying the operation itself.

**Implementation:**
- Use the Decorator pattern.
- `GuardedAction` implements `core.Action`.
- It wraps an inner `core.Action`.
- It executes `Trace -> Authorize -> Inner.Execute -> Validate`.

**When Implementing:** Always use `client.Protect()` to apply this pattern. Never manually construct `guard.GuardedAction` in user code.

### 12.2 The "Universal Adapter" Pattern

**Context:** We want to support many external libraries (Genkit, LangChain) without tight coupling.

**Implementation:**
- Define a thin adapter struct that holds the external client.
- Implement `core.Action` on the adapter.
- Map `core.Envelope` (Payload) to the external client's input format.
- Map the external client's output back to `core.Envelope`.

**When Implementing:** Create new packages in `adapters/` for new external libraries. Keep them thin.

### 12.3 Zero-Config Reflection

**Context:** We need to validate arbitrary Go structs in Datalog without writing manual mappers.

**Implementation:**
- Use `core/reflection` to walk the struct.
- Generate facts: `field(ID, "FieldName", Value)`.
- Use `engine.Reflector` in the Policy Engine.

**When Implementing:** Rely on `engine.Authorize/Validate` to handle conversion. Do not write custom `ToFacts` methods unless absolutely necessary for performance.

---

## 13. Provider Test Architecture

### 13.1 Strategy 1: Unit Test (Adapter Logic)

*   **Use Case:** Testing that an adapter correctly marshals data to/from the external driver.
*   **Method:** Mock the external driver (e.g., use a mock Genkit model or HTTP client).
*   **Goal:** Verify `core.Envelope` conversion.

### 13.2 Strategy 2: Integration Test (Policy Enforcement)

*   **Use Case:** Testing that `client.Protect()` correctly enforces rules.
*   **Method:**
    1.  Create a real `manglekit.Client` with a test policy file.
    2.  Wrap a mock Action.
    3.  Execute and assert that `PolicyViolationError` occurs when expected.
*   **Goal:** Verify the Guard/Engine interaction.

---

## 14. Documentation Cross-References

| Document | Purpose | When to Use |
|-----------|---------|------------|
| `docs/CONTEXT.md` | Live architecture snapshot | Reasoning about current system state |
| `docs/HLD.md` | High-level design & layering | Understanding system boundaries |
| `docs/LLD.md` | Low-level implementation details | Implementing new adapters or engine logic |
| `docs/ADR.md` | Architecture decisions | Understanding *why* (e.g., why Builder was removed) |
| `docs/LOGGING.md` | Observability conventions | Implementing logging |
| `docs/TRACING.md` | OTel span hierarchy | Debugging trace issues |
| `docs/ARCHITECTURE_RULES.md` | Architecture rules |  |

---

## 15. Agent Diagnostic Checklist

Before committing any code changes, verify:

### Code Quality Checks
- [ ] All new functions/types have clear documentation
- [ ] No global state
- [ ] Context used for Logger/Tracer access

### Architecture Compliance Checks
- [ ] **Veto Rules Check**: Verified against `docs/ARCHITECTURE_RULES.md`
- [ ] No illegal cross-layer imports
- [ ] Adapters do not depend on Guard or Engine
- [ ] Guard always creates a span

### Documentation Checks
- [ ] `docs/CONTEXT.md` updated
- [ ] `docs/LLD.md` updated if logic changed
- [ ] `last_updated` timestamp bumped in all modified docs

### Safety Checks
- [ ] No hallucinated code — all changes based on diff evidence
- [ ] YAML metadata preserved in all docs
- [ ] No removal of documented sections or warnings
- [ ] Commit is atomic and logically coherent

---

## 16. Summary & Core Principles

**The Manglekit SDK embodies these core principles:**

1.  **Universal Governance**: Everything is an Action; every Action is Guarded.
2.  **Composition**: Wrap existing objects; don't reinvent them.
3.  **Observability**: Tracing and Logging are first-class citizens.
4.  **Policy-Driven**: Logic Engine controls execution flow.

> *"An agent is only as smart as its context — keep it fresh, structured, and faithful."*