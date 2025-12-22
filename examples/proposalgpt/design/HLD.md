# ProposalGPT High-Level Design (HLD)

## 1. System Overview

ProposalGPT is an agentic application designed to automate the creation of comprehensive, professional software proposal drafts from Request for Proposal (RFP) documents. It utilizes a **Neuro-Symbolic** architecture, combining the probabilistic power of Large Language Models (LLMs) for unstructured text processing with the deterministic logic of Datalog for decision-making and governance.

### Core Philosophy
- **Dynamic Adaptability:** The system selects the most appropriate "Solution Design Playbook" based on the specific context of the RFP (e.g., Greenfiled vs. Modernization).
- **Governance-First:** Critical decisions (which playbook to use, whether to reject a request) are governed by transparent, auditable rules, not opaque prompt engineering.
- **Content-Guided Generation:** The generation process is strictly guided by the selected playbook's content, ensuring the output aligns with organizational best practices.

## 2. Architecture Diagram

```mermaid
graph TD
    User[User / CLI] -->|RFP File| Ingest[Ingestion Action]
    Ingest -->|Raw Text| Extractor[RFP Extractor (LLM)]
    Extractor -->|Structured Facts| Engine[Manglekit Engine (Datalog)]
    
    subgraph "Reasoning Layer"
        Engine -->|Facts| Selection[Selection Rules]
        Selection -->|Playbook ID & File| Config[Action Config]
    end
    
    Config -->|Load| Library[Playbook Library (Markdown)]
    Library -->|Playbook Content| Planner[Proposal Planner (LLM)]
    Engine -->|Context| Planner
    
    Planner -->|Draft Proposal| User
```

## 3. Key Components

### 3.1 Ingestion & Extraction Layer
- **Ingestion (`actions.IngestRFP`)**: Reads the raw markdown content of the provided RFP file.
- **RFP Extractor (`actions.NewExtractor`)**: A specialized LLM Action that analyzes the raw text to extract a structured schema (`ExtractedFacts`).
    - **Inputs**: Raw RFP text.
    - **Outputs**: JSON containing `Summary`, `Keywords`, `Budget`, `Compliance`, and `OriginalContent`.
    - **Role**: Converts unstructured data into facts that the reasoning engine can process.

### 3.2 Reasoning Engine (The "Brain")
The Manglekit Engine is the central orchestrator, powered by Datalog policies.

- **Fact Loading**: Extracted facts are converted into Datalog predicates (e.g., `has_keyword("cloud")`, `value("rfp", "budget", 50000)`).
- **Selection Logic (`selection.dl`)**:
    - analyzing keywords to determine the "Project Type".
    - identifying the correct Playbook ID (e.g., `modernization`, `greenfield`).
    - Mapping the ID to a specific filename (e.g., `modernization.md`).
- **Steering Logic**: Can define "Stop" conditions, such as missing critical information (e.g., Budget), prompting the user for clarification before proceeding.

### 3.3 Knowledge Management (Playbooks)
- **Playbook Library**: A collection of markdown files in `internal/knowledge/library/`. Each file represents a standardized approach to a specific type of project.
- **Dynamic Loading**: The system dynamically loads only the selected playbook. It parses structured metadata (NFRs, Risks, Pattern) while preserving the `RawContent` of the file to serve as the generation guideline.

### 3.4 Generation Layer (Planner)
- **Proposal Planner (`actions.NewPlanner`)**: The generative component.
- **Prompt Engineering**:
    - Injects the `RawContent` of the selected Playbook as the authoritative "Guideline".
    - Injects the `OriginalContent` of the RFP for context.
    - Explicitly asks for a long-form, multi-page output.
- **Output**: A `ProposalDraft` struct containing the full markdown proposal, which is printed to the console.

## 4. Data Flow

1.  **Input**: `project_brief.md` is passed to the CLI.
2.  **Extract**: LLM identifies keywords: `["build from scratch", "new app"]`.
3.  **Reason**: Datalog Selection Rule fires: `selected_playbook("greenfield") :- has_keyword("build from scratch")`.
4.  **Configure**: Engine returns `playbook_file="greenfield.md"`.
5.  **Load**: System reads `examples/proposalgpt/internal/knowledge/library/greenfield.md`.
6.  **Plan**: LLM generates proposal using `greenfield.md` as the instructions and `project_brief.md` as the data source.
7.  **Output**: Final Markdown Proposal.

## 5. Design Decisions

- **Markdown as Logic**: Playbooks are stored as human-readable markdown files. This allows solution architects to update the "logic" of the proposal generation (the structure, the required sections) by simply editing a document, without touching Go code.
- **Separation of Concerns**: The *Selection* of the strategy is deterministic (Rules), while the *Execution* of the strategy is creative (LLM). This prevents the LLM from hallucinating an inappropriate architecture for the given problem.
