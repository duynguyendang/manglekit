---
context_type: kernel_source_dump
project: manglekit
language: go, datalog
last_updated: 2025-12-19
scan_mode: logic_focused
---

## 1. THE COMPLETE FILE MAP

.
├── adapters
│   ├── ai                  # Genkit Wrapper (LLM Integration)
│   ├── extractor           # Structured Data Extraction
│   ├── func                # Go Function Wrapper
│   ├── knowledge           # Graph/RDF Loading
│   ├── logger              # Logger Adapters (Zap, etc.)
│   ├── mcp                 # Model Context Protocol (Tool Discovery)
│   ├── resilience          # Circuit Breaker & Retry
│   └── vector              # Vector Store & RAG
├── cmd
│   └── mkit                # CLI Tool (gen, inspect, eval, serve)
├── config                  # Configuration Loading (YAML)
├── core                    # The Contracts (Interfaces & Types)
├── internal
│   ├── engine              # The Neuro-Symbolic Core (Mangle Runtime)
│   │   ├── memory          # Volatile Memory Store
│   │   └── resources       # Embedded Datalog Rules (std.dl)
│   ├── logger              # Default Logger Implementation
│   ├── resources           # Shared Resources (ICL Golden Samples)
│   ├── statehelper         # Conversation State Management
│   ├── supervisor          # The Action Sandwich (Trace-Assess-Exec-Reflect)
│   ├── telemetry           # OpenTelemetry Setup
│   ├── testproviders       # Mocks for Testing
│   ├── tools               # Internal Tooling
│   └── util                # Utilities (Schema, etc.)
├── providers               # Plugin Factories
│   ├── google
│   ├── memory
│   └── openai
└── sdk                     # The Orchestration Layer (Client, Loop, Planner)

#### 3. COMPONENTS (The Logic)

Component: sdk
1. Responsibilities:
   - Orchestrates the execution lifecycle (Client).
   - Manages the "Semantic State Machine" (RunLoop).
   - Generates and executes plans (Planner).
   - Connects Core interfaces to internal implementations.
2. Core Structs:
   - **Client**: The main entry point. Holds references to Engine, Supervisor, Memory, and Action Registry.
   - **ExecutionParams**: Context object for the RunLoop, tracking retries and feedback.
   - **PlanStep**: Represents a step in a generated plan.
3. Key Functions:
   - func NewClient(opts ...Option) *Client - [Initializes the SDK client with configured options].
   - func (c *Client) ExecuteByName(ctx context.Context, actionName string, payload any, opts ...ExecuteOption) (core.Envelope, error) - [Executes a named action within the governance loop].
   - func (c *Client) Plan(ctx context.Context, goalName string) ([]PlanStep, error) - [Generates a plan to achieve a goal using Datalog reasoning].

Component: internal/engine
1. Responsibilities:
   - Encapsulates the Mangle Datalog runtime.
   - Evaluates policies (Assess, Reflect, Steering).
   - Converts Go structs and JSON to Datalog facts (Reflection/Flattener).
2. Core Structs:
   - **PolicyEngine**: The main logic engine. Wraps MangleRuntime.
   - **MangleRuntime**: Low-level wrapper around `google/mangle`.
   - **Evaluator**: Ad-hoc rule evaluator.
3. Key Functions:
   - func (e *PolicyEngine) Assess(ctx, meta, input) error - [Checks Pre-Check policies].
   - func (e *PolicyEngine) Reflect(ctx, meta, output) (Envelope, error) - [Checks Post-Check policies].
   - func (e *PolicyEngine) EvaluateSteering(ctx, input) (string, map, error) - [Determines next step (Retry/Route)].
   - func ToFacts(entityID string, val any) ([]string, error) - [Reflects a struct into Datalog facts].

Component: internal/supervisor
1. Responsibilities:
   - Implements the "Guarded Action" pattern.
   - Enforces the "Trace -> Assess -> Execute -> Reflect" lifecycle.
2. Core Structs:
   - **SupervisedAction**: Decorator that wraps `core.Action`.
3. Key Functions:
   - func (g *SupervisedAction) Execute(ctx, input) (Envelope, error) - [Executes the wrapped action with full governance].

Component: core
1. Responsibilities:
   - Defines the system's contract interfaces.
   - Pure abstract layer with no dependencies.
2. Core Structs:
   - **Action**: Interface for executable units.
   - **Envelope**: Standard data carrier.
   - **Evaluator**: Interface for the Policy Engine.
   - **AgentMemory**: Interface for RAG and History.
3. Key Functions:
   - type Action interface { Execute(ctx, Envelope) (Envelope, error) }

Component: adapters/ai
1. Responsibilities:
   - Wraps Genkit AI models as `core.Action`.
   - Handles LLM generation and streaming.
2. Core Structs:
   - **LLMAction**: Implements `core.Action` for Genkit models.
   - **genkitAdapter**: Wrapper for `genkit.Genkit`.
3. Key Functions:
   - func NewLLMAction(name string, generator core.TextGenerator) *LLMAction - [Creates a new AI action].

Component: adapters/mcp
1. Responsibilities:
   - Connects to Model Context Protocol servers.
   - Discovers tools and exposes them as `core.Action`.
2. Core Structs:
   - **Loader**: Handles MCP connection and tool discovery.
   - **MCPAction**: Wraps an MCP tool.
3. Key Functions:
   - func (l *Loader) Load(ctx) ([]Action, error) - [Connects to MCP and returns actions].

Component: adapters/extractor
1. Responsibilities:
   - Uses an LLM to extract structured data from text.
   - Validates output against a generated JSON Schema.
2. Core Structs:
   - **ExtractorAction**: Wraps an LLM Action for extraction.
3. Key Functions:
   - func New(name string, generator core.Action, schema any) (*ExtractorAction, error) - [Creates an extractor for a target struct].
   - func (e *ExtractorAction) Execute(ctx, input) (Envelope, error) - [Extracts data and returns a struct envelope].

Component: adapters/func
1. Responsibilities:
   - Adapts any Go function (generic `func(In) (Out, error)`) into a `core.Action`.
   - Handles type assertion and envelope wrapping.
2. Core Structs:
   - **Wrapper[In, Out]**: Generic wrapper struct.
3. Key Functions:
   - func NewWrapper[In, Out](name string, fn ToolFunc[In, Out]) *Wrapper[In, Out] - [Wraps a Go function].
   - func (w *Wrapper[In, Out]) Execute(ctx, input) (Envelope, error) - [Invokes the function].

Component: adapters/knowledge
1. Responsibilities:
   - Loads Knowledge Graphs (RDF/Turtle/N-Quads) into Datalog facts.
   - Parses external graph files.
2. Core Structs:
   - **RDFLoader**: Handles file parsing using `knakk/rdf`.
3. Key Functions:
   - func (l *RDFLoader) Parse(path string) ([]string, error) - [Parses RDF file to 'triple(S,P,O)' facts].

Component: adapters/vector
1. Responsibilities:
   - Abstracts Vector DB operations (Search/Upsert).
   - Provides RAG capabilities via `RetrieverAction`.
2. Core Structs:
   - **RetrieverAction**: Wraps a `DocumentRetriever`.
   - **Document**: Standard snippet format.
3. Key Functions:
   - func NewRetrieverAction(name string, retriever DocumentRetriever) *RetrieverAction - [Creates a retrieval action].
   - func (r *RetrieverAction) Execute(ctx, input) (Envelope, error) - [Performs search and returns JSON doc list].

Component: providers/google
1. Responsibilities:
   - Initializes Google GenAI (Gemini) plugin.
   - Implements a Proxy Pattern to handle configuration issues in the official plugin.
2. Core Structs:
   - N/A (Functional initialization).
3. Key Functions:
   - func Init(ctx, globalG, apiKey, modelName) (string, error) - [Registers Google model and returns global name].

#### 4. CRITICAL PATH & DATA (The Flow)

**1. Execution Sequence (High Level)**
```mermaid
sequenceDiagram
    participant User
    participant SDK as sdk.Client
    participant Loop as RunLoop
    participant Sup as Supervisor
    participant Eng as Engine (Datalog)
    participant Act as Adapter/Action

    User->>SDK: ExecuteByName("chat", payload)
    SDK->>Loop: runLoopInternal()
    loop Semantic Loop
        Loop->>Loop: Inject Context (Memory, Feedback)
        Loop->>Sup: Action.Execute(Envelope)

        rect rgb(240, 240, 240)
            note right of Sup: Governance Lifecycle
            Sup->>Eng: Assess(Input) [Pre-Check]
            Eng-->>Sup: OK / Block
            Sup->>Act: Execute(Input)
            Act-->>Sup: Result
            Sup->>Eng: Reflect(Result) [Post-Check]
            Eng-->>Sup: OK / Block
            Sup->>Eng: EvaluateSteering(Result)
            Eng-->>Sup: Decision (Retry/Route)
        end

        Sup-->>Loop: Envelope + Metadata

        alt Decision == RETRY
            Loop->>Loop: Increment Retry Count
            Loop->>Loop: Inject Feedback
            note right of Loop: Backoff & Repeat
        else Decision == ROUTE
            Loop->>Loop: Switch Action Target
        else Decision == PROCEED
            Loop-->>SDK: Result
        end
    end
    SDK-->>User: Final Result
```

**2. Data Transformation Flow**
```mermaid
flowchart LR
    Input[User Input (Struct/JSON)] --> Env[core.Envelope]
    Env --> Reflector{Reflection Engine}

    Reflector -- TypeStruct --> StructToFacts[internal/engine/reflection.go]
    Reflector -- TypeJSON --> Flattener[internal/engine/flattener.go]

    StructToFacts --> Facts[(Datalog Facts)]
    Flattener --> Facts

    Facts --> Solver[Mangle Runtime]
    Policies[std.dl / User Policies] --> Solver

    Solver --> QueryHalt{Halt/Deny?}
    QueryHalt -- Yes --> Block[AlignmentError]
    QueryHalt -- No --> Action[Execute Action]

    Action --> Result[Result Envelope]
    Result --> Solver
    Solver --> Steering{Steering?}
    Steering -- retry(Hint) --> Retry[Decision: RETRY]
    Steering -- route(Target) --> Route[Decision: ROUTE]
    Steering -- None --> Proceed[Decision: PROCEED]
```

#### 5. SOURCE CODE DUMP

---
## [internal/engine/resources/std.dl]
```prolog
% --- Manglekit Standard Library (v2.0) ---
% Auto-loaded on engine startup.

% ==========================================
% 1. DATA REFLECTION (DO NOT REMOVE)
% These predicates allow the engine to read JSON inputs and Graph data.
% ==========================================

% JSON Primitives (Flattened)
Decl json_str(Parent, Key, Val).
Decl json_num(Parent, Key, Val).
Decl json_bool(Parent, Key, Val).
Decl json_link(Parent, Key, Child).
Decl json_null(Parent, Key).

% Knowledge Graph Primitives (N-Quads/Triples)
Decl quad(S, P, O, G).
Decl triple(S, P, O).
triple(S, P, O) :- quad(S, P, O, _).

% ==========================================
% 2. SYSTEM CONTROL (Standard Vocabulary)
% ==========================================

% Arity 1: Global/Single-Context (JSON)
Decl deny(Reason).
Decl halt(Reason).
Decl route(NextStep).
Decl retry(Feedback).

% Arity 2: Entity-Specific (Graph/Batch)
% Allows pinning the decision to a specific node (Entity).
Decl deny(Entity, Reason).
Decl halt(Entity, Reason).
Decl route(Entity, NextStep).
Decl retry(Entity, Feedback).

% Config is always Key-Value (Arity 2) or Entity-Key-Value (Arity 3)
Decl config(Key, Value).
Decl config(Entity, Key, Value).

% Semantic Alias for backward compatibility
halt(Reason) :- deny(Reason).
halt(Entity, Reason) :- deny(Entity, Reason).

% ==========================================
% 3. CONTEXT INJECTION
% These predicates are injected by the runtime environment.
% ==========================================

Decl attempt(N).            % Current retry count (0, 1, 2...).
Decl meta(Key, Value).      % Envelope metadata (e.g., user_id, session_id).
Decl label(Tag).            % Security taint labels (e.g., "pii", "unsafe").

% Telemetry Predicate (Arity 2: Entity, RuleID)
Decl violation_rule(Entity, RuleID).

% --- Helper: String Matching ---

% input_contains(Req, Keyword)
% Checks if the input text payload contains the keyword (case-insensitive if supported).
% NOTE: fn:contains is not supported in Mangle v0.3.0, so this helper is currently disabled.
% input_contains(Req, Keyword) :-
%    value(Req, "text", Text),
%    fn:contains(Text, Keyword).
Decl goal(Name) .
Decl subgoal(Parent, Child, Order) .
Decl plan_step(Action, Order) .

plan_step(Action, Order) :- goal(G), subgoal(G, Action, Order).
```
---
## [core/types.go]
```go
package core

import (
	"errors"

	"github.com/google/uuid"
)

// Common Constants for Metadata Keys and OpenTelemetry Attributes.
const (
	// Metadata Keys (Vocabulary)
	KeyReason       = "reason"
	KeyDecision     = "decision"
	KeyFeedback     = "mangle_feedback"
	KeyPrevFeedback = "prev_feedback" // History of feedback
	KeyNextStep     = "next_step"
	KeyRiskScore    = "risk_score"
	KeyTraceID      = "trace_id"
	KeySessionID    = "session_id"
	KeyContext      = "context" // RAG Context
	KeyHistory      = "history" // Chat History

	// Decision Outcomes
	DecisionProceed = "PROCEED"
	DecisionHalt    = "HALT"
	DecisionRetry   = "RETRY"
	DecisionRoute   = "ROUTE"

	// OTel Attributes
	AttrActionName = "policy.action"
	AttrOutcome    = "policy.outcome"
	AttrRuleID     = "policy.rule_id"
	AttrReason     = "policy.reason"
	AttrLatency    = "policy.latency_ms"
	AttrLabels     = "policy.labels"
	AttrAttempt    = "policy.attempt"

	// Span Names
	SpanPreCheck  = "Datalog.Assess"
	SpanPostCheck = "Datalog.Reflect"
	SpanMemory    = "Memory.Recall"
	SpanLLM       = "LLM.Generate"

	// Standard Outcome Values
	OutcomeProceed = "allow"
	OutcomeHalt    = "deny"

	// Datalog Predicates (Canonical)
	PredHalt      = "halt"
	PredRetry     = "retry"
	PredRoute     = "route"
	PredViolation = "violation_msg"

	// Prefixes
	PrefixPromptConfig = "prompt_config_"
	EntityInput        = "Req"
	EntityOutput       = "Resp"
)

// ContentType defines how the payload should be interpreted by the engine.
type ContentType string

const (
	TypeStruct ContentType = "struct"
	TypeJSON   ContentType = "json"
	TypeText   ContentType = "text"
)

// Envelope is the standard unit of data exchange in Manglekit.
// It wraps the payload with metadata, identification, and security context.
type Envelope struct {
	ID             uuid.UUID      `json:"id"`
	Payload        any            `json:"data"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	SecurityLabels []string       `json:"security_labels,omitempty"`
	Facts          []string       `json:"facts,omitempty"` // Explicit Datalog facts
	ContentType    ContentType    `json:"content_type,omitempty"`
}

// NewEnvelope creates a new Envelope with a generated UUID.
func NewEnvelope(payload any) Envelope {
	return Envelope{
		ID:          uuid.New(),
		Payload:     payload,
		Metadata:    make(map[string]any),
		ContentType: TypeStruct, // Default
	}
}

// SetMeta sets a metadata key-value pair.
func (e *Envelope) SetMeta(key string, value any) {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
}

// GetFeedback retrieves the semantic feedback (if any) from metadata.
func (e *Envelope) GetFeedback() string {
	if v, ok := e.Metadata[KeyFeedback]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetFeedback injects feedback into the metadata.
func (e *Envelope) SetFeedback(msg string) {
	e.SetMeta(KeyFeedback, msg)
}

// SetHistory injects chat history into the payload (if it's a map) or metadata.
// For structured prompts, it prefers metadata injection for the template to use.
func (e *Envelope) SetHistory(history []Message) {
	e.SetMeta(KeyHistory, history)
}

// MergeLabels adds unique security labels to the envelope.
func (e *Envelope) MergeLabels(labels []string) {
	existing := make(map[string]bool)
	for _, l := range e.SecurityLabels {
		existing[l] = true
	}
	for _, l := range labels {
		if !existing[l] {
			e.SecurityLabels = append(e.SecurityLabels, l)
			existing[l] = true
		}
	}
}

// ActionMetadata describes a registered capability.
type ActionMetadata struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // e.g., "llm", "tool", "api"
	Description string `json:"description,omitempty"`
	InputType   string `json:"input_type,omitempty"`
	OutputType  string `json:"output_type,omitempty"`
}

// Decision represents the outcome of a policy evaluation.
type Decision struct {
	Outcome string            `json:"outcome"` // allow, deny, retry, route
	Reasons []string          `json:"reasons,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
```
---
## [core/logic.go]
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
```
---
## [core/governance.go]
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

	// Query executes a Datalog query and returns all matching solutions.
	// It is used by the planner to reason about goals and generate action sequences.
	Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error)

	// Logger returns the engine's logger.
	Logger() Logger
}
```

---
## [internal/supervisor/supervisor.go]
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

	// Set output ID without overwriting the outcome
	span.SetAttributes(map[string]any{
		"mangle.output_id": result.ID.String(),
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
	// Always block on Input/Validation errors (bypass fail-open)
	if core.IsInputError(err) {
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
// It orchestrates: Logger Setup → Pre-Check → Config → Execute → Post-Check → Steering.
func (g *SupervisedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Inject logger and get metadata
	ctx = core.ContextWithLogger(ctx, g.engine.Logger())
	logger := core.LoggerFromContext(ctx)
	meta := g.inner.Metadata()

	logger.Info("Action started", "action", meta.Name, "input_id", input.ID.String())

	// Phase 1: Pre-Check (Assessment)
	if err := g.performAssessment(ctx, logger, meta, input); err != nil {
		return core.Envelope{}, err
	}

	// Phase 2: Dynamic Configuration Injection
	g.injectDynamicConfig(ctx, logger, &input)

	// Phase 3: Execute Inner Action
	result, err := g.executeAction(ctx, logger, meta, input)
	if err != nil {
		return core.Envelope{}, err
	}

	// Phase 4: Post-Check (Reflection)
	validatedResult, err := g.performReflection(ctx, logger, meta, result)
	if err != nil {
		return core.Envelope{}, err
	}

	// Phase 5: Steering
	validatedResult = g.applySteering(ctx, logger, meta, validatedResult)

	logger.Info("Action completed", "action", meta.Name, "result", "success")
	return validatedResult, nil
}

// performAssessment executes the pre-check phase (blueprint validation of input).
func (g *SupervisedAction) performAssessment(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) error {
	if err := g.engine.Assess(ctx, meta, input); err != nil {
		if g.shouldBlock(err) {
			msg := "assessment failed"
			if core.IsInputError(err) {
				msg = "assessment blocked due to invalid input"
			}
			logger.Warn(msg, core.AttrActionName, meta.Name, "error", err.Error())
			return fmt.Errorf("%s: %w", msg, err)
		}
		// Fail-Open
		logger.Warn("engine assessment failed but Fail-Open active. Proceeding.", "error", err)
	}
	return nil
}

// injectDynamicConfig queries the engine for configuration overrides and injects them into input metadata.
func (g *SupervisedAction) injectDynamicConfig(ctx context.Context, logger core.Logger, input *core.Envelope) {
	config, err := g.engine.GetActionConfig(ctx, *input)
	if err != nil {
		logger.Warn("failed to retrieve action config", "error", err)
		return
	}

	if len(config) == 0 {
		return
	}

	if input.Metadata == nil {
		input.Metadata = make(map[string]any)
	}
	for k, v := range config {
		input.Metadata[core.PrefixPromptConfig+k] = v
	}
}

// executeAction runs the inner action with context propagation and security label inheritance.
func (g *SupervisedAction) executeAction(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) (core.Envelope, error) {
	// Context Propagation: set input ID as parent
	childCtx := core.WithParentID(ctx, input.ID.String())

	// Execute
	result, err := g.inner.Execute(childCtx, input)
	if err != nil {
		logger.Error("action execution failed", core.AttrActionName, meta.Name, "error", err.Error())
		return core.Envelope{}, fmt.Errorf("action execution failed: %w", err)
	}

	// Security label propagation
	if len(input.SecurityLabels) > 0 {
		result.MergeLabels(input.SecurityLabels)
	}

	// Link output to input
	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata["derived_from"] = input.ID.String()

	return result, nil
}

// performReflection executes the post-check phase (blueprint validation of output).
func (g *SupervisedAction) performReflection(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) (core.Envelope, error) {
	validatedResult, err := g.engine.Reflect(ctx, meta, result)
	if err != nil {
		if g.shouldBlock(err) {
			msg := "reflection failed"
			if core.IsInputError(err) {
				msg = "reflection blocked due to invalid input"
			}
			logger.Warn(msg, "action", meta.Name, "error", err.Error())
			return core.Envelope{}, fmt.Errorf("%s: %w", msg, err)
		}
		// Fail-Open: use original result
		logger.Warn("engine reflection failed but Fail-Open active. Proceeding.", "error", err)
		validatedResult = result
	}
	return validatedResult, nil
}

// applySteering evaluates steering decisions and stamps metadata.
func (g *SupervisedAction) applySteering(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) core.Envelope {
	decision, steeringMeta, err := g.engine.EvaluateSteering(ctx, result)
	if err != nil {
		logger.Warn("steering evaluation failed", "action", meta.Name, "error", err.Error())
		// Continue with empty decision on steering error
		decision = ""
		steeringMeta = nil
	}

	// Stamp metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata[core.KeyDecision] = decision
	for k, v := range steeringMeta {
		result.Metadata[k] = v
	}

	return result
}
```
---
## [internal/engine/solver.go]
```go
package supervisor

import (
	"context"
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/telemetry"

	"go.opentelemetry.io/otel"
)

// SupervisedAction is a decorator that wraps any `core.Action` to enforce governance blueprints.
type SupervisedAction struct {
	inner       core.Action
	engine      core.Evaluator
	tracer      core.Tracer
	failureMode string
}

// PolicyEngine is the core decision-making component of Manglekit.
// It orchestrates the loading of policies, maintaining the Datalog runtime,
// and executing authorization (Pre-Check) and validation (Post-Check) logic.
type PolicyEngine struct {
	tracer  core.Tracer
	logger  core.Logger
	runtime *MangleRuntime
}

// New creates a new PolicyEngine with default no-op observability.
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

// NewWithObservability creates a new PolicyEngine with both tracing and logging enabled.
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

// Logger returns the engine's configured Logger instance.
func (e *PolicyEngine) Logger() core.Logger {
	if e.logger == nil {
		return core.NopLogger{}
	}
	return e.logger
}

// LoadFacts injects a list of raw Datalog fact strings into the runtime's base knowledge.
func (e *PolicyEngine) LoadFacts(facts []string) error {
	return e.runtime.LoadFacts(facts)
}

// RegisterAction injects metadata about a registered action into the Datalog runtime.
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
func (e *PolicyEngine) LoadPolicy(ctx context.Context, policy string) error {
	if policy == "" {
		return nil
	}
	if err := e.runtime.AddPolicy(policy); err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}
	return nil
}

// AssessPlan implements the core.Evaluator interface.
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
			continue
		}
		facts = append(facts, atom)
	}

	// Inject Metadata facts
	for k, v := range input.Metadata {
		safeK := escapeString(k)
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}
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

	return config, nil
}

// Assess performs the Pre-Check phase of governance.
func (e *PolicyEngine) Assess(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.tracer == nil {
		return e.assessInternal(ctx, actionMeta, input)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPreCheck)
	defer span.End()

	if len(input.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: input.SecurityLabels})
	}

	err := e.assessInternal(ctx, actionMeta, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
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

func (e *PolicyEngine) assessInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	var extraFacts []ast.Atom
	if actionMeta.Name != "" {
		safeName := core.EntityInput
		safeOp := actionMeta.Name
		opFactStr := fmt.Sprintf("action_operation(\"%s\", \"%s\")", escapeString(safeName), escapeString(safeOp))
		opAtom, err := parse.Atom(opFactStr)
		if err == nil {
			extraFacts = append(extraFacts, opAtom)
		}
	}

	span.SetAttributes(map[string]any{
		"mangle.output_id": result.ID.String(),
	})
	return result, nil
}

// Reflect performs the Post-Check phase of governance.
func (e *PolicyEngine) Reflect(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.tracer == nil {
		return e.reflectInternal(ctx, actionMeta, output)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPostCheck)
	defer span.End()

	if len(output.SecurityLabels) > 0 {
		span.SetAttributes(map[string]any{core.AttrLabels: output.SecurityLabels})
	}

	result, err := e.reflectInternal(ctx, actionMeta, output)
	if err != nil {
		span.RecordError(err)
		return core.Envelope{}, err
	}
	if g.failureMode == "open" {
		return false
	}
	return true
}

func (g *SupervisedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	ctx = core.ContextWithLogger(ctx, g.engine.Logger())
	logger := core.LoggerFromContext(ctx)
	meta := g.inner.Metadata()

	logger.Info("Action started", "action", meta.Name, "input_id", input.ID.String())

	// Phase 1: Pre-Check (Assessment)
	if err := g.performAssessment(ctx, logger, meta, input); err != nil {
		return core.Envelope{}, err
	}

// evaluateGate centralizes the logic for "Check -> Deny -> Explain".
func (e *PolicyEngine) evaluateGate(ctx context.Context, actionName string, entityID string, env core.Envelope, extraFacts ...ast.Atom) error {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return nil
	}

	// Phase 3: Execute Inner Action
	result, err := g.executeAction(ctx, logger, meta, input)
	if err != nil {
		return core.Envelope{}, err
	}

	facts = append(facts, extraFacts...)

	// Inject Labels
	labelFacts, err := LabelsToFacts(entityID, env.SecurityLabels)
	if err != nil {
		return core.Envelope{}, err
	}
	for _, f := range labelFacts {
		atom, err := parse.Atom(f)
		if err == nil { facts = append(facts, atom) }
	}
	return nil
}

	// Inject Explicit Facts
	for _, f := range env.Facts {
		atom, err := parse.Atom(f)
		if err == nil { facts = append(facts, atom) }
	}

	// Inject Metadata
	for k, v := range env.Metadata {
		safeK := escapeString(k)
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}
		if k == "retry_count" {
			attemptFact := fmt.Sprintf("attempt(%s)", vStr)
			if atom, err := parse.Atom(attemptFact); err == nil {
				facts = append(facts, atom)
			}
		}
	}

	// Query: halt(Entity, Reason)
	queryHalt := fmt.Sprintf("%s(\"%s\", Reason)", core.PredHalt, entityID)
	var violationMsg, ruleID string
	var blocked bool

	err = e.runtime.QueryWithSolutions(facts, queryHalt, func(solution map[string]any) error {
		if reason, ok := solution["Reason"].(string); ok {
			violationMsg = reason
			blocked = true
			return ErrSolutionFound
		}
		return nil
	})

	if errors.Is(err, ErrSolutionFound) { err = nil }
	if err != nil { return fmt.Errorf("halt query error: %w", err) }

	if blocked {
		e.runtime.QueryWithSolutions(facts, "violation_rule(ID)", func(solution map[string]any) error {
			if id, ok := solution["ID"].(string); ok {
				ruleID = id
				return ErrSolutionFound
			}
			return nil
		})
		if e.logger != nil {
			e.logger.Debug("gate violation detected (halt)", "action", actionName, "msg", violationMsg, "rule_id", ruleID)
		}
		return &core.AlignmentError{Message: violationMsg, RuleID: ruleID}
	}

	// Priority 2: deny(Entity)
	queryDeny := fmt.Sprintf("%s(\"%s\")", core.PredHalt, entityID)
	denied, err := e.runtime.ExecuteQuery(facts, queryDeny)
	if err != nil { return fmt.Errorf("policy evaluation error: %w", err) }

	if denied {
		e.runtime.QueryWithSolutions(facts, fmt.Sprintf("%s(Msg)", core.PredViolation), func(solution map[string]any) error {
			if msg, ok := solution["Msg"].(string); ok {
				violationMsg = msg
				return ErrSolutionFound
			}
			return nil
		})
		e.runtime.QueryWithSolutions(facts, "violation_rule(ID)", func(solution map[string]any) error {
			if id, ok := solution["ID"].(string); ok {
				ruleID = id
				return ErrSolutionFound
			}
			return nil
		})
		return &core.AlignmentError{Message: violationMsg, RuleID: ruleID}
	}

	return nil
}

// CheckRequirement queries: requires("req_id", "capability")
func (e *PolicyEngine) CheckRequirement(ctx context.Context, input core.Envelope, reqName string) (bool, error) {
	if e.runtime == nil { return false, nil }
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil { return false, fmt.Errorf("fact conversion failed: %w", err) }
	query := fmt.Sprintf(\`requires("%s", "%s")\`, core.EntityInput, reqName)
	return e.ExecuteQuery(ctx, facts, query)
}

// EvaluateSteering executes "Steering Policies".
func (e *PolicyEngine) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	decision := core.DecisionProceed
	metadata := make(map[string]string)
	if e.runtime == nil || e.runtime.programInfo == nil {
		return decision, metadata, nil
	}

	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		return decision, metadata, fmt.Errorf("failed to convert input to facts: %w", err)
	}

	// Inject Explicit Facts
	for _, f := range input.Facts {
		atom, err := parse.Atom(f)
		if err == nil { facts = append(facts, atom) }
	}

	// Inject Metadata
	for k, v := range input.Metadata {
		safeK := escapeString(k)
		vStr := fmt.Sprintf("%v", v)
		safeV := escapeString(vStr)
		metaFact := fmt.Sprintf("meta(\"%s\", \"%s\")", safeK, safeV)
		if atom, err := parse.Atom(metaFact); err == nil {
			facts = append(facts, atom)
		}
		if k == "retry_count" {
			attemptFact := fmt.Sprintf("attempt(%s)", vStr)
			if atom, err := parse.Atom(attemptFact); err == nil {
				facts = append(facts, atom)
			}
		}
	}

	err = e.runtime.QueryWithSolutions(facts, fmt.Sprintf("%s(Hint)", core.PredRetry), func(solution map[string]any) error {
		if hint, ok := solution["Hint"].(string); ok {
			decision = core.DecisionRetry
			metadata[core.KeyFeedback] = hint
			return ErrSolutionFound
		}
		return nil
	})
	if errors.Is(err, ErrSolutionFound) { err = nil }
	if decision == core.DecisionRetry { return decision, metadata, nil }

	err = e.runtime.QueryWithSolutions(facts, fmt.Sprintf("%s(Target)", core.PredRoute), func(solution map[string]any) error {
		if target, ok := solution["Target"].(string); ok {
			decision = core.DecisionRoute
			metadata[core.KeyNextStep] = target
			return ErrSolutionFound
		}
		return nil
	})
	if errors.Is(err, ErrSolutionFound) { err = nil }

	return decision, metadata, nil
}

func (e *PolicyEngine) ExecuteQuery(ctx context.Context, facts []ast.Atom, queryStr string) (bool, error) {
	if e.tracer == nil { return e.runtime.ExecuteQuery(facts, queryStr) }
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

func (e *PolicyEngine) Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error) {
	var results []map[string]string
	if e.runtime == nil { return nil, fmt.Errorf("runtime not initialized") }

	var atomFacts []ast.Atom
	for _, f := range facts {
		atom, err := parse.Atom(f)
		if err != nil { return nil, fmt.Errorf("failed to parse fact '%s': %w", f, err) }
		atomFacts = append(atomFacts, atom)
	}

	err := e.runtime.QueryWithSolutions(atomFacts, queryStr, func(solution map[string]any) error {
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
	if err != nil { return nil, err }
	return results, nil
}

func toMangleFacts(entityID string, input any, contentType core.ContentType) ([]ast.Atom, error) {
	if input == nil { return nil, nil }
	var atoms []ast.Atom
	var facts []string
	var err error

	if contentType == core.TypeJSON {
		facts, err = Flatten(entityID, input)
	} else {
		facts, err = ToFacts(entityID, input)
	}
	if err != nil { return nil, err }

	for _, factStr := range facts {
		atom, err := parse.Atom(factStr)
		if err != nil { return nil, fmt.Errorf("failed to parse fact '%s': %w", factStr, err) }
		atoms = append(atoms, atom)
	}
	return atoms, nil
}
```

---
## [internal/engine/evaluator.go]
```go
package engine

import (
	"fmt"
	"reflect"
	"strings"

	mangleanalysis "github.com/google/mangle/analysis"
	mangleast "github.com/google/mangle/ast"
	mangleengine "github.com/google/mangle/engine"
	manglefactstore "github.com/google/mangle/factstore"
	mangleparse "github.com/google/mangle/parse"
)

// Evaluator provides capabilities for evaluating a single Datalog rule against a Go struct.
type Evaluator struct {
	rule     string
	clause   mangleast.Clause
	ruleHead string // e.g., "deny", "allow", "route"
}

// NewEvaluator creates a new Evaluator instance from a Datalog rule string.
func NewEvaluator(rule string) (*Evaluator, error) {
	if rule == "" {
		return nil, fmt.Errorf("rule cannot be empty")
	}

	clause, err := mangleparse.Clause(rule)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rule: %w", err)
	}

	ruleHead := ""
	if clause.Head.Predicate.Symbol != "" {
		ruleHead = clause.Head.Predicate.Symbol
	}

	return &Evaluator{
		rule:     rule,
		clause:   clause,
		ruleHead: ruleHead,
	}, nil
}

// EvaluateResult encapsulates the outcome of a rule evaluation.
type EvaluateResult struct {
	Matched bool
	EntityID string
	RuleHead string
}

// Evaluate executes the configured Datalog rule against a provided Go struct.
func (e *Evaluator) Evaluate(entityID string, entity any) (EvaluateResult, error) {
	result := EvaluateResult{
		EntityID: entityID,
		RuleHead: e.ruleHead,
	}

	facts, err := structToFacts(entityID, entity)
	if err != nil {
		return result, fmt.Errorf("failed to convert entity to facts: %w", err)
	}

	store := manglefactstore.NewSimpleInMemoryStore()
	knownPredicates := make(map[mangleast.PredicateSym]mangleast.Decl)

	for _, atom := range facts {
		store.Add(atom)
		if _, ok := knownPredicates[atom.Predicate]; !ok {
			knownPredicates[atom.Predicate] = mangleast.NewSyntheticDeclFromSym(atom.Predicate)
		}
	}

	program := []mangleast.Clause{e.clause}
	programInfo, err := mangleanalysis.AnalyzeOneUnit(mangleparse.SourceUnit{Clauses: program}, knownPredicates)
	if err != nil {
		return result, fmt.Errorf("failed to analyze program: %w", err)
	}

	if err := mangleengine.EvalProgram(programInfo, store); err != nil {
		return result, fmt.Errorf("failed to evaluate program: %w", err)
	}

	queryStr := fmt.Sprintf(`%s("%s")`, e.ruleHead, entityID)
	queryAtom, err := mangleparse.Atom(queryStr)
	if err != nil {
		return result, fmt.Errorf("failed to parse query: %w", err)
	}

	result.Matched = store.Contains(queryAtom)
	return result, nil
}

func structToFacts(entityID string, entity any) ([]mangleast.Atom, error) {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, fmt.Errorf("entity cannot be nil")
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("entity must be a struct, got %v", val.Kind())
	}

	typ := val.Type()
	var atoms []mangleast.Atom

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		if !field.IsExported() { continue }

		tag := field.Tag.Get("mangle")
		if tag == "-" { continue }
		if tag == "" { tag = strings.ToLower(field.Name) }

		var atom mangleast.Atom
		switch fieldVal.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.Number(fieldVal.Int()))
		case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.Number(int64(fieldVal.Uint())))
		case reflect.Float32, reflect.Float64:
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.String(fmt.Sprintf("%f", fieldVal.Float())))
		case reflect.String:
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.String(fieldVal.String()))
		case reflect.Bool:
			boolStr := "false"
			if fieldVal.Bool() { boolStr = "true" }
			atom = mangleast.NewAtom(tag, mangleast.String(entityID), mangleast.String(boolStr))
		default:
			continue
		}
		atoms = append(atoms, atom)
	}
}
```

### `sdk/client.go`
```go
package sdk

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/trace"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/supervisor"
)

const (
	TracerName = "github.com/duynguyendang/manglekit/sdk"
	FailModeOpen   = "open"
	FailModeClosed = "closed"
)

type Client struct {
	engine core.Evaluator
	tracer core.Tracer
	otelTracer trace.Tracer
	logger core.Logger
	agentMemory core.AgentMemory
	registry map[string]core.Action
	failureMode string
	blueprintPath string
	shutdownFunc func(context.Context) error
	llm core.TextGenerator
}

	if len(atoms) == 0 {
		return nil, fmt.Errorf("no valid facts could be extracted from entity")
	}
	return atoms, nil
}
```
---
## [internal/engine/reflection.go]
```go
package engine

import (
	"fmt"
	"reflect"
	"strings"
)

// ToFacts converts an arbitrary Go struct into a list of Datalog fact strings.
// It recursively traverses the struct and generates facts for each field.
//
// Format:
//   - Root: predicate("entityID", value)
//   - Nested: predicate("parentID", "childID") + child facts
func ToFacts(entityID string, val any) ([]string, error) {
	var facts []string
	if val == nil {
		return facts, nil
	}

	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return facts, nil
		}
		v = v.Elem()
	}
	if err := ensureDependencies(c); err != nil {
		return nil, err
	}
	return c, nil
}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("ToFacts expects a struct or pointer to struct, got %v", v.Kind())
	}

	err := structToFactsRecursive(entityID, v, &facts)
	if err != nil {
		return nil, err
	}

	return facts, nil
}

func structToFactsRecursive(parentID string, v reflect.Value, facts *[]string) error {
	t := v.Type()
	safeParentID := escapeString(parentID)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		val := v.Field(i)

		if !field.IsExported() {
			continue
		}

		// Handle 'mangle' tag
		predName := field.Tag.Get("mangle")
		if predName == "-" {
			continue
		}
		if predName == "" {
			predName = strings.ToLower(field.Name)
		}
		// Sanitize predicate name
		predName = escapeString(predName)

		// Resolve Ptr/Interface
		for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
			if val.IsNil() {
				goto NextField
			}
			val = val.Elem()
		}

		switch val.Kind() {
		case reflect.Struct:
			// Recurse: predName("parentID", "childID")
			childID := fmt.Sprintf("%s_%s", safeParentID, predName)
			fact := fmt.Sprintf("%s(\"%s\", \"%s\")", predName, safeParentID, childID)
			*facts = append(*facts, fact)
			if err := structToFactsRecursive(childID, val, facts); err != nil {
				return err
			}

		case reflect.Slice, reflect.Array:
			// List handling: predName("parentID", value) for each item
			// Note: This treats slices as repeated predicates (multiset).
			for j := 0; j < val.Len(); j++ {
				elem := val.Index(j)
				// Resolve Ptr/Interface
				for elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
					if elem.IsNil() {
						continue
					}
					elem = elem.Elem()
				}

				if elem.Kind() == reflect.Struct {
					// Complex list item
					childID := fmt.Sprintf("%s_%s_%d", safeParentID, predName, j)
					fact := fmt.Sprintf("%s(\"%s\", \"%s\")", predName, safeParentID, childID)
					*facts = append(*facts, fact)
					if err := structToFactsRecursive(childID, elem, facts); err != nil {
						return err
					}
				} else {
					// Primitive list item
					fact := formatPrimitiveFact(predName, safeParentID, elem)
					if fact != "" {
						*facts = append(*facts, fact)
					}
				}
			}

		case reflect.Map:
			// Map handling: value("parentID", "key", "val")
			iter := val.MapRange()
			for iter.Next() {
				k := iter.Key()
				v := iter.Value()

				// Resolve Ptr/Interface
				for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
					if v.IsNil() {
						continue
					}
					v = v.Elem()
				}

				// Only support primitive values in maps for now
				if isPrimitive(v.Kind()) {
					keyStr := fmt.Sprintf("%v", k.Interface())
					valStr := fmt.Sprintf("%v", v.Interface())
					// value(Entity, Key, Value)
					fact := fmt.Sprintf("value(\"%s\", \"%s\", \"%s\")", safeParentID, escapeString(keyStr), escapeString(valStr))
					*facts = append(*facts, fact)
				}
			}

		default:
			// Primitive
			fact := formatPrimitiveFact(predName, safeParentID, val)
			if fact != "" {
				*facts = append(*facts, fact)
			}
		}

	NextField:
	}
	return nil
}

func formatPrimitiveFact(pred, entity string, val reflect.Value) string {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%s(\"%s\", %d)", pred, entity, val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%s(\"%s\", %d)", pred, entity, val.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%s(\"%s\", %g)", pred, entity, val.Float()) // Use %g for unquoted float
	case reflect.Bool:
		if val.Bool() {
			return fmt.Sprintf("%s(\"%s\", \"true\")", pred, entity)
		}
		return fmt.Sprintf("%s(\"%s\", \"false\")", pred, entity)
	case reflect.String:
		return fmt.Sprintf("%s(\"%s\", \"%s\")", pred, entity, escapeString(val.String()))
	default:
		return ""
	}
}

func isPrimitive(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
		return true
	}
	return false
}

// LabelsToFacts converts a list of security labels into Datalog facts.
// Format: label("tag")
func LabelsToFacts(entityID string, labels []string) ([]string, error) {
	var facts []string
	for _, l := range labels {
		safeLabel := escapeString(l)
		facts = append(facts, fmt.Sprintf("label(\"%s\")", safeLabel))
	}
	return facts, nil
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
```
---
## [internal/engine/flattener.go]
```go
package engine

import (
	"fmt"
	"reflect"
	"strconv"
)

// Flatten converts any JSON-like object (map[string]any, slice, primitive) into a flat list of Datalog facts.
// It generates a graph-like representation of the JSON tree.
//
// Predicates:
//   - json_str(NodeID, Key, Value)
//   - json_num(NodeID, Key, Value)
//   - json_bool(NodeID, Key, Value)
//   - json_null(NodeID, Key)
//   - json_link(NodeID, Key, ChildNodeID)
//
// Parameters:
//   - rootID: The ID of the root node (usually "Req" or "Resp").
//   - input: The data to flatten.
func Flatten(rootID string, input any) ([]string, error) {
	var facts []string
	counter := 0 // Node ID generator

	// If input is nil, return empty
	if input == nil {
		return facts, nil
	}

	visited := make(map[uintptr]bool) // Cycle detection

	// We wrap the input in a dummy map key if it's complex, to match the recursive signature
	// But usually input is the root object.
	// Let's assume input is map[string]any or []any.

	err := flattenRecursive(rootID, reflect.ValueOf(input), &facts, &counter, visited)
	if err != nil {
		return nil, err
	}

	return facts, nil
}

func flattenRecursive(nodeID string, v reflect.Value, facts *[]string, counter *int, visited map[uintptr]bool) error {
	// Unwrap
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil // treat as null?
		}
		v = v.Elem()
	}
	return supervisor.NewSupervisedAction(action, c.engine, c.failureMode)
}

	// Cycle detection for pointers/maps/slices
	if v.CanAddr() {
		addr := v.UnsafeAddr()
		if visited[addr] {
			return fmt.Errorf("cycle detected")
		}
		visited[addr] = true
		defer delete(visited, addr)
	}

	switch v.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()

			keyStr := fmt.Sprintf("%v", key.Interface())
			safeKey := escapeString(keyStr)

			effVal := val
			for effVal.Kind() == reflect.Interface || effVal.Kind() == reflect.Ptr {
				if effVal.IsNil() { break }
				effVal = effVal.Elem()
			}

			if isComplexKind(effVal.Kind()) {
				*counter++
				childID := fmt.Sprintf("node_%d", *counter)
				fact := fmt.Sprintf("json_link(\"%s\", \"%s\", \"%s\")", escapeString(nodeID), safeKey, childID)
				*facts = append(*facts, fact)
				if err := flattenRecursive(childID, val, facts, counter, visited); err != nil {
					return err
				}
			} else {
				addPrimitiveReflect(nodeID, safeKey, effVal, facts)
			}
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			val := v.Index(i)
			keyStr := strconv.Itoa(i)

			effVal := val
			for effVal.Kind() == reflect.Interface || effVal.Kind() == reflect.Ptr {
				if effVal.IsNil() { break }
				effVal = effVal.Elem()
			}

			if isComplexKind(effVal.Kind()) {
				*counter++
				childID := fmt.Sprintf("node_%d", *counter)
				fact := fmt.Sprintf("json_link(\"%s\", \"%s\", \"%s\")", escapeString(nodeID), keyStr, childID)
				*facts = append(*facts, fact)
				if err := flattenRecursive(childID, val, facts, counter, visited); err != nil {
					return err
				}
			} else {
				addPrimitiveReflect(nodeID, keyStr, effVal, facts)
			}
		}
	}
	return nil
}

func isComplexKind(k reflect.Kind) bool {
	return k == reflect.Map || k == reflect.Slice || k == reflect.Array
}

func addPrimitiveReflect(nodeID, key string, v reflect.Value, facts *[]string) {
	nodeID = escapeString(nodeID)
	switch v.Kind() {
	case reflect.String:
		fact := fmt.Sprintf("json_str(\"%s\", \"%s\", \"%s\")", nodeID, key, escapeString(v.String()))
		*facts = append(*facts, fact)
	case reflect.Bool:
		sVal := "false"
		if v.Bool() { sVal = "true" }
		fact := fmt.Sprintf("json_bool(\"%s\", \"%s\", \"%s\")", nodeID, key, sVal)
		*facts = append(*facts, fact)
	case reflect.Int, reflect.Int64, reflect.Int32:
		fact := fmt.Sprintf("json_num(\"%s\", \"%s\", %d)", nodeID, key, v.Int())
		*facts = append(*facts, fact)
	case reflect.Float64, reflect.Float32:
		fact := fmt.Sprintf("json_num(\"%s\", \"%s\", %g)", nodeID, key, v.Float())
		*facts = append(*facts, fact)
	default:
		sVal := fmt.Sprintf("%v", v.Interface())
		fact := fmt.Sprintf("json_str(\"%s\", \"%s\", \"%s\")", nodeID, key, escapeString(sVal))
		*facts = append(*facts, fact)
	}
}
```

---
## [sdk/client.go]
```go
package sdk

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/supervisor"
)

const (
	// TracerName is the instrumentation scope name for Manglekit tracing.
	TracerName = "github.com/duynguyendang/manglekit/sdk"

	// Failure modes determine the system's resilience strategy.
	FailModeOpen   = "open"   // Allow execution on system error (Fail-Open)
	FailModeClosed = "closed" // Block execution on system error (Fail-Closed)
)

// Client is the primary entry point for the Manglekit system.
// It acts as the governance kernel, managing blueprints, observability, and action execution.
type Client struct {
	engine        core.Evaluator
	tracer        core.Tracer
	otelTracer    trace.Tracer
	logger        core.Logger
	agentMemory   core.AgentMemory
	registry      map[string]core.Action
	failureMode   string
	blueprintPath string
	shutdownFunc  func(context.Context) error
	llm           core.TextGenerator
}

// NewClient initializes a new Manglekit Client with the provided options.
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	c := &Client{
		logger: logger.NewDefault(),
		// Default to HybridMemory with Nop stores
		agentMemory: NewHybridMemory(core.NopStore{}, core.NopVectorStore{}, core.NopEmbedder{}),
		registry:    make(map[string]core.Action),
		failureMode: FailModeClosed, // Default to closed
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if err := ensureDependencies(c); err != nil {
		return nil, err
	}

	return c, nil
}

// NewClientFromFile initializes a Client by loading configuration from a YAML file.
func NewClientFromFile(ctx context.Context, configPath string, opts ...ClientOption) (*Client, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	newOpts := append([]ClientOption{WithConfig(cfg)}, opts...)
	return NewClient(ctx, newOpts...)
}

// Supervise wraps a raw core.Action in a SupervisedAction.
func (c *Client) Supervise(action core.Action) core.Action {
	if c.tracer != nil {
		return supervisor.NewSupervisedActionWithTracer(action, c.engine, c.tracer, c.failureMode)
	}
	return supervisor.NewSupervisedAction(action, c.engine, c.failureMode)
}

// Engine returns the underlying policy engine (Evaluator).
func (c *Client) Engine() core.Evaluator {
	return c.engine
}

// LoadFacts allows manually injecting straight Datalog facts into the engine.
func (c *Client) LoadFacts(facts []string) error {
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	return c.engine.LoadFacts(facts)
}

// RegisterAction adds an action to the client's internal registry.
func (c *Client) RegisterAction(name string, action core.Action) {
	c.registry[name] = action
	if c.engine != nil {
		if err := c.engine.RegisterAction(action.Metadata()); err != nil {
			c.logger.Warn("failed to register action metadata to engine", "action", name, "error", err)
		}
	}
}

// Memory returns the active memory provider (if any).
func (c *Client) Memory() core.AgentMemory {
	return c.agentMemory
}
```
---
## [sdk/loop.go]
```go
package sdk

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/core"
	engine_memory "github.com/duynguyendang/manglekit/internal/engine/memory"
)

const (
	DefaultMaxRetries = 3
	DefaultMaxSteps   = 10
	BackoffBase       = 100 * time.Millisecond
)

// ExecutionParams tracks the state of a single execution flow (RunLoop).
type ExecutionParams struct {
	SessionID       string
	RetryCount      int
	CurrentHistory  []core.Message
	FeedbackHistory []string
	LastFeedback    string
	Metadata        map[string]any
	Store           core.AgentMemory // Using generic AgentMemory interface subset for history
	MemoryMode      core.MemoryMode
}

// runLoopInternal manages the semantic state machine (Action -> Decision -> Action).
// It handles retries (correction) and routing (next step).
func (c *Client) runLoopInternal(ctx context.Context, startAction string, payload any, params ExecutionParams) (core.Envelope, error) {
	ctx = core.ContextWithLogger(ctx, c.logger)

	// 1. Determine Store Strategy
	switch params.MemoryMode {
	case core.MemoryModePersist:
		params.Store = c.agentMemory
		if params.SessionID != "" {
			var err error
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
		if err := ctx.Err(); err != nil {
			return core.Envelope{}, err
		}

		c.logger.Info("RunLoop step", "step", step, "action", currentAction)

		result, err := c.ExecuteSingleStep(ctx, currentAction, currentPayload, &params)
		if err != nil {
			return core.Envelope{}, err
		}

		decision := result.Metadata[core.KeyDecision]
		if decision == core.DecisionRoute {
			next, ok := result.Metadata[core.KeyNextStep].(string)
			if !ok || next == "" {
				return core.Envelope{}, fmt.Errorf("route decision missing next_step")
			}
			c.logger.Info("RunLoop: Routing to next action", "from", currentAction, "to", next)
			currentAction = next
			currentPayload = result.Payload
			continue
		}

		if decision == core.DecisionRetry {
			continue
		}

		return result, nil
	}
	return core.Envelope{}, fmt.Errorf("max steps exceeded")
}

// ExecuteSingleStep runs one step of the action and returns the decision.
func (c *Client) ExecuteSingleStep(ctx context.Context, actionName string, payload any, params *ExecutionParams) (core.Envelope, error) {
	action, ok := c.registry[actionName]
	if !ok {
		return core.Envelope{}, fmt.Errorf("action not found: %s", actionName)
	}

	env := core.NewEnvelope(payload)
	env.ContentType = action.Metadata().InputContentType
	c.injectContext(ctx, &env, payload, params)

	result, err := action.Execute(ctx, env)
	if err != nil {
		return c.handleExecutionError(ctx, err, payload, params)
	}
	params.LastFeedback = ""

	c.updateHistory(ctx, payload, result, params)

	return c.handleDecision(ctx, actionName, result, payload, params)
}

// injectContext populates the envelope with feedback, history, RAG context, metadata, and facts.
func (c *Client) injectContext(ctx context.Context, env *core.Envelope, payload any, params *ExecutionParams) {
	if len(params.FeedbackHistory) > 0 {
		env.Metadata[core.KeyPrevFeedback] = strings.Join(params.FeedbackHistory, "; ")
	}
	if params.LastFeedback != "" {
		env.SetFeedback(params.LastFeedback)
		env.Metadata["mangle_feedback"] = params.LastFeedback
	}
	if len(params.CurrentHistory) > 0 && params.MemoryMode != core.MemoryModeNone {
		env.SetHistory(params.CurrentHistory)
	}

	c.recallContext(ctx, payload, env)

	for k, v := range params.Metadata { env.Metadata[k] = v }
	for k, v := range core.ContextFacts(ctx) { env.Metadata[k] = v }
}

func (c *Client) handleExecutionError(ctx context.Context, err error, payload any, params *ExecutionParams) (core.Envelope, error) {
	var alignErr *core.AlignmentError
	if !errors.As(err, &alignErr) {
		return core.Envelope{}, err
	}

	if params.RetryCount >= DefaultMaxRetries {
		return core.Envelope{}, fmt.Errorf("max retries exceeded: %w", err)
	}

	params.RetryCount++
	params.LastFeedback = alignErr.Message
	c.logger.Warn("RunLoop: Blueprint Alignment Issue", "feedback", params.LastFeedback, "attempt", params.RetryCount)

	if err := c.backoff(ctx, params.RetryCount); err != nil {
		return core.Envelope{}, err
	}

	res := core.NewEnvelope(payload)
	res.Metadata[core.KeyDecision] = core.DecisionRetry
	return res, nil
}

func (c *Client) updateHistory(ctx context.Context, payload any, result core.Envelope, params *ExecutionParams) {
	if params.MemoryMode == core.MemoryModeNone {
		return
	}
	newExchange := []core.Message{
		{Role: "user", Content: safelyStringify(payload)},
		{Role: "assistant", Content: safelyStringify(result.Payload)},
	}
	params.CurrentHistory = append(params.CurrentHistory, newExchange...)

	if params.Store != nil && params.SessionID != "" && params.MemoryMode == core.MemoryModePersist {
		if err := params.Store.Append(ctx, params.SessionID, params.CurrentHistory); err != nil && c.logger != nil {
			c.logger.Warn("RunLoop failed to persist history", "error", err)
		}
	}
}

func (c *Client) handleDecision(ctx context.Context, actionName string, result core.Envelope, payload any, params *ExecutionParams) (core.Envelope, error) {
	decision := result.Metadata[core.KeyDecision]

	if decision == "" || decision == core.DecisionProceed {
		c.asyncMemorize(payload, result.Payload)
	}

	c.logger.Debug("RunLoop decision", "decision", decision, "action", actionName)

	switch decision {
	case core.DecisionRetry:
		return c.handleRetryDecision(ctx, actionName, result, params)
	case core.DecisionRoute:
		params.RetryCount = 0
		params.FeedbackHistory = nil
		c.logger.Info("RunLoop: Feedback history cleared for new action route")
		return result, nil
	case core.DecisionHalt:
		return core.Envelope{}, c.buildHaltError(result)
	}
	return result, nil
}

func (c *Client) handleRetryDecision(ctx context.Context, actionName string, result core.Envelope, params *ExecutionParams) (core.Envelope, error) {
	if params.RetryCount >= DefaultMaxRetries {
		return core.Envelope{}, fmt.Errorf("max retries exceeded for action %s", actionName)
	}
	params.RetryCount++
	hint := result.GetFeedback()
	params.LastFeedback = hint
	params.FeedbackHistory = append(params.FeedbackHistory, hint)
	c.logger.Warn("RunLoop: RETRY triggered", "feedback", hint)

	if err := c.backoff(ctx, params.RetryCount); err != nil {
		return core.Envelope{}, err
	}
	return result, nil
}

func (c *Client) backoff(ctx context.Context, retryCount int) error {
	sleepDuration := time.Duration(retryCount) * BackoffBase
	select {
	case <-ctx.Done(): return ctx.Err()
	case <-time.After(sleepDuration): return nil
	}
}

func safelyStringify(v any) string {
	if v == nil { return "" }
	if s, ok := v.(string); ok { return s }
	return fmt.Sprintf("%v", v)
}

func (c *Client) recallContext(ctx context.Context, payload any, env *core.Envelope) {
	if c.agentMemory == nil { return }
	if c.engine != nil {
		needed, err := c.engine.CheckRequirement(ctx, *env, "memory")
		if err != nil || !needed { return }
	}

	var span core.Span
	if c.tracer != nil {
		ctx, span = c.tracer.Start(ctx, core.SpanMemory)
		defer span.End()
	}

	inputStr := safelyStringify(payload)
	contextData, err := c.agentMemory.Recall(ctx, inputStr)
	if err != nil {
		c.logger.Warn("Memory Recall failed", "error", err)
		if span != nil { span.RecordError(err) }
		return
	}
	if contextData != "" {
		env.SetMeta(core.KeyContext, contextData)
		c.logger.Debug("Injected memory context", "len", len(contextData))
	}
}

func (c *Client) asyncMemorize(input any, output any) {
	if c.agentMemory == nil { return }
	inputStr := safelyStringify(input)
	outputStr := safelyStringify(output)
	go func(q, a string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.agentMemory.Memorize(ctx, q, a); err != nil {
			c.logger.Warn("Memory Memorize failed", "error", err)
		}
	}(inputStr, outputStr)
}

func (c *Client) buildHaltError(result core.Envelope) error {
	reason := result.Metadata["reason"]
	if reason == "" { reason = result.Metadata["violation_msg"] }
	if reason == "" { reason = "blueprint violation" }
	return fmt.Errorf("action halted by blueprint: %s", reason)
}
```
---

### `adapters/func/wrapper.go`
```go
package function

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
)

type ToolFunc[In any, Out any] func(context.Context, In) (Out, error)

type Wrapper[In any, Out any] struct {
	name        string
	fn          ToolFunc[In, Out]
	contentType core.ContentType
	inputType   string
	outputType  string
}

func New[In any, Out any](name string, fn ToolFunc[In, Out]) *Wrapper[In, Out] {
	inType := reflect.TypeOf((*In)(nil)).Elem()
	if inType.Kind() == reflect.Ptr {
		inType = inType.Elem()
	}
	inName := inType.Name()

	outType := reflect.TypeOf((*Out)(nil)).Elem()
	if outType.Kind() == reflect.Ptr {
		outType = outType.Elem()
	}
	outName := outType.Name()

	return &Wrapper[In, Out]{
		name:        name,
		fn:          fn,
		contentType: core.TypeStruct,
		inputType:   inName,
		outputType:  outName,
	}
}

func (w *Wrapper[In, Out]) SetContentType(ct core.ContentType) {
	w.contentType = ct
}

func (w *Wrapper[In, Out]) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	in, ok := input.Payload.(In)
	if !ok {
		return core.Envelope{}, fmt.Errorf("%w: invalid input type, expected %T but got %T", core.ErrSystemError, *new(In), input.Payload)
	}

	out, err := w.fn(ctx, in)
	if err != nil {
		return core.Envelope{}, err
	}

	return core.NewEnvelope(out), nil
}

func (w *Wrapper[In, Out]) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name:             w.name,
		Type:             "function",
		InputContentType: w.contentType,
		InputType:        w.inputType,
		OutputType:       w.outputType,
		IsDynamic:        false,
	}
}
```

* 2025-12-19: Kernel Resync. Added Datalog StdLib and Reflection Logic. Generated Component Specs.
