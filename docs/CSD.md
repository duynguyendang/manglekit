# Conceptual Solution Design (CSD) for Manglekit

## Overview
Manglekit addresses key business needs in AI-driven knowledge management by providing a lightweight framework for controlled Retrieval-Augmented Generation (RAG). It integrates rule-based logic, semantic retrieval, and AI orchestration to deliver explainable, policy-compliant responses, reducing risks like hallucinations or data leaks in enterprise applications. This CSD maps high-level business requirements to a conceptual solution, ensuring alignment with product goals such as scalability for growing user bases, compliance with regulations (e.g., GDPR for redaction), and cost efficiency through modular integration.

The solution enables organizations to build trustworthy AI tools – from internal support systems to customer-facing chatbots – without vendor lock-in, supporting rapid iteration and ROI through embeddable Go-based deployment.

## Business Requirements
Based on stakeholder inputs, core requirements include:
- **Functional**: Support ontology-aware search with expansion to handle ambiguous queries (e.g., "app crash" → domain-specific troubleshooting), while enforcing policies for access control and content filtering.
- **Non-Functional**: Achieve <500ms response times for 100+ concurrent users; ensure 99% uptime via resilient design; maintain explainability for audit trails (e.g., regulatory compliance in finance/healthcare).
- **Product Goals**: Modular framework for easy adoption (e.g., integrate into existing apps); cost-effective (open-source core, pluggable paid LLMs); scalable to handle 1M+ documents without performance degradation.
- **Constraints**: Focus on text-based RAG; avoid complex graph databases to keep development lean (under 3 months to MVP); prioritize developer productivity over full enterprise features.

## Mapping Requirements to Solution
- **Requirement: Controlled Query Handling** → Solution: Sandwich Pattern (rules pre/post RAG) prevents drift, mapping to business need for accurate, scoped responses in knowledge-intensive workflows.
- **Requirement: Explainability & Compliance** → Solution: Rule annotations and redaction ensure traceable decisions, addressing audit needs and reducing liability (e.g., "why was this info filtered?").
- **Requirement: Scalability & Integration** → Solution: Pluggable components (e.g., Vector DB, LLM providers) allow horizontal scaling and easy embedding, supporting business growth without re-architecting.
- **Requirement: Cost Efficiency** → Solution: Go-based library minimizes runtime overhead; hybrid retrieval optimizes API calls to expensive LLMs.

## Key Concepts (Business Perspective)
- **Sandwich Pattern**: Wraps RAG with rules to align AI outputs with business policies, ensuring responses are relevant and safe – critical for trust in AI adoption.
- **Hybrid Retrieval**: Semantic + keyword search improves result quality, directly supporting requirements for high-recall knowledge discovery (e.g., in support tickets).
- **Rule Engine (Mangle)**: Declarative policies on facts (e.g., user roles, ontologies) enable customization per business unit, like restricting sensitive data in HR vs. engineering.
- **Orchestration (Genkit)**: Manages flows to route intents (e.g., Q&A vs. exploration), optimizing resource use and aligning with UX goals for seamless interactions.
- **Guardrails & Traceability**: Built-in annotations promote accountability, meeting compliance reqs while providing insights for product improvements (e.g., analyze rule firings for feature gaps).

## Use Cases
- **Internal Knowledge Base**: Developers query docs with role-based access; business benefit: Faster onboarding, reduced support tickets (20-30% time savings).
- **Compliance-Checked Chatbot**: Customer service bots redact PII and explain denials; ROI: Avoid fines, improve satisfaction via transparent AI.
- **Exploratory Analytics**: Sales teams expand queries over reports; value: Deeper insights without data scientist involvement, accelerating decisions.
- **Edge Deployment**: Offline-capable library in mobile/field apps; advantage: Low-latency, no cloud dependency for remote ops.

## Component Interactions and Data Flows (High-Level)
- **Query Flow**: User input → Orchestration (intent routing) → Pre-rules (scope/expand) → Retrieval (hybrid search) → Post-rules (validate/redact) → Synthesis (AI response with citations).
- **Ingestion Flow**: Async document upload → Chunking/embedding → Index update, supporting business need for dynamic knowledge updates.
- **Diagram**:
  ```
  [Business User/App] --> [Orchestrator (Flow Mgmt)]
                          |
                          v
  [Pre-Rules (Policy Scope)] --> [Retrieval (Semantic + Keyword)]
                          |                          |
                          v                          v
  [Post-Rules (Compliance Check)] --> [AI Synthesis (Guarded Response)]
                          |                          |
                          v                          |
  [Traceable Output (Auditable)] <-------------------
  ```
  Flows ensure end-to-end compliance, with modularity for business-specific customizations.

## Design Decisions and Rationale (Tied to Requirements)
- **Modular, Pluggable Design**: Addresses integration reqs by allowing swaps (e.g., open-source vs. enterprise Vector DBs), reducing vendor costs by 40-50%.
- **Rule-First Approach**: Prioritizes business control over raw AI power, mitigating risks like biased outputs – essential for regulated industries.
- **Lightweight Go Framework**: Meets performance/scalability goals with low overhead; rationale: Embeddable for diverse environments (cloud/edge), faster dev cycles.
- **Hybrid Over Pure Semantic**: Balances accuracy/cost; business impact: Higher user satisfaction, fewer follow-up queries.
- **Non-Functional Alignment**:
  - **Scalability**: Horizontal design supports growth; e.g., auto-scale for peak loads.
  - **Resilience/Security**: Fallbacks and policies ensure availability/compliance.
  - **Observability**: Metrics for ROI tracking (e.g., query success rate).

## Assumptions and Constraints
- Assumes pre-defined ontologies/policies provided by business; no auto-learning (to avoid complexity).
- Text-focused; multi-modal as phase 2.
- Budget: Open-source dev, optional cloud costs for prod DBs.
- Team: 3-5 engineers familiar with Go/AI.

## Next Steps
- Validate with stakeholders: Map to specific reqs via workshop.
- Prototype key use cases to demo business value.
- Refine based on feedback; transition to technical HLD/LLD.

This CSD ensures Manglekit delivers tangible business outcomes, from enhanced productivity to risk reduction.