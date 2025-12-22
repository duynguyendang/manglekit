# Greenfield Application Development
- **I. Project Profile:**
    - **Description:** Building a completely new user-facing application (web or mobile) from the ground up.
    - **Core Challenge:** Defining a viable scope (MVP), achieving fast time-to-market, and establishing a scalable foundation for future growth.
- **II. Requirements Analysis Guide:**
    - **Key Information to Extract:** Core user personas, key user journeys, new revenue streams, competitive landscape.
    - **Critical NFRs to Prioritize (ISO 25010 aligned):**
        - **Performance Efficiency:** User experience (e.g., page load, API response). SLO: P95 API response time for core user journeys must be under 300ms.
        - **Reliability:** System availability and uptime.
        - **Maintainability:** Code quality, modularity, and ease of future updates.
    - **Primary Stakeholders to Identify:** Product Owner/Manager, End-Users, Business Sponsor.
- **III. High-Level Design & Proposal Guide:**
    - **Executive Summary & Vision Focus:** Emphasize innovation, speed-to-market, and creating a modern user experience to capture a new opportunity.
    - **Guiding Principles to Emphasize (WAF Aligned):**
        - **Operational Excellence:** The entire platform will be deployed via Infrastructure-as-Code (IaC) and integrated into a CI/CD pipeline, enabling automated and repeatable deployments.
        - **Performance Efficiency & Reliability:** We will leverage cloud-native, auto-scaling services and patterns like modularity and loose coupling to ensure the system is responsive and resilient under load.
        - **Cost Optimization:** We will prioritize serverless and managed services to minimize idle costs and ensure a low Total Cost of Ownership (TCO).
    - **Baseline Architecture Analysis Focus:** N/A (Omit section).
    - **Target Architecture Specification:**
        - **Core Architectural Pattern:** Microservices or Serverless.
        - **Diagram Type:** Layered C4 Model (Level 1: System Context & Level 2: Container Diagram).
        - **Key Components:** Web/Mobile Client, API Gateway, Authentication Service, domain-specific Microservices (e.g., Product Service, Order Service), Data Stores (SQL/NoSQL), Event Bus.
    - **Key Technology Decisions to Analyze:**
        1.  Cloud Provider (e.g., AWS, Azure, GCP).
        2.  Frontend Framework (e.g., React, Angular, Vue).
        3.  Backend Architecture (e.g., Microservices vs. Serverless vs. Modular Monolith).
    - **Implementation Roadmap Phases:**
        1.  **Phase 1: Foundation & MVP:** Setup CI/CD, IaC, core auth & user models. Build and deploy the Minimum Viable Product.
        2.  **Phase 2: Core Feature Expansion:** Build out the primary feature set based on prioritized user stories and feedback.
        3.  **Phase 3: Scale & Optimize:** Performance tuning, A/B testing, advanced monitoring, and hardening.
    - **Proposed Project Team Roles:** Project Manager, Product Owner, Lead Architect, Frontend/Backend Developers, UX/UI Designer, QA Engineer.
    - **Governance & Operating Model Focus:** Propose an **Agile/Scrum product team model** with a dedicated Product Owner to drive the backlog.
    - **Security & Compliance Focus:** Application Security (OWASP Top 10), Identity and Access Management (IAM), Secrets Management, Dependency Scanning, Data Encryption.
    - **Post-Project Operations & Maintenance Focus:**
        - **SRE (Site Reliability Engineering):** Implement monitoring against SLOs and establish an error budget to balance new feature velocity with stability.
        - **FinOps (Financial Operations):** Establish continuous cloud cost monitoring and reporting from day one to ensure TCO goals are met.
    - **Key Risks & Mitigations:**
        - **Risk: Scope Creep Delaying MVP.** Mitigation: Enforce a strict MVP definition and a disciplined backlog grooming process managed by the Product Owner.
        - **Risk: Poor User Adoption.** Mitigation: Incorporate user feedback loops early via wireframes, prototypes, and a beta testing program.
        - **Risk: Technical Debt from Rushing to Market.** Mitigation: Schedule regular refactoring sprints and maintain a high standard of automated test coverage.
