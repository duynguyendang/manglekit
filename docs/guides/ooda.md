# Building OODA Applications

> Moved from the README in v0.6. For the high-level design see
> [manglekit-hld.md](../../../docs/design/manglekit-hld.md); for current-state
> context see [docs/context/architecture/ooda-loop.md](../../../docs/context/architecture/ooda-loop.md).

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
> built** — see [ROADMAP.md](../../ROADMAP.md). Today you author policies directly as
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

    "github.com/duynguyendang/manglekit"
    "github.com/duynguyendang/manglekit/core"
)

func main() {
    ctx := context.Background()

    // QuickClient wires the live engine + fail-closed supervisor and loads
    // the policy in one step. (sdk.NewClient + sdk.WithPolicyPath is the
    // explicit equivalent: sdk.NewClient + sdk.WithPolicyPath.)
    client, err := manglekit.QuickClient(ctx, "policies/security_gate.dl")
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

    // Or wrap a real core.Action so its Execute() is gated and register it
    // in one step:
    // client.RegisterSupervised("kubectl_deploy", myAction)
    // out, err := client.ExecuteByName(ctx, "kubectl_deploy", payload, env)
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

Manglekit provides structured error types for proper error categorization.
Deny errors are **actionable**: they carry the action name, the matched rule
text, and the tier, populated from the engine's audit trail (real rule
instantiation, not heuristics). `IsPolicyViolationError` matches both policy
violations and alignment errors, giving one block-detection idiom across the
Assess / Execute paths (`AssessPlan` reports denies as `DecisionHalt` instead).

```go
import (
    "errors"
    "github.com/duynguyendang/manglekit/core"
)

// Check for policy violations
if core.IsPolicyViolationError(err) {
    var pve *core.PolicyViolationError
    if errors.As(err, &pve) {
        log.Printf("action %q blocked by policy: rule %q (tier %s)",
            pve.ActionName, pve.MatchedRule, pve.Tier)
    }
}
```

For a full derivation tree of *why* an action was denied, use
`Client.Explain(ctx, query, facts)` or `mkit eval --explain` — see the
[Datalog guide](./datalog.md#explain-derivation-trees).

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

### Supervised Streaming

LLM actions can stream while staying behind the governance gate. The OpenAI
provider implements the Genkit streaming contract, and
`mkai.NewStreamingSupervisedAction` gates the stream: the supervisor's
pre-check runs **before** the first chunk (a policy deny means zero provider
requests), chunks stream to the caller as they arrive, and the post-check
(Reflect, fail-closed) runs on the assembled full response.

```go
action, err := mkai.NewStreamingSupervisedAction("summarize", generator, client.Engine())
if err != nil {
    panic(err)
}

chunks, err := action.Stream(ctx, envelope)
if err != nil {
    // Pre-check denied — no provider request was ever made.
    log.Printf("blocked: %v", err)
    return
}
for chunk := range chunks {
    if chunk.Err != nil {
        // Post-check denial surfaces as a terminal chunk carrying the
        // policy error; FinalEnvelope is only set when it passes.
        log.Printf("stream error: %v", chunk.Err)
        break
    }
    fmt.Print(chunk.Text)
}
_ = action.FinalEnvelope() // post-checked envelope (set on success)
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

