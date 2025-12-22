# Lift & Shift Migration (Rehosting)
- **I. Project Profile:**
    - **Description:** Moving an existing application from an on-premises data center to a cloud IaaS provider with minimal or no code changes.
    - **Core Challenge:** Ensuring compatibility, managing network connectivity, and executing the cutover with minimal downtime.
- **II. Requirements Analysis Guide:**
    - **Key Information to Extract:** Server inventory, application dependency mapping, network topology, acceptable downtime window, data center exit deadline.
    - **Critical NFRs to Prioritize (ISO 25010 aligned):**
        - **Cost Optimization:** Achieving target cloud spend and TCO reduction.
        - **Reliability:** Minimizing downtime during cutover. SLO: Downtime during the final cutover window must not exceed the agreed maintenance window (e.g., 4 hours).
        - **Security:** Establishing a secure network perimeter in the cloud.
    - **Primary Stakeholders to Identify:** Infrastructure/Ops Team, Application Owners, CFO/Finance.
- **III. High-Level Design & Proposal Guide:**
    - **Executive Summary & Vision Focus:** Emphasize cost savings from hardware decommissioning, improved operational stability, and achieving a fast exit from the data center.
    - **Guiding Principles to Emphasize (WAF Aligned):**
        - **Cost Optimization:** The primary driver is TCO reduction. We will use right-sizing, reserved instances, and savings plans to achieve cost targets.
        - **Operational Excellence:** We will use Infrastructure as Code (IaC) to create a repeatable, auditable cloud foundation.
        - **Reliability:** We will leverage cloud-native backup and disaster recovery services to improve resilience over the on-premise environment.
    - **Baseline Architecture Analysis Focus:** Physical/virtual server specs, storage types, network dependencies, existing backup/DR process.
    - **Target Architecture Specification:**
        - **Core Architectural Pattern:** Infrastructure as a Service (IaaS) replica.
        - **Diagram Type:** Cloud Network/Infrastructure Diagram.
        - **Key Components:** On-Prem Data Center, Cloud VPC, Subnets (Public/Private), Security Groups/NACLs, VPN/Direct Connect, Migrated VMs/Servers (EC2/Azure VM), Cloud Storage (S3/Blob).
    - **Key Technology Decisions to Analyze:**
        1.  Cloud Provider (e.g., AWS, Azure, GCP).
        2.  Migration Tooling (e.g., AWS MGN, Azure Migrate).
        3.  Connectivity Method (e.g., Site-to-Site VPN vs. Direct Connect).
    - **Implementation Roadmap Phases:**
        1.  **Phase 1: Discovery & Foundation:** Analyze dependencies, provision target VPC and networking.
        2.  **Phase 2: Pilot Migration:** Migrate a non-critical component to test the process and tooling.
        3.  **Phase 3: Wave-Based Migration & Cutover:** Migrate applications in logical groups, perform cutover, and validate.
    - **Proposed Project Team Roles:** Project Manager, Cloud Infrastructure Architect, Network Engineer, Systems Administrator, Security Specialist.
    - **Governance & Operating Model Focus:** Propose a **Cloud Center of Excellence (CCoE)** to manage the new cloud environment, focusing on cost management, security, and operations.
    - **Security & Compliance Focus:** Network Perimeter Security (Security Groups, WAF), IAM Roles & Policies for infrastructure, Data Encryption (at rest and in transit), Patch Management strategy for VMs.
    - **Post-Project Operations & Maintenance Focus:**
        - **FinOps:** This is critical. Implement continuous cost monitoring, budget alerts, and automated shutdown policies for non-production environments to prevent cost overruns.
        - **SRE:** Establish baseline performance monitoring for migrated VMs and set up automated alerts for infrastructure health issues.
    - **Key Risks & Mitigations:**
        - **Risk: Unforeseen Application Dependencies.** Mitigation: Use automated discovery tools (e.g., AWS Application Discovery Service) and conduct thorough manual analysis.
        - **Risk: Performance Degradation.** Mitigation: Conduct pre- and post-migration performance benchmarking and right-size cloud instances.
        - **Risk: Cloud Cost Overrun.** Mitigation: Enforce a strict FinOps practice with cost monitoring, budget alerts, and a right-sizing strategy post-migration.
