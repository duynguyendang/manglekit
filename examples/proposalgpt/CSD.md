# Conceptual Solution Design (CSD) — ProposalGPT

**Edition:** 2.0 (Integrated Playbooks & Governance)
**Concept:** Autonomous Solution Architect & RFP Response Agent
**Core Tech:** Go, Manglekit (Logic Engine), Google Genkit (AI).

## 1. Executive Summary

ProposalGPT is not merely a text generator; it is a **Governed Neuro-Symbolic Agent**. It automates the extraction of requirements from RFPs, selects the appropriate architectural strategy (Playbook), and generates compliant technical proposals.

It solves the "Stochastic Runtime" problem by using **Manglekit** to enforce:

1. **Strict Compliance:** Adherence to "Kill Criteria" (e.g., Budget, ISO Standards).
2. **Strategic Consistency:** Logic-based selection of Architecture (e.g., Microservices vs. Monolith) based on defined Playbooks.
3. **Self-Correction:** A feedback loop that forces the AI to fix hallucinations or missing technical details (e.g., "Strangler Fig" pattern).

---

## 2. System Architecture: The "Dual-Brain"

The system follows the Manglekit **Left Brain / Right Brain** topology.

| Component | Technology | Role | Responsibility |
| --- | --- | --- | --- |
| **The Left Brain** | **Mangle Engine** | **The Architect** | Performs deterministic reasoning. It selects the **Playbook**, identifies **Gaps**, and enforces **Validation Rules**. |
| **The Right Brain** | **Genkit / LLM** | **The Worker** | Performs probabilistic tasks: Extracting entities from raw Markdown, drafting prose, and summarizing risks. |
| **The Body** | **Go (SDK)** | **The Orchestrator** | Manages the state machine, loads Playbook data, and handles I/O. |

### 2.1 The Logic Pipeline

1. **Ingest:** Read `RFP.md`.
2. **Extract:** AI converts text to `ExtractedFacts` struct.
3. **Reflect:** Go Structs are flattened into Datalog Facts (e.g., `has_keyword("legacy")`).
4. **Reason (Gap Analysis):** Check for missing critical info (Budget, Deadline).
5. **Reason (Selection):** Select the correct **Playbook** (e.g., "Modernization") and **Tech Stack** (e.g., "Databricks").
6. **Execute:** Generate proposal content using Playbook templates.
7. **Validate:** Check output against Playbook constraints (e.g., "Must include Strangler Fig").

---

## 3. Knowledge Layer: The Playbooks

ProposalGPT is grounded in the **Solution Design Playbooks**. These are not just text chunks; they are structured knowledge objects loaded into the system.

### 3.1 Playbook Data Model

We map the Markdown content into Go structs for logic injection.

```go
// internal/knowledge/playbook.go
type Playbook struct {
    ID           string   `mangle:"playbook_id"`
    Name         string   
    Description  string   
    CriticalNFRs []string `mangle:"nfrs"` // e.g., "Reliability", "Performance Efficiency"
    Risks        []string // e.g., "Scope Creep", "Big Bang Failure"
    ArchPattern  string   `mangle:"arch_pattern"` // e.g., "Strangler Fig", "Lakehouse"
}

```

### 3.2 Supported Strategies (derived from file)

* **Greenfield:** For new apps. Focus on MVP & Speed.
* **Modernization:** For legacy refactoring. Focus on Strangler Fig & Reliability.
* **Lift & Shift:** For Datacenter exit. Focus on TCO & Migration Tooling.
* **Data Migration:** Focus on Data Integrity & Validation.
* **...and 6 others.**

---

## 4. Logical Blueprints (The "How")

This logic sits in `rules/*.dl` and drives the Manglekit Engine.

### 4.1 Gap Analysis (`rules/gap.dl`)

Before planning, ensure we know enough.

```datalog
% Alert if Budget is missing
missing_info("budget") :- not value(_, "budget", _).

% Alert if Cloud Preference is missing (unless agnostic)
missing_info("cloud_pref") :- 
    not value(_, "cloud_pref", _), 
    not has_keyword("multi-cloud").

% Steering: If critical info missing, ask questions instead of proposing.
route("ask_clarification") :- missing_info("budget").

```

### 4.2 Playbook Selection (`rules/selection.dl`)

Logic to pick the strategy based on RFP keywords.

```datalog
% Select "Modernization" if keywords "legacy" or "monolith" exist.
selected_playbook("modernization") :- has_keyword("monolith").
selected_playbook("modernization") :- has_keyword("legacy system").

% Select "Greenfield" if "scratch" or "new app" exists.
selected_playbook("greenfield") :- has_keyword("build from scratch").

% Select "Data Platform Modernization" if "data warehouse" and "slow queries" exist.
selected_playbook("data_platform_mod") :- 
    has_keyword("data warehouse"), 
    has_keyword("performance issues").

```

### 4.3 Validation & Guardrails (`rules/validation.dl`)

Enforcing the Playbook's specific constraints.

```datalog
% Rule: Modernization proposals MUST mention "Strangler Fig".
%
retry("Compliance Error: Modernization strategy requires the 'Strangler Fig' pattern.") :-
    selected_playbook("modernization"),
    value(_, "proposal_content", Content),
    not input_contains(Content, "Strangler Fig").

% Rule: Greenfield proposals MUST define an "MVP".
%
retry("Strategy Error: Greenfield projects must explicitly define an MVP scope.") :-
    selected_playbook("greenfield"),
    value(_, "proposal_content", Content),
    not input_contains(Content, "MVP").

% Rule: Halt if Budget < $50k for Databricks (Business Rule).
halt("Budget insufficient for Databricks architecture.") :-
    platform_choice("databricks"),
    value(_, "budget", B), B < 50000.

```

---

## 5. Implementation Structure

The source tree reflects the separation of Logic, Knowledge, and Action.

```text
proposalgpt/
├── cmd/
│   └── app/
│       └── main.go              # Entry Point
├── internal/
│   ├── actions/                 # AI Capabilities
│   │   ├── ingestor.go          # Markdown Reader
│   │   ├── extractor.go         # Genkit: RFP Text -> Struct
│   │   ├── planner.go           # Genkit: Playbook + Facts -> Outline
│   │   └── writer.go            # Genkit: Outline -> Prose
│   ├── knowledge/               # Static Knowledge Base
│   │   └── playbooks.go         # Playbook Definitions (Loaded from MD)
│   └── types/                   # Data Schema
│       └── rfp.go               # ExtractedFacts, ProposalDraft
├── rules/                       # Manglekit Logic
│   ├── gap.dl                   # Missing Info Detection
│   ├── selection.dl             # Playbook & Tech Stack Selection
│   └── validation.dl            # Compliance & Retry Logic
├── test/
│   └── rfps/                    # Sample RFPs (Greenfield.md, Migration.md)
└── go.mod

```

---

## 6. Execution Flow Example

**Scenario:** User uploads `LegacyCRM_RFP.md` asking to "replace our slow monolithic CRM with a microservices-based Salesforce integration."

1. **Ingestion:** `ingestor.go` reads the file.
2. **Extraction:** `extractor.go` creates `ExtractedFacts`:
* `Keywords: ["monolith", "CRM", "slow"]`
* `Budget: 200,000`


3. **Reflection:** Facts loaded into Engine.
4. **Selection:**
* Engine matches `monolith` + `CRM`.
* **Decision:** `selected_playbook("modernization")` AND `platform_choice("salesforce")`.


5. **Steering:** Route to `planner.go`.
6. **Context Injection:**
* Planner retrieves **Playbook 2 (Modernization)** & **Playbook 10 (CRM)**.
* Injects Risks: *"Big Bang Integration Failure"*.
* Injects NFRs: *"Data sync latency < 5 mins"*.


7. **Drafting:** AI writes the plan using these specific constraints.
8. **Validation:**
* Engine checks output.
* *Check:* Does it mention "Strangler Fig"? (Yes)
* *Check:* Does it address "Big Bang Risk"? (Yes)
* **Result:** `PROCEED`.



## 7. Next Steps for Developer

1. **Hydrate Knowledge:** Copy the content from `Solution Design Playbooks.md` into the `internal/knowledge/playbooks.go` map.
2. **Write Extraction Prompt:** Update `extractor.go` to specifically look for the "Key Information to Extract" listed in each Playbook (e.g., "Server inventory" for Playbook 3).
3. **Deploy Rules:** Save the Datalog blocks above into the `rules/` directory.