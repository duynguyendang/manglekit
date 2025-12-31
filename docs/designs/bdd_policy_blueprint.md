# High-Level Design (HLD): BDD Policy Blueprint (Gherkin Integration)

## 1. Design Objectives

The Gherkin Integration aims to provide a human-readable Domain Specific Language (DSL) for defining Manglekit governance policies.

* **Business-Logic Alignment**: Bridges the gap between high-level business requirements and low-level Datalog predicates.
* **Executable Specifications**: Allows Behavior-Driven Development (BDD) scenarios to function as active security and alignment gates.
* **Declarative Simplicity**: Reduces the need for developers to write raw Mangle/Datalog syntax directly.

---

## 2. Conceptual Mapping

To translate natural language into symbolic logic, Gherkin keywords are mapped to specific Manglekit primitives.

| Gherkin Keyword | Manglekit Structural Mapping | Datalog Implementation |
| --- | --- | --- |
| **Feature** | **Policy Set** | A logical grouping or a single `.dl` resource file. |
| **Scenario** | **Rule / Clause** | A single Datalog clause with a specific head (outcome). |
| **Given** | **Pre-conditions / Facts** | Queries against `label(L)`, `meta(K, V)`, or `json_*` facts. |
| **When** | **Trigger / Operation** | Bound to `action_operation(Entity, Name)`. |
| **Then** | **Decision Outcome** | System predicates: `halt(R)`, `retry(H)`, or `route(T)`. |

---

## 3. Architecture Overview: The Gherkin Compiler

A new **`GherkinCompiler`** component is introduced within the `internal/engine` layer to transform `.feature` files into executable logic.

### 3.1 Translation Pipeline

1. **Lexer/Parser**: Scans the Gherkin file and extracts Scenario metadata and Steps.
2. **Step Resolver**: Matches each Step against a registry of **Step Definitions** (pre-defined logic fragments).
3. **Datalog Generator**: Synthesizes the fragments into a formal Mangle clause.
4. **Runtime Injection**: Loads the resulting rules into the `MangleRuntime` during engine initialization.

---

## 4. Logical Transformation Flow

The transformation follows a strict logical structure where "Given" and "When" form the body of the Datalog rule, and "Then" forms the head.

**Scenario Logic Representation:**


**Example Blueprint:**

* **Gherkin**:
* `Given the user has "pii" label`
* `When calling "llm_generate"`
* `Then halt with "PII leakage detected"`


* **Generated Datalog**:
* `halt(Req, "PII leakage detected") :- action_operation(Req, "llm_generate"), label("pii").`



---

## 5. Execution Sequence

The BDD policies are enforced during the **Assess** (Pre-Check) and **Reflect** (Post-Check) phases of the Supervised Action lifecycle.

1. **Request Capture**: The `Supervisor` receives an `Envelope`.
2. **Logic Query**: The `PolicyEngine` executes the compiled Gherkin scenarios against the `Envelope` facts.
3. **Match Detection**: If a Scenario's conditions are met, the associated `Then` outcome (e.g., `halt`) is triggered.
4. **Feedback Injection**: The `Reason` defined in the Gherkin scenario is injected into the `Envelope` metadata as `mangle_feedback`.

---

## 6. Implementation Strategy: Step Definitions

To make Gherkin extensible, Manglekit provides a standard library of **Step Patterns**:

* **Contextual Steps**: "the input contains {value}", "the entity is labeled {label}".
* **Action Steps**: "calling the action {action_name}".
* **Validation Steps**: "the output schema matches {struct}", "the risk score is above {threshold}".

---