# Conceptual Solution Design (CSD) — Manglekit

**Edition:** Genesis (v1.0 Reboot)
**Context:** The AI Control Plane for Reliable Go Applications

## 1. The Core Problem We Solve (The Pain Point)

Building modern AI applications is fragile. When you chain multiple LLMs and tools, the system suffers from **"Garbage In, Garbage Out"** and **"Unpredictable Black Boxes."**

1.  **Code Fragility:** Writing explicit checks for safety and business logic directly in application code is messy, error-prone, and hard to maintain.
2.  **The Black Box Problem:** Controlling AI behavior via **Prompt Engineering** is fragile and non-testable. It’s impossible to guarantee compliance when the AI operates on **stochastic probabilities**.
3.  **The Integration Chaos:** Cloud-Native apps use Go, but AI logic is often Python-based, creating a slow, heavy, and complex stack.

---

## 2. The Genesis Philosophy: Control & Simplicity

Manglekit solves these problems by establishing an **AI Kernel**. We create a hard separation between **Policy (The Brain)** and **Execution (The Muscle)**.

### 2.1 The Core Concept: Kernel & Drivers

* **The Kernel (Manglekit):** Acts as the **Control Unit**. It handles Policy, Routing, and Audit. It is deterministic (logic-based).
* **The Drivers (Adapters):** Act as the **I/O mechanism**. Genkit and other libraries are drivers for the underlying models or tools.

### 2.2 Engineering Principles

* **Action-Centric:** Every operation (LLM call, DB query, API request) is a uniform **`Action`**. The Kernel treats them all equally.
* **Smart Guardrails (Logic + AI):** We combine **AI (Neuro)** for semantic understanding with **Datalog (Symbolic)** for enforcing math-based rules.
* **Safety by Default:** A default-deny architecture ensures no action executes without passing a logic gate.

---

## 3. The Architecture: The Safety Wrapper

Manglekit introduces the **Guarded Action** primitive (replacing ad-hoc chains). This is a transparent layer that wraps your business logic.

### 3.1 The Guarded Action Lifecycle
The Guard enforces a standard governance loop around every operation:

1.  **Reflection Layer:** Automatically projects your Go Struct state into Datalog Facts.
2.  **Pre-Check (Authorize):** The logic gate checks Authorization (RBAC), Input Validation (Schema), and Symbolic Routing (selects provider).
    * *Outcome:* Deny (zero cost) or Proceed.
3.  **Execution:** The inner Action runs (RAG, Tool, or Local function).
4.  **Post-Check (Validate & Audit):** The logic gate performs Data Masking, Consistency Checks, and records the Audit Log.

### 3.2 Native Go Struct Mapping
* **Zero Boilerplate:** The engine uses advanced reflection to traverse complex Go types (Pointers, Maps, Slices) and converts them into Datalog predicates automatically.
* **Benefit:** Developers define their data structures once, and Manglekit handles the messy **Struct-to-Fact translation** without requiring any manual mapping code.

---

## 4. Orchestration Modes

### 4.1 Auto-Pilot Routing (Agent Mode)
* **Pattern:** Dynamic Dispatch.
* **Use Case:** Complex problem solving requiring multi-step execution.
* **Mechanism:** The **Logic Router** evaluates symbolic rules to dynamically select the next tool.
    * *Example:* The Datalog rule `next_step(tool_B) :- result(tool_A, "incomplete")` automatically dictates the flow.

### 4.2 Pipeline Mode
* **Pattern:** Linear Chain.
* **Use Case:** High-throughput, low-latency tasks (e.g., simple filtering or processing).
* **Mechanism:** A pre-compiled sequence of checks and actions, providing high speed with zero reasoning overhead.

---

## 5. Key Capabilities (What Genesis Gives You)

### 5.1 The Logic Kernel (The Brain)
* **Deep Reflector:** Specialized module for converting complex Go types into a queryable Fact Base.
* **In-Memory Datalog:** Runs Google Mangle locally. Policies are **pre-loaded and validated at startup** (Immutable Policies). This ensures decisions are fast (microsecond latency) and stable.

### 5.2 Universal Adapters
* **LLM/Embedder:** Wraps Genkit Models.
* **Semantic Extraction:** Dedicated agents for converting messy text (Docs, Logs) into structured Facts for the logic engine.
* **Action:** Generic interface for wrapping native Go functions (`func(ctx, input) output`).

---

## 6. Strategic Use Cases

### Pattern A: The Semantic Firewall

**Goal:** Ensure every input and output adheres to business logic and safety rules.
**Solution:**
1.  **Extract:** Use Semantic Extraction to convert prompts into logical facts (e.g., `intent="attack"`).
2.  **Filter:** Execute Datalog rules on the facts to guarantee safety.
    **Outcome:** Deterministic blocking of malicious inputs based on *meaning*, not just keywords.

### Pattern B: The Reliable Agent Chain

**Goal:** Build multi-step workflows that are robust and self-correcting.
**Solution:**
1.  **Guarded Action:** Ensures every step is validated before passing data forward.
2.  **Data Ancestry:** Tracks data origin across the entire chain.
    **Outcome:** Enables complex workflows like **Architectural Linting** (checking generated code against structural rules) or **Self-Correction** when facing validation errors.

### Pattern C: Edge AI Control

**Goal:** Enable secure, low-latency AI governance in constrained environments.
**Solution:**
1.  **Local Runtime:** Run Manglekit as a static Go binary on Edge Gateways (IoT).
2.  **Local Logic:** Enforce safety and execution rules locally (e.g., "Do not open door if risk is high") without relying on cloud connectivity.

---

## 7. Summary

Manglekit is the **Infrastructure Layer** for your AI. It moves the complexity of AI governance from "Prompt Engineering" (Fragile) to **"Systems Engineering"** (Robust).

* **Language:** Go.
* **Paradigm:** Neuro-Symbolic.
* **Deliverable:** Single Static Binary / Library.