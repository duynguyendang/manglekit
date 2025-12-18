# Conceptual Solution Design (CSD) — Manglekit

**Edition:** 1.0
**Concept:** The Neuro-Symbolic Engine for Agents
**Core Tech:** Go, Google Genkit, Google Mangle.

## 1. The Challenge: Taming the Stochastic Runtime

Building reliable agents on probabilistic models (LLMs) presents a fundamental engineering paradox:
1. **Probabilistic Core:** LLMs are non-deterministic and creative.
2. **Deterministic Requirement:** Applications require strict adherence to logic, protocols, and type safety.

Current frameworks often over-abstract this problem or rely on fragile prompt engineering. Manglekit solves this by introducing a **Deterministic Control Plane** that wraps the probabilistic runtime.

The Mission: To provide a rigorous **Neuro-Symbolic Runtime** that enables developers to build agents that are inherently observable, reliable, and self-correcting.

---

## 2. The Solution: Dual-Brain Architecture

Manglekit unifies logic and execution into a single **"Supervised Action"**. It follows a **Left Brain / Right Brain architecture** to separate concerns:

### System Topology

| Component | Technology | Role | Responsibility |
| :--- | :--- | :--- | :--- |
| **The Left Brain** | **Google Mangle** | **The Logic Engine** | Holds the **Blueprints**. Performs deterministic reasoning, State Evaluation, and Routing. |
| **The Right Brain** | **Genkit / LLM** | **The Intuition** | Wraps the LLMs (Gemini, OpenAI, Llama). Performs probabilistic tasks (Reasoning, Generation). |
| **The Body** | **Go (Golang)** | **The Runtime** | The high-concurrency nervous system that binds Logic and Intuition together. |

### The Philosophy: "Wrap, Don't Build"

Manglekit is not a framework that dictates your application structure. It is a **Middleware Engine**.
* You bring the capabilities (Genkit flows, Tools, APIs).
* Manglekit wraps them in a **Supervisor** shell.
* The shell handles the **Feedback Loop**: If the Intuition fails (Logic Alignment Error), the Logic generates corrective feedback for a retry.

---

## 3. Core Architecture: The "Supervised Action"

The fundamental atomic unit of Manglekit is the **Supervised Action**. It wraps raw AI capabilities inside a deterministic logic shell.

**The Execution Lifecycle:**

1.  **Context Injection (Go):** The SDK captures the request and injects context (User Role, Budget, History) into the Runtime.
2.  **Blueprint Check (Mangle):** The Engine consults the **Blueprint** (Datalog).
    * *Logic:* "Does this request meet the preconditions?"
    * *Outcome:* If valid, proceed. If invalid, return a structured `AlignmentError`.
3.  **Execution (Genkit):** The Request is passed to the AI Adapter. The AI performs the task (e.g., Generate Code).
4.  **Evaluation & Steering (Mangle):** The output is checked against the Blueprint. The Engine makes a Decision:
    * *Correction (Retry):* If the error is recoverable (e.g., Syntax Error), feed feedback to AI and loop.
    * *Steering (Route):* If the result requires escalation (e.g., High Risk), route to a different handler (e.g., Human Review) without re-prompting the AI.
    * *Rejection (Deny):* If the violation is critical, halt execution immediately.

---

## 4. System Building Blocks

| Component | New Terminology | Role |
| :--- | :--- | :--- |
| **Logic Store** | **The Blueprint** | The "Source of Truth". Defines the SOP and Logic Structure. Decoupled from execution. |
| **SDK** | **Client** | The entry point. Developers use client.Manage() to wrap capabilities. |
| **Interceptor** | **Supervisor** | The middleware that enforces the **Blueprint**. Manages the state machine. |
| **Drivers** | **Integrations** | Bridges to external capabilities (LLMs, Vector DBs). Uses the **Clean Adapter Pattern**. |

---

## 5. Technical Capabilities

### 5.1 Neuro-Symbolic Steering
We use logic predicates to control the execution flow.
* **Mechanism:** `next_step("review_phase") :- output.confidence < 0.9.`
* **Result:** The runtime dynamically routes the execution based on logic inference, not hardcoded switches.

### 5.2 Resilience Layer (Circuit Breaker)
* **Problem:** Downstream AI providers may experience outages or latency spikes.
* **Solution:** A native, zero-dependency **Circuit Breaker** ensures the runtime Fails Fast to preserve system resources.

### 5.3 Zero-Config Reflection
* **Feature:** Automatic mapping of Go structs to Datalog facts.
* **Benefit:** Your Go types *are* the schema. No manual serialization code required.

### 5.4 Logical Observability
* **Feature:** Deep tracing of the reasoning process.
* **Output:** OpenTelemetry spans show exactly which **Logic Rule** triggered a decision, providing full auditability of the agent's behavior.

---

## 6. Developer Experience:

1.  **Go-Native:** No Python bloat. No sidecars. Manglekit compiles to a single, lightweight binary perfect for Kubernetes or Edge.
2.  **Protocol-First Development:** Don't bury logic in if/else statements. Write protocols in .dl files and hot-swap them instantly without recompiling.
3.  **Zero-Guesswork:** With `GenerateStruct[T]`, your Go types are the schema. The AI speaks your language, not the other way around.
4. **Type-Safety:** Enforces strong typing at the boundaries between Neural (AI) and Symbolic (Logic) components.
5.  **Observability:** Every "thought" (LLM call) and every "decision" (Mangle rule) is traced via OpenTelemetry. You can see exactly *why* the Agent acted the way it did.

---

## 7. Summary

**Manglekit is the Neuro-Symbolic Engine for Go**

It provides the **Control Plane** for probabilistic AI, ensuring that agents are:
* **Deterministic** (Guided by Blueprints)
* **Reliable** (Self-Correcting)
* **Observable** (Logical Tracing)
