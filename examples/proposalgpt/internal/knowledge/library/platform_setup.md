# Platform Setup
- **I. Project Profile:**
    - **Description:** Building a shared, self-service internal platform to enable other development teams (e.g., a Kubernetes platform, a CI/CD platform, an MLOps platform).
    - **Core Challenge:** Balancing enablement (flexibility for developers) with governance (guardrails for security, cost, standards).
- **II. Requirements Analysis Guide:**
    - **Key Information to Extract:** Target developer personas, common workflows to be standardized, existing tooling pain points, governance requirements.
    - **Critical NFRs to Prioritize (ISO 25010 aligned):**
        - **Usability:** Developer Experience (DX) and ease of use.
        - **Reliability:** Uptime and stability of the platform itself. SLO: Core platform services (e.g., CI/CD) must have 99.9% availability.
        - **Security:** Enforcing security guardrails by default.
    - **Primary Stakeholders to Identify:** Internal Development Teams (as customers), Head of Platform/Engineering, Security/Compliance Team.
- **III. High-Level Design & Proposal Guide:**
    - **Executive Summary & Vision Focus:** Emphasize accelerating developer velocity, standardizing security and compliance, and reducing duplicated effort across the organization.
    - **Guiding Principles to Emphasize (WAF Aligned):**
        - **Operational Excellence:** The platform's goal is to enable operational excellence for its users through fully automated, self-service workflows and "paved road" templates.
        - **Security:** We will build security directly into the platform, providing secure-by-default templates, integrated scanners, and policy-as-code enforcement.
        - **Cost Optimization:** The platform will provide cost visibility and guardrails to its tenants, enabling federated cost management.
    - **Baseline Architecture Analysis Focus:** N/A (Omit section).
    - **Target Architecture Specification:**
        - **Core Architectural Pattern:** Platform as a Product.
        - **Diagram Type:** Layered C4 Model (Level 1: System Context & Level 2: Container Diagram).
        - **Key Components:** "The Platform" (as central system), "Developer" (as user), CI/CD Pipeline, Code Repository, Artifact Registry, Monitoring System, Security Scanners.
    - **Key Technology Decisions to Analyze:**
        1.  Core Platform Technology (e.g., Kubernetes vs. Serverless base).
        2.  CI/CD Tooling (e.g., Jenkins vs. GitLab CI vs. GitHub Actions).
        3.  Observability Stack (e.g., Prometheus/Grafana vs. Datadog).
    - **Implementation Roadmap Phases:**
        1.  **Phase 1: Core Infrastructure & Tooling:** Build the foundational cluster/service, networking, and CI/CD.
        2.  **Phase 2: Pilot Team Onboarding:** Work with a single "friendly" team to prove the platform's value and refine the developer experience.
        3.  **Phase 3: Self-Service Enablement & Scaling:** Build automation, documentation, and "paved road" templates for broad adoption.
    - **Proposed Project Team Roles:** Platform Product Manager, Lead Platform Engineer, DevOps/SRE Engineers, Developer Advocate/Support.
    - **Governance & Operating Model Focus:** Propose a dedicated **Platform Engineering Team** operating with a product mindset, responsible for the platform's reliability, features, and roadmap.
    - **Security & Compliance Focus:** "Policy as Code" (e.g., OPA Gatekeeper), Secure defaults in templates, Integrated SAST/DAST scanning in pipelines, Container security.
    - **Post-Project Operations & Maintenance Focus:**
        - **SRE:** The platform team's primary function is SRE for the platform itself, managing its reliability and performance for its internal customers.
        - **FinOps:** The platform team is responsible for managing the overall platform cost and providing showback/chargeback capabilities to tenant teams.
    - **Key Risks & Mitigations:**
        - **Risk: Building a Platform Nobody Wants.** Mitigation: Treat developers as customers. Involve pilot teams from day one to gather requirements and feedback.
        - **Risk: Platform Becomes a Bottleneck.** Mitigation: Focus heavily on self-service automation, clear documentation, and Infrastructure as Code (IaC).
        - **Risk: Poor Security Posture.** Mitigation: Implement "policy as code" and security scanning directly into the platform pipelines, making security the easy path.
