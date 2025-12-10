# Conceptual Solution Design (CSD) — Manglekit

**Edition:** 1.0
**Concept:** The Neuro-Symbolic Engine for Agents
**Core Tech:** Go, Google Genkit, Google Mangle.

## 1. The Challenge: From "Chat Rooms" to "Assembly Lines"

The current AI landscape is dominated by probabilistic paradigms where agents talk to each other loosely. While creative, this approach is unworkable for  operations due to:

1.  **Non-Determinism:** You cannot build critical AI applications on top of probability. Business logic requires 100% adherence to protocols.
2.  **The "Infinite Loop" Risk:** Agents left to chat often hallucinate or get stuck. They need a "Conductor" to keep them on track.
3.  **The "Prototype Trap":** Python-based stacks are great for labs but struggle with the concurrency, latency, and type safety required for production scale.

**The Mission:** To shift AI from **Artisanal Creation** (Hand-crafted prompts) to **Industrial Production** (Reliable, Scalable, Self-Correcting Assembly Lines).

---

## 2. The Solution: A Neuro-Symbolic Agent Runtime

Manglekit is not just a "Guardrail" (which limits AI). It is an **Agent Operating System** that empowers AI to work correctly.
It unifies three distinct forces into a single **"Supervised Action"** architecture:

### 2.1 The "Power Trio" Architecture

| Component | Technology | Role | Responsibility |
| :--- | :--- | :--- | :--- |
| **The Left Brain** | **Google Mangle** | **The Logic Engine** | Holds the **Blueprints** (Datalog). It performs deterministic reasoning to Plan, Route, and Evaluate. |
| **The Right Brain** | **Genkit / LLM** | **The Intuition** | Wraps the LLMs (Gemini, OpenAI, Llama). Performs probabilistic tasks (Reasoning, Generation). |
| **The Body** | **Go (Golang)** | **The Runtime** | Provides the high-speed nervous system to bind Logic and Intuition together. |

### 2.2 The Philosophy: "Empower, Don't Just Block"

Instead of a Firewall (Allow/Deny), Manglekit acts as a **Senior Staff Engineer** supervising a Junior Developer (The AI):
* If the AI makes a mistake, Manglekit doesn't just block it; it provides **Corrective Feedback** and asks the AI to try again.
* This creates a **Self-Correcting Loop** that increases success rates from 80% to 99.9%.

---

## 3. Core Architecture: The "Supervised Action"

The fundamental atomic unit of Manglekit is the **Supervised Action**. It wraps raw AI capabilities inside a deterministic logic shell.

**The Execution Lifecycle:**

1.  **Context Injection (Go):** The SDK captures the request and injects context (User Role, Budget, History) into the Runtime.
2.  **Blueprint Check (Mangle):** The Engine consults the **Blueprint** (Datalog).
    * *Logic:* "Does this request meet the preconditions?"
    * *Outcome:* If valid, proceed. If invalid, return a structured `AlignmentError`.
3.  **Execution (Genkit):** The Request is passed to the AI Adapter. The AI performs the task (e.g., Generate Code).
4.  **Evaluation & Decision (Mangle):** The output is checked against the Blueprint. The Engine makes a Decision:
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

## 5. Strategic Capabilities

### 5.1 Programmable Flow Control We use logic to determine the Next Best Action, not just blindly retry.
* **Mechanism:** The Blueprint dictates whether to **Loop (with feedback)**, **Route (to another agent/human)**, or **Fail**.
* **Benefit:** Prevents "Infinite Retry Loops" on unrecoverable errors and enables complex workflows like "Escalation to Human".

### 5.2 Resilience Layer (Circuit Breaker)
* **Problem:** AI APIs fail or stall.
* **Solution:** Manglekit includes a native **Circuit Breaker**. If the AI provider has an outage, Manglekit "Fails Fast" to protect the system resources, preventing thundering herds.

### 5.3 Zero-Config Reflection
* **Feature:** Manglekit automatically maps your existing Go structs (used by Genkit) into Mangle predicates.
* **Benefit:** You don't need to write manual conversion code. Your Go types *are* your policy schema.

### 5.4 Deep Observability (Logical Spans)
We trace not just the Genkit execution, but the Mangle reasoning.
* **Output:** OpenTelemetry traces show the exact **Mangle Rule ID** that triggered a decision, alongside the **Genkit Latency**.

### 5.5 Adaptive Personas (Prompt-as-Config)

* **Feature:** Decouple prompts from code. Define personas in YAML and let the Logic Engine select the best persona for the context dynamically.
* **Benefit:** Change the Agent's behavior (from "Creative" to "Strict") via config/policy without redeploying binary.

---

## 6. Developer Experience: "Liquid Software"

1.  **Go-Native:** No Python bloat. No sidecars. Manglekit compiles to a single, lightweight binary perfect for Kubernetes or Edge.
2.  **Protocol-First Development:** Don't bury logic in if/else statements. Write protocols in .dl files and hot-swap them instantly without recompiling.
3.  **Zero-Guesswork:** With GenerateStruct[T], your Go types are the schema. The AI speaks your language, not the other way around.
4.  **Industrial Observability:** Every "thought" (LLM call) and every "decision" (Mangle rule) is traced via OpenTelemetry. You can see exactly *why* the Agent acted the way it did.

---

## 7. Summary

**Manglekit is the Neuro-Symbolic Engine for Agents**

It combines the execution power of Google Genkit with the governance precision of Google Mangle, creating a unified stack for building safe, reliable, and observable AI systems.

* **Reliable.** (Deterministic Protocols)
* **Scalable.** (Go Runtime)
* **Self-Correcting.** (Feedback Loops)
