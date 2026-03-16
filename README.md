[![Go](https://img.shields.io/badge/Go-1.25%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is the **Sovereign Neuro-Symbolic Logic Kernel** for Go.

It solves the **Stochastic Runtime Paradox** of modern AI: applications require **Deterministic Reliability** (strict protocols, type safety, logic), but LLMs are inherently **Probabilistic** (creative, non-deterministic).

Manglekit bridges this gap by formalizing the agent lifecycle into an **OODA Loop** (Observe, Orient, Decide, Verify, Act) protected by a **Zero-Trust Supervisor** architecture:
1.  **The Brain (Symbolic)**: The Datalog Engine and **Tiered GenePool** that handle verifiable reasoning and Shadow Audits.
2.  **The Planner (Neural)**: The Execution Runtime (Genkit) that drafts generative plans.
3.  **The Memory (Silo)**: A persistent BadgerDB storage layer for SPO facts and vector embeddings.

## Core Capabilities

1.  **OODA Loop Execution**: Orchestrates AI workflows using a structural Observe, Orient, Decide, Verify, Act pipeline.
2.  **Shadow Audit (Self-Correction)**: The *Verify* step mathematically proves AI-generated plans against Tier 0 Axioms in the GenePool using Datalog *before* execution. If a policy is violated, the loop self-corrects using real-time generative feedback.
3.  **The Silo (Persistent Knowledge)**: Native BadgerDB integration providing high-performance SPOg (Subject-Predicate-Object-Graph) quad indexing and vector storage for long-term memory.
4.  **Source-to-Knowledge Pipeline**: Built-in extractors capable of ingesting Markdown/Code and dynamically inducing Tier 2 Datalog policies.
5.  **Deep Observability**: Fully integrated OpenTelemetry tracing that links Genkit spans directly to logic rules, showing exactly *why* a decision was made.

## System Building Blocks

| Component | Role | Responsibility |
| :--- | :--- | :--- |
| **SDK** | **Client** | The entry point. Developers use `client.SupervisedAction()` to wrap capabilities. |
| **GenePool** | **Logic Store** | Datalog files (`.dl`) defining the Tier 0, 1, and 2 "Standard Operating Procedures". |
| **The Silo** | **Persistent Memory**| BadgerDB backed SPOg quad fact and vector storage. |
| **Supervisor** | **Interceptor** | The zero-trust gateway that enforces the GenePool on every action. |
| **Adapters** | **Drivers** | Universal adapters for LLMs (Genkit), Extractors, Tools (MCP), Functions, and Resilience. |

---

# Building OODA Applications

This section provides a comprehensive guide on building applications using the OODA (Observe-Orient-Decide-Verify-Act) cognitive loop in Manglekit.

## Understanding the OODA Loop

The OODA loop is a structured approach to AI agent cognition:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           OODA LOOP                                      │
│                                                                         │
│    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐      │
│    │ OBSERVE  │───▶│  ORIENT  │───▶│  DECIDE  │───▶│   ACT    │      │
│    └──────────┘    └──────────┘    └──────────┘    └──────────┘      │
│         │                │               │               │              │
│         │                │               │               │              │
│         └────────────────┴───────────────┴───────────────┘              │
│                              │                                           │
│                              ▼                                           │
│                       ┌──────────────┐                                    │
│                       │   VERIFY     │                                    │
│                       └──────────────┘                                    │
└─────────────────────────────────────────────────────────────────────────┘
```

| Phase | Description | Your Responsibility |
|-------|-------------|---------------------|
| **Observe** | Analyze and normalize raw input | Implement `Observer` interface |
| **Orient** | Retrieve domain context and rules | Implement `Orienter` interface |
| **Decide** | Formulate execution plan | Implement `Decider` interface |
| **Verify** | Validate plan against policies | Implement `Verifier` interface |
| **Act** | Execute and produce output | Implement `Actor` interface |

---

## The CognitiveFrame

Every OODA execution operates on a **CognitiveFrame** - the complete state of a single reasoning epoch:

```go
package ooda

// CognitiveFrame is the complete state of a single reasoning epoch.
type CognitiveFrame struct {
    ID        uuid.UUID
    Timestamp time.Time
    Intent    IntentStr
    Phase     Phase

    // Task Metadata
    TaskType   TaskType    // INDUCTION, GENERATION, AUDIT, RECOVERY
    OutputType OutputType   // PLAN (JSON/Markdown) or RULE (Datalog)

    // Input Stimulus
    Input string

    // Memory & Logic
    Context       []Atom         // Soft Logic (INT8) - Pruneable facts
    AttentionSink []Atom         // Hard Logic (FP32) - Immutable Axioms (Tier 0)
    ActiveGenes   []DomainGene   // Crystallized rules for this epoch
    RawContext    map[string]any // Legacy escape hatch

    // Reasoning
    Draft  any          // Neural proposal
    Proof  *AuditResult // Verification trace
    Status VerifyStatus

    // Telemetry
    TraceID        string
    SessionHistory []AuditResult
    EAST           EASTState

    // Staging
    IsProposal bool
}

// NewCognitiveFrame initializes a fresh cognitive epoch.
func NewCognitiveFrame(input string, intent IntentStr, taskType TaskType) *CognitiveFrame
```

### Key Types

```go
// Phase represents the phases in the OODA loop
type Phase string

const (
    PhaseObserve Phase = "observe"
    PhaseOrient  Phase = "orient"
    PhaseDecide  Phase = "decide"
    PhaseVerify  Phase = "verify"
    PhaseAct     Phase = "act"
)

// TaskType represents the operational mode for this epoch
type TaskType string

const (
    TaskTypeInduction  TaskType = "INDUCTION"  // Learning from raw input
    TaskTypeGeneration TaskType = "GENERATION" // Creating structured output
    TaskTypeAudit      TaskType = "AUDIT"      // System verification
    TaskTypeRecovery   TaskType = "RECOVERY"   // Error remediation
)

// VerifyStatus represents the result of the Datalog verification
type VerifyStatus string

const (
    VerifyStatusPending  VerifyStatus = "PENDING"
    VerifyStatusPassed   VerifyStatus = "FP32_PASSED"
    VerifyStatusFailed   VerifyStatus = "LOGIC_VIOLATION"
    VerifyStatusWarning  VerifyStatus = "WARNING"
)

// TrustTier represents the 4-level system of logical axiom trust
type TrustTier string

const (
    Tier0Kernel TrustTier = "TIER_0" // Immutable Core Axioms (Hard Logic - FP32)
    Tier1Admin  TrustTier = "TIER_1" // Human Operator / Governance
    Tier2AI     TrustTier = "TIER_2" // Induced / Learned Logic (Soft Logic - INT8)
    Tier3User   TrustTier = "TIER_3" // Untrusted External Input
)
```

---

## Building Your First OODA App

### Step 1: Implement OODA Phase Interfaces

```go
package myapp

import (
    "context"
    "fmt"

    "github.com/duynguyendang/manglekit/sdk/ooda"
)

// MyObserver implements the Observer phase
type MyObserver struct{}

func (o *MyObserver) Observe(ctx context.Context, frame *ooda.CognitiveFrame) error {
    fmt.Printf("[OBSERVE] Analyzing input: %s\n", frame.Input)
    
    // Extract facts from input
    frame.Context = append(frame.Context, ooda.Atom{
        Predicate: "input_received",
        Subject:   "user",
        Object:    frame.Input,
        Weight:    1.0,
    })
    
    // Set intent based on input analysis
    frame.Intent = ooda.IntentStr("document_generation")
    
    return nil
}

// MyOrienter implements the Orienter phase
type MyOrienter struct{}

func (o *MyOrienter) Orient(ctx context.Context, frame *ooda.CognitiveFrame) error {
    fmt.Printf("[ORIENT] Retrieving context for intent: %s\n", frame.Intent)
    
    // Retrieve relevant knowledge (e.g., from vector store)
    frame.Context = append(frame.Context, ooda.Atom{
        Predicate: "domain_knowledge",
        Subject:   "architecture",
        Object:    "enterprise patterns",
        Weight:    0.8,
    })
    
    return nil
}

// MyDecider implements the Decider phase
type MyDecider struct{}

func (o *MyDecider) Decide(ctx context.Context, frame *ooda.CognitiveFrame) error {
    fmt.Printf("[DECIDE] Creating plan for: %s\n", frame.Intent)
    
    // Create execution plan
    frame.Draft = map[string]string{
        "action": "generate_document",
        "type":   "architecture",
        "format": "markdown",
    }
    
    return nil
}

// MyVerifier implements the Verifier phase
type MyVerifier struct{}

func (o *MyVerifier) Verify(ctx context.Context, frame *ooda.CognitiveFrame) error {
    fmt.Printf("[VERIFY] Validating plan: %v\n", frame.Draft)
    
    // Verify against policies (Datalog rules)
    frame.Status = ooda.VerifyStatusPassed
    frame.Proof = &ooda.AuditResult{
        Pass:          true,
        ViolationTier: "",
    }
    
    return nil
}

// MyActor implements the Actor phase
type MyActor struct{}

func (o *MyActor) Act(ctx context.Context, frame *ooda.CognitiveFrame) error {
    fmt.Printf("[ACT] Executing: %v\n", frame.Draft)
    
    // Generate actual output
    output := fmt.Sprintf("# Architecture Document\n\nGenerated for: %s\n\nContent...", frame.Input)
    
    // Store in frame for retrieval
    frame.RawContext["output"] = output
    
    return nil
}
```

### Step 2: Create and Run the OODA Loop

```go
package myapp

import (
    "context"
    "fmt"

    "github.com/duynguyendang/manglekit/sdk/ooda"
)

func main() {
    ctx := context.Background()

    // 1. Create OODA phase implementations
    observer := &MyObserver{}
    orienter := &MyOrienter{}
    decider := &MyDecider{}
    verifier := &MyVerifier{}
    actor := &MyActor{}

    // 2. Create the OODA Loop
    loop := ooda.NewLoop(observer, orienter, decider, verifier, actor)

    // 3. Run the loop
    input := "Generate an architecture document for a cloud migration project"
    
    frame, err := loop.Run(ctx, input, nil)
    if err != nil {
        fmt.Printf("OODA loop failed: %v\n", err)
        return
    }

    // 4. Retrieve the output
    output := frame.RawContext["output"]
    fmt.Printf("\nResult: %s\n", output)
}
```

---

## Advanced: Integrating with GenePool (Policy Verification)

The real power of Manglekit comes from integrating the OODA loop with the **GenePool** for policy-based verification:

### Step 1: Define Datalog Policies

```prolog
% policies/main.dl

% ==========================================
% TIER 0: Immutable Core Axioms
% ==========================================

% Allow by default
allow(Req) :- request(Req).

% ==========================================
% TIER 1: Governance Rules
% ==========================================

% Require approval for high-risk actions
deny(Req) :-
    request(Req),
    req_action(Req, Action),
    Action == "delete_production".

violation_msg("Cannot delete production resources without approval") :- deny(Req).

% Budget limits
deny(Req) :-
    request(Req),
    req_action(Req, "deploy"),
    req_cost(Req, Cost),
    Cost > 10000.

violation_msg("Cost exceeds budget limit of $10,000") :- deny(Req).

% ==========================================
% TIER 2: AI-Induced Rules
% =========================================%

% Pattern-based security rules
security_check(Req) :-
    request(Req),
    req_payload(Req, Text),
    contains(Text, "password"),
    contains(Text, "plaintext").
```

### Step 2: Implement Verifier with GenePool Integration

```go
package myapp

import (
    "context"
    "fmt"

    "github.com/duynguyendang/manglekit/sdk/ooda"
    "github.com/duynguyendang/manglekit/internal/genepool"
)

type PolicyVerifier struct {
    pool *genepool.GenePool
}

func NewPolicyVerifier(pool *genepool.GenePool) *PolicyVerifier {
    return &PolicyVerifier{pool: pool}
}

func (v *PolicyVerifier) Verify(ctx context.Context, frame *ooda.CognitiveFrame) error {
    fmt.Printf("[VERIFY] Checking policies for: %s\n", frame.Intent)
    
    // Build Datalog query from frame
    query := fmt.Sprintf(`request("%s"), req_action(Action).`, frame.Input)
    
    // Query GenePool
    results, err := v.pool.Query(ctx, query)
    if err != nil {
        return fmt.Errorf("policy check failed: %w", err)
    }
    
    // Check results
    if len(results) == 0 {
        frame.Status = ooda.VerifyStatusPassed
        frame.Proof = &ooda.AuditResult{
            Pass: true,
        }
        return nil
    }
    
    // Check for violations
    for _, result := range results {
        if action, ok := result["Action"]; ok {
            if action == "delete_production" {
                frame.Status = ooda.VerifyStatusFailed
                frame.Proof = &ooda.AuditResult{
                    Pass:           false,
                    ViolationTier:  ooda.Tier1Admin,
                    TierID:         "governance-001",
                    ConflictPath:   "main.dl:15",
                    EntropyDelta:   0.5,
                }
                return fmt.Errorf("policy violation: cannot delete production")
            }
        }
    }
    
    frame.Status = ooda.VerifyStatusPassed
    return nil
}
```

---

## Advanced: Multi-Turn Conversation with Memory

For applications requiring memory across multiple OODA iterations:

```go
package myapp

import (
    "context"
    "fmt"

    "github.com/duynguyendang/manglekit/sdk/ooda"
)

type StatefulOODA struct {
    loop       *ooda.Loop
    sessionID  string
    history    []ooda.CognitiveFrame
}

func NewStatefulOODA(loop *ooda.Loop, sessionID string) *StatefulOODA {
    return &StatefulOODA{
        loop:      loop,
        sessionID: sessionID,
        history:   []ooda.CognitiveFrame{},
    }
}

func (s *StatefulOODA) Execute(ctx context.Context, input string) (*ooda.CognitiveFrame, error) {
    // Create frame with session context
    frame := ooda.NewCognitiveFrame(input, "", ooda.TaskTypeGeneration)
    frame.TraceID = s.sessionID
    
    // Inject prior context if available
    if len(s.history) > 0 {
        lastFrame := s.history[len(s.history)-1]
        
        // Carry forward relevant context atoms
        for _, atom := range lastFrame.Context {
            if atom.Weight > 0.5 {
                frame.Context = append(frame.Context, atom)
            }
        }
        
        // Carry forward immutable axioms
        frame.AttentionSink = lastFrame.AttentionSink
    }
    
    // Run the loop
    result, err := s.loop.Run(ctx, input, frame)
    if err != nil {
        return result, err
    }
    
    // Store in history
    s.history = append(s.history, *result)
    
    return result, nil
}

// Usage
func main() {
    loop := ooda.NewLoop(&MyObserver{}, &MyOrienter{}, &MyDecider{}, &MyVerifier{}, &MyActor{})
    
    session := NewStatefulOODA(loop, "user-123")
    
    ctx := context.Background()
    
    // First turn
    result1, _ := session.Execute(ctx, "I need an architecture document")
    fmt.Printf("Turn 1: %v\n", result1.Status)
    
    // Second turn - carries context from turn 1
    result2, _ := session.Execute(ctx, "Make it for AWS")
    fmt.Printf("Turn 2: %v\n", result2.Status)
}
```

---

## Best Practices

### 1. Keep Phases Focused

Each OODA phase should have a single, clear responsibility:

```go
// ✅ Good: Focused responsibility
func (o *MyObserver) Observe(ctx context.Context, frame *ooda.CognitiveFrame) error {
    // Only analyze and classify input
    frame.Intent = classifyIntent(frame.Input)
    return nil
}

// ❌ Bad: Doing too much in one phase
func (o *MyObserver) Observe(ctx context.Context, frame *ooda.CognitiveFrame) error {
    // Don't retrieve knowledge here (that's ORIENT)
    // Don't create plans here (that's DECIDE)
    return nil
}
```

### 2. Use Trust Tiers Appropriately

```go
// Tier 0: Immutable facts (always verified)
frame.AttentionSink = append(frame.AttentionSink, ooda.Atom{
    Predicate: "system_requirement",
    Subject:   "compliance",
    Object:    "SOC2",
    Weight:    1.0,  // Hard fact
})

// Tier 2: AI-generated (verifiable but not guaranteed)
frame.Context = append(frame.Context, ooda.Atom{
    Predicate: "suggestion",
    Subject:   "llm",
    Object:    "use_microservices",
    Weight:    0.7,  // AI recommendation
})
```

### 3. Handle Verification Failures Gracefully

```go
func (d *MyDecider) Decide(ctx context.Context, frame *ooda.CognitiveFrame) error {
    // Create initial plan
    plan := createPlan(frame.Input)
    
    // Check if previous verification failed
    if frame.Status == ooda.VerifyStatusFailed {
        // Adjust plan based on feedback
        plan = adjustPlanForPolicy(frame.Proof, plan)
        frame.IsProposal = true  // Mark for re-verification
    }
    
    frame.Draft = plan
    return nil
}
```

---

## Error Handling

Manglekit provides structured error types for proper error categorization:

```go
import (
    "errors"
    "github.com/duynguyendang/manglekit-wip/core"
)

// Check for policy violations
if core.IsPolicyViolationError(err) {
    var pve *core.PolicyViolationError
    if errors.As(err, &pve) {
        log.Printf("Blocked by policy: %s at %s", pve.Tier, pve.RuleID)
    }
}
```

---

## Directory Structure

```
manglekit/
├── adapters/           # Drivers for External Systems (AI, MCP, Vector)
│   ├── ai/             # Google Genkit & LLM Adapters
│   ├── knowledge/      # N-Quads/RDF Knowledge Loaders
│   ├── mcp/            # Model Context Protocol Tools
│   └── resilience/     # Circuit Breaker
├── cmd/                # CLI Tools
│   └── mkit/           # The 'mkit' Developer Utility
├── config/             # Configuration Loading
├── core/               # Public Interfaces & Types (Action, Envelope)
├── docs/               # Architecture Documentation
├── internal/           # Private Implementation
│   ├── engine/         # The Datalog Logic Engine (Solver, Runtime)
│   ├── supervisor/     # The Governance Interceptor
│   ├── genepool/       # Tiered Policy Management
│   └── ...
├── sdk/                # The User-Facing API (Client, Loop)
│   └── ooda/          # OODA Loop Implementation
└── examples/           # Runnable Demo Projects
    └── proposalgpt/   # Example OODA Application
```

---

## Quick Start

This example demonstrates the **Self-Correcting Loop**. We wrap an LLM capability in a "Supervised Action".

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/duynguyendang/manglekit-wip/core"
    "github.com/duynguyendang/manglekit-wip/sdk"
    "github.com/joho/godotenv"
)

func main() {
    ctx := context.Background()
    _ = godotenv.Load() // Load GOOGLE_API_KEY from .env

    // 1. Initialize Client from YAML Configuration
    client, err := sdk.NewClientFromFile(ctx, "mangle.yaml")
    if err != nil {
        log.Fatalf("Client Init Failed: %v", err)
    }
    defer client.Shutdown(ctx)

    // 2. Create a supervised action
    // The Supervisor automatically checks input/output against the GenePool.
    action := client.SupervisedAction(&myLLMAction{})

    // 3. Execute with governance
    result, err := action.Execute(ctx, core.NewEnvelope("Tell me a joke about security."))
    if err != nil {
        log.Fatalf("Task Failed: %v", err)
    }

    fmt.Printf("Result: %s\n", result.Payload)
}

// myLLMAction is a placeholder for your LLM capability
type myLLMAction struct{}

func (m *myLLMAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
    // Your LLM logic here
    return core.Envelope{Payload: "Here's a joke..."}, nil
}
```

### Configuration File (`mangle.yaml`)

```yaml
# Policy configuration
policy:
  path: "${POLICY_PATH:-./policies/main.dl}"
  evaluation_timeout: 30

# Observability configuration
observability:
  enabled: true
  service_name: "${SERVICE_NAME:-manglekit-app}"
  log_level: "${LOG_LEVEL:-info}"

# Pre-defined Actions
actions:
  llm_google:
    type: llm
    provider: google
    options:
      model: gemini-pro
      temperature: 0.7
```

### Defining Policies (`main.dl`)

Manglekit uses **Datalog** to define governance logic. It's like SQL but for rules.

```prolog
// Allow requests by default
allow(Req) :- request(Req).

// The "Quality Control" Rule
// If the joke contains "password", deny it.
deny(Req) :-
    request(Req),
    req_payload(Req, Text),
    fn:contains(Text, "password").

violation_msg("Do not mention passwords in jokes.") :- deny(Req).
```

---

## Architecture

Manglekit is a **Sovereign Logic Kernel** built on four core layers:

### Layer 1: The Client (SDK)

*   **Role**: Orchestrates the entire governance flow
*   **Responsibilities**: Holds configuration, manages the Cognitive Loop, and coordinates observability.
*   **Entry Point**: `manglekit.NewClient()` initializes the kernel with policy rules.

### Layer 2: The Cognitive Loop (OODA)

*   **Role**: An intelligent orchestration layer that binds logic to execution.
*   **Lifecycle**: `Observe -> Orient -> Decide -> Verify -> Act`
    *   **Observe**: Ingest raw signals and extract logical quad facts (SPOg) and embeddings into The Silo.
    *   **Orient**: Align input context against The Silo and Tiered Policy Rules.
    *   **Decide**: Generate an execution plan via the LLM Driver.
    *   **Verify**: Mathematically prove the execution plan against Datalog GenePool policies (Shadow Audit).
    *   **Act**: Safely execute capability (Tool, API Call) through the Zero-Trust Supervisor.

### Layer 3: The Zero-Trust Supervisor (Interceptor)

*   **Role**: The mechanical port that physically blocks unverified Actions.
*   **Lifecycle**: `Trace -> Check Proof -> Emit Spans`
*   **Pattern**: Middleware / Decorator for execution protocols.

### Layer 4: The Brain (Memory & Logic Store)

*   **Role**: The deterministic reasoning and storage layer.
*   **Components**:
    *   **The Silo**: Persistent BadgerDB storage for metadata, vectors, and facts (Quads).
    *   **Tiered GenePool**: Segregates policies by trust level limits (Axioms, Governance, AI-induced).
    *   **Policy Solver**: Robust Datalog Evaluator natively supporting built-in comparison mapping and stratified execution.
*   **Guarantees**: Fast (microsecond latency), deterministic, testable, verifiable.

### Universal Adapters

Bridge external libraries into the kernel:

*   **`ai` Adapter**: Wraps Google Genkit models and embedders.
*   **`func` Adapter**: Wraps native Go functions as Actions.
*   **`mcp` Adapter**: Integrates Model Context Protocol (MCP) servers.
*   **`extractor` Adapter**: Performs semantic extraction using LLMs.
*   **`vector` Adapter**: Handles vector search and retrieval operations.
*   **`resilience` Adapter**: Provides Circuit Breaker functionality for failure resilience.

#### Resilience Adapter

The `resilience` adapter provides a zero-dependency Circuit Breaker that prevents failure amplification.

```go
import (
    "time"

    "github.com/duynguyendang/manglekit-wip/adapters/resilience"
    "github.com/duynguyendang/manglekit-wip/core"
)

func main() {
    var myAction core.Action

    config := resilience.CircuitBreakerConfig{
        FailureThreshold: 5,
        ResetTimeout:     30 * time.Second,
    }

    safeAction := resilience.NewCircuitBreaker(myAction, config)
    // If myAction fails repeatedly, safeAction returns resilience.ErrCircuitOpen
}
```

---

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache 2.0
