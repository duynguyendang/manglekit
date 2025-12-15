---
context_type: full_source_dump
project: manglekit
language: go
last_updated: 2025-12-15
scan_mode: exhaustive
---

#### 2. THE COMPLETE FILE MAP

```
.
├── .env.example
├── .gitignore
├── AGENTS.md # Operational manual for AI agents
├── CONTRIBUTING.md
├── LICENSE
├── Makefile
├── README.md
├── adapters # Integration layer for external capabilities
│   ├── ai # Genkit & LLM Integration
│   │   ├── adapter.go
│   │   ├── adapter_test.go
│   │   ├── genkit.go
│   │   └── utils.go
│   ├── extractor # Structured Data Extraction
│   │   ├── adapter.go
│   │   └── adapter_test.go
│   ├── func # Generic Function Wrapper
│   │   └── wrapper.go
│   ├── knowledge # RDF/Graph ETL
│   │   ├── graph_loader.go
│   │   ├── graph_loader_test.go
│   │   ├── nquads.go
│   │   ├── nquads_test.go
│   │   ├── ntriples.go
│   │   ├── rdf.go
│   │   └── rdf_stub.go
│   ├── logger # Logger Adapters
│   │   └── zap_adapter.go
│   ├── mcp # Model Context Protocol
│   │   ├── action.go
│   │   ├── loader.go
│   │   └── loader_test.go
│   ├── resilience # Circuit Breakers & Retries
│   │   ├── circuit_breaker.go
│   │   └── circuit_breaker_test.go
│   └── vector # Vector Store Retrievers
│       ├── genkit_retriever.go
│       ├── retriever_adapter.go
│       └── retriever_adapter_test.go
├── cmd # Entry points
│   └── mkit # CLI Tooling
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
│       │   └── kg
│       │       ├── convert.go
│       │       └── root.go
│       └── main.go
├── config # Configuration Loading & Schema
│   ├── loader.go
│   ├── loader_test.go
│   └── schema.go
├── core # Interfaces & Contracts (The "Kernel")
│   ├── context_facts.go
│   ├── context_lineage.go
│   ├── data.go
│   ├── errors.go
│   ├── governance.go
│   ├── infra.go
│   ├── logger.go
│   ├── logger_test.go
│   ├── logic.go
│   ├── state.go
│   ├── tracer.go
│   └── types.go
├── docs # Documentation & Architecture Records
│   ├── ADR.md
│   ├── ARCHITECTURE_RULES.md
│   ├── CONFIG.md
│   ├── CONTEXT.md
│   ├── CSD.md
│   ├── HLD.md
│   ├── LLD.md
│   ├── LOGGING.md
│   ├── Mangle-quickstart.md
│   ├── TRACING.md
│   ├── VOCABULARY.md
│   └── reports
│       ├── CONFIG_LOADER_MIGRATION_SUMMARY.md
│       ├── IMPLEMENTATION_CHECKLIST.md
│       ├── IMPLEMENTATION_SUMMARY.md
│       ├── POLICY_COPILOT_GUIDE.md
│       ├── POLICY_COPILOT_IMPLEMENTATION.md
│       ├── code_audit_20250604.md
│       └── feature_audit_v1.0.0.md
├── examples # Usage Examples
│   ├── AGENTS.md
│   ├── README.md
│   ├── chat_chit
│   ├── context_aware_rag
│   ├── dynamic_pricing
│   ├── extractor_demo
│   ├── fintech_approval
│   ├── kernel_knowledge_demo
│   ├── lineage_demo
│   ├── openrouter_demo
│   ├── planner
│   ├── policy-copilot
│   ├── rag_flow
│   ├── semantic_feedback
│   ├── sre_guardrail
│   ├── steering
│   └── taint_demo
├── go.mod
├── go.sum
├── internal # Internal Implementation Details
│   ├── engine # Logic Engine (Datalog Runtime)
│   │   ├── dual_mode_test.go
│   │   ├── evaluator.go
│   │   ├── evaluator_test.go
│   │   ├── flattener.go
│   │   ├── flattener_test.go
│   │   ├── memory
│   │   │   ├── volatile.go
│   │   │   └── volatile_test.go
│   │   ├── reflection.go
│   │   ├── reflection_test.go
│   │   ├── resources
│   │   │   ├── embed.go
│   │   │   ├── planner.dl
│   │   │   └── std.dl
│   │   ├── runtime.go
│   │   ├── solver.go
│   │   └── solver_test.go
│   ├── logger # Internal Logging
│   │   └── std.go
│   ├── resources # Shared Resources
│   │   └── icl
│   │       ├── embed.go
│   │       └── golden.dl
│   ├── statehelper # State Utilities
│   │   └── statehelper.go
│   ├── supervisor # Governance Supervisor (The "Guard")
│   │   ├── supervisor.go
│   │   ├── supervisor_test.go
│   │   └── trace_test.go
│   ├── telemetry # OpenTelemetry Helpers
│   │   └── otel.go
│   ├── testproviders # Test Mocks
│   │   ├── mock
│   │   │   └── mock.go
│   │   ├── mocks.go
│   │   └── noop
│   │       └── noop_testhooks.go
│   ├── tools # Build Tools
│   │   └── rulegen
│   │       └── example_test.go
│   └── util # Utilities
│       └── schema
│           ├── generator.go
│           ├── schema_test.go
│           └── validator.go
├── mangle.yaml
├── manglekit.go # Public Facade
├── providers # Standard Plugins
│   ├── google
│   │   └── gemini.go
│   └── openrouter
│       └── client.go
└── sdk # Developer SDK & Steering Loop
    ├── client.go
    ├── client_execute.go
    ├── config_loader.go
    ├── context.go
    ├── executor.go
    ├── generics.go
    ├── helpers.go
    ├── loop.go
    ├── options.go
    ├── registry.go
    └── policy_generator.go
```

#### 3. COMPONENT ANALYSIS (The Logic)

**Engine (`internal/engine`)**
*   **Key Structs:** `PolicyEngine` (Solver), `MangleRuntime` (Wrapper), `Evaluator` (Single Rule).
*   **Responsibilities:**
    *   Manages the lifecycle of the Google Mangle Datalog runtime.
    *   Loads policies from strings/files (`Load`, `LoadFromSource`).
    *   Performs `Authorize` (Pre-Check) and `Validate` (Post-Check) using Datalog queries.
    *   Evaluates `Steering` logic (Correction/Routing/Retry).
    *   Handles Reflection (`ToFacts`) and Flattening (`Flatten`) to convert Go structs/JSON to Datalog facts.
    *   Injects security labels as facts for taint tracking.

**SDK (`sdk`)**
*   **Key Structs:** `Client`, `Generator` (Policy Copilot), `ExecutionParams`.
*   **Responsibilities:**
    *   Provides the high-level API for users (`NewClient`, `ExecuteByName`).
    *   Manages the "Steering Loop" (`ExecuteByName` -> `runLoopInternal`) which handles retries, routing, and feedback.
    *   Registers Actions (`RegisterAction`) and routes execution.
    *   Manages Context injection (Logger, Tracer, Facts).
    *   Implements the "Policy Copilot" (`policy_generator.go`) for LLM-based rule generation.

**Adapters (`adapters`)**
*   **Key Structs:** `genkitAdapter` (AI), `MCPAction`, `CircuitBreaker`.
*   **Responsibilities:**
    *   Wraps external capabilities (Genkit, MCP, Functions) into the `core.Action` interface.
    *   Handles protocol translation (e.g., JSON -> Envelope).
    *   Implements resilience patterns (Circuit Breaker) and data loading (Knowledge/Vector).

**Supervisor (`internal/supervisor`)**
*   **Key Structs:** `SupervisedAction`.
*   **Responsibilities:**
    *   Implements the "Universal Guarded Action" (UGA) pattern.
    *   Decorates any `core.Action` with:
        1.  **Trace:** Auto-starts OpenTelemetry spans (`manglekit` tracer).
        2.  **Authorization:** Calls `Engine.Authorize`.
        3.  **Config Injection:** Dynamically injects config from facts (`GetActionConfig`).
        4.  **Execution:** Invokes the inner action with propagated context.
        5.  **Taint Propagation:** Merges input labels to output.
        6.  **Validation:** Calls `Engine.Validate`.
        7.  **Steering:** Calls `Engine.EvaluateSteering`.

**Config (`config`)**
*   **Key Structs:** `Config`, `PolicyConfig`, `ObservabilityConfig`.
*   **Responsibilities:**
    *   Loads configuration from YAML files with environment variable expansion.
    *   Defines the schema for system configuration.

**CLI (`cmd/mkit`)**
*   **Key Commands:** `mkit gen rule` (Rule Generation), `mkit inspect`.
*   **Responsibilities:**
    *   Provides developer tooling.
    *   Implements Neuro-Symbolic Feedback Loop for rule generation (`cmd/mkit/commands/gen/logic.go`).

#### 4. CRITICAL PATH & DATA (The Flow)

**Execution Flow (`ExecuteByName`)**
1.  **Entry:** User calls `client.ExecuteByName(ctx, "action", payload)`.
2.  **Loop Start:** `sdk.runLoopInternal` initializes execution parameters (Memory, History).
3.  **Step Execution (`ExecuteSingleStep`):**
    *   Resolves `core.Action` from registry (typically a `SupervisedAction`).
    *   Creates `core.Envelope` and injects Metadata (Feedback, History, Facts).
    *   **Supervisor (`SupervisedAction.Execute`):**
        *   **Trace:** Auto-starts span using global `manglekit` tracer.
        *   **AuthZ:** `Engine.Authorize` checks `deny(Req)`.
        *   **Config:** `Engine.GetActionConfig` injects dynamic parameters (e.g. system prompt).
        *   **Context:** Propagates `parent_id` for lineage.
        *   **Run:** Inner Action executes (e.g., calls LLM).
        *   **Taint:** Propagates labels from input to output.
        *   **Validate:** `Engine.Validate` checks `deny(Output)`.
        *   **Steering:** `Engine.EvaluateSteering` checks `retry(Hint)` or `route(Target)`.
4.  **Decision Handling:**
    *   **RETRY:** If Steering returns `RETRY` (or `AlignmentError` occurs), Loop increments retry count, sleeps (backoff), and retries with feedback.
    *   **ROUTE:** If Steering returns `ROUTE`, Loop switches current action to `next_step` and continues.
    *   **ALLOW:** Loop returns result to user.
    *   **DENY:** Loop returns error.

**Data Structures**
*   **Envelope (`core.Envelope`):** The standard container.
    *   `ID` (UUID), `Payload` (Any), `Metadata` (Map), `SecurityLabels` (Slice), `Facts` (Slice).
*   **Facts (Datalog):**
    *   `value(ID, Field, Value)`: Reflection of payload.
    *   `has_label(ID, Label)`: Taint tags.
    *   `action_operation(ID, OpName)`: Context.
    *   `retry(Hint)`, `route(Target)`: Steering signals.

#### 5. SOURCE CODE DUMP

## core/types.go
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
	KeyDecision     = "manglekit.decision"   // Values: "ALLOW", "DENY", "RETRY", "ROUTE"
	KeyFeedback     = "manglekit.feedback"   // Human/LLM readable reason
	KeyPrevFeedback = "prev_feedback"        // Loopback for retry
	KeyNextStep     = "manglekit.next_step"  // Next action routing

	// Risk & Analysis
	KeyRiskScore = "manglekit.risk_score" // 0-100

	// Performance & Observability
	KeyLatencyMs = "manglekit.latency_ms"
	KeyTraceID   = "manglekit.trace_id"
	KeyModel     = "manglekit.model"
	KeyHistory   = "manglekit_history"

	// Configuration
	PrefixPromptConfig = "prompt."
)

// Standard Decision Values
const (
	DecisionAllow = "ALLOW"
	DecisionDeny  = "DENY"
	DecisionRetry = "RETRY"
	DecisionRoute = "ROUTE"
)

// Datalog System Constants
const (
	EntityInput  = "Req"    // ID for Input Envelope
	EntityOutput = "Output" // ID for Output Envelope
)

// Observability & Trace Attributes
const (
	// Span Names
	SpanPreCheck  = "Datalog.PreCheck"
	SpanPostCheck = "Datalog.PostCheck"

	// Attribute Keys
	AttrPolicyName   = "policy.name"
	AttrPolicyType   = "policy.type"
	AttrDecisionType = "decision.type"
	AttrOutcome      = "outcome"       // "ALLOWED", "DENIED"
	AttrLabels       = "mangle.labels" // Taint Propagation
	AttrActionName   = "action.name"
	AttrActionType   = "action.type"
	AttrRuleID       = "mangle.rule_id" // Replaces AttrPolicyRuleID
	AttrAttempt      = "mangle.attempt" // Replaces AttrPolicyAttempt


)

// Outcome Values (for Tracing)
const (
	OutcomeAllowed = "ALLOWED"
	OutcomeDenied  = "DENIED"
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
	Outcome string            // Matches DecisionAllow, DecisionDeny, etc.
	Target  string            // Used if Outcome == DecisionRoute
	Reasons []string          // Explanations
	Meta    map[string]string // Side-channel data (risk scores, latency budget)
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

// Answer represents a structured system response.
type Answer struct {
	Text string         `json:"text"`
	Meta map[string]any `json:"meta,omitempty"`
}
```

## core/logic.go
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
type GenerateOption func(o any)

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

## core/governance.go
```go
package core

import "context"

// Evaluator: The Mangle Logic Engine.
// It defines the contract for policy execution, validation, and steering.
type Evaluator interface {
	// Assess evaluates the policy for a given input (General purpose).
	Assess(ctx context.Context, input Envelope) (Decision, error)

	// Authorize performs the Pre-Check phase (input validation).
	Authorize(ctx context.Context, actionMeta ActionMetadata, input Envelope) error

	// Validate performs the Post-Check phase (output validation).
	Validate(ctx context.Context, actionMeta ActionMetadata, output Envelope) (Envelope, error)

	// EvaluateSteering determines the next step (Retry/Route) based on the output.
	EvaluateSteering(ctx context.Context, input Envelope) (string, map[string]string, error)

	// GetActionConfig queries the engine for dynamic configuration parameters.
	GetActionConfig(ctx context.Context, input Envelope) (map[string]string, error)

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

## sdk/client.go
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
	c.engine = engine.NewWithObservability(c.tracer, c.logger)

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
// This is useful when configuration is deserialized from a file or external source.
//
// Parameters:
//   - ctx: The context.
//   - cfg: The loaded configuration struct.
//   - opts: Additional functional options (override config settings).
//
// Returns:
//   - A pointer to the Client, or an error.
func NewClientWithConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error) {
	// Initialize logger (use default for now)
	log := logger.NewDefault()

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
	c.engine = engine.NewWithObservability(c.tracer, c.logger)

	// Load policy from the configured path
	if cfg != nil && cfg.Policy.Path != "" {
		content, err := os.ReadFile(cfg.Policy.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read policy file %q: %w", cfg.Policy.Path, err)
		}
		if err := c.engine.LoadPolicy(ctx, string(content)); err != nil {
			return nil, fmt.Errorf("failed to load policy from %q: %w", cfg.Policy.Path, err)
		}
	}

	// Load knowledge from the configured path
	if cfg != nil && cfg.Knowledge.Path != "" {
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

	// Set failure mode from config
	if cfg != nil && cfg.FailureMode != "" {
		c.failureMode = cfg.FailureMode
	}

	// Log configuration loaded successfully
	if cfg != nil {
		c.logger.Info("Manglekit client initialized with config",
			"service_name", cfg.Observability.ServiceName,
			"observability_enabled", cfg.Observability.Enabled,
			"failure_mode", c.failureMode)
	}

	return c, nil
}

// NewClientFromConfig initializes a Client by loading configuration from a YAML file.
// It supports environment variable expansion in the config file.
//
// Parameters:
//   - ctx: The context.
//   - configPath: Path to the YAML configuration file.
//   - opts: Additional functional options.
//
// Returns:
//   - A pointer to the Client, or an error.
func NewClientFromConfig(ctx context.Context, configPath string, opts ...ClientOption) (*Client, error) {
	// Load configuration from file
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	c, err := NewClientWithConfig(ctx, cfg, opts...)
	if err != nil {
		return nil, err
	}

	// 1. Load Actions from Config
	for name, actCfg := range cfg.Actions {
		action, err := NewActionFromConfig(ctx, name, actCfg)
		if err != nil {
			// Log warning but don't fail entire client load? Or fail?
			// Config usually implies "desired state", so failure is critical.
			c.logger.Warn("failed to load action from config", "name", name, "error", err)
			continue
		}
		// Register it (Metadata Name might need update if factory didn't set it)
		c.RegisterAction(name, action)
	}

	// Load MCP Actions
	if len(cfg.MCP) > 0 {
		for _, mcpCfg := range cfg.MCP {
			loader := mcpAdapter.NewLoader(mcpCfg).WithLogger(c.logger)
			actions, err := loader.Load(ctx)
			if err != nil {
				// Because loader.Load now handles Soft Failure internally (returning nil error),
				// any error returned here implies FailOnStartup=true or a critical loader error.
				return nil, fmt.Errorf("critical tool '%s' failed to load: %w", mcpCfg.Name, err)
			}

			for _, action := range actions {
				// Supervise It
				safeAction := c.Supervise(action)
				// Register It
				c.RegisterAction(safeAction.Metadata().Name, safeAction)
				c.logger.Info("Discovered MCP Tool", "name", safeAction.Metadata().Name)
			}
		}
	}

	return c, nil
}

// Supervise wraps a raw core.Action in a SupervisedAction.
// This applies the "Trace -> Authorize -> Execute -> Validate" governance lifecycle.
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
```

## sdk/loop.go
```go
package sdk

import (
	"context"
	"encoding/json"
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
		env.Metadata["manglekit.feedback"] = params.LastFeedback
	}

	// 1.2 Inject Chat History
	if len(params.CurrentHistory) > 0 && params.MemoryMode != core.MemoryModeNone {
		env.SetHistory(params.CurrentHistory)
	}

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

	case core.DecisionAllow, "":
		return result, nil

	case core.DecisionDeny:
		reason := result.Metadata["reason"]
		if reason == "" {
			reason = result.Metadata["violation_msg"]
		}
		if reason == "" {
			reason = "blueprint violation"
		}
		return core.Envelope{}, fmt.Errorf("action denied by blueprint: %s", reason)
	}

	// Should not reach here for standard decisions
	return result, nil
}

// Helper for better logging
func safelyStringify(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	// Try JSON first for structured data
	b, err := json.Marshal(v)
	if err == nil {
		return string(b)
	}
	// Fallback
	return fmt.Sprintf("%v", v)
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
```

## internal/engine/solver.go
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
func New() *PolicyEngine {
	pe := &PolicyEngine{
		tracer:  &core.NopTracer{},
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}

	// Auto-load Standard Library
	if err := pe.runtime.AddPolicy(resources.GetStdLib()); err != nil {
		panic("manglekit: failed to load std.dl: " + err.Error())
	}

	return pe
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
func NewWithObservability(tracer core.Tracer, logger core.Logger) *PolicyEngine {
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
	if err := pe.runtime.AddPolicy(resources.GetStdLib()); err != nil {
		if logger != nil {
			logger.Error("failed to load standard library", "error", err)
		}
		// Failure to load stdlib is critical for dynamic features
		panic("manglekit: failed to load std.dl: " + err.Error())
	}

	return pe
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

// Assess implements the core.Evaluator interface.
// It performs a high-level assessment of the input, mapping Authorize logic to a Decision.
func (e *PolicyEngine) Assess(ctx context.Context, input core.Envelope) (core.Decision, error) {
	// Simple mapping: use empty metadata for generic assessment
	err := e.Authorize(ctx, core.ActionMetadata{}, input)
	if err != nil {
		// If authorization fails, it's a DENY
		var alignErr *core.AlignmentError
		if errors.As(err, &alignErr) {
			return core.Decision{
				Outcome: core.DecisionDeny,
				Reasons: []string{alignErr.Message},
				Meta:    map[string]string{"rule_id": alignErr.RuleID},
			}, nil
		}
		return core.Decision{Outcome: core.DecisionDeny, Reasons: []string{err.Error()}}, err
	}
	return core.Decision{Outcome: core.DecisionAllow}, nil
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

// Authorize performs the Pre-Check phase of governance.
// It checks if the input is allowed to proceed based on the loaded policies.
// If the `deny(Req)` predicate is derived, it returns `core.ErrAlignment`.
//
// It automatically starts a tracing span (`Datalog.PreCheck`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being authorized.
//   - input: The input envelope containing the payload and security labels.
//
// Returns:
//   - core.ErrAlignment if blocked, or nil if allowed.
func (e *PolicyEngine) Authorize(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.tracer == nil {
		return e.authorizeInternal(ctx, actionMeta, input)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPreCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(input.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: input.SecurityLabels})
	}

	err := e.authorizeInternal(ctx, actionMeta, input)
	if err != nil {
		span.RecordError(err)
	} else {
		span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeAllowed})
	}
	return err
}

// authorizeInternal executes the core authorization logic.
func (e *PolicyEngine) authorizeInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
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

	return e.evaluateGate(ctx, actionMeta.Name, core.EntityInput, input, `deny("%s")`, extraFacts...)
}

// Validate performs the Post-Check phase of governance.
// It checks if the output is allowed to be returned to the caller.
// If the `deny(Output)` predicate is derived, it returns `core.ErrAlignment`.
//
// It automatically starts a tracing span (`Datalog.PostCheck`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being validated.
//   - output: The output envelope containing the result.
//
// Returns:
//   - The validated envelope (potentially modified, though currently pass-through), or an error.
func (e *PolicyEngine) Validate(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.tracer == nil {
		return e.validateInternal(ctx, actionMeta, output)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPostCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(output.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: output.SecurityLabels})
	}

	result, err := e.validateInternal(ctx, actionMeta, output)
	if err != nil {
		span.RecordError(err)
		return core.Envelope{}, err
	}
	span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeAllowed})
	return result, nil
}

// validateInternal executes the core validation logic.
func (e *PolicyEngine) validateInternal(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	err := e.evaluateGate(ctx, actionMeta.Name, core.EntityOutput, output, `deny("%s")`)
	if err != nil {
		return core.Envelope{}, err
	}
	return output, nil
}

// evaluateGate centralizes the logic for "Check -> Deny -> Explain".
// It is used by both Authorize (Pre-Check) and Validate (Post-Check).
func (e *PolicyEngine) evaluateGate(ctx context.Context, actionName string, entityID string, env core.Envelope, queryTemplate string, extraFacts ...ast.Atom) error {
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

	// 6. Run Query (deny(EntityID))
	denied, err := e.runtime.ExecuteQuery(facts, fmt.Sprintf(queryTemplate, entityID))
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("policy evaluation failed", "error", err)
		}
		return fmt.Errorf("policy evaluation error: %w", err)
	}

	if denied {
		// Teacher-Student Protocol: Extract violation message and rule ID
		var violationMsg, ruleID string

		// Query: violation_msg(Msg)
		_ = e.runtime.QueryWithSolutions(facts, "violation_msg(Msg)", func(solution map[string]any) error {
			if msg, ok := solution["Msg"].(string); ok {
				violationMsg = msg
				return fmt.Errorf("found") // Stop searching
			}
			return nil
		})

		// Query: violation_rule(ID)
		_ = e.runtime.QueryWithSolutions(facts, "violation_rule(ID)", func(solution map[string]any) error {
			if id, ok := solution["ID"].(string); ok {
				ruleID = id
				return fmt.Errorf("found") // Stop searching
			}
			return nil
		})

		if e.logger != nil {
			e.logger.Debug("gate violation detected", "action", actionName, "msg", violationMsg, "rule_id", ruleID)
		}
		return &core.AlignmentError{Message: violationMsg, RuleID: ruleID}
	}

	return nil
}

// EvaluateSteering executes "Steering Policies" which determine what to do next.
// Unlike Authorize/Validate (which are binary Allow/Deny), Steering returns decisions like "Retry" or "Route".
//
// Logic Priority:
//  1. Correction (Retry): If `correction(Req, Hint)` is derived, we return `RETRY` with the hint.
//  2. Routing (Route): If `next_step(Req, Target)` is derived, we return `ROUTE` with the target.
//  3. Default: `ALLOW` (Proceed as normal).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input envelope.
//
// Returns:
//   - decision: The decision string (e.g., "RETRY", "ROUTE", "ALLOW").
//   - metadata: A map containing steering details (e.g., {"feedback": "hint"}).
//   - error: An error if evaluation fails.
func (e *PolicyEngine) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	decision := core.DecisionAllow
	metadata := make(map[string]string)

	if e.runtime == nil || e.runtime.programInfo == nil {
		return decision, metadata, nil
	}

	// Convert the input payload to Mangle facts
	// We use "Req" as the entity ID, consistent with Authorize
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
	_ = e.runtime.QueryWithSolutions(facts, "retry(Hint)", func(solution map[string]any) error {
		if hint, ok := solution["Hint"].(string); ok {
			decision = core.DecisionRetry
			metadata[core.KeyFeedback] = hint
			// Stop searching after first match
			return fmt.Errorf("found") // Use error to break early
		}
		return nil
	})

	if decision == core.DecisionRetry {
		return decision, metadata, nil
	}

	// 2. Check Routing
	// Query: route(Target)
	_ = e.runtime.QueryWithSolutions(facts, "route(Target)", func(solution map[string]any) error {
		if target, ok := solution["Target"].(string); ok {
			decision = core.DecisionRoute
			metadata[core.KeyNextStep] = target
			return fmt.Errorf("found") // Use error to break early
		}
		return nil
	})

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

## internal/supervisor/supervisor.go
```go
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/duynguyendang/manglekit/core"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SupervisedAction is a decorator that wraps any `core.Action` to enforce governance blueprints.
// It implements the standard "Trace -> Authorize -> Execute -> Validate" lifecycle.
//
// Lifecycle:
//  1. Trace: Starts an OpenTelemetry span for the operation.
//  2. Authorize: Checks Pre-Check blueprints (e.g., "deny(Req)").
//  3. Execute: Runs the inner action (e.g., calls the LLM).
//  4. Validate: Checks Post-Check blueprints (e.g., "deny(Output)").
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
//  3. Runs Authorize(). If it fails, execution halts (unless Fail-Open).
//  4. Runs the inner Action.Execute().
//  5. Propagates taint labels from input to output.
//  6. Runs Validate(). If it fails, the result is blocked.
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
	// We use the global OTel tracer "manglekit" to create spans automatically.
	// This supersedes the legacy g.tracer usage for the main span,
	// ensuring consistent observability without user configuration.
	tracer := otel.Tracer("manglekit")
	meta := g.inner.Metadata()

	ctx, span := tracer.Start(ctx, fmt.Sprintf("Action.%s", meta.Name),
		trace.WithAttributes(
			attribute.String("mangle.action_name", meta.Name),
			attribute.String("mangle.action_type", string(meta.Type)),
			attribute.String("mangle.input_id", input.ID.String()),
		),
	)
	defer span.End()

	result, err := g.executeInternal(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Distinguish between Blueprint DENIAL and System ERROR
		if g.isAlignmentIssue(err) {
			span.SetAttributes(attribute.String(core.AttrOutcome, "deny"))
			var alignErr *core.AlignmentError
			if errors.As(err, &alignErr) {
				span.SetAttributes(attribute.String(core.KeyFeedback, alignErr.Message))
				if alignErr.RuleID != "" {
					span.SetAttributes(attribute.String(core.AttrRuleID, alignErr.RuleID))
				}
			} else {
				span.SetAttributes(attribute.String(core.KeyFeedback, err.Error()))
			}
			// Legacy attribute for backward compatibility
			span.SetAttributes(attribute.String(core.AttrOutcome, "DENIED"))
		} else {
			span.SetAttributes(attribute.String(core.AttrOutcome, "ERROR"))
		}
		return core.Envelope{}, err
	}

	// Success Path: Determine outcome (Allow/Route/Retry)
	decision := result.Metadata[core.KeyDecision]
	switch decision {
	case core.DecisionRetry:
		span.SetAttributes(attribute.String(core.AttrOutcome, "retry"))
		if hint, ok := result.Metadata[core.KeyFeedback]; ok {
			if s, ok := hint.(string); ok {
				span.SetAttributes(attribute.String(core.KeyFeedback, s))
			}
		}
	case core.DecisionRoute:
		span.SetAttributes(attribute.String(core.AttrOutcome, "route"))
		if target, ok := result.Metadata[core.KeyNextStep]; ok {
			if s, ok := target.(string); ok {
				span.SetAttributes(attribute.String(core.AttrActionName, s))
			}
		}
	default:
		span.SetAttributes(attribute.String(core.AttrOutcome, "allow"))
	}

	// Inject Retry Count if present
	if attemptVal, ok := input.Metadata["retry_count"]; ok {
		// handle both string and int
		if s, ok := attemptVal.(string); ok {
			if n, err := strconv.Atoi(s); err == nil {
				span.SetAttributes(attribute.Int(core.AttrAttempt, n))
			}
		} else if n, ok := attemptVal.(int); ok {
			span.SetAttributes(attribute.Int(core.AttrAttempt, n))
		}
	}

	span.SetAttributes(
		attribute.String(core.AttrOutcome, "ALLOWED"),
		attribute.String("mangle.output_id", result.ID.String()),
	)
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

	// 2. Pre-Check: Authorization
	if err := g.engine.Authorize(ctx, g.inner.Metadata(), input); err != nil {
		if g.shouldBlock(err) {
			logger.Warn("authorization failed",
				core.AttrActionName, meta.Name,
				"error", err.Error(),
			)
			return core.Envelope{}, fmt.Errorf("authorization failed: %w", err)
		}
		// Fail-Open
		logger.Warn("engine failed but Fail-Open active. Proceeding.", "error", err)
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

	// 7. Post-Check: Validation
	validatedResult, err := g.engine.Validate(ctx, g.inner.Metadata(), result)
	if err != nil {
		if g.shouldBlock(err) {
			logger.Warn("validation failed",
				"action", meta.Name,
				"error", err.Error(),
			)
			return core.Envelope{}, fmt.Errorf("validation failed: %w", err)
		}
		// Fail-Open: use result as validatedResult
		logger.Warn("engine validation failed but Fail-Open active. Proceeding.", "error", err)
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

## adapters/ai/genkit.go
```go
package ai

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// genkitAdapter adapts the Firebase Genkit ai.Model interface to the Manglekit core.TextGenerator interface.
type genkitAdapter struct {
	model ai.Model
	gk    *genkit.Genkit
}

// NewGenkitAdapter creates a new adapter from a pre-initialized Genkit model.
//
// Parameters:
//   - model: The Genkit model instance.
//   - gk: The Genkit runtime instance.
//
// Returns:
//   - A core.TextGenerator implementation.
func NewGenkitAdapter(model ai.Model, gk *genkit.Genkit) core.TextGenerator {
	return &genkitAdapter{
		model: model,
		gk:    gk,
	}
}

// Complete generates text using the underlying Genkit model.
func (g *genkitAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	resp, err := g.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// Generate implements the core.TextGenerator interface using Genkit.
func (g *genkitAdapter) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	var messages []*ai.Message

	// Dynamic Prompt Configuration
	facts := core.ContextFacts(ctx)
	systemPrompt := ""
	if facts != nil {
		if val, ok := facts[core.PrefixPromptConfig+"tone"]; ok {
			systemPrompt += "\n[INSTRUCTION]: Maintain a " + val + " tone."
		}
		if val, ok := facts[core.PrefixPromptConfig+"strategy"]; ok && val == "cot" {
			systemPrompt += "\n[STRATEGY]: Think step-by-step."
		}
	}

	if systemPrompt != "" {
		messages = append(messages, &ai.Message{
			Role:    ai.RoleSystem,
			Content: []*ai.Part{ai.NewTextPart(systemPrompt)},
		})
	}

	messages = append(messages, &ai.Message{
		Role:    ai.RoleUser,
		Content: []*ai.Part{ai.NewTextPart(prompt)},
	})

	req := &ai.ModelRequest{
		Messages: messages,
		// Output describes the desired response format.
		Output: &ai.ModelOutputConfig{
			Format: ai.OutputFormatJSON,
		},
	}

	resp, err := g.model.Generate(ctx, req, nil)
	if err != nil {
		return nil, err
	}

	// Extract token usage if available
	usage := make(map[string]int)
	if resp.Usage != nil {
		usage["prompt"] = int(resp.Usage.InputTokens)
		usage["completion"] = int(resp.Usage.OutputTokens)
		usage["total"] = int(resp.Usage.TotalTokens)
	}

	return &core.LLMResponse{
		Text:  resp.Text(),
		Usage: usage,
	}, nil
}

// Stream implements the core.TextGenerator interface.
// Currently returns error as streaming is not fully adapted here.
func (g *genkitAdapter) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	// Simple non-streaming fallback or error
	return nil, fmt.Errorf("streaming not implemented in genkit adapter yet")
}
```

## sdk/config_loader.go
```go
package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
)

// HydrateActions iterates through the configuration and instantiates the defined actions.
// It acts as a high-level factory for converting config maps into executable core.Actions.
func HydrateActions(ctx context.Context, actions map[string]config.ActionConfig) ([]core.Action, error) {
	var hydrated []core.Action

	for name, cfg := range actions {
		action, err := NewActionFromConfig(ctx, name, cfg)
		if err != nil {
			// We log/return error. Returning error stops the boot, which is usually safer for config correctness.
			return nil, fmt.Errorf("failed to hydrate action %q: %w", name, err)
		}
		hydrated = append(hydrated, action)
	}
	return hydrated, nil
}

// NewActionFromConfig creates a new Action instance based on the provided configuration.
func NewActionFromConfig(ctx context.Context, name, cfg config.ActionConfig) (core.Action, error) {
	switch cfg.Type {
	case "llm":
		return createLLMAction(ctx, name, cfg)
	default:
		return nil, fmt.Errorf("unsupported action type: %s", cfg.Type)
	}
}

func createLLMAction(ctx context.Context, name string, cfg config.ActionConfig) (core.Action, error) {
	// 1. Try to find a registered provider
	factory, err := GetProvider(cfg.Provider)
	if err == nil {
		return factory(ctx, name, cfg)
	}

	// 2. Fallback for Mock (Built-in)
	if cfg.Provider == "mock" {
		return createMockLLMAction(name, cfg)
	}

	// 3. Error if not found
	return nil, fmt.Errorf("failed to create action '%s': %w (Did you forget to call sdk.RegisterProvider in main?)", name, err)
}

func createMockLLMAction(name string, cfg config.ActionConfig) (core.Action, error) {
	prompt := ""
	if p, ok := cfg.Options["prompt"].(string); ok {
		prompt = p
	}

	gen := &mockGenerator{
		systemPrompt: prompt,
	}

	return ai.NewLLMAction(name, gen)
}

// mockGenerator implements core.TextGenerator for testing/fallback.
type mockGenerator struct {
	systemPrompt string
}

func (m *mockGenerator) Complete(ctx context.Context, prompt string) (string, error) {
	return fmt.Sprintf("%s %s", m.systemPrompt, prompt), nil
}

func (m *mockGenerator) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	// Basic echo behavior
	resp := fmt.Sprintf("%s %s", m.systemPrompt, prompt)
	return &core.LLMResponse{
		Text: resp,
		Usage: map[string]int{
			"input":  len(prompt),
			"output": len(resp),
		},
	}, nil
}

func (m *mockGenerator) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		ch <- fmt.Sprintf("%s %s", m.systemPrompt, prompt)
	}()
	return ch, nil
}
```

## sdk/client_execute.go
```go
package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// Execute processes an envelope by determining the initial action via the policy engine.
// It relies on the 'next_step' predicate to route the request.
func (c *Client) Execute(ctx context.Context, input core.Envelope, opts ...ExecuteOption) (core.Envelope, error) {
	// 1. Evaluate Steering to find the entry point
	decision, meta, err := c.engine.EvaluateSteering(ctx, input)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("failed to evaluate entry route: %w", err)
	}

	if decision == core.DecisionRoute {
		actionName := meta[core.KeyNextStep]
		if actionName == "" {
			return core.Envelope{}, fmt.Errorf("routing decision returned empty action name")
		}
		// Forward payload and metadata
		return c.ExecuteByName(ctx, actionName, input.Payload, opts...)
	}

	return core.Envelope{}, fmt.Errorf("no execution route defined for this input (decision=%s, meta=%v)", decision, meta)
}
```

#### 6. CHANGELOG

*   **2025-12-15**: Multi-Provider Architecture (FOP).
    *   Refactored `providers/google` to use Functional Options Pattern.
    *   Added `providers/openrouter` for OpenAI-compatible APIs (OpenRouter, Groq).
    *   Verified standardized `Register/New` pattern across providers.
*   **2025-12-15**: Full Repository Exhaustive Scan.
    *   Updated File Map to reflect current directory structure (removed `core/action.go`, `core/constants.go`, added `core/logic.go`, `core/data.go`, `core/governance.go`, `internal/resources`, etc.).
    *   Updated Component Analysis to include `internal/resources` and CLI tools.
    *   Updated Source Code Dump with critical files: `core/types.go`, `core/logic.go`, `core/governance.go`, `sdk/client.go`, `sdk/loop.go`, `internal/engine/solver.go`, `internal/supervisor/supervisor.go`, `internal/engine/reflection.go`.
*   **2025-12-15**: Standard Providers & Registry Pattern.
    *   Created `providers/google/gemini.go` implementing standard Google GenAI registration.
    *   Refactored `examples/config_driven_bot/main.go` to use `google.Register()` and `sdk.NewClientFromFile`.
*   **2025-12-14**: Low-Code Gateway Implementation.
    *   Implemented `sdk/config_loader.go` for `HydrateActions`.
    *   Updated `sdk/client.go` to support `NewClientFromConfig` (Config Struct) and `NewClientFromFile` (Path).
    *   Added `sdk/client_execute.go` for Policy-Driven Execution (`Execute`).
    *   Added `examples/config_driven_bot` demonstrating YAML+Datalog bot configuration.
    *   Aligned `internal/logger` with Config LogLevel.
*   **2025-11-29**: Added Rich Telemetry.
