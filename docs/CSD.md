# Conceptual Solution Design (CSD) — Manglekit

**Edition:** 1.0
**Concept:** The AI Control Plane for Go
**Core Tech:** Go, Google Genkit, Google Mangle.

## 1. The Problem: The "Control Crisis" in AI

As organizations move from AI demos to production systems, they face three fundamental gaps:

1.  **The Reliability Gap:** Business logic requires **100% certainty**, but LLMs operate on **probability**. You cannot "prompt" an LLM into strict compliance.
2.  **The Visibility Gap:** Autonomous Agents are often "Black Boxes." Developers lack the mechanisms to understand *why* an agent made a decision or to intervene safely.
3.  **The Integration Gap:** Integrating Python-based AI logic with Go-based infrastructure creates a fragmented "Spaghetti Stack" that is hard to test and scale.

**The Need:** We need a way to enforce **deterministic rules** over **probabilistic AI**, leveraging the speed and type-safety of Go.

---

## 2. The Solution: A Native AI Control Plane

Manglekit solves this by establishing an **AI Control Plane** that unifies three powerful technologies into a single **"Guarded Action"** architecture.

### 2.1 The "Power Trio" Architecture

We do not reinvent the wheel. We orchestrate the best-in-class tools:

* **The Runtime (Go):** The foundation. Manglekit leverages Go's concurrency and static typing to create a single, high-performance binary with zero Python dependencies.
* **The Execution Layer (Google Genkit):** We use **Genkit** as the standardized driver for LLMs, Prompts, and Flows. It handles the "doing" (The Muscle).
* **The Governance Layer (Google Mangle):** We embed **Mangle**, a high-performance Datalog engine, directly into the Go binary. It handles the "thinking" (The Brain).

### 2.2 How They Work Together (Kernel vs. Driver)

| Layer | Technology | Role | Responsibility |
| :--- | :--- | :--- | :--- |
| **Control Plane** | **Google Mangle** | **The Manager** | Holds Policy, Context, and State. It makes deterministic decisions (Allow, Deny, Route) based on logic rules. |
| **Data Plane** | **Google Genkit** | **The Worker** | Wraps the LLMs and Tools. It executes the prompts and returns raw data, but *never* makes governance decisions. |
| **Host** | **Go (Golang)** | **The Container** | Provides the Type System (Reflection), Concurrency, and compilation target. |

---

## 3. Core Architecture: The "Guarded Action"

The fundamental building block is the **Guarded Action**. It wraps a **Genkit Flow** (or any Go function) inside a **Mangle Policy** shell.

**The Conceptual Flow:**

1.  **Intercept (Go):** The SDK intercepts the request. Using Go reflection, it converts the input struct into **Mangle Facts**.
2.  **Authorize (Mangle):** The **Mangle Engine** evaluates the facts against `.dl` policies.
    * *Rule:* `deny(Req) :- input(Req), contains_pii(Req).`
    * *Result:* If denied, execution stops immediately (Zero Cost).
3.  **Execute (Genkit):** If allowed, the request is passed to the **Genkit Adapter**. Genkit calls the LLM or Tool.
4.  **Validate (Go + Mangle):** The output is inspected. Mangle checks the result against safety constraints (Schema, Lineage).

---

## 4. System Building Blocks

| Component | Tech Stack | Role |
| :--- | :--- | :--- |
| **The SDK** | **Go Standard Lib** | The user-facing API. It allows developers to "Protect" native Go functions or Genkit Actions with a single line of code. |
| **The Guard** | **Go Middleware** | The interceptor that enforces the lifecycle. It ensures no Genkit action runs without a Mangle policy check. |
| **The Logic Engine** | **Google Mangle** | The embedded solver. It runs locally in-memory (no network calls), ensuring microsecond latency for policy checks. |
| **The Drivers** | **Genkit / MCP** | The I/O layer. Pre-built adapters allow Manglekit to control any Genkit-compatible model or MCP tool. |

---

## 5. Key Capabilities

### 5.1 Logic-Driven Routing (Neuro-Symbolic)
Instead of hard-coding flow logic in Go, we use **Mangle** to steer **Genkit**.
* *Mechanism:* The Mangle engine returns a `next_step` signal based on the current state.
* *Application:* The Go runtime reads this signal and invokes the corresponding Genkit tool.
* *Result:* **Software-Defined Agents** where behavior is defined in Datalog, not compiled code.

### 5.2 Zero-Config Reflection
* *Feature:* Manglekit automatically maps your existing Go structs (used by Genkit) into Mangle predicates.
* *Benefit:* You don't need to write manual conversion code. Your Go types *are* your policy schema.

### 5.3 Deep Observability (Logical Spans)
We trace not just the Genkit execution, but the Mangle reasoning.
* *Output:* OpenTelemetry traces show the exact **Mangle Rule ID** that triggered a decision, alongside the **Genkit Latency**.

---

## 6. Strategic Use Cases

### Pattern A: The Semantic Firewall
* **Goal:** Protect Genkit Flows from malicious inputs.
* **Approach:** Before invoking `genkit.Generate()`, Manglekit runs a `deny` check on the prompt. Malicious inputs are blocked at the logic layer, saving token costs.

### Pattern B: The Self-Correcting Agent
* **Goal:** Reliable Code Generation.
* **Approach:** If Genkit generates invalid code, the Mangle validator fails. The Go runtime catches this, triggers a `RETRY`, and feeds the error back to Genkit to fix the code automatically.

---

## 7. Developer Experience Goals

1.  **Go-Native:** No Python, no sidecars. Just `go get` and build a single binary.
2.  **Genkit-Compatible:** If you know Genkit, you already know how to use Manglekit. Just wrap your flows.
3.  **Policy-as-Code:** Define your business logic in `.dl` files, separate from your Go application code.

---

## 8. Summary

Manglekit is the **AI Control Plane** for Go.

It combines the execution power of **Google Genkit** with the governance precision of **Google Mangle**, creating a unified stack for building safe, reliable, and observable AI systems.

* **Runtime:** Go (Fast, Static).
* **Execution:** Genkit (The Muscle).
* **Governance:** Mangle (The Brain).