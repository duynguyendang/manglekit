[![Go](https://img.shields.io/badge/Go-1.25%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is the **Sovereign Neuro-Symbolic Logic Kernel** for Go.

> **Genkit 1.7 Integration**: Manglekit now supports all Genkit 1.7 middleware features including Retry, Fallback, Tool Approval, Filesystem, and custom middleware. See [Genkit 1.7 Middleware Features](#genkit-17-middleware-features) for details.

It solves the **Stochastic Runtime Paradox** of modern AI: applications require **Deterministic Reliability** (strict protocols, type safety, logic), but LLMs are inherently **Probabilistic** (creative, non-deterministic).

Manglekit bridges this gap by formalizing the agent lifecycle into an **OODA Loop** (Observe, Orient, Decide, Verify, Act) protected by a **Zero-Trust Supervisor** architecture:
1.  **The Brain (Symbolic)**: The Datalog Engine and **Tiered GenePool** (the `.dl` policy set) that handle verifiable reasoning and fail-closed verification.
2.  **The Planner (Neural)**: The Execution Runtime (Genkit) that drafts generative plans.
3.  **The Memory (Silo)**: A persistent BadgerDB storage layer for SPO facts and vector embeddings.

## Core Capabilities

1.  **OODA Loop Execution**: Orchestrates AI workflows using a structural Observe, Orient, Decide, Verify, Act pipeline.
2.  **Shadow Audit (Self-Correction)**: The *Verify* step evaluates AI-generated plans against the Tier 0/1/2 GenePool policies using Datalog *before* execution, in a **fail-closed** manner. If a policy is violated, the action is blocked rather than allowed through.
3.  **The Silo (Persistent Knowledge)**: Native BadgerDB integration providing high-performance SPOg (Subject-Predicate-Object-Graph) quad indexing and vector storage for long-term memory.
4.  **Extractors**: Built-in extractors capable of ingesting Markdown/Code into structured data. (Dynamic rule induction is planned — see [ROADMAP.md](./ROADMAP.md).)
5.  **Deep Observability**: Fully integrated OpenTelemetry tracing that links Genkit spans directly to logic rules, showing exactly *why* a decision was made.

## System Building Blocks

| Component | Role | Responsibility |
| :--- | :--- | :--- |
| **SDK** | **Client** | The entry point. Developers use `client.SupervisedAction()` to wrap capabilities. |
| **GenePool** | **Logic Store** | Datalog files (`.dl`) defining the Tier 0, 1, and 2 "Standard Operating Procedures" enforced by the engine. |
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

## Advanced: Policy Verification with the GenePool (Datalog)

The real power of Manglekit is a fail-closed **supervisor** that verifies every
action against a tiered set of Datalog policies — the **GenePool**. The GenePool is
the *set of `.dl` policy files* the engine loads in tiers (Tier 0 axioms, Tier 1
standard library, Tier 2 user policy); there is no separate Go "GenePool engine".
Verification is performed by the live engine on every supervised action.

> Rule induction (learning new Tier-2 rules from experience) is **planned, not yet
> built** — see [ROADMAP.md](./ROADMAP.md). Today you author policies directly as
> `.dl` files.

### Step 1: Define a Datalog Policy

```prolog
% policies/security_gate.dl

% Allow by default
allow(Action) :- request(Action).

% Block unapproved production deploys
deny(kubectl_deploy) :-
    request(kubectl_deploy),
    meta(target_env, "production"),
    meta(has_approval, "false").

% Block oversized replica scales (numeric pre-computed in Go)
deny(kubectl_scale) :-
    request(kubectl_scale),
    scale_too_high(_).
```

### Step 2: Enforce the Policy via the Supervisor

```go
package myapp

import (
    "context"
    "fmt"

    "github.com/duynguyendang/manglekit/core"
    "github.com/duynguyendang/manglekit/sdk"
)

func main() {
    ctx := context.Background()

    // NewClient wires the live engine + fail-closed supervisor.
    client, err := sdk.NewClient(ctx, sdk.WithBlueprintPath("policies/security_gate.dl"))
    if err != nil {
        panic(err)
    }

    // Build the action envelope (input facts + metadata) for an action.
    env := core.NewEnvelope(nil)
    env.Metadata["target_env"] = "production"
    env.Metadata["has_approval"] = "false"

    // Assess the action against the GenePool. The supervisor blocks
    // this action before any side effect happens.
    if err := client.Engine().Assess(ctx, core.ActionMetadata{Name: "kubectl_deploy"}, env); core.IsAlignmentError(err) {
        fmt.Printf("🚫 Blocked by policy: %v\n", err)
    }

    // Approve it and re-assess — now it passes.
    env.Metadata["has_approval"] = "true"
    if err := client.Engine().Assess(ctx, core.ActionMetadata{Name: "kubectl_deploy"}, env); err != nil {
        fmt.Printf("🚫 Blocked: %v\n", err)
    } else {
        fmt.Println("✅ Allowed: approved production deploy.")
    }

    // Or wrap a real core.Action so its Execute() is gated:
    // supervised := client.Supervise(myAction)
}

// Any core.Action (e.g. an adapter from adapters/func, adapters/ai, or your own)
// can be passed to client.Supervise so its Execute() runs behind the same gate.
```

A fully worked, runnable version of this gate (replica limits, business-hour
destroys, open security groups) lives in
[`manglekit-examples/devops_policy_gate`](https://github.com/duynguyendang/manglekit-examples).

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
    "github.com/duynguyendang/manglekit/core"
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

## OODA + Genkit Flows

Manglekit integrates the OODA loop with Genkit Flows, enabling **traced OODA phases** in the Genkit Dev UI, **streaming responses**, **middleware application**, and **flow registration** with Genkit's runtime.

### Running OODA as a Genkit Flow

```go
import (
    mkai "github.com/duynguyendang/manglekit/adapters/ai"
    "github.com/duynguyendang/manglekit/sdk/ooda"
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/googlegenai"
)

func main() {
    ctx := context.Background()
    g := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}))

    flow := mkai.NewOODAFlow(&mkai.OODAFlowConfig{
        MaxRetries:     3,
        Timeout:        5 * time.Minute,
        KnowledgeStore: myKnowledgeStore,
        TransientStore: myTransientStore,
    })

    flow.DefineFlow(g, "myOodaFlow")
    flow.DefineStreamingFlow(g, "myOodaStreamingFlow")

    result, err := flow.Run(ctx, &mkai.OODAFlowInput{
        Input:    "Generate an architecture document",
        Intent:   "document_generation",
        TaskType: ooda.TaskTypeGeneration,
    })
}
```

### Genkit Bridge & Flow Registry

```go
bridge := mkai.NewOODAGenkitBridge(g, &mkai.OODAFlowConfig{
    MaxRetries: 3,
})

bridge.DefineOODAFlow("documentGeneration")

registry := mkai.NewFlowRegistry(g)
registry.RegisterAndDefine("docGen", mkai.NewOODAFlow(cfg1))
registry.RegisterAndDefine("codeReview", mkai.NewOODAFlow(cfg2))
```

### Flow Input/Output Types

```go
type OODAFlowInput struct {
    Input    string          `json:"input"`
    Intent   string          `json:"intent,omitempty"`
    TaskType ooda.TaskType  `json:"task_type,omitempty"`
}

type OODAFlowOutput struct {
    Output         string                       `json:"output"`
    Status         ooda.VerifyStatus            `json:"status"`
    PhaseDurations map[ooda.Phase]time.Duration `json:"phase_durations"`
    RetryCount     int                          `json:"retry_count"`
    AuditTrail     string                      `json:"audit_trail,omitempty"`
    Error          string                      `json:"error,omitempty"`
}
```

### HTTP Endpoint for OODA Flow

Genkit automatically creates an HTTP endpoint for each defined flow:

```bash
curl -X POST http://localhost:8080/myOodaFlow \
  -H "Content-Type: application/json" \
  -d '{"data": {"input": "Generate architecture", "intent": "doc_gen"}}'
```

### Middleware in OODA Flows

All Genkit 1.7 middleware features are available for OODA-based generation. Middleware is applied per-generation when the OODA loop executes LLM-based actions.

### Available Middleware

| Middleware | Description | Use Case |
|-----------|-------------|----------|
| **Retry** | Automatic retry with exponential backoff | Transient failures, rate limits |
| **Fallback** | Automatic model fallback on failure | High availability, cost optimization |
| **Tool Approval** | Human-in-the-loop tool approval | Security-sensitive operations |
| **Filesystem** | Scoped file system access for models | Document processing, code generation |
| **Datalog Validator** | Pre/post-generation validation | Policy enforcement |
| **Telemetry** | Generation metrics collection | Observability, monitoring |
| **Logging** | Request/response logging | Debugging, audit trails |

### Using Middleware

```go
// Retry failed API calls up to 3 times
resp, err := generator.Generate(ctx, prompt, mkai.WithRetry(3))

// Fallback to alternative models on failure
resp, err := generator.Generate(ctx, prompt,
    mkai.WithFallback([]ai.ModelRef{
        googlegenai.ModelRef("googleai/gemini-2.5-flash", nil),
    }),
)

// Require approval for sensitive tools
resp, err := generator.Generate(ctx, prompt,
    mkai.WithToolApproval([]string{"read_file", "query_database"}),
)

// Enable scoped filesystem access
resp, err := generator.Generate(ctx, prompt,
    mkai.WithFilesystem("/app/data", true),
)
```

### Composing Multiple Middleware

```go
resp, err := generator.Generate(ctx, prompt,
    mkai.WithRetry(3),
    mkai.WithFallback(fallbackModels),
    mkai.WithToolApproval(allowedTools),
    mkai.WithFilesystem("/app/data", false),
    mkai.WithTelemetry(func(ctx context.Context, duration time.Duration, model string, inTokens, outTokens int) {
        metrics.Record(duration, model, inTokens, outTokens)
    }),
    mkai.WithLogging(logger),
)
```

### Custom Datalog Validator

```go
resp, err := generator.Generate(ctx, prompt,
    mkai.WithDatalogValidator(func(ctx context.Context, phase string, req *ai.ModelRequest, resp *ai.ModelResponse) error {
        if phase == "pre" {
            return validateInput(ctx, req)
        }
        return validateOutput(ctx, resp)
    }),
)
```

### OODA Frame with GenerateOptions

Configure the CognitiveFrame with middleware options that are passed to the TextGenerator during the Act phase:

```go
frame := ooda.NewCognitiveFrame(input, intent, taskType).
    WithGenerateOptions(
        mkai.WithRetry(3),
        mkai.WithFallback(fallbackModels),
    )

result, err := ooda.RunOODA(ctx, frame)
```

### MCP Tool Approval

Require approval for MCP tool execution:

```go
loader := mcp.NewLoader(cfg).
    WithMiddleware(mkai.WithToolApproval([]string{"read_file", "list_files"}))
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

    "github.com/duynguyendang/manglekit/core"
    "github.com/duynguyendang/manglekit/sdk"
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

## Datalog Engine Capabilities

The Manglekit Datalog engine (powered by [mangle-go](https://codeberg.org/TauCeti/mangle-go)) supports a rich set of predicates, functions, and operators for declarative logic.

### Comparisons (`:ge`, `:le`, `:gt`, `:lt`)

Compare values in rule bodies. Uses **integer** values (scale floats by 1000).

```prolog
% Quality gate: pass if score >= threshold
passes_gate(DocType) :-
    completeness_pct(DocType, Score),
    min_completeness_pct(DocType, Min),
    :ge(Score, Min).

% Overfitting detection: fail if effort > max
task_exceeds_max(Task) :-
    task_effort_pct(Task, E),
    max_effort_pct(Max),
    :gt(E, Max).
```

| Operator | Meaning | Example |
|----------|---------|---------|
| `:ge(A, B)` | A >= B | `:ge(870, 850)` → true |
| `:le(A, B)` | A <= B | `:le(100, 200)` → true |
| `:gt(A, B)` | A > B | `:gt(200, 150)` → true |
| `:lt(A, B)` | A < B | `:lt(700, 850)` → true |

> **Note**: mangle-go comparisons use `int64` only. For float comparisons, scale to integers (e.g., `0.85` → `850`).

### Negation (`!`)

Find cases where a condition is **not** met. All predicates must be defined in the program.

```prolog
% Find missing capabilities
has_capability("cloud", "deploy").
has_capability("cloud", "monitor").
needs_capability("cloud", "scale").
needs_capability("cloud", "deploy").

% "scale" is needed but not available
missing_capability(Project, Cap) :-
    needs_capability(Project, Cap),
    !has_capability(Project, Cap).
```

**Result**: `missing_capability("cloud", "scale")` — only `scale` is missing.

### Aggregation (`fn:count`, `fn:sum`, `fn:max`, `fn:min`)

Aggregate values using `|> do fn:group_by(...)` transforms.

```prolog
task_effort("M1.1", 3).
task_effort("M1.2", 5).
task_effort("M1.3", 2).

% Sum all efforts
total_effort(Total) :-
    task_effort(Task, E) |> do fn:group_by(Task), let Total = fn:sum(E).

% Find max effort
max_effort(MaxE) :-
    task_effort(Task, E) |> do fn:group_by(Task), let MaxE = fn:max(E).

% Count tasks
task_count(Count) :-
    task_effort(_, _) |> do fn:group_by(_), let Count = fn:count(_).
```

| Function | Description | Input | Output |
|----------|-------------|-------|--------|
| `fn:count(X)` | Count elements | list | int |
| `fn:sum(E)` | Sum values | list of int | int |
| `fn:max(E)` | Maximum value | list of int | int |
| `fn:min(E)` | Minimum value | list of int | int |
| `fn:avg(E)` | Average value | list of floats | float64 |

### Arithmetic (`fn:mult`, `fn:div`, `fn:plus`, `fn:minus`)

Arithmetic functions work within `|>` transforms.

```prolog
% PERT estimation: E = (O + 4*M + P) / 6
task_pert("M1.1", 2, 3, 5).

pert_estimate(Task, E) :-
    task_pert(Task, O, M, P),
    let E = fn:div(fn:plus(O, fn:plus(fn:mult(4, M), P)), 6).
```

| Function | Description | Example |
|----------|-------------|---------|
| `fn:plus(A, B)` | Integer addition | `fn:plus(3, 5)` → 8 |
| `fn:minus(A, B)` | Integer subtraction | `fn:minus(10, 3)` → 7 |
| `fn:mult(A, B)` | Integer multiplication | `fn:mult(4, 3)` → 12 |
| `fn:div(A, B)` | Integer division | `fn:div(19, 6)` → 3 |
| `fn:float:plus(A, B)` | Float addition | `fn:float:plus(1.5, 2.5)` → 4.0 |
| `fn:float:mult(A, B)` | Float multiplication | `fn:float:mult(10.0, 0.5)` → 5.0 |
| `fn:float:div(A, B)` | Float division | `fn:float:div(10.0, 3.0)` → 3.333... |

> **Note**: `let` bindings work inside `|>` transforms. Standalone `let X = fn:...` in rule bodies is not supported.

### String Operations

```prolog
% String contains
has_password(Text) :- fn:contains(Text, "password").

% String starts with
is_api_endpoint(URL) :- fn:starts_with(URL, "/api/").

% String concatenation
full_path(Base, Seg, Result) :- let Result = fn:string:concat(Base, Seg).
```

### Full Example: Quality Gate

```prolog
% Document metrics (integer-scaled: 0.87 → 870)
completeness_pct("BRD", 870).
consistency_pct("BRD", 920).
generic_pct("BRD", 100).

% Thresholds
min_completeness("BRD", 850).
min_consistency("BRD", 880).
max_generic("BRD", 200).

% Derived predicates
passes_completeness(D) :- completeness_pct(D, S), min_completeness(D, M), :ge(S, M).
passes_consistency(D) :- consistency_pct(D, S), min_consistency(D, M), :ge(S, M).
passes_generic(D) :- generic_pct(D, R), max_generic(D, M), :le(R, M).

passes_quality_gate(D) :-
    passes_completeness(D),
    passes_consistency(D),
    passes_generic(D).
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
    *   **Verify**: Evaluate the execution plan against Datalog GenePool policies (fail-closed Shadow Audit).
    *   **Act**: Safely execute capability (Tool, API Call) through the Zero-Trust Supervisor.

### Layer 3: The Zero-Trust Supervisor (Interceptor)

*   **Role**: The mechanical port that physically blocks unverified Actions.
*   **Lifecycle**: `Trace -> Check Proof -> Emit Spans`
*   **Pattern**: Middleware / Decorator for execution protocols.

### Layer 4: The Brain (Memory & Logic Store)

*   **Role**: The deterministic reasoning and storage layer.
*   **Components**:
    *   **The Silo**: Persistent BadgerDB storage for metadata, vectors, and facts (Quads).
    *   **Tiered GenePool**: The set of `.dl` policy files the engine loads by trust level (Axioms, Governance, User). AI-induced rule learning is planned — see ROADMAP.md.
    *   **Policy Solver**: Robust Datalog Evaluator supporting comparisons (`:ge`/`:le`/`:gt`/`:lt`), negation (`!`), aggregation (`fn:sum`/`fn:max`/`fn:min`/`fn:count`), arithmetic (`fn:mult`/`fn:div`/`fn:plus`/`fn:minus`), and stratified execution.
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

    "github.com/duynguyendang/manglekit/adapters/resilience"
    "github.com/duynguyendang/manglekit/core"
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
