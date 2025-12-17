---
context_type: full_source_dump
project: manglekit
language: go
last_updated: 2025-12-17
scan_mode: exhaustive
---

## 1. THE COMPLETE FILE MAP

```
.
├── AGENTS.md                                   # Operational manual for AI agents
├── CONTRIBUTING.md
├── LICENSE
├── Makefile
├── README.md
├── adapters                                    # Integration layer for external systems
│   ├── ai
│   │   ├── adapter.go                          # LLMAction implementation
│   │   ├── adapter_test.go
│   │   ├── genkit.go                           # Genkit -> TextGenerator adapter
│   │   └── utils.go
│   ├── extractor
│   │   ├── adapter.go                          # LLM-based structured data extraction
│   │   └── adapter_test.go
│   ├── func
│   │   └── wrapper.go                          # Generic function wrapper (ToolFunc)
│   ├── knowledge
│   │   ├── graph_loader.go
│   │   ├── graph_loader_test.go
│   │   ├── nquads.go                           # N-Quads parser
│   │   ├── nquads_test.go
│   │   ├── ntriples.go                         # N-Triples parser
│   │   ├── rdf.go                              # RDF loader interface
│   │   └── rdf_stub.go
│   ├── logger
│   │   └── zap_adapter.go                      # Zap logger adapter
│   ├── mcp                                     # Model Context Protocol
│   │   ├── action.go                           # MCP Tool -> Action adapter
│   │   ├── loader.go                           # MCP Server loader
│   │   └── loader_test.go
│   ├── resilience
│   │   ├── circuit_breaker.go                  # Circuit Breaker decorator
│   │   ├── circuit_breaker_test.go
│   └── vector
│       ├── genkit_retriever.go
│       ├── retriever_adapter.go
│       └── retriever_adapter_test.go
├── cmd
│   └── mkit                                    # CLI Tool
│       ├── commands
│       │   ├── eval
│       │   │   └── run.go
│       │   ├── gen
│       │   │   ├── inductor
│       │   │   │   ├── graph.go
│       │   │   │   ├── inductor.go
│       │   │   │   ├── inductor_test.go
│       │   │   │   └── topology_test.go
│       │   │   ├── logic.go
│       │   │   ├── logic_test.go
│       │   │   ├── resources.go
│       │   │   ├── root.go
│       │   │   └── rule.go
│       │   ├── inspect
│       │   │   ├── root.go
│       │   │   └── struct.go
│       │   ├── kg
│       │   │   ├── convert.go
│       │   │   └── root.go
│       │   └── serve
│       │       ├── serve.go
│       │       └── serve_test.go
│       └── main.go
├── config                                      # Configuration Management
│   ├── loader.go                               # Config loading logic
│   ├── loader_test.go
│   └── schema.go                               # Configuration struct definitions
├── core                                        # Domain Kernel (No dependencies)
│   ├── context_facts.go                        # Context propagation helpers
│   ├── context_lineage.go
│   ├── data.go                                 # Data interfaces (Memory, Knowledge)
│   ├── errors.go
│   ├── governance.go                           # Engine interfaces (Evaluator)
│   ├── infra.go                                # Infrastructure interfaces (Tracer)
│   ├── logger.go                               # Logger interface
│   ├── logger_test.go
│   ├── logic.go                                # Action & Generator interfaces
│   ├── memory.go                               # Semantic Memory interface
│   ├── state.go                                # StateProvider interface
│   ├── tracer.go
│   └── types.go                                # Domain Types (Envelope, Decision)
├── docs
│   ├── ADR.md
│   ├── ARCHITECTURE_RULES.md
│   ├── CONFIG.md
│   ├── CONTEXT.md                              # This file
│   ├── CSD.md
│   ├── HLD.md
│   ├── LLD.md
│   ├── LOGGING.md
│   ├── Mangle-quickstart.md
│   ├── TRACING.md
│   ├── VOCABULARY.md
│   └── reports
│       └── code-review.md
├── examples                                    # Usage Examples
│   ├── AGENTS.md
│   ├── README.md
│   ├── chat_chit
│   │   ├── main.go
│   │   └── protocol.dl
│   ├── config_driven_bot
│   │   ├── main.go
│   │   ├── mangle.yaml
│   │   └── policy.dl
│   ├── context_aware_rag
│   │   ├── blueprint.dl
│   │   ├── knowledge_base.nq
│   │   └── main.go
│   ├── dynamic_pricing
│   │   ├── inventory.csv
│   │   ├── main.go
│   │   └── pricing.dl
│   ├── extractor_demo
│   │   └── main.go
│   ├── fintech_approval
│   │   ├── credit.dl
│   │   ├── data.ttl
│   │   └── main.go
│   ├── kernel_knowledge_demo
│   │   └── main.go
│   ├── lineage_demo
│   │   └── main.go
│   ├── openrouter_demo
│   │   ├── README.md
│   │   ├── blueprint.dl
│   │   └── main.go
│   ├── planner
│   │   └── main.go
│   ├── policy_bot
│   │   ├── main.go
│   │   └── mangle.yaml
│   ├── rag_flow
│   │   └── main.go
│   ├── semantic_feedback
│   │   ├── blueprint.dl
│   │   └── main.go
│   ├── sre_guardrail
│   │   ├── main.go
│   │   └── safety.dl
│   ├── steering
│   │   ├── blueprint.dl
│   │   └── main.go
│   ├── taint_demo
│   │   └── main.go
│   ├── typed_invocation
│   │   └── main.go
├── go.mod
├── go.sum
├── internal                                    # Internal Implementation Details
│   ├── engine                                  # The Mangle Policy Engine
│   │   ├── dual_mode_test.go
│   │   ├── evaluator.go                        # Evaluator implementation
│   │   ├── evaluator_test.go
│   │   ├── flattener.go                        # JSON to Facts converter
│   │   ├── flattener_test.go
│   │   ├── memory
│   │   │   ├── volatile.go
│   │   │   └── volatile_test.go
│   │   ├── reflection.go                       # Struct to Facts converter
│   │   ├── reflection_test.go
│   │   ├── resources
│   │   │   ├── embed.go
│   │   │   ├── planner.dl
│   │   │   └── std.dl
│   │   ├── runtime.go                          # Mangle Runtime wrapper
│   │   ├── solver.go                           # PolicyEngine logic
│   │   └── solver_test.go
│   ├── logger
│   │   └── std.go                              # Standard library logger
│   ├── resources
│   │   └── icl
│   │       ├── embed.go
│   │       └── golden.dl
│   ├── statehelper
│   │   └── statehelper.go
│   ├── supervisor                              # The Guard Component
│   │   ├── supervisor.go                       # SupervisedAction Decorator
│   │   ├── supervisor_test.go
│   │   └── trace_test.go
│   ├── telemetry
│   │   └── otel.go                             # OpenTelemetry setup
│   ├── testproviders
│   │   ├── mock
│   │   │   └── mock.go
│   │   ├── mocks.go
│   │   └── noop
│   │       └── noop_testhooks.go
│   ├── tools
│   │   └── rulegen
│   │       └── example_test.go
│   └── util
│       └── schema
│           ├── generator.go
│           ├── schema_test.go
│           └── validator.go
├── main
├── mangle.yaml
├── manglekit.go                                # Root Facade
├── providers                                   # Standard Providers
│   ├── google
│   │   └── plugin.go
│   ├── memory
│   │   ├── inmem
│   │       └── store.go
│   ├── openai
│   │   └── plugin.go
│   └── openrouter
│       └── client.go
└── sdk                                         # Software Development Kit
    ├── action_test.go
    ├── client.go                               # Main Client entry point
    ├── client_execute.go
    ├── config_loader.go
    ├── context.go
    ├── execute_typed.go
    ├── execute_typed_test.go
    ├── executor.go
    ├── generics.go
    ├── handle.go
    ├── helpers.go
    ├── integration_test.go
    ├── loader_ext.go
    ├── loop.go                                 # Execution Loop (Semantic State Machine)
    ├── options.go
    ├── options_ext.go
    ├── planner.go
    ├── policy_generator.go
    ├── policy_generator_test.go
    ├── reflection_test.go
    ├── registry.go
    ├── tracing.go
    └── types.go
```

## 2. COMPONENT ANALYSIS

### 2.1 Engine (`internal/engine`)
**Key Structs:** `PolicyEngine`, `MangleRuntime`.
**Responsibilities:**
- Implements `core.Evaluator`.
- Manages the Datalog runtime (Mangle v0.3.0).
- Orchestrates `Assess` (Pre-Check) and `Reflect` (Post-Check).
- Executes "Steering Policies" (Retry/Route) via `EvaluateSteering`.
- Handles data conversion (Struct/JSON -> Datalog Facts) via `reflection.go` and `flattener.go`.

### 2.2 Supervisor (`internal/supervisor`)
**Key Structs:** `SupervisedAction`.
**Responsibilities:**
- Implements the "Guarded Action" pattern (Decorator).
- Wraps `core.Action` to enforce the governance lifecycle: Trace -> Assess -> Execute -> Reflect.
- Handles Taint Propagation (`SecurityLabels`).
- Enforces Resilience (`FailureMode`).

### 2.3 SDK (`sdk`)
**Key Structs:** `Client`, `Runnable[In, Out]`.
**Responsibilities:**
- **Orchestration:** `Client` wires together the Engine, Supervisor, and Adapters.
- **Execution Loop:** `ExecuteByName` implements the Semantic State Machine (Action -> Feedback -> Retry/Route).
- **Configuration:** Loads `config.Config` and hydrates components.
- **Facade:** Provides user-friendly APIs (`NewClient`, `Supervise`).

### 2.4 Adapters (`adapters/`)
**Key Structs:** `LLMAction`, `MCPAction`, `CircuitBreaker`.
**Responsibilities:**
- **AI:** Wraps Genkit and other LLMs into `core.Action`.
- **MCP:** Connects to Model Context Protocol servers.
- **Resilience:** Provides `CircuitBreaker` and other reliability patterns.
- **Knowledge:** Loads RDF/Graph data.

### 2.5 Core (`core/`)
**Key Structs:** `Envelope`, `Action`, `Decision`, `ActionMetadata`.
**Responsibilities:**
- Defines the **Hexagonal Architecture Ports** (Interfaces).
- Contains the **Ubiquitous Language** (Constants, Types).
- Dependency-free kernel.

### 2.6 CLI (`cmd/mkit`)
**Key Structs:** `EvalCmd`, `GenCmd`, `ServeCmd`, `InspectCmd`.
**Responsibilities:**
- **Code Generation:** `gen rule` automates Datalog policy creation from natural language.
- **Evaluation:** `eval` runs Datalog queries against static data or knowledge bases.
- **Server:** `serve` exposes the SDK via HTTP.
- **Inspection:** `inspect` visualizes how data maps to Datalog facts.

### 2.7 Providers (`providers/`)
**Key Structs:** `openai.Config`, `google.Config`.
**Responsibilities:**
- **OpenAI:** Implements the OpenAI provider plugin, bridging `genkit` to `openai-go`.
- **Google:** Implements the Google GenAI provider plugin.
- **Memory:** Reference implementations for memory stores.

## 3. CRITICAL PATH & DATA

### 3.1 Execution Flow (`Client.ExecuteByName`)
1. **Resolution:** Client looks up `core.Action` by name in its registry.
2. **Context Injection:** Client injects History, RAG Context, and Metadata into a new `Envelope`.
3. **Loop Entry:** `runLoopInternal` starts the Semantic State Machine.
4. **Step Execution (`ExecuteSingleStep`):**
    a. **Supervision:** `SupervisedAction.Execute` is called.
    b. **Tracing:** Span started.
    c. **Assess:** Engine checks `infeasible(Req)` or `deny(Req)`. If blocked, returns `AlignmentError`.
    d. **Action:** Inner Action executes (e.g., calls LLM).
    e. **Reflect:** Engine checks `infeasible(Output)`.
    f. **Steering:** Engine evaluates `retry(Hint)` or `route(Target)`.
5. **Decision Handling:**
    - **RETRY:** Client increments counter, updates context with Feedback, and loops back.
    - **ROUTE:** Client switches to the new Target Action and loops back.
    - **PROCEED:** Loop terminates, result returned.

### 3.2 Core Data Models
- **Envelope:** The universal container for Payload (`any`), Metadata (`map[string]any`), and SecurityLabels (`[]string`).
- **Facts:** Datalog atoms derived from the Envelope Payload.
- **Decision:** The outcome of a policy evaluation (`PROCEED`, `HALT`, `RETRY`, `ROUTE`).

## 4. SOURCE CODE DUMP

### `core/types.go`
```go
package core

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// --- CONSTANTS: The Ubiquitous Language ---

// Standard Metadata Keys used for Control Plane signaling.
const (
	// Governance & Routing
	KeyDecision     = "manglekit.decision"  // Values: "PROCEED", "HALT", "RETRY", "ROUTE"
	KeyFeedback     = "manglekit.feedback"  // Human/LLM readable reason
	KeyPrevFeedback = "prev_feedback"       // Loopback for retry
	KeyNextStep     = "manglekit.next_step" // Next action routing

	// Risk & Analysis
	KeyRiskScore = "manglekit.risk_score" // 0-100

	// Performance & Observability
	KeyLatencyMs = "manglekit.latency_ms"
	KeyTraceID   = "manglekit.trace_id"
	KeyModel     = "manglekit.model"
	KeyHistory   = "manglekit_history"
	KeyContext   = "manglekit.context" // RAG data injected here
	KeySummary   = "manglekit.summary" // Conversation summary

	// Configuration
	PrefixPromptConfig = "prompt."
)

// Standard Decision Values
const (
	DecisionProceed = "PROCEED" // Formerly "ALLOW"
	DecisionHalt    = "HALT"    // Formerly "DENY"
	DecisionRetry   = "RETRY"
	DecisionRoute   = "ROUTE"
)

// Datalog System Constants
const (
	EntityInput   = "Req"           // ID for Input Envelope
	EntityOutput  = "Output"        // ID for Output Envelope
	PredHalt      = "halt"          // Was "deny" or "infeasible"
	PredRetry     = "retry"         // Correction signal
	PredRoute     = "route"         // Dynamic routing signal
	PredViolation = "violation_msg" // To extract error messages
)

// Observability & Trace Attributes
const (
	// Span Names
	SpanPreCheck  = "Datalog.Assess"  // Formerly "Datalog.PreCheck"
	SpanPostCheck = "Datalog.Reflect" // Formerly "Datalog.PostCheck"
	SpanMemory    = "Mangle.Recall"   // RAG lookup

	// Attribute Keys
	AttrPolicyName   = "policy.name"
	AttrPolicyType   = "policy.type"
	AttrDecisionType = "decision.type"
	AttrOutcome      = "outcome"       // "PROCEED", "HALT"
	AttrLabels       = "mangle.labels" // Taint Propagation
	AttrActionName   = "action.name"
	AttrActionType   = "action.type"
	AttrRuleID       = "mangle.rule_id" // Replaces AttrPolicyRuleID
	AttrAttempt      = "mangle.attempt" // Replaces AttrPolicyAttempt
)

// Outcome Values (for Tracing)
const (
	OutcomeProceed = "PROCEED" // Formerly "ALLOWED"
	OutcomeHalt    = "HALT"    // Formerly "DENIED"
	OutcomeSuccess = "success"
)

// --- STRUCTS ---

// ContentType defines the nature of the data payload.
type ContentType string

const (
	// TypeStruct indicates the payload is a strong Go struct.
	// This is the default mode, optimized for internal services.
	TypeStruct ContentType = "STRUCT"

	// TypeJSON indicates the payload is a flexible map[string]any.
	// This is used for AI agents and external webhooks.
	TypeJSON ContentType = "JSON"
)

// Envelope: The unified data container.
type Envelope struct {
	// ID is the unique identifier for this specific data envelope.
	ID uuid.UUID `json:"id"`
	// Payload is the actual data being transported.
	// Note: Field name preserved as Payload for compatibility, tagged as "data".
	Payload any `json:"data"`
	// Metadata stores key-value pairs for control plane signaling.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Error stores any error encountered during processing.
	Error error `json:"error,omitempty"`

	// SecurityLabels holds taint tags (e.g., "secret", "pii") for information flow control.
	SecurityLabels []string `json:"security_labels,omitempty"`
	// Facts holds structured logical facts extracted from the payload.
	Facts []string `json:"facts,omitempty"`
	// ContentType indicates whether the payload is a Struct or JSON.
	ContentType ContentType `json:"content_type,omitempty"`
}

// NewEnvelope creates a new envelope with the provided payload.
func NewEnvelope(payload any) Envelope {
	return Envelope{
		ID:             uuid.New(),
		Payload:        payload,
		Metadata:       make(map[string]any),
		SecurityLabels: []string{},
		ContentType:    TypeStruct, // Default to Typed Mode
	}
}

// SetMeta sets a value in the envelope's metadata map.
func (e *Envelope) SetMeta(k string, v any) {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[k] = v
}

// GetMeta retrieves a value from the envelope's metadata map as a string.
// If the value is not a string, it returns an empty string (or simple string representation).
func (e *Envelope) GetMeta(k string) string {
	if e.Metadata == nil {
		return ""
	}
	v, ok := e.Metadata[k]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// SetFeedback injects the "Teacher's" feedback into metadata
func (e *Envelope) SetFeedback(msg string) {
	e.SetMeta(KeyFeedback, msg)
}

// GetFeedback retrieves the feedback for the "Student" (AI/Logic)
func (e *Envelope) GetFeedback() string {
	return e.GetMeta(KeyFeedback)
}

// AddLabel adds a security label to the envelope if it does not already exist.
func (e *Envelope) AddLabel(label string) {
	if !e.HasLabel(label) {
		e.SecurityLabels = append(e.SecurityLabels, label)
	}
}

// HasLabel checks for the existence of a specific security label on the envelope.
func (e *Envelope) HasLabel(label string) bool {
	for _, l := range e.SecurityLabels {
		if l == label {
			return true
		}
	}
	return false
}

// MergeLabels appends distinct labels from another source to this one.
func (e *Envelope) MergeLabels(other []string) {
	for _, l := range other {
		e.AddLabel(l)
	}
}

// SetHistory serializes a list of chat messages into the envelope's metadata.
func (e *Envelope) SetHistory(msgs []Message) {
	b, err := json.Marshal(msgs)
	if err == nil {
		e.SetMeta(KeyHistory, string(b))
	}
}

// Decision: Structured result from the Policy Engine.
type Decision struct {
	Outcome string            // Matches DecisionProceed, DecisionHalt, etc.
	Target  string            // Used if Outcome == DecisionRoute
	Reasons []string          // Explanations
	Meta    map[string]string // Side-channel data (risk scores, latency budget)
}

// ConfigEvent: For Hot-Swap mechanisms.
type ConfigEvent struct {
	Key     string
	Content []byte
	Type    string
}

// ActionMetadata provides metadata about an action.
type ActionMetadata struct {
	// Name is the unique identifier for the action.
	Name string
	// Type describes the category of the action.
	Type string
	// InputContentType specifies the expected input format.
	InputContentType ContentType
	// InputType is the string name of the Go input type.
	InputType string
	// OutputType is the string name of the Go output type.
	OutputType string
	// IsDynamic indicates if the input type is generic.
	IsDynamic bool
}

// Message represents a single message in a conversation flow.
type Message struct {
	// Role indicates the sender of the message.
	Role string `json:"role"`
	// Content is the textual body of the message.
	Content string `json:"content"`
}

// ConversationHistory represents a sequence of messages in a dialogue.
type ConversationHistory struct {
	// Messages is the ordered list of messages in the conversation.
	Messages []Message `json:"messages"`
}

// Query represents a structured user request.
type Query struct {
	Text string         `json:"text"`
	Meta map[string]any `json:"meta,omitempty"`
}

// GenerationConfig holds standard LLM parameters.
type GenerationConfig struct {
	Temperature   float64
	MaxTokens     int
	TopP          float64
	StopSequences []string
	Model         string
	JSONMode      bool
	// OutputType is used by Genkit to enforce structured output (schema).
	OutputType any
}

// Document represents a snippet of knowledge/memory.
type Document struct {
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Score    float32        `json:"score,omitempty"` // Re-ranking score
}

// Answer represents a structured system response.
type Answer struct {
	Text string         `json:"text"`
	Meta map[string]any `json:"meta,omitempty"`
}
```

### `core/governance.go`
```go
package core

import "context"

// Evaluator: The Mangle Logic Engine.
// It defines the contract for policy execution, validation, and steering.
type Evaluator interface {
	// AssessPlan evaluates the policy for a given input (General purpose).
	// It returns a Decision struct with the outcome.
	// Formerly: Assess
	AssessPlan(ctx context.Context, input Envelope) (Decision, error)

	// Assess performs the Pre-Check phase (input validation).
	// Formerly: Authorize
	Assess(ctx context.Context, actionMeta ActionMetadata, input Envelope) error

	// Reflect evaluates the outcome of an action (Post-Check).
	// Formerly: Validate
	Reflect(ctx context.Context, actionMeta ActionMetadata, output Envelope) (Envelope, error)

	// EvaluateSteering determines the next step (Retry/Route) based on the output.
	EvaluateSteering(ctx context.Context, input Envelope) (string, map[string]string, error)

	// GetActionConfig queries the engine for dynamic configuration parameters.
	GetActionConfig(ctx context.Context, input Envelope) (map[string]string, error)

	// CheckRequirement checks if a capability is needed. e.g., requires(Req, "memory").
	CheckRequirement(ctx context.Context, input Envelope, reqName string) (bool, error)

	// LoadPolicy loads policy rules from a source string or file content.
	LoadPolicy(ctx context.Context, source string) error

	// LoadFacts injects dynamic facts into the engine.
	LoadFacts(facts []string) error

	// RegisterAction registers action metadata for discovery/planning.
	RegisterAction(meta ActionMetadata) error

	// Logger returns the engine's logger.
	Logger() Logger
}

// PreProcessor: Fast/Stateless checks (CEL/Expr).
type PreProcessor interface {
	Process(ctx context.Context, input Envelope) (map[string]any, error)
}

// RiskEngine: specialized interface for calculating risk.
type RiskEngine interface {
	CalculateRisk(ctx context.Context, input Envelope) (float64, error)
}

// ResourceMonitor: Cost & Rate Limiting.
type ResourceMonitor interface {
	CountTokens(ctx context.Context, text string) (int, error)
	CheckBudget(ctx context.Context, key string, cost int) (bool, error)
}
```

### `core/logic.go`
```go
package core

import "context"

// Action defines the interface for a unit of work in the Manglekit system.
type Action interface {
	// Execute performs the action's logic.
	Execute(ctx context.Context, input Envelope) (Envelope, error)

	// Metadata returns the action's metadata.
	Metadata() ActionMetadata
}

// GenerateOption is a functional option for text generation.
type GenerateOption func(o *GenerationConfig)

// LLMResponse contains the generated text and token usage metadata.
type LLMResponse struct {
	Text  string
	Usage map[string]int
}

// TextGenerator abstracts the LLM.
type TextGenerator interface {
	// Complete generates text from a prompt.
	Complete(ctx context.Context, prompt string) (string, error)

	// Generate generates text with options.
	Generate(ctx context.Context, prompt string, opts ...GenerateOption) (*LLMResponse, error)

	// Stream generates a stream of text.
	Stream(ctx context.Context, prompt string) (<-chan string, error)
}

// Extractor converts raw text into structured data.
type Extractor interface {
	Extract(ctx context.Context, input string, schema any) error
}
```

### `sdk/client.go`
```go
package sdk

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/duynguyendang/manglekit/adapters/knowledge"
	mcpAdapter "github.com/duynguyendang/manglekit/adapters/mcp"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/supervisor"
	"github.com/duynguyendang/manglekit/internal/telemetry"
)

const (
	// TracerName is the instrumentation scope name for Manglekit tracing.
	TracerName = "github.com/duynguyendang/manglekit/sdk"
)

// Client is the primary entry point for the Manglekit system.
// It acts as the governance kernel, managing blueprints, observability, and action execution.
// Applications should create a single Client instance and reuse it.
type Client struct {
	// engine is the internal Policy Engine responsible for Datalog evaluation.
	engine core.Evaluator
	// tracer is the Manglekit core.Tracer wrapper.
	tracer core.Tracer
	// otelTracer is the raw OpenTelemetry tracer instance.
	otelTracer trace.Tracer
	// logger is the structured logger used by the client and its components.
	logger core.Logger
	// memory is the persistence layer for chat history (optional).
	memory core.MemoryStore
	// agentMemory is the semantic memory (RAG) provider (optional).
	agentMemory core.AgentMemory
	// registry holds registered actions for dynamic routing.
	registry map[string]core.Action
	// failureMode determines the system's resilience strategy ("open" or "closed").
	failureMode string
	// blueprintPath stores the path loaded at startup (for debugging/reloading).
	blueprintPath string
	// shutdownFunc is a cleanup function to stop exporters/tracers.
	shutdownFunc func(context.Context) error
	// llm is the plugged-in text generation backend.
	llm core.TextGenerator
}

// NewClient initializes a new Manglekit Client with the provided options.
// It sets up the Policy Engine, Observability (Logging/Tracing), and default configurations.
//
// Parameters:
//   - ctx: The initialization context.
//   - opts: A variadic list of ClientOption configuration functions.
//
// Returns:
//   - A pointer to the initialized Client, or an error if initialization fails.
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	c := &Client{
		logger:   logger.NewDefault(),
		registry: make(map[string]core.Action),
		memory:   core.NopStore{},
	}

	for _, opt := range opts {
		opt(c)
	}

	// SAFETY FIX: Ensure tracer is never nil to prevent nil pointer dereferences
	if c.tracer == nil {
		c.otelTracer = trace.NewNoopTracerProvider().Tracer(TracerName)
		c.tracer = telemetry.NewOTelTracer(c.otelTracer)
	}

	// Initialize Engine with observability
	eng, err := engine.NewWithObservability(c.tracer, c.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize policy engine: %w", err)
	}
	c.engine = eng

	// Load blueprint from file if provided
	if c.blueprintPath != "" {
		content, err := os.ReadFile(c.blueprintPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read blueprint file: %w", err)
		}
		if err := c.engine.LoadPolicy(ctx, string(content)); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// NewClientWithConfig initializes a Client using a loaded Config object.
// It performs full wiring: Engine, Policy, Knowledge, and Actions.
//
// Parameters:
//   - ctx: The context.
//   - cfg: The loaded configuration struct.
//   - opts: Additional functional options (override config settings).
//
// Returns:
//   - A pointer to the Client, or an error.
func NewClientWithConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error) {
	// Initialize logger
	var log core.Logger
	if cfg != nil && cfg.Observability.LogLevel != "" {
		log = logger.New(cfg.Observability.LogLevel)
	} else {
		log = logger.NewDefault()
	}

	// Create client with loaded configuration
	c := &Client{
		logger:   log,
		registry: make(map[string]core.Action),
		memory:   core.NopStore{},
	}

	for _, opt := range opts {
		opt(c)
	}

	// SAFETY FIX: Ensure tracer is never nil to prevent nil pointer dereferences
	if c.tracer == nil {
		c.otelTracer = trace.NewNoopTracerProvider().Tracer(TracerName)
		c.tracer = telemetry.NewOTelTracer(c.otelTracer)
	}

	// Initialize Engine with observability
	eng, err := engine.NewWithObservability(c.tracer, c.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize policy engine: %w", err)
	}
	c.engine = eng

	if cfg == nil {
		return c, nil
	}

	// 1. Load Policy
	if cfg.Policy.Path != "" {
		content, err := os.ReadFile(cfg.Policy.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read policy file %q: %w", cfg.Policy.Path, err)
		}
		if err := c.engine.LoadPolicy(ctx, string(content)); err != nil {
			return nil, fmt.Errorf("failed to load policy from %q: %w", cfg.Policy.Path, err)
		}
	}

	// 2. Load Knowledge
	if cfg.Knowledge.Path != "" {
		path := cfg.Knowledge.Path
		var facts []string
		var err error

		if strings.HasSuffix(path, ".nt") {
			f, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("failed to open knowledge file %q: %w", path, err)
			}
			loader := knowledge.NewNTriplesLoader()
			facts, err = loader.Parse(f)
			f.Close()
		} else {
			// Default to RDF Loader (Turtle/XML)
			loader := knowledge.NewRDFLoader()
			facts, err = loader.Parse(path)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to parse knowledge from %q: %w", path, err)
		}

		if err := c.engine.LoadFacts(facts); err != nil {
			return nil, fmt.Errorf("failed to load knowledge facts from %q: %w", path, err)
		}
	}

	// 3. Hydrate Memory
	if cfg.Memory.Provider != "" {
		mem, err := createMemory(ctx, *cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create memory: %w", err)
		}
		c.agentMemory = mem
	}

	// 4. Hydrate Actions
	if len(cfg.Actions) > 0 {
		actions, err := HydrateActions(ctx, cfg.Actions)
		if err != nil {
			return nil, err
		}
		for _, action := range actions {
			// Register it
			c.RegisterAction(action.Metadata().Name, action)
		}
	}

	// 5. Load MCP Actions
	if len(cfg.MCP) > 0 {
		for _, mcpCfg := range cfg.MCP {
			loader := mcpAdapter.NewLoader(mcpCfg).WithLogger(c.logger)
			actions, err := loader.Load(ctx)
			if err != nil {
				return nil, fmt.Errorf("critical tool '%s' failed to load: %w", mcpCfg.Name, err)
			}

			for _, action := range actions {
				safeAction := c.Supervise(action)
				c.RegisterAction(safeAction.Metadata().Name, safeAction)
				c.logger.Info("Discovered MCP Tool", "name", safeAction.Metadata().Name)
			}
		}
	}

	// Set failure mode
	if cfg.FailureMode != "" {
		c.failureMode = cfg.FailureMode
	}

	// Log configuration loaded successfully
	c.logger.Info("Manglekit client initialized with config",
		"service_name", cfg.Observability.ServiceName,
		"observability_enabled", cfg.Observability.Enabled,
		"failure_mode", c.failureMode)

	return c, nil
}

// NewClientFromFile initializes a Client by loading configuration from a YAML file.
// It supports environment variable expansion in the config file.
//
// Parameters:
//   - ctx: The context.
//   - configPath: Path to the YAML configuration file.
//   - opts: Additional functional options.
//
// Returns:
//   - A pointer to the Client, or an error.
func NewClientFromFile(ctx context.Context, configPath string, opts ...ClientOption) (*Client, error) {
	// Load configuration from file
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return NewClientWithConfig(ctx, cfg, opts...)
}

// NewClientFromConfig is a wrapper for NewClientWithConfig to match user requirements.
func NewClientFromConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error) {
	return NewClientWithConfig(ctx, cfg, opts...)
}

// Supervise wraps a raw core.Action in a SupervisedAction.
// This applies the "Trace -> Assess -> Execute -> Reflect" governance lifecycle.
// This is the core function of the Manglekit framework.
//
// Parameters:
//   - action: The action to supervise.
//
// Returns:
//   - A new core.Action that enforces blueprints.
func (c *Client) Supervise(action core.Action) core.Action {
	if c.tracer != nil {
		return supervisor.NewSupervisedActionWithTracer(action, c.engine, c.tracer, c.failureMode)
	}
	return supervisor.NewSupervisedAction(action, c.engine, c.failureMode)
}

func (c *Client) Engine() core.Evaluator {
	return c.engine
}

// LoadFacts allows manually injecting straight Datalog facts into the engine.
// This supports the "Explicit Loading" workflow where adapters parse data first.
func (c *Client) LoadFacts(facts []string) error {
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	return c.engine.LoadFacts(facts)
}

// Tracer returns the OpenTelemetry Tracer used by the client.
// This allows users to start their own spans that are linked to the Manglekit trace context.
func (c *Client) Tracer() trace.Tracer {
	return c.otelTracer
}

// Logger returns the configured Logger instance.
func (c *Client) Logger() core.Logger {
	return c.logger
}

// NewDefault initializes a Client with sensible default settings:
//   - Default internal logger (slog).
//   - No-op tracer.
//   - No policy loaded (allow-all default).
//
// Returns:
//   - A pointer to the Client, or an error.
func NewDefault() (*Client, error) {
	return NewClient(context.Background())
}

// RegisterAction adds an action to the client's internal registry.
// Registered actions can be invoked by name using ExecuteByName, enabling dynamic routing.
//
// Parameters:
//   - name: The unique name for the action.
//   - action: The action instance.
func (c *Client) RegisterAction(name string, action core.Action) {
	c.registry[name] = action
	if c.engine != nil {
		if err := c.engine.RegisterAction(action.Metadata()); err != nil {
			c.logger.Warn("failed to register action metadata to engine", "action", name, "error", err)
		}
	}
}

// Shutdown cleans up resources used by the client, such as flushing traces.
func (c *Client) Shutdown(ctx context.Context) error {
	if c.shutdownFunc != nil {
		return c.shutdownFunc(ctx)
	}
	return nil
}

// Memory returns the active memory provider (if any).
// This is useful for manual data seeding or debugging.
func (c *Client) Memory() core.AgentMemory {
	return c.agentMemory
}
```

### `sdk/loop.go`
```go
package sdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/core"
	engine_memory "github.com/duynguyendang/manglekit/internal/engine/memory"
)

// Define constants or config struct
const (
	DefaultMaxSteps   = 10
	DefaultMaxRetries = 3
	BackoffBase       = 100 * time.Millisecond
)

// ExecuteByName executes a registered action by its name, handling the Semantic State Machine loop.
func (c *Client) ExecuteByName(ctx context.Context, actionName string, input any, opts ...ExecuteOption) (core.Envelope, error) {
	params := ExecutionParams{
		MemoryMode: core.MemoryModeNone,
	}
	for _, opt := range opts {
		opt(&params)
	}
	return c.runLoopInternal(ctx, actionName, input, params)
}

func (c *Client) runLoopInternal(ctx context.Context, startAction string, payload any, params ExecutionParams) (core.Envelope, error) {
	if c.logger != nil {
		ctx = core.ContextWithLogger(ctx, c.logger)
	}

	// 1. Determine Store Strategy
	switch params.MemoryMode {
	case core.MemoryModePersist:
		params.Store = c.memory
		if params.SessionID != "" {
			var err error
			// Load initial history
			params.CurrentHistory, err = params.Store.Read(ctx, params.SessionID)
			if err != nil && c.logger != nil {
				c.logger.Warn("RunLoop failed to hydrate history", "error", err)
			}
		}
	case core.MemoryModeTransient:
		params.Store = &engine_memory.VolatileStore{}
	default:
		params.Store = &core.NopStore{}
	}

	currentAction := startAction
	currentPayload := payload

	for step := 0; step < DefaultMaxSteps; step++ {
		// Check context before starting new step
		if err := ctx.Err(); err != nil {
			return core.Envelope{}, err
		}

		if c.logger != nil {
			c.logger.Info("RunLoop step", "step", step, "action", currentAction)
		}

		result, err := c.ExecuteSingleStep(ctx, currentAction, currentPayload, &params)
		if err != nil {
			return core.Envelope{}, err
		}

		decision := result.Metadata[core.KeyDecision]
		if decision == core.DecisionRoute {
			// Update flow for next loop
			next, ok := result.Metadata[core.KeyNextStep].(string)
			if !ok || next == "" {
				return core.Envelope{}, fmt.Errorf("route decision missing next_step")
			}

			// Validate/Log Payload Handover
			if c.logger != nil {
				c.logger.Info("RunLoop: Routing to next action", "from", currentAction, "to", next, "payload_type", fmt.Sprintf("%T", result.Payload))
			}

			currentAction = next
			currentPayload = result.Payload
			continue
		}

		if decision == core.DecisionRetry {
			// Params internal state (RetryCount, Feedback) already updated by ExecuteSingleStep
			// Validation (MaxRetries) and Backoff (Sleep) also handled by ExecuteSingleStep
			// Just loop again with same action/payload
			continue
		}

		// ALLOW or empty -> Done
		return result, nil
	}
	return core.Envelope{}, fmt.Errorf("max steps exceeded")
}

// ExecuteSingleStep runs one step of the action and returns the decision.
// It handles: Action Execution, History Persistence, Blueprint Alignment Backoff, and Steering Logic (Retry/Route updates).
func (c *Client) ExecuteSingleStep(ctx context.Context, actionName string, payload any, params *ExecutionParams) (core.Envelope, error) {
	// 1. Resolve Action
	action, ok := c.registry[actionName]
	if !ok {
		return core.Envelope{}, fmt.Errorf("action not found: %s", actionName)
	}

	env := core.NewEnvelope(payload)
	env.ContentType = action.Metadata().InputContentType

	// --- Phase 1: Context Injection ---
	// 1.1 Inject Feedback (History & Last Hint)
	if len(params.FeedbackHistory) > 0 {
		env.Metadata[core.KeyPrevFeedback] = strings.Join(params.FeedbackHistory, "; ")
	}
	if params.LastFeedback != "" {
		env.SetFeedback(params.LastFeedback)
		env.Metadata["mangle_feedback"] = params.LastFeedback
	}

	// 1.2 Inject Chat History
	if len(params.CurrentHistory) > 0 && params.MemoryMode != core.MemoryModeNone {
		env.SetHistory(params.CurrentHistory)
	}

	// 1.3 Inject Semantic Memory (RAG)
	c.recallContext(ctx, payload, &env)

	// 1.3 Inject Explicit Metadata
	if params.Metadata != nil {
		for k, v := range params.Metadata {
			env.Metadata[k] = v
		}
	}

	// 1.4 Inject Context Facts (e.g. User Role, Budget from sdk.WithFact)
	// This ensures facts propagate to the Engine for Blueprint Checks.
	if facts := ContextFacts(ctx); facts != nil {
		for k, v := range facts {
			env.Metadata[k] = v
		}
	}

	// --- Phase 2: Blueprint Check (Pre-check) & Phase 3: Execution (Intuition) ---
	// The Action.Execute wrapper handles the Pre-check (Blueprint) and the actual Genkit Execution.
	result, err := action.Execute(ctx, env)
	if err != nil {
		var pve *core.AlignmentError
		if errors.As(err, &pve) {
			if params.RetryCount >= DefaultMaxRetries {
				return core.Envelope{}, fmt.Errorf("max retries exceeded: %w", err)
			}
			params.RetryCount++
			params.LastFeedback = pve.Message

			if c.logger != nil {
				c.logger.Warn("RunLoop: Blueprint Alignment Issue", "feedback", params.LastFeedback, "attempt", params.RetryCount)
			}

			// Context-aware Backoff
			if err := c.backoff(ctx, params.RetryCount); err != nil {
				return core.Envelope{}, err
			}

			// Return a mock result to signal RETRY to caller
			res := core.NewEnvelope(payload)
			res.Metadata[core.KeyDecision] = core.DecisionRetry
			return res, nil
		}
		return core.Envelope{}, err
	}

	params.LastFeedback = ""

	// 4. Update History
	if params.MemoryMode != core.MemoryModeNone {
		userContent := safelyStringify(payload)
		assistContent := safelyStringify(result.Payload)

		newExchange := []core.Message{
			{Role: "user", Content: userContent},
			{Role: "assistant", Content: assistContent},
		}
		params.CurrentHistory = append(params.CurrentHistory, newExchange...)
	}

	// 5. Persist
	if params.Store != nil && params.SessionID != "" && params.MemoryMode == core.MemoryModePersist {
		if err := params.Store.Write(ctx, params.SessionID, params.CurrentHistory); err != nil && c.logger != nil {
			c.logger.Warn("RunLoop failed to persist history", "error", err)
		}
	}

	// --- Phase 4: Evaluation & Correction (Post-check) ---
	// The Engine evaluates the result against the Blueprint and decides next steps (Retry/Route/Allow).
	decision := result.Metadata[core.KeyDecision]

	// --- HOOK: MEMORIZE ---
	// Only learn if no error and Decision is PROCEED (or empty/legacy)
	if err == nil && (decision == "" || decision == core.DecisionProceed) {
		c.asyncMemorize(payload, result.Payload)
	}

	if c.logger != nil {
		c.logger.Debug("RunLoop decision", "decision", decision, "action", actionName)
	}

	switch decision {
	case core.DecisionRetry:
		if params.RetryCount >= DefaultMaxRetries {
			return core.Envelope{}, fmt.Errorf("max retries exceeded for action %s", actionName)
		}
		params.RetryCount++
		hint := result.GetFeedback()
		params.LastFeedback = hint
		params.FeedbackHistory = append(params.FeedbackHistory, hint)

		if c.logger != nil {
			c.logger.Warn("RunLoop: RETRY triggered", "feedback", hint)
		}

		// Context-aware Backoff
		if err := c.backoff(ctx, params.RetryCount); err != nil {
			return core.Envelope{}, err
		}
		// Return result so caller loops
		return result, nil

	case core.DecisionRoute:
		// Reset retry count for new action
		params.RetryCount = 0
		params.FeedbackHistory = nil

		if c.logger != nil {
			c.logger.Info("RunLoop: Feedback history cleared for new action route")
		}
		// Return result so caller loops
		return result, nil

	case core.DecisionProceed, "":
		return result, nil

	case core.DecisionHalt:
		reason := result.Metadata["reason"]
		if reason == "" {
			reason = result.Metadata["violation_msg"]
		}
		if reason == "" {
			reason = "blueprint violation"
		}
		return core.Envelope{}, fmt.Errorf("action halted by blueprint: %s", reason)
	}

	// Should not reach here for standard decisions
	return result, nil
}

// backoff handles the sleep and context cancellation check
func (c *Client) backoff(ctx context.Context, retryCount int) error {
	sleepDuration := time.Duration(retryCount) * BackoffBase
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(sleepDuration):
		return nil
	}
}

// ---------------------------------------------------------
// PRIVATE HELPERS (Memory & Context)
// ---------------------------------------------------------

// safelyStringify converts any payload to string for embedding.
func safelyStringify(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	// Fallback to generic formatting
	return fmt.Sprintf("%v", v)
}

// recallContext handles the RAG lookup logic.
// It fails silently (logs only) to not break the main flow.
func (c *Client) recallContext(ctx context.Context, payload any, env *core.Envelope) {
	if c.agentMemory == nil {
		return
	}

	// [GATE] Check if memory is required
	if c.engine != nil {
		// Ask: requires(Req, "memory") ?
		needed, err := c.engine.CheckRequirement(ctx, *env, "memory")
		if err != nil {
			if c.logger != nil {
				c.logger.Warn("Engine check failed, skipping memory", "err", err)
			}
			return
		}
		if !needed {
			return // Skip RAG if not required
		}
	}

	// Start Span for Observability
	var span core.Span
	if c.tracer != nil {
		ctx, span = c.tracer.Start(ctx, core.SpanMemory)
		defer span.End()
	}

	inputStr := safelyStringify(payload)

	// Call Memory Provider
	contextData, err := c.agentMemory.Recall(ctx, inputStr)
	if err != nil {
		if c.logger != nil {
			c.logger.Warn("Memory Recall failed", "error", err)
		}
		if span != nil {
			span.RecordError(err)
		}
		return
	}

	// Inject if found
	if contextData != "" {
		env.SetMeta(core.KeyContext, contextData)
		if c.logger != nil {
			c.logger.Debug("Injected memory context", "len", len(contextData))
		}
	}
}

// asyncMemorize handles the Fire-and-Forget storage logic.
func (c *Client) asyncMemorize(input any, output any) {
	if c.agentMemory == nil {
		return
	}

	inputStr := safelyStringify(input)
	outputStr := safelyStringify(output)

	// Launch Goroutine to not block response latency
	go func(q, a string) {
		// Create a detached context with timeout to prevent leaks
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := c.agentMemory.Memorize(ctx, q, a); err != nil {
			if c.logger != nil {
				c.logger.Warn("Memory Memorize failed", "error", err)
			}
		}
	}(inputStr, outputStr)
}
```

### `internal/engine/solver.go`
```go
package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine/resources"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/parse"
)

var ErrSolutionFound = errors.New("solution found")

// PolicyEngine is the core decision-making component of Manglekit.
// It orchestrates the loading of policies, maintaining the Datalog runtime,
// and executing authorization (Pre-Check) and validation (Post-Check) logic.
// It also integrates with observability (Tracing/Logging) to provide transparent governance.
type PolicyEngine struct {
	tracer  core.Tracer
	logger  core.Logger
	runtime *MangleRuntime
}

// New creates a new PolicyEngine with default no-op observability.
// This is suitable for basic usage where tracing and structured logging are not required.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func New() (*PolicyEngine, error) {
	pe := &PolicyEngine{
		tracer:  &core.NopTracer{},
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}

	// Auto-load Standard Library
	if err := pe.runtime.AddPolicy(resources.StdLib()); err != nil {
		return nil, fmt.Errorf("manglekit: failed to load std.dl: %w", err)
	}

	return pe, nil
}

// NewWithTracer creates a new PolicyEngine with tracing enabled.
//
// Deprecated: Use NewWithObservability instead.
//
// Parameters:
//   - tracer: The tracer implementation to use.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func NewWithTracer(tracer core.Tracer) *PolicyEngine {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	return &PolicyEngine{
		tracer:  tracer,
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}
}

// NewWithObservability creates a new PolicyEngine with both tracing and logging enabled.
// This is the recommended constructor for production use.
//
// Parameters:
//   - tracer: The tracer implementation (e.g., OpenTelemetry).
//   - logger: The logger implementation.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func NewWithObservability(tracer core.Tracer, logger core.Logger) (*PolicyEngine, error) {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	if logger == nil {
		logger = core.NopLogger{}
	}

	pe := &PolicyEngine{
		tracer:  tracer,
		logger:  logger,
		runtime: NewMangleRuntime(),
	}

	// Load Planner Core Rules
	if err := pe.runtime.AddPolicy(resources.GetPlannerRules()); err != nil {
		if logger != nil {
			logger.Error("failed to load planner core schema", "error", err)
		}
	}

	// Load Manglekit Standard Library (std.dl)
	if err := pe.runtime.AddPolicy(resources.StdLib()); err != nil {
		if logger != nil {
			logger.Error("failed to load standard library", "error", err)
		}
		// Failure to load stdlib is critical for dynamic features
		return nil, fmt.Errorf("manglekit: failed to load std.dl: %w", err)
	}

	return pe, nil
}

// RecordLineage records a data lineage relationship between a child and a parent.
// Note: In the current architecture, lineage is primarily handled via context propagation
// and tracing spans rather than explicit in-memory storage.
//
// Parameters:
//   - ctx: The context.
//   - childID: The ID of the derived data.
//   - parentID: The ID of the source data.
func (e *PolicyEngine) RecordLineage(ctx context.Context, childID, parentID string) {
	if e.tracer != nil {
		// Lineage linking is handled via context propagation in GuardedAction.
		// If explicit linking span events are needed, they can be added here.
	}
}

// Logger returns the engine's configured Logger instance.
// This allows other components (like GuardedAction) to reuse the engine's logger.
//
// Returns:
//   - The configured Logger, or a NopLogger if none was set.
func (e *PolicyEngine) Logger() core.Logger {
	if e.logger == nil {
		return core.NopLogger{}
	}
	return e.logger
}

// LoadFacts injects a list of raw Datalog fact strings into the runtime's base knowledge.
// This allows adding dynamic context or configuration at runtime (e.g., feature flags).
//
// Parameters:
//   - facts: A slice of Datalog fact strings.
//
// Returns:
//   - An error if parsing or evaluation fails.
func (e *PolicyEngine) LoadFacts(facts []string) error {
	return e.runtime.LoadFacts(facts)
}

// RegisterAction injects metadata about a registered action into the Datalog runtime.
// It generates facts that describe the action's interface, enabling the Planner to reason about it.
//
// Generated Facts:
//   - action("name")
//   - has_input("name", "InputType")
//   - has_output("name", "OutputType")
//
// Parameters:
//   - meta: The metadata of the action.
//
// Returns:
//   - An error if fact loading fails.
func (e *PolicyEngine) RegisterAction(meta core.ActionMetadata) error {
	var facts []string
	safeName := escapeString(meta.Name)

	facts = append(facts, fmt.Sprintf("action(\"%s\")", safeName))

	if meta.InputType != "" {
		facts = append(facts, fmt.Sprintf("has_input(\"%s\", \"%s\")", safeName, escapeString(meta.InputType)))
	}

	if meta.OutputType != "" {
		facts = append(facts, fmt.Sprintf("has_output(\"%s\", \"%s\")", safeName, escapeString(meta.OutputType)))
	}

	return e.LoadFacts(facts)
}

// LoadPolicy loads policy rules from a raw Datalog string.
// This decouples the engine from file I/O.
//
// Parameters:
//   - ctx: The execution context (unused in current implementation but required by interface).
//   - policy: The Datalog rules as a string.
//
// Returns:
//   - An error if parsing or loading fails.
func (e *PolicyEngine) LoadPolicy(ctx context.Context, policy string) error {
	if policy == "" {
		return nil
	}

	// Load the rules into the Mangle runtime
	if err := e.runtime.AddPolicy(policy); err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Log successful load
	if e.logger != nil {
		e.logger.Debug("policy loaded from string", "length", len(policy))
	}

	return nil
}

// AssessPlan implements the core.Evaluator interface.
// It performs a high-level assessment of the input, mapping Assess logic to a Decision.
// Formerly: Assess
func (e *PolicyEngine) AssessPlan(ctx context.Context, input core.Envelope) (core.Decision, error) {
	// Simple mapping: use empty metadata for generic assessment
	err := e.Assess(ctx, core.ActionMetadata{}, input)
	if err != nil {
		// If authorization fails, it's a DENY
		var alignErr *core.AlignmentError
		if errors.As(err, &alignErr) {
			return core.Decision{
				Outcome: core.DecisionHalt,
				Reasons: []string{alignErr.Message},
				Meta:    map[string]string{"rule_id": alignErr.RuleID},
			}, nil
		}
		return core.Decision{Outcome: core.DecisionHalt, Reasons: []string{err.Error()}}, err
	}
	return core.Decision{Outcome: core.DecisionProceed}, nil
}

// GetActionConfig queries the engine for dynamic configuration parameters.
// It executes the query `action_config(Key, Value)` and returns a map of results.
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input envelope.
//
// Returns:
//   - A map of configuration keys and values.
//   - An error if execution fails.
func (e *PolicyEngine) GetActionConfig(ctx context.Context, input core.Envelope) (map[string]string, error) {
	config := make(map[string]string)

	if e.runtime == nil || e.runtime.programInfo == nil {
		return config, nil
	}

	// Convert input to facts
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts for config", "error", err)
		}
		// Return empty config on fact conversion failure to avoid blocking
		return config, nil
	}

	// Inject Envelope Facts
	for _, factStr := range input.Facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse envelop fact", "fact", factStr, "error", err)
			}
			// Continue without this fact
			continue
		}
		facts = append(facts, atom)
	}

	// Inject Metadata facts: meta(Key, Value) and attempt(N)
	for k, v := range input.Metadata {
		// Ensure k is string
		safeK := escapeString(k)
		// Ensure v is string or convertible to string
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)

		// meta("key", "val")
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}

		// attempt(N) from retry_count
		if k == "retry_count" {
			attemptFact := fmt.Sprintf("attempt(%s)", vStr)
			if atom, err := parse.Atom(attemptFact); err == nil {
				facts = append(facts, atom)
			}
		}
	}

	// Execute query: config(Key, Value)
	err = e.runtime.QueryWithSolutions(facts, "config(Key, Value)", func(solution map[string]any) error {
		key, kOk := solution["Key"].(string)
		val, vOk := solution["Value"].(string)
		if kOk && vOk {
			config[key] = val
		}
		return nil
	})

	if err != nil && e.logger != nil {
		e.logger.Debug("failed to query action config", "error", err)
	}

	return config, nil
}

// Assess performs the Pre-Check phase of governance.
// It checks if the input is allowed to proceed based on the loaded policies.
// If the `infeasible(Req, Reason)` or `deny(Req)` predicate is derived, it returns `core.ErrAlignment`.
//
// It automatically starts a tracing span (`Datalog.Assess`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being authorized.
//   - input: The input envelope containing the payload and security labels.
//
// Returns:
//   - core.ErrAlignment if blocked, or nil if allowed.
// Formerly: Authorize
func (e *PolicyEngine) Assess(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.tracer == nil {
		return e.assessInternal(ctx, actionMeta, input)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPreCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(input.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: input.SecurityLabels})
	}

	err := e.assessInternal(ctx, actionMeta, input)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})
	}
	return err
}

// assessInternal executes the core authorization logic.
func (e *PolicyEngine) assessInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	var extraFacts []ast.Atom

	// Inject Action Metadata facts: action_operation("Req", "Name")
	if actionMeta.Name != "" {
		safeName := core.EntityInput
		safeOp := actionMeta.Name
		opFactStr := fmt.Sprintf("action_operation(\"%s\", \"%s\")", escapeString(safeName), escapeString(safeOp))
		opAtom, err := parse.Atom(opFactStr)
		if err == nil {
			extraFacts = append(extraFacts, opAtom)
		}
	}

	// Use infeasible(Req, Reason) with fallback to deny(Req)
	return e.evaluateGate(ctx, actionMeta.Name, core.EntityInput, input, extraFacts...)
}

// Reflect performs the Post-Check phase of governance.
// It checks if the output is allowed to be returned to the caller.
// If the `infeasible(Output, Reason)` predicate is derived, it returns `core.ErrAlignment`.
//
// It automatically starts a tracing span (`Datalog.Reflect`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being validated.
//   - output: The output envelope containing the result.
//
// Returns:
//   - The validated envelope (potentially modified, though currently pass-through), or an error.
// Formerly: Validate
func (e *PolicyEngine) Reflect(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.tracer == nil {
		return e.reflectInternal(ctx, actionMeta, output)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPostCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(output.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: output.SecurityLabels})
	}

	result, err := e.reflectInternal(ctx, actionMeta, output)
	if err != nil {
		span.RecordError(err)
		return core.Envelope{}, err
	}
	span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})
	return result, nil
}

// reflectInternal executes the core validation logic.
func (e *PolicyEngine) reflectInternal(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	err := e.evaluateGate(ctx, actionMeta.Name, core.EntityOutput, output)
	if err != nil {
		return core.Envelope{}, err
	}
	return output, nil
}

// evaluateGate centralizes the logic for "Check -> Deny -> Explain".
// It is used by both Assess (Pre-Check) and Reflect (Post-Check).
// Updated to check `infeasible(Entity, Reason)` first, then `deny(Entity)`.
func (e *PolicyEngine) evaluateGate(ctx context.Context, actionName string, entityID string, env core.Envelope, extraFacts ...ast.Atom) error {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return nil // No runtime or program loaded, allow by default
	}

	// 1. ToFacts: Convert Payload
	facts, err := toMangleFacts(entityID, env.Payload, env.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert payload to facts", "error", err)
		}
		// Return actual error to allow Fail-Open handling if configured upstream
		return fmt.Errorf("fact conversion error: %w", err)
	}

	// 2. Inject Extra Facts (e.g. Action Operation)
	facts = append(facts, extraFacts...)

	// 3. Inject Labels
	labelFacts, err := LabelsToFacts(entityID, env.SecurityLabels)
	if err != nil {
		if e.logger != nil {
			e.logger.Error("failed to generate label facts", "error", err)
		}
		return fmt.Errorf("label conversion error: %w", err)
	}
	for _, f := range labelFacts {
		atom, err := parse.Atom(f)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse label fact", "fact", f, "error", err)
			}
			return fmt.Errorf("label parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// 4. Inject Explicit Facts from Envelope
	for _, f := range env.Facts {
		atom, err := parse.Atom(f)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse envelop fact", "fact", f, "error", err)
			}
			return fmt.Errorf("envelope fact parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// 5. Inject Metadata
	for k, v := range env.Metadata {
		safeK := escapeString(k)
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)
		// meta("key", "val")
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}
		// attempt(N) from retry_count
		if k == "retry_count" {
			attemptFact := fmt.Sprintf("attempt(%s)", vStr)
			if atom, err := parse.Atom(attemptFact); err == nil {
				facts = append(facts, atom)
			}
		}
	}

	// 6. Run Query
	// Priority 1: halt(Entity, Reason)
	var violationMsg, ruleID string
	var blocked bool

	// Query: halt(Entity, Reason)
	queryHalt := fmt.Sprintf("%s(\"%s\", Reason)", core.PredHalt, entityID)
	err = e.runtime.QueryWithSolutions(facts, queryHalt, func(solution map[string]any) error {
		if reason, ok := solution["Reason"].(string); ok {
			violationMsg = reason
			blocked = true
			return ErrSolutionFound // Stop searching
		}
		return nil
	})

	// Check if search was stopped due to finding a solution
	if errors.Is(err, ErrSolutionFound) {
		err = nil // Clear sentinel error
	}
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to execute halt query", "error", err)
		}
		return fmt.Errorf("halt query error: %w", err)
	}

	if blocked {
		// Try to find rule ID if available (optional)
		// Query: violation_rule(ID)
		qErr := e.runtime.QueryWithSolutions(facts, "violation_rule(ID)", func(solution map[string]any) error {
			if id, ok := solution["ID"].(string); ok {
				ruleID = id
				return ErrSolutionFound
			}
			return nil
		})
		if errors.Is(qErr, ErrSolutionFound) {
			qErr = nil
		}
		// Log but do not block on metadata query failure
		if qErr != nil && e.logger != nil {
			e.logger.Warn("failed to query violation rule ID", "error", qErr)
		}

		if e.logger != nil {
			e.logger.Debug("gate violation detected (halt)", "action", actionName, "msg", violationMsg, "rule_id", ruleID)
		}
		return &core.AlignmentError{Message: violationMsg, RuleID: ruleID}
	}

	// Priority 2: deny(Entity) (Backward Compatibility)
	// Map legacy "deny" to core.PredHalt ("halt")
	queryDeny := fmt.Sprintf("%s(\"%s\")", core.PredHalt, entityID)
	denied, err := e.runtime.ExecuteQuery(facts, queryDeny)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("policy evaluation failed (deny check)", "error", err)
		}
		return fmt.Errorf("policy evaluation error: %w", err)
	}

	if denied {
		// Query: violation_msg(Msg)
		qErr := e.runtime.QueryWithSolutions(facts, fmt.Sprintf("%s(Msg)", core.PredViolation), func(solution map[string]any) error {
			if msg, ok := solution["Msg"].(string); ok {
				violationMsg = msg
				return ErrSolutionFound
			}
			return nil
		})
		if errors.Is(qErr, ErrSolutionFound) {
			qErr = nil
		}
		if qErr != nil && e.logger != nil {
			e.logger.Warn("failed to query violation message", "error", qErr)
		}

		// Query: violation_rule(ID)
		qErr = e.runtime.QueryWithSolutions(facts, "violation_rule(ID)", func(solution map[string]any) error {
			if id, ok := solution["ID"].(string); ok {
				ruleID = id
				return ErrSolutionFound
			}
			return nil
		})
		if errors.Is(qErr, ErrSolutionFound) {
			qErr = nil
		}
		if qErr != nil && e.logger != nil {
			e.logger.Warn("failed to query violation rule ID", "error", qErr)
		}

		if e.logger != nil {
			e.logger.Debug("gate violation detected (deny)", "action", actionName, "msg", violationMsg, "rule_id", ruleID)
		}
		return &core.AlignmentError{Message: violationMsg, RuleID: ruleID}
	}

	return nil
}

// CheckRequirement queries: requires("req_id", "capability")
func (e *PolicyEngine) CheckRequirement(ctx context.Context, input core.Envelope, reqName string) (bool, error) {
	if e.runtime == nil {
		return false, nil
	}

	// 1. Convert Input to Facts using the PRIVATE helper
	// Signature: toMangleFacts(entityID, payload, contentType)
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		return false, fmt.Errorf("fact conversion failed: %w", err)
	}

	// 2. Construct Query
	// Query format: requires("Req", "memory")
	query := fmt.Sprintf(`requires("%s", "%s")`, core.EntityInput, reqName)

	// 3. Execute Query
	// Returns (bool, error) directly as per current engine design
	return e.ExecuteQuery(ctx, facts, query)
}

// EvaluateSteering executes "Steering Policies" which determine what to do next.
// Unlike Assess/Reflect (which are binary Proceed/Infeasible), Steering returns decisions like "Retry" or "Route".
//
// Logic Priority:
//  1. Correction (Retry): If `retry(Hint)` is derived, we return `RETRY` with the hint.
//  2. Routing (Route): If `route(Target)` is derived, we return `ROUTE` with the target.
//  3. Default: `PROCEED` (Proceed as normal).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input envelope.
//
// Returns:
//   - decision: The decision string (e.g., "RETRY", "ROUTE", "PROCEED").
//   - metadata: A map containing steering details (e.g., {"feedback": "hint"}).
//   - error: An error if evaluation fails.
func (e *PolicyEngine) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	decision := core.DecisionProceed
	metadata := make(map[string]string)

	if e.runtime == nil || e.runtime.programInfo == nil {
		return decision, metadata, nil
	}

	// Convert the input payload to Mangle facts
	// We use "Req" as the entity ID, consistent with Assess
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts for steering", "error", err)
		}
		return decision, metadata, fmt.Errorf("failed to convert input to facts: %w", err)
	}

	// [NEW] Inject Explicit Facts from Envelope
	for _, factStr := range input.Facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse envelop fact", "fact", factStr, "error", err)
			}
			return decision, metadata, fmt.Errorf("envelope fact parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// Inject Metadata facts: meta(Key, Value) and attempt(N)
	for k, v := range input.Metadata {
		safeK := escapeString(k)
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)
		// meta("key", "val")
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}

		// attempt(N) from retry_count
		if k == "retry_count" {
			attemptFact := fmt.Sprintf("attempt(%s)", vStr)
			if atom, err := parse.Atom(attemptFact); err == nil {
				facts = append(facts, atom)
			}
		}
	}

	// 1. Check Correction (Retry)
	// Query: retry(Hint)
	err = e.runtime.QueryWithSolutions(facts, fmt.Sprintf("%s(Hint)", core.PredRetry), func(solution map[string]any) error {
		if hint, ok := solution["Hint"].(string); ok {
			decision = core.DecisionRetry
			metadata[core.KeyFeedback] = hint
			// Stop searching after first match
			return ErrSolutionFound // Use error to break early
		}
		return nil
	})

	if errors.Is(err, ErrSolutionFound) {
		err = nil
	}
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to query retry", "error", err)
		}
	}

	if decision == core.DecisionRetry {
		return decision, metadata, nil
	}

	// 2. Check Routing
	// Query: route(Target)
	err = e.runtime.QueryWithSolutions(facts, fmt.Sprintf("%s(Target)", core.PredRoute), func(solution map[string]any) error {
		if target, ok := solution["Target"].(string); ok {
			decision = core.DecisionRoute
			metadata[core.KeyNextStep] = target
			return ErrSolutionFound // Use error to break early
		}
		return nil
	})

	if errors.Is(err, ErrSolutionFound) {
		err = nil
	}
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to query route", "error", err)
		}
	}

	return decision, metadata, nil
}

// ExecuteQuery executes a raw Datalog query against the current program state.
// It wraps the underlying runtime execution with observability (tracing).
//
// Parameters:
//   - ctx: The execution context.
//   - facts: Temporary facts to include in this specific query execution.
//   - queryStr: The Datalog query atom.
//
// Returns:
//   - true if derived, false otherwise.
//   - error if execution fails.
func (e *PolicyEngine) ExecuteQuery(ctx context.Context, facts []ast.Atom, queryStr string) (bool, error) {
	if e.tracer == nil {
		return e.runtime.ExecuteQuery(facts, queryStr)
	}

	ctx, span := e.tracer.Start(ctx, "Datalog.ExecuteQuery")
	defer span.End()

	span.SetAttributes(map[string]any{"datalog.query": queryStr})

	res, err := e.runtime.ExecuteQuery(facts, queryStr)
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	span.SetAttributes(map[string]any{"datalog.result": res})
	return res, nil
}

// Query executes a Datalog query and returns all matching solutions.
// Each solution is a map where keys are variable names (e.g., "Action") and values are stringified constants.
//
// Parameters:
//   - ctx: The execution context.
//   - facts: Temporary facts (strings) to include.
//   - queryStr: The Datalog query with variables (e.g., 'plan_step(Action, Order)').
//
// Returns:
//   - A list of solution maps.
//   - An error if execution fails.
func (e *PolicyEngine) Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error) {
	var results []map[string]string

	if e.runtime == nil {
		return nil, fmt.Errorf("runtime not initialized")
	}

	// Parse temporary facts
	var atomFacts []ast.Atom
	for _, f := range facts {
		atom, err := parse.Atom(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", f, err)
		}
		atomFacts = append(atomFacts, atom)
	}

	if e.tracer != nil {
		var span core.Span
		ctx, span = e.tracer.Start(ctx, "Datalog.Query")
		defer span.End()
		span.SetAttributes(map[string]any{"datalog.query": queryStr})
	}

	err := e.runtime.QueryWithSolutions(atomFacts, queryStr, func(solution map[string]any) error {
		// Convert map[string]any to map[string]string
		strMap := make(map[string]string)
		for k, v := range solution {
			if s, ok := v.(string); ok {
				strMap[k] = s
			} else {
				strMap[k] = fmt.Sprintf("%v", v)
			}
		}
		results = append(results, strMap)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// toMangleFacts helper converts a Go struct to Mangle atoms via the Reflection API.
func toMangleFacts(entityID string, input any, contentType core.ContentType) ([]ast.Atom, error) {
	if input == nil {
		return nil, nil
	}

	var atoms []ast.Atom
	var facts []string
	var err error

	// Choose strategy based on ContentType
	if contentType == core.TypeJSON {
		facts, err = Flatten(entityID, input)
	} else {
		// Default to Reflection (Struct)
		facts, err = ToFacts(entityID, input)
	}

	if err != nil {
		return nil, err
	}

	// Parse each fact string back into an ast.Atom
	for _, factStr := range facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", factStr, err)
		}
		atoms = append(atoms, atom)
	}

	return atoms, nil
}
```

### `internal/supervisor/supervisor.go`
```go
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/telemetry"

	"go.opentelemetry.io/otel"
)

// SupervisedAction is a decorator that wraps any `core.Action` to enforce governance blueprints.
// It implements the standard "Trace -> Assess -> Execute -> Reflect" lifecycle.
//
// Lifecycle:
//  1. Trace: Starts an OpenTelemetry span for the operation.
//  2. Assess: Checks Pre-Check blueprints (e.g., "infeasible(Req)").
//  3. Execute: Runs the inner action (e.g., calls the LLM).
//  4. Reflect: Checks Post-Check blueprints (e.g., "infeasible(Output)").
//  5. Steering: Evaluates steering blueprints for routing or correction.
type SupervisedAction struct {
	inner       core.Action
	engine      core.Evaluator
	tracer      core.Tracer
	failureMode string
}

// NewSupervisedAction creates a new SupervisedAction with default settings (no tracing).
//
// Parameters:
//   - action: The inner action to supervise.
//   - eng: The policy engine (evaluator) to use for governance.
//   - failureMode: The resilience strategy ("open" or "closed").
//
// Returns:
//   - A new SupervisedAction instance.
func NewSupervisedAction(action core.Action, eng core.Evaluator, failureMode string) *SupervisedAction {
	return &SupervisedAction{
		inner:       action,
		engine:      eng,
		tracer:      &core.NopTracer{},
		failureMode: failureMode,
	}
}

// NewSupervisedActionWithTracer creates a new SupervisedAction with tracing enabled.
//
// Parameters:
//   - action: The inner action to supervise.
//   - eng: The policy engine (evaluator).
//   - tracer: The tracer implementation.
//   - failureMode: "open" (log only on system error) or "closed" (block on system error).
//
// Returns:
//   - A new SupervisedAction instance.
func NewSupervisedActionWithTracer(action core.Action, eng core.Evaluator, tracer core.Tracer, failureMode string) *SupervisedAction {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	return &SupervisedAction{
		inner:       action,
		engine:      eng,
		tracer:      tracer,
		failureMode: failureMode,
	}
}

// Execute runs the supervised action, orchestrating the full governance lifecycle.
//
// It performs the following steps:
//  1. Starts a span.
//  2. Injects the logger into the context.
//  3. Runs Assess(). If it fails, execution halts (unless Fail-Open).
//  4. Runs the inner Action.Execute().
//  5. Propagates taint labels from input to output.
//  6. Runs Reflect(). If it fails, the result is blocked.
//  7. Runs EvaluateSteering() to determine next steps (Retry/Route).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The data envelope.
//
// Returns:
//   - The result envelope (possibly modified by blueprint), or an error.
func (g *SupervisedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Auto-Tracing (Phase 5)
	// We use the injected core.Tracer if available, otherwise fallback to global OTel.
	tracer := g.tracer
	if tracer == nil {
		tracer = telemetry.NewOTelTracer(otel.Tracer("manglekit"))
	}
	meta := g.inner.Metadata()

	ctx, span := tracer.Start(ctx, fmt.Sprintf("Action.%s", meta.Name))
	defer span.End()

	span.SetAttributes(map[string]any{
		"mangle.action_name": meta.Name,
		"mangle.action_type": string(meta.Type),
		"mangle.input_id":    input.ID.String(),
	})

	result, err := g.executeInternal(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		// Distinguish between Blueprint HALT and System ERROR
		if g.isAlignmentIssue(err) {
			span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeHalt})
			var alignErr *core.AlignmentError
			if errors.As(err, &alignErr) {
				attrs := map[string]any{core.KeyFeedback: alignErr.Message}
				if alignErr.RuleID != "" {
					attrs[core.AttrRuleID] = alignErr.RuleID
				}
				span.SetAttributes(attrs)
			} else {
				span.SetAttributes(map[string]any{core.KeyFeedback: err.Error()})
			}
		} else {
			span.SetAttributes(map[string]any{core.AttrOutcome: "ERROR"})
		}
		return core.Envelope{}, err
	}

	// Success Path: Determine outcome (Proceed/Route/Retry)
	decision := result.Metadata[core.KeyDecision]
	switch decision {
	case core.DecisionRetry:
		span.SetAttributes(map[string]any{core.AttrOutcome: "retry"})
		if hint, ok := result.Metadata[core.KeyFeedback]; ok {
			if s, ok := hint.(string); ok {
				span.SetAttributes(map[string]any{core.KeyFeedback: s})
			}
		}
	case core.DecisionRoute:
		span.SetAttributes(map[string]any{core.AttrOutcome: "route"})
		if target, ok := result.Metadata[core.KeyNextStep]; ok {
			if s, ok := target.(string); ok {
				span.SetAttributes(map[string]any{core.AttrActionName: s})
			}
		}
	default:
		span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})
	}

	// Inject Retry Count if present
	if attemptVal, ok := input.Metadata["retry_count"]; ok {
		// handle both string and int
		if s, ok := attemptVal.(string); ok {
			if n, err := strconv.Atoi(s); err == nil {
				span.SetAttributes(map[string]any{core.AttrAttempt: n})
			}
		} else if n, ok := attemptVal.(int); ok {
			span.SetAttributes(map[string]any{core.AttrAttempt: n})
		}
	}

	span.SetAttributes(map[string]any{
		core.AttrOutcome:     core.OutcomeProceed,
		"mangle.output_id":   result.ID.String(),
	})
	return result, nil
}

// isAlignmentIssue checks if the error is a wrapped alignment check violation
func (g *SupervisedAction) isAlignmentIssue(err error) bool {
	return core.IsAlignmentError(err)
}

// Metadata delegates to the inner action's Metadata method.
// This allows the SupervisedAction to transparently represent the underlying capability.
func (g *SupervisedAction) Metadata() core.ActionMetadata {
	return g.inner.Metadata()
}

// shouldBlock determines if the action should be blocked based on the error and failure mode.
func (g *SupervisedAction) shouldBlock(err error) bool {
	if err == nil {
		return false
	}
	// Always block on explicit alignment issues
	if core.IsAlignmentError(err) {
		return true
	}
	// If mode is "open" (Fail-Open), allow execution (return false)
	// Otherwise (default/closed), block execution (return true)
	if g.failureMode == "open" {
		return false
	}
	return true
}

// executeInternal contains the actual execution logic.
// It receives the context with the active span so child spans can be created.
// The logger is injected into the context here, ensuring all downstream code
// can access it via core.LoggerFromContext(ctx).
func (g *SupervisedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Inject the logger into the context for downstream access
	ctx = core.ContextWithLogger(ctx, g.engine.Logger())

	logger := core.LoggerFromContext(ctx)
	meta := g.inner.Metadata()
	logger.Info("Action started",
		"action", meta.Name,
		"input_id", input.ID.String(),
	)

	// 1. Ingestion: Link Input to Parent (Tracing only)
	// if parentID, ok := core.GetParentID(ctx); ok {
	// 	// Evaluator doesn't support RecordLineage directly.
	// 	// g.engine.RecordLineage(ctx, input.ID.String(), parentID)
	// }

	// 2. Pre-Check: Assessment
	// Formerly: Authorize
	if err := g.engine.Assess(ctx, g.inner.Metadata(), input); err != nil {
		if g.shouldBlock(err) {
			logger.Warn("assessment failed",
				core.AttrActionName, meta.Name,
				"error", err.Error(),
			)
			return core.Envelope{}, fmt.Errorf("assessment failed: %w", err)
		}
		// Fail-Open
		logger.Warn("engine assessment failed but Fail-Open active. Proceeding.", "error", err)
	}

	// [NEW] Dynamic Configuration Injection
	// We query the engine for any configuration overrides (e.g. prompt params)
	config, err := g.engine.GetActionConfig(ctx, input)
	if err != nil {
		logger.Warn("failed to retrieve action config", "error", err)
	} else if len(config) > 0 {
		if input.Metadata == nil {
			input.Metadata = make(map[string]any)
		}
		for k, v := range config {
			input.Metadata[core.PrefixPromptConfig+k] = v
		}
	}

	// 3. Context Propagation: Pass the Gene
	// Propagate the current input ID as the new parent for the inner action
	childCtx := core.WithParentID(ctx, input.ID.String())

	// 4. Execution: Run inner action
	result, err := g.inner.Execute(childCtx, input)
	if err != nil {
		logger.Error("action execution failed",
			core.AttrActionName, meta.Name,
			"error", err.Error(),
		)
		return core.Envelope{}, fmt.Errorf("action execution failed: %w", err)
	}

	// 5. Propagation: Output inherits Input's security labels
	if len(input.SecurityLabels) > 0 {
		result.MergeLabels(input.SecurityLabels)
	}

	// 6. Linking: Link Output to Input (Tracing only)
	// g.engine.RecordLineage(ctx, result.ID.String(), input.ID.String())
	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata["derived_from"] = input.ID.String()

	// 7. Post-Check: Reflection
	// Formerly: Validate
	validatedResult, err := g.engine.Reflect(ctx, g.inner.Metadata(), result)
	if err != nil {
		if g.shouldBlock(err) {
			logger.Warn("reflection failed",
				"action", meta.Name,
				"error", err.Error(),
			)
			return core.Envelope{}, fmt.Errorf("reflection failed: %w", err)
		}
		// Fail-Open: use result as validatedResult
		logger.Warn("engine reflection failed but Fail-Open active. Proceeding.", "error", err)
		validatedResult = result
	}

	// 8. Steering: Evaluate next steps (Correction/Routing)
	decision, steeringMeta, err := g.engine.EvaluateSteering(ctx, validatedResult)
	if err != nil {
		logger.Warn("steering evaluation failed",
			"action", meta.Name,
			"error", err.Error(),
		)
		return core.Envelope{}, fmt.Errorf("steering evaluation failed: %w", err)
	}

	// Stamp metadata
	if validatedResult.Metadata == nil {
		validatedResult.Metadata = make(map[string]any)
	}
	validatedResult.Metadata[core.KeyDecision] = decision
	for k, v := range steeringMeta {
		validatedResult.Metadata[k] = v
	}

	logger.Info("Action completed",
		"action", meta.Name,
		"result", "success", // Simplified as per doc
	)

	return validatedResult, nil
}
```

### `cmd/mkit/commands/gen/logic.go`
```go
package gen

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/gen/inductor"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/duynguyendang/manglekit/internal/engine/resources"
)

// GeneratedPolicy represents the structured response from the LLM.
type GeneratedPolicy struct {
	// DatalogContent contains the raw .dl code.
	DatalogContent string `json:"datalog_content"`
	// Explanation provides a human-readable summary of the logic.
	Explanation string `json:"explanation"`
}

// ValidatePolicySyntax checks if the generated Datalog code is valid using the Mangle engine.
func ValidatePolicySyntax(datalog, schemaDeclarations string) error {
	// Initialize a runtime.
	eng := engine.NewMangleRuntime()

	// Prepend standard declarations to the validator so usages of std lib don't fail.
	// We use the single source of truth from internal/engine/resources.
	// We also strictly declare 'deny' as it is the expected output interface.
	stdDecls := resources.StdLib()
	denyDecl := "Decl deny(Source, Reason) ."
	fullProgram := stdDecls + "\n" + denyDecl + "\n" + schemaDeclarations + "\n" + datalog

	// Attempt to parse and compile the policy
	return eng.LoadFromSource(fullProgram)
}

// GenerateWithFeedback orchestrates the Teacher-Student protocol.
func GenerateWithFeedback(ctx context.Context, gen core.TextGenerator, userReq string, domainVocab []string, schema *inductor.SchemaHint, iclContent string) (*GeneratedPolicy, error) {
	// 1. Construct System Prompt
	factsList := ""
	for _, f := range domainVocab {
		factsList += "- " + f + "\n"
	}

	autoVocab := ""
	if schema != nil {
		if schema.FileType == "graph" {
			// For Graph: "Decl is_vip(S, O)."
			// We format declarations as a list
			decls := strings.Join(schema.Declarations, "\n")
			autoVocab = fmt.Sprintf(`
### Auto-Detected Vocabulary (Graph):
The data uses these predicates. Use them directly:
%s
`, decls)
		} else if schema.FileType == "json" {
			// For JSON: "amount (number)", "desc (string)"
			keys := strings.Join(schema.JsonKeys, "\n")
			autoVocab = fmt.Sprintf(`
### Auto-Detected JSON Structure (Path -> Type):
%s
### Handling Nested & Array Paths:
1. **Dot Notation** ("deployment.replicas"): Use 'json_link' to traverse objects.
   json_link(Root, "deployment", DeployNode), json_num(DeployNode, "replicas", Val).

2. **Object Arrays** ("servers[].ip"): Use 'json_link' to list, then 'json_link' with '_' (wildcard) to iterate items.
   json_link(Root, "servers", List), json_link(List, _, Server), json_str(Server, "ip", IP).

3. **Primitive Arrays** ("env_vars (array of string)"): Use 'json_link' to list, then 'json_str' with '_' to iterate values.
   json_link(Root, "env_vars", List), json_str(List, _, "TargetValue").
`, keys)
		}
	}

	systemPrompt := fmt.Sprintf(`You are a Senior Knowledge Engineer specializing in Google Mangle Datalog.
Your task is to translate natural language requirements into strict, compilable Datalog rules.

### Standard Library (Always Available):
- json_num(Source, Key, Value)  // Int/Float fields
- json_str(Source, Key, Value)  // String fields
- json_bool(Source, Key, Value) // Boolean fields
- json_link(Parent, Key, Child) // Nested objects
- deny(Source, Reason)          // Main policy output (Do NOT redeclare this)

### Telemetry & Compliance (MANDATORY):
1. You MUST declare a violation rule predicate: Decl violation_rule(Entity, RuleID).
2. For EVERY 'deny' rule you create, you MUST create a corresponding 'violation_rule'.
   - The RuleID must be UPPERCASE_SNAKE_CASE (e.g., "COST_LIMIT_EXCEEDED").
   - It captures the same conditions as the deny rule.

Example:
   deny(Req, "Cost too high") :- exceeds_cost(Req).
   violation_rule(Req, "COST_LIMIT_CHECK_01") :- exceeds_cost(Req).

### Domain Vocabulary:
%s
%s
### 4. Code Style Reference (Golden Rules)
The following are verified Manglekit Datalog examples.
Pay close attention to how 'json_link' is used for nested objects and how predicates are declared.

--- BEGIN REFERENCE ---
%s
--- END REFERENCE ---

### Syntax Rules:
1. Use 'Decl name(Arg1, Arg2, ...).' to declare predicates.
   - CRITICAL: Do NOT use '.Decl', '.decl', or 'decl'. MUST be 'Decl' (Case-sensitive, no dot).
   - Args MUST start with Uppercase (e.g., Decl amount(Source, Val).).
2. Do NOT redeclare 'deny'. It is already declared.
3. Prioritize using the Domain Vocabulary over raw json_xxx predicates if available.
4. If mapping raw JSON, prefer creating a "Helper Predicate" first (e.g., amount(R, V) :- json_num...) to keep the deny rule clean.
5. Strings must be double-quoted.
6. Variables start with uppercase (e.g., P, Amount).
7. Do NOT use aggregation (max, count) unless absolutely necessary.

Output JSON only: {"datalog_content": "...", "explanation": "..."}`, factsList, autoVocab, iclContent)

	currentReq := userReq
	var lastErr error

	// The Loop (Max 5 Retries)
	for i := 0; i < 5; i++ {
		// Step B (Generation)
		policy, err := ai.GenerateStruct[GeneratedPolicy](ctx, gen, systemPrompt, currentReq)
		if err != nil {
			// If generation itself fails (e.g. network), we probably shouldn't blindly retry unless it's transient.
			lastErr = err
			// Feedback for JSON error
			feedback := fmt.Sprintf("Generation failed: %v", err)
			currentReq = fmt.Sprintf("%s\n\n[SYSTEM CORRECTION]: Previous attempt invalid.\nError: %s\nFix syntax immediately.", userReq, feedback)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Step C (Validation)
		schemaDeclsVal := ""
		if schema != nil && len(schema.Declarations) > 0 {
			schemaDeclsVal = strings.Join(schema.Declarations, "\n")
		}
		if err := ValidatePolicySyntax(policy.DatalogContent, schemaDeclsVal); err != nil {
			lastErr = err
			// Step D (Decision) - Update feedback
			feedback := err.Error()
			fmt.Printf("DEBUG: Validation failed for Datalog:\n%s\nError: %s\n", policy.DatalogContent, feedback)
			// Step A (Prompting) - Update prompt for next iteration
			currentReq = fmt.Sprintf("%s\n\n[SYSTEM CORRECTION]: Previous attempt invalid.\nError: %s\nCheck your decl syntax. Do NOT use .Decl or .decl.", userReq, feedback)

			// Sleep briefly
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Success
		return &policy, nil
	}

	return nil, fmt.Errorf("failed after 5 retries. Last error: %w", lastErr)
}
```

### `cmd/mkit/commands/gen/inductor/inductor.go`
```go
package inductor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SchemaHint holds the inferred schema information.
type SchemaHint struct {
	Declarations []string // For Graph: "Decl is_vip(S, O)."
	JsonKeys     []string // For JSON: "amount (number)", "desc (string)"
	FileType     string   // "graph" or "json"
}

// InferFromFile scans the file at path and returns a SchemaHint.
func InferFromFile(path string) (*SchemaHint, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return parseJSON(path)
	case ".nq", ".nt":
		return parseGraph(path)
	case ".ttl":
		return parseTurtle(path)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func parseJSON(path string) (*SchemaHint, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}

	hint := &SchemaHint{FileType: "json"}
	var targetMap map[string]any

	switch v := raw.(type) {
	case map[string]any:
		targetMap = v
	case []any:
		if len(v) > 0 {
			if m, ok := v[0].(map[string]any); ok {
				targetMap = m
			}
		}
	}

	if targetMap == nil {
		// Could not find a suitable object to infer schema from
		return hint, nil // Empty hint but valid file type detected
	}

	walkJSON("", targetMap, &hint.JsonKeys)
	return hint, nil
}

func walkJSON(prefix string, data map[string]any, keys *[]string) {
	for k, v := range data {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]any:
			walkJSON(fullKey, val, keys)
		case []any:
			if len(val) > 0 {
				first := val[0]
				if m, ok := first.(map[string]any); ok {
					walkJSON(fullKey+"[]", m, keys)
				} else {
					switch first.(type) {
					case string:
						*keys = append(*keys, fmt.Sprintf("%s (array of string)", fullKey))
					case float64:
						*keys = append(*keys, fmt.Sprintf("%s (array of number)", fullKey))
					case bool:
						*keys = append(*keys, fmt.Sprintf("%s (array of boolean)", fullKey))
					}
				}
			}
		case float64:
			*keys = append(*keys, fmt.Sprintf("%s (number)", fullKey))
		case string:
			*keys = append(*keys, fmt.Sprintf("%s (string)", fullKey))
		case bool:
			*keys = append(*keys, fmt.Sprintf("%s (boolean)", fullKey))
		}
	}
}
```

### `cmd/mkit/commands/kg/convert.go`
```go
package kg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cayleygraph/quad"
	"github.com/cayleygraph/quad/jsonld"
	"github.com/cayleygraph/quad/nquads"
	"github.com/spf13/cobra"
)

var (
	inFile  string
	outFile string
	inFmt   string
	outFmt  string
)

var ConvertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert between knowledge graph formats (e.g., TTL to NQ)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Open Input
		f, err := os.Open(inFile)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer f.Close()

		// 2. Decoder
		format := inFmt
		if format == "" {
			ext := filepath.Ext(inFile)
			switch ext {
			case ".ttl":
				// Turtle not supported in current quad version apparently, use nquads as fallback or error?
				// return fmt.Errorf("turtle format not supported in this version")
				// User explicitly asked for it, but if it's missing...
				// I'll leave a TODO or try to map .ttl to something else if possible? No.
				return fmt.Errorf("turtle format (.ttl) is currently not supported")
			case ".nq":
				format = "nquads"
			case ".nt":
				format = "ntriples"
			case ".jsonld":
				format = "jsonld"
			default:
				return fmt.Errorf("unknown input extension %s, please specify --from", ext)
			}
		}

		var r quad.Reader
		switch format {
		case "nquads", "ntriples":
			// nquads.NewReader likely takes (io.Reader, bool) for strict mode. Default to false?
			r = nquads.NewReader(f, false)
		case "jsonld":
			r = jsonld.NewReader(f)
		default:
			return fmt.Errorf("unsupported input format: %s", format)
		}

		// 3. Open Output
		outF, err := os.Create(outFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outF.Close()

		// 4. Encoder
		var w quad.Writer
		switch outFmt {
		case "nquads", "ntriples":
			// nquads writer is used for both
			w = nquads.NewWriter(outF)
		case "jsonld":
			w = jsonld.NewWriter(outF)
		default:
			return fmt.Errorf("unsupported output format: %s", outFmt)
		}

		// Close writer if it needs closing
		defer func() {
			if c, ok := w.(io.Closer); ok {
				c.Close()
			}
		}()

		// 5. Copy
		n, err := quad.Copy(w, r)
		if err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}

		// 6. UX
		cmd.Printf("Converted %d quads to %s\n", n, outFile)
		return nil
	},
}

func init() {
	ConvertCmd.Flags().StringVarP(&inFile, "input", "i", "", "Input file path")
	ConvertCmd.MarkFlagRequired("input")

	ConvertCmd.Flags().StringVarP(&outFile, "output", "o", "", "Output file path")
	ConvertCmd.MarkFlagRequired("output")

	ConvertCmd.Flags().StringVar(&inFmt, "from", "", "Input format (auto-detect if empty)")
	ConvertCmd.Flags().StringVar(&outFmt, "to", "nquads", "Output format (options: \"nquads\", \"ntriples\", \"jsonld\")")
}
```

### `cmd/mkit/commands/eval/run.go`
```go
package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit/adapters/knowledge"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/spf13/cobra"
)

var (
	policyPath    string
	dataPath      string
	knowledgePath string
	queryString   string
)

var EvalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluate a Datalog query against policy and data",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validation
		if dataPath == "" && knowledgePath == "" {
			return fmt.Errorf("must provide either --data (JSON) or --facts/--knowledge (Graph)")
		}

		// 1. Read Files
		policyBytes, err := os.ReadFile(policyPath)
		if err != nil {
			return fmt.Errorf("failed to read policy file: %w", err)
		}

		var dataBytes []byte
		if dataPath != "" {
			dataBytes, err = os.ReadFile(dataPath)
			if err != nil {
				return fmt.Errorf("failed to read data file: %w", err)
			}
		}

		// 2. Initialize Engine
		// New() now auto-loads std.dl, so no manual declaration fixes needed!
		e, err := engine.New()
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		// 3. Process Knowledge (Facts) First to Extract Schema
		if knowledgePath != "" {
			// Using the new Graph Loader that supports .nq and .ttl
			triples, err := knowledge.ParseGraphFile(knowledgePath)
			if err != nil {
				return fmt.Errorf("failed to parse knowledge base: %w", err)
			}

			// Extract Schema Declarations
			preds := knowledge.GetPredicates(triples)
			if len(preds) > 0 {
				var decls []string
				for _, p := range preds {
					// Default to binary declaration: Decl p(Subject, Object).
					decls = append(decls, fmt.Sprintf("Decl %s(S, O).", p))
				}
				schemaBlock := strings.Join(decls, "\n")

				// Load Declarations BEFORE Policy
				if err := e.LoadPolicy(cmd.Context(), schemaBlock); err != nil {
					return fmt.Errorf("failed to inject schema declarations: %w", err)
				}
				fmt.Printf("Injected %d schema declarations.\n", len(preds))
			}

			// Load Facts
			facts := knowledge.TriplesToFacts(triples)
			if err := e.LoadFacts(facts); err != nil {
				return fmt.Errorf("failed to load knowledge facts: %w", err)
			}
			fmt.Printf("Loaded %d knowledge facts.\n", len(facts))
		}

		// 4. Load Policy (User Policy)
		// Now that Decls are present (if any), loading the policy should succeed.
		if err := e.LoadPolicy(cmd.Context(), string(policyBytes)); err != nil {
			// This might still error if the user uses non-standard predicates without declaring them,
			// but json_str, quad etc are now covered.
			return fmt.Errorf("failed to load policy: %w", err)
		}

		// 5. Inject Data (JSON)
		if dataPath != "" {
			var data any
			if err := json.Unmarshal(dataBytes, &data); err != nil {
				return fmt.Errorf("failed to parse data JSON: %w", err)
			}

			// Flatten the data into facts
			// "input" is used as the root ID for the data
			dataFacts, err := engine.Flatten("input", data)
			if err != nil {
				return fmt.Errorf("failed to flatten data: %w", err)
			}

			if err := e.LoadFacts(dataFacts); err != nil {
				return fmt.Errorf("failed to load data facts: %w", err)
			}
		}

		// 6. Execute Query
		ctx := context.Background()
		// Using Query instead of ExecuteQuery to get results/bindings
		results, err := e.Query(ctx, nil, queryString)
		if err != nil {
			return fmt.Errorf("query execution failed: %w", err)
		}

		// 7. Output
		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		// Print as JSON array for nice formatting
		outputBytes, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
		fmt.Println(string(outputBytes))

		return nil
	},
}

func init() {
	EvalCmd.Flags().StringVarP(&policyPath, "policy", "p", "", "Path to the .dl file")
	EvalCmd.MarkFlagRequired("policy")

	EvalCmd.Flags().StringVarP(&dataPath, "data", "d", "", "Path to the .json input file. Optional if --knowledge is provided.")

	EvalCmd.Flags().StringVarP(&knowledgePath, "knowledge", "k", "", "Path to the .nq, .nt, or .ttl knowledge base")
	EvalCmd.Flags().StringVar(&knowledgePath, "facts", "", "Alias for --knowledge")

	EvalCmd.Flags().StringVarP(&queryString, "query", "q", "", "The Datalog query to execute")
	EvalCmd.MarkFlagRequired("query")
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(EvalCmd)
}
```

### `providers/openai/plugin.go`
```go
package openai

import (
	"context"
	"fmt"
	"os"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	openai "github.com/sashabaranov/go-openai"
)

// Config holds connection parameters.
type Config struct {
	APIKey  string
	BaseURL string // Crucial for Ollama/LocalAI/OpenRouter support
}

// Init registers a specific model (e.g., "gpt-4o") into Genkit's global registry.
// It maps "openai/{modelID}" in Genkit to "{modelID}" in OpenAI API.
func Init(g *genkit.Genkit, modelID string, cfg Config) error {
	// 1. Env Var Fallback
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	// Allow empty Key only if BaseURL is set (e.g. Local Ollama)
	if apiKey == "" && cfg.BaseURL == "" {
		return fmt.Errorf("openai provider: API Key is missing and no BaseURL provided")
	}

	// 2. Setup Client
	c := openai.DefaultConfig(apiKey)
	if cfg.BaseURL != "" {
		c.BaseURL = cfg.BaseURL
	}
	client := openai.NewClientWithConfig(c)

	// 3. Define Metadata
	meta := &ai.ModelOptions{
		Label: modelID,
		Supports: &ai.ModelSupports{
			Multiturn: true, SystemRole: true, Tools: false, Media: false,
		},
	}

	// 4. Register
	// Genkit Name: "openai/gpt-4o"
	// We use genkit.DefineModel to register it to the instance.
	// Name: "openai/" + modelID
	fullName := "openai/" + modelID
	genkit.DefineModel(g, fullName, meta, func(ctx context.Context, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
		// Pass the explicit 'modelID' to ensure we don't send "openai/gpt-4o" to the API
		return generate(ctx, client, modelID, req, cb)
	})

	return nil
}

// generate handles the translation logic
func generate(ctx context.Context, client *openai.Client, explicitModelID string, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
	// CONSTRAINT: No Streaming Support yet
	if cb != nil {
		return nil, fmt.Errorf("streaming is not supported by this provider yet")
	}

	// 1. Map Messages
	msgs := []openai.ChatCompletionMessage{}
	for _, m := range req.Messages {
		role := openai.ChatMessageRoleUser
		switch m.Role {
		case ai.RoleModel:
			role = openai.ChatMessageRoleAssistant
		case ai.RoleSystem:
			role = openai.ChatMessageRoleSystem
		}

		content := ""
		for _, p := range m.Content {
			if p.Text != "" {
				content += p.Text
			}
		}
		msgs = append(msgs, openai.ChatCompletionMessage{Role: role, Content: content})
	}

	// 2. Prepare Request
	// IMPORTANT: Use explicitModelID
	oaiReq := openai.ChatCompletionRequest{
		Model:    explicitModelID,
		Messages: msgs,
	}

	// 3. Map Config
	// Config is 'any' in ModelRequest. We need to cast it.
	// Genkit passes 'GenerationCommonConfig' usually.
	// Fields are values (not pointers) in this version.
	if cfg, ok := req.Config.(ai.GenerationCommonConfig); ok {
		if cfg.Temperature != 0 {
			oaiReq.Temperature = float32(cfg.Temperature)
		}
		if cfg.MaxOutputTokens != 0 {
			oaiReq.MaxTokens = cfg.MaxOutputTokens
		}
		if cfg.TopP != 0 {
			oaiReq.TopP = float32(cfg.TopP)
		}
	}

	// 4. Execute
	resp, err := client.CreateChatCompletion(ctx, oaiReq)
	if err != nil {
		return nil, fmt.Errorf("openai completion error: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	// 5. Return Response
	return &ai.ModelResponse{
		Message: ai.NewModelTextMessage(resp.Choices[0].Message.Content),
		// Fill Usage if available
		Usage: &ai.GenerationUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}, nil
}
```

### `cmd/mkit/commands/serve/serve.go`
```go
package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	port         string
	policyPath   string
	mcpConfig    string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Manglekit HTTP Server",
	Long:  `Exposes the Manglekit SDK via an HTTP API, enforcing governance policies on every request.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func AddCommands(root *cobra.Command) {
	serveCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to listen on")
	serveCmd.Flags().StringVarP(&policyPath, "policy", "f", "", "Path to the Datalog policy file")
	serveCmd.Flags().StringVarP(&mcpConfig, "mcp", "m", "", "Path to MCP configuration file")
	root.AddCommand(serveCmd)
}

func runServer() {
	opts := []sdk.ClientOption{}
	if policyPath != "" {
		opts = append(opts, sdk.WithBlueprintPath(policyPath))
	}

	ctx := context.Background()
	client, err := sdk.NewClient(ctx, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize client: %v\n", err)
		os.Exit(1)
	}

	handler := createHandler(client)
	http.Handle("/", handler)

	fmt.Printf("Manglekit HTTP Server listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
		os.Exit(1)
	}
}

// Handler handles the HTTP request.
// It is separated for testability.
func createHandler(client *sdk.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var envelope core.Envelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		// FIX: Initialize metadata if nil
		if envelope.Metadata == nil {
			envelope.Metadata = make(map[string]any)
		}

		// FIX: Generate UUID if missing
		if envelope.ID == uuid.Nil {
			envelope.ID = uuid.New()
		}

		// Execution
		result, err := client.Execute(r.Context(), envelope)

		// Response Mapping
		if err != nil {
			// Check for Policy Violation (AlignmentError)
			if core.IsAlignmentError(err) {
				// Case B: Policy Violation
				w.WriteHeader(http.StatusForbidden)

				// Construct JSON body with deny reasons
				var alignErr *core.AlignmentError
				errors.As(err, &alignErr)

				// Create a structured response
				resp := map[string]any{
					"error": "Policy Violation",
					"reasons": []string{alignErr.Message},
					"rule_id": alignErr.RuleID,
					"decision": core.DecisionHalt,
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			// Case A: Internal Error
			http.Error(w, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
			return
		}

		// Case B: Policy Violation (Check Metadata)
		// Even if err is nil, check decision metadata
		if d, ok := result.Metadata[core.KeyDecision]; ok && d == core.DecisionHalt {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(result) // Return full result as body
			return
		}

		// Case C: Success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}
```

### `cmd/mkit/commands/inspect/struct.go`
```go
package inspect

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/spf13/cobra"
)

var jsonFlag string

var StructCmd = &cobra.Command{
	Use:   "struct",
	Short: "Inspect a struct and see the generated Mangle facts.",
	Long:  `Parses a JSON string or file representing a struct and uses the core reflection engine to generate Mangle Datalog facts. This helps users understand how Manglekit perceives their data structures.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var jsonData []byte
		var err error

		// Check if the input is a file path or a raw JSON string
		if _, err := os.Stat(jsonFlag); err == nil {
			jsonData, err = os.ReadFile(jsonFlag)
			if err != nil {
				return fmt.Errorf("failed to read JSON file: %w", err)
			}
		} else {
			jsonData = []byte(jsonFlag)
		}

		var data map[string]any
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		facts, err := engine.ToFacts("request", data)
		if err != nil {
			return fmt.Errorf("failed to generate facts: %w", err)
		}

		for _, fact := range facts {
			fmt.Println(fact)
		}

		return nil
	},
}

func init() {
	StructCmd.Flags().StringVar(&jsonFlag, "json", "", "JSON string or file path representing a struct")
	StructCmd.MarkFlagRequired("json")
	InspectCmd.AddCommand(StructCmd)
}
```

## 10. Known Gaps

*   **Google Provider Dependency**: The `googleai` plugin package is missing in the `genkit/go` `v1.2.0` module. We are polyfilling it using the `googlegenai` plugin (which exists) combined with a manual proxy registration pattern to ensure compatibility with `Init(ctx, apiKey)` semantics.

## 14. CHANGELOG

- **2025-12-05**: Refactored `providers/google` to use the `googlegenai` plugin with a proxy pattern due to missing `googleai` package in Genkit v1.2.0. Updated `sdk` loader to support new `Init` pattern.
- **2025-12-17**: Full Context Resync. Exhaustive scan and source dump of critical kernel components (`core`, `sdk`, `engine`, `supervisor`), plus `cmd/mkit` generation logic and `providers/openai` plugin.
