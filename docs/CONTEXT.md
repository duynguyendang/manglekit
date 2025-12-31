---
context_type: kernel_source_dump
project: manglekit
language: go, datalog
last_updated: 2025-12-31
scan_mode: logic_focused
---

## 1. THE COMPLETE FILE MAP

.
├── adapters
│   ├── ai                  # Genkit Wrapper (LLM Integration)
│   ├── memory              # Hybrid HNSW+Graph Memory
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
│   │   └── resources           # Embedded Datalog Rules (std.dl)
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
	KeyDecision     = "manglekit.decision"
	KeyFeedback     = "manglekit.feedback"
	KeyPrevFeedback = "prev_feedback" // History of feedback
	KeyNextStep     = "manglekit.next_step"
	KeyRiskScore    = "manglekit.risk_score"
	KeyTraceID      = "manglekit.trace_id"
	KeySessionID    = "manglekit.session_id"
	KeyContext      = "manglekit.context" // RAG Context
	KeyHistory      = "manglekit_history" // Chat History

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
	SpanMemory    = "Mangle.Recall"
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
	EntityOutput       = "Output"
)

// ContentType defines how the payload should be interpreted by the engine.
type ContentType string

const (
	TypeStruct ContentType = "STRUCT"
	TypeJSON   ContentType = "JSON"
)

// Envelope is the standard unit of data exchange in Manglekit.
// It wraps the payload with metadata, identification, and security context.
type Envelope struct {
	ID             uuid.UUID      `json:"id"`
	Payload        any            `json:"data"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Error       error          `json:"error,omitempty"`

	// SecurityLabels holds taint tags (e.g., "secret", "pii") for information flow control.
	SecurityLabels []string       `json:"security_labels,omitempty"`
	// Facts holds structured logical facts extracted from the payload.
	Facts          []string       `json:"facts,omitempty"` // Explicit Datalog facts
	// ContentType indicates whether the payload is a Struct or JSON.
	ContentType    ContentType    `json:"content_type,omitempty"`
}

// NewEnvelope creates a new Envelope with a generated UUID.
func NewEnvelope(payload any) Envelope {
	return Envelope{
		ID:          uuid.New(),
		Payload:     payload,
		Metadata:    make(map[string]any),
		SecurityLabels: []string{},
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
	if v, ok := e.Metadata[KeyFeedback]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetHistory injects chat history into the payload (if it's a map) or metadata.
// For structured prompts, it prefers metadata injection for the template to use.
func (e *Envelope) SetHistory(msgs []Message) {
	b, err := json.Marshal(msgs)
	if err == nil {
		e.SetMeta(KeyHistory, string(b))
	}
}

// MergeLabels adds unique security labels to the envelope.
func (e *Envelope) MergeLabels(other []string) {
	existing := make(map[string]bool)
	for _, l := range e.SecurityLabels {
		existing[l] = true
	}
	for _, l := range other {
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
	IsDynamic   bool   `json:"is_dynamic,omitempty"`
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

// Extractor converts raw text into structured data.
type Extractor interface {
	Extract(ctx context.Context, input string, schema any) error
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
## [core/memory.go]
```go
package core

import "context"

// AgentMemory defines the capability to store and retrieve past experiences.
// It unifies sequential History (chat logs) and Semantic Memory (RAG).
type AgentMemory interface {
	// --- Sequential History ---
	// Read retrieves the chat history for a given session.
	Read(ctx context.Context, sessionID string) ([]Message, error)
	// Append adds new messages to the history.
	Append(ctx context.Context, sessionID string, msgs []Message) error

	// --- Semantic Memory (RAG) ---
	// Recall retrieves relevant context based on the current query.
	Recall(ctx context.Context, query string) (string, error)
	// Memorize stores a new interaction (Input/Output) for future recall.
	Memorize(ctx context.Context, query string, answer string) error

	// Init performs any necessary setup (e.g. connecting to DB).
	Init(ctx context.Context) error
}

// AgentMemoryWithFacts is an optional interface for Memory providers that can return
// additional metadata (facts) along with the text context.
type AgentMemoryWithFacts interface {
	AgentMemory
	// RecallWithFacts retrieves context and associated metadata (e.g. doc IDs).
	RecallWithFacts(ctx context.Context, query string) (string, map[string]any, error)
}
```
---
## [core/infra.go]
```go
package core

import "context"

// Breaker implements the Circuit Breaker pattern.
type Breaker interface {
	Execute(req func() (any, error)) (any, error)
	Name() string
}

// Meter abstracts metrics (Counters, Gauges).
type Meter interface {
	Counter(name string, val int64, tags map[string]string)
	Histogram(name string, val float64, tags map[string]string)
}

// Tracer abstracts distributed tracing.
type Tracer interface {
	// Start creates a new span with the given name and returns a context and span.
	Start(ctx context.Context, name string) (context.Context, Span)
}

// Span represents an active operation being traced.
type Span interface {
	End()
	SetAttributes(attributes map[string]any)
	SetStatus(code string, msg string)
	RecordError(err error)
}
```
---
## [core/errors.go]
```go
package core

import (
	"errors"
	"fmt"
)

var (
	// ErrAlignment is returned when a blueprint alignment check blocks an action.
	ErrAlignment = errors.New("alignment error")
	// ErrSystemError is returned when an unexpected error occurs.
	ErrSystemError = errors.New("system error")
	// ErrInputValidation is returned when input conversion or parsing fails.
	ErrInputValidation = errors.New("input validation error")
)

// AlignmentError is a structured error that carries a specific intervention message.
// It wraps ErrAlignment to ensure standard error matching works.
type AlignmentError struct {
	Message string
	RuleID  string
}

func (e *AlignmentError) Error() string {
	if e.RuleID != "" {
		return fmt.Sprintf("[INTERVENTION] [%s]: %s", e.RuleID, e.Message)
	}
	return fmt.Sprintf("[INTERVENTION]: %s", e.Message)
}

func (e *AlignmentError) Is(target error) bool {
	return target == ErrAlignment
}

func (e *AlignmentError) Unwrap() error {
	return ErrAlignment
}

// IsAlignmentError checks if the error is an AlignmentError.
func IsAlignmentError(err error) bool {
	return errors.Is(err, ErrAlignment)
}

// InputError is a structured error that indicates a failure in input validation or fact conversion.
// It wraps ErrInputValidation to ensure distinction from system errors.
type InputError struct {
	Err error
}

func (e *InputError) Error() string {
	return fmt.Sprintf("input validation error: %v", e.Err)
}

func (e *InputError) Is(target error) bool {
	return target == ErrInputValidation
}

func (e *InputError) Unwrap() error {
	return ErrInputValidation
}

// IsInputError checks if the error is an InputError.
func IsInputError(err error) bool {
	return errors.Is(err, ErrInputValidation)
}

// NewAlignmentError creates a new AlignmentError with the given message and rule ID.
func NewAlignmentError(message, ruleID string) *AlignmentError {
	return &AlignmentError{
		Message: message,
		RuleID:  ruleID,
	}
}

// WrapInputError wraps an error as an InputError.
func WrapInputError(err error) *InputError {
	return &InputError{Err: err}
}
```
---
## [core/session_state.go]
```go
package core

import (
	"encoding/json"
	"fmt"
)

// SessionState packages all components needed for full recovery of a Manglekit session.
// It is used by the Durable State Manager to checkpoint and restore execution state.
type SessionState struct {
	// SessionID is the unique identifier for the persistent thread.
	SessionID string `json:"session_id"`

	// ActiveEnvelope is the current envelope including Payload, Metadata, and Labels.
	ActiveEnvelope Envelope `json:"active_envelope"`

	// ExecutionCtx contains the current RetryCount, FeedbackHistory, and CurrentHistory.
	ExecutionCtx ExecutionContext `json:"execution_context"`

	// LogicalFacts are the Datalog facts derived during the last successful reflection.
	LogicalFacts []string `json:"logical_facts,omitempty"`

	// PayloadType stores the Go type name of the payload for reconstruction.
	// This is used during hydration to unmarshal the payload back to its original type.
	PayloadType string `json:"payload_type,omitempty"`
}

// ExecutionContext captures the runtime state of an execution session.
// This is used by the Durable State Manager to preserve execution continuity across restarts.
type ExecutionContext struct {
	// RetryCount tracks the number of retry attempts for the current action.
	RetryCount int `json:"retry_count"`
	// FeedbackHistory stores all feedback messages from previous retry attempts.
	FeedbackHistory []string `json:"feedback_history,omitempty"`
	// CurrentHistory contains the conversation history for this session.
	CurrentHistory []Message `json:"current_history,omitempty"`
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
	ID       string         `json:"id,omitempty"`
	Content  string         `json:"content"`
	Vector   []float32      `json:"vector,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Score    float32        `json:"score,omitempty"` // Re-ranking score
}

// Answer represents a structured system response.
type Answer struct {
	Text string         `json:"text"`
	Meta map[string]any `json:"meta,omitempty"`
}
```
---
## [core/state.go]
```go
package core

import "context"

// State defines the interface for state management.
// It allows components to persist and retrieve execution state.
type State interface {
	// Get retrieves the state for a given session ID.
	Get(ctx context.Context, sessionID string) (*SessionState, error)
	// Set stores the state for a given session ID.
	Set(ctx context.Context, sessionID string, state *SessionState) error
	// Delete removes the state for a given session ID.
	Delete(ctx context.Context, sessionID string) error
}
```
---
## [core/data.go]
```go
package core

// NopStore is a no-op implementation of State for testing.
type NopStore struct{}

func (n *NopStore) Get(ctx context.Context, sessionID string) (*SessionState, error) {
	return nil, nil
}

func (n *NopStore) Set(ctx context.Context, sessionID string, state *SessionState) error {
	return nil
}

func (n *NopStore) Delete(ctx context.Context, sessionID string) error {
	return nil
}
```
---
## [core/logger.go]
```go
package core

import "context"

// Logger defines the logging interface.
type Logger interface {
	// Debug logs a debug message with optional key-value pairs.
	Debug(msg string, kv ...interface{})
	// Info logs an informational message with optional key-value pairs.
	Info(msg string, kv ...interface{})
	// Warn logs a warning message with optional key-value pairs.
	Warn(msg string, kv ...interface{})
	// Error logs an error message with optional key-value pairs.
	Error(msg string, kv ...interface{})
}

// NopLogger is a no-op implementation of Logger for testing.
type NopLogger struct{}

func (n *NopLogger) Debug(msg string, kv ...interface{}) {}
func (n *NopLogger) Info(msg string, kv ...interface{}) {}
func (n *NopLogger) Warn(msg string, kv ...interface{}) {}
func (n *NopLogger) Error(msg string, kv ...interface{}) {}
```
---
## [core/tracer.go]
```go
package core

import "context"

// NopTracer is a no-op implementation of Tracer for testing.
type NopTracer struct{}

func (n *NopTracer) Start(ctx context.Context, name string) (context.Context, Span) {
	return ctx, &NopSpan{}
}

// NopSpan is a no-op implementation of Span for testing.
type NopSpan struct{}

func (s *NopSpan) End() {}
func (s *NopSpan) SetAttributes(attributes map[string]any) {}
func (s *NopSpan) SetStatus(code string, msg string) {}
func (s *NopSpan) RecordError(err error) {}
```
---
## [core/context_facts.go]
```go
package core

import "context"

// ContextWithLogger injects a logger into the context.
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(loggerKey{}, logger)
}

type loggerKey struct{}

// LoggerFromContext retrieves a logger from the context.
// Returns NopLogger if no logger is present.
func LoggerFromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return logger
	}
	return NopLogger{}
}

// WithParentID injects a parent ID into the context for lineage tracking.
func WithParentID(ctx context.Context, parentID string) context.Context {
	return context.WithValue(parentIDKey{}, parentID)
}

type parentIDKey struct{}

// ParentIDFromContext retrieves a parent ID from the context.
func ParentIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(parentIDKey{}).(string); ok {
		return id
	}
	return ""
}
```
---
## [core/context_lineage.go]
```go
package core

import "context"

// LineageKey is the context key for storing lineage information.
type lineageKey struct{}

// WithLineage injects lineage information into the context.
func WithLineage(ctx context.Context, lineage map[string]string) context.Context {
	return context.WithValue(lineageKey{}, lineage)
}

// LineageFromContext retrieves lineage information from the context.
func LineageFromContext(ctx context.Context) map[string]string {
	if lineage, ok := ctx.Value(lineageKey{}).(map[string]string); ok {
		return lineage
	}
	return nil
}
```
---
## [core/embedder.go]
```go
package core

import "context"

// Embedder defines the interface for text embedding operations.
type Embedder interface {
	// Embed generates embeddings for a given text.
	Embed(ctx context.Context, text string) ([]float32, error)
}

// NopEmbedder is a no-op implementation of Embedder for testing.
type NopEmbedder struct{}

func (n *NopEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, nil
}
```
---
## [manglekit.go]
```go
package manglekit

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// --- Aliases ---
type Client = sdk.Client
type ClientOption = sdk.ClientOption
type ExecuteOption = sdk.ExecuteOption

// --- Facade Functions ---

// NewClient initializes a client with defaults.
// It implements the "Batteries Included" philosophy by leveraging SDK defaults.
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	return sdk.NewClient(ctx, opts...)
}

// Must helper for panic-on-error initialization
func Must(c *Client, err error) *Client {
	return sdk.Must(c, err)
}

// Define is the public entry point for creating Actions
func Define[In any, Out any](
	c *Client,
	name string,
	handler func(context.Context, In) (Out, error),
) *sdk.Runnable[In, Out] {
	return sdk.Define(c, name, handler)
}

// --- Option Wrappers ---
func WithBlueprintPath(path string) ClientOption { return sdk.WithBlueprintPath(path) }

// Deprecated: Use WithBlueprintPath instead.
func WithPolicyPath(path string) ClientOption         { return sdk.WithBlueprintPath(path) }
func WithFailMode(mode string) ClientOption           { return sdk.WithFailMode(mode) }
func WithLogger(l core.Logger) ClientOption           { return sdk.WithLogger(l) }
func WithHistory(store core.HistoryStore) ClientOption { return sdk.WithHistory(store) }
func WithMemory(mem core.AgentMemory) ClientOption    { return sdk.WithMemory(mem) }

func WithSessionID(id string) ExecuteOption        { return sdk.WithSessionID(id) }
func WithTransientMemory() ExecuteOption           { return sdk.WithTransientMemory() }
func WithMetadata(key, value string) ExecuteOption { return sdk.WithMetadata(key, value) }
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
	// agentMemory is the unified memory provider (History + RAG).
	agentMemory core.AgentMemory
	// registry holds registered actions for dynamic routing.
	registry map[string]core.Action
	// failureMode determines the system's resilience strategy ("open" or "closed").
	failureMode string
	// blueprintPath stores the path loaded at startup (for debugging/reloading).
	blueprintPath string
	// shutdownFunc is a cleanup function to stop exporters/tracers.
	shutdownFunc func(context.Context) error
	// llm is the plugged-in text generation backend (e.g., Google, OpenAI).
	llm core.TextGenerator
	// stateManager handles durable state persistence and recovery.
	stateManager interface {
		Hydrate(ctx context.Context, sessionID string) (*core.SessionState, error)
		Checkpoint(ctx context.Context, state *core.SessionState) error
		ExtractFacts(ctx context.Context, envelope core.Envelope) ([]string, error)
	}
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

	// Ensure dependencies (Engine/Tracer) are initialized if JIT init didn't happen
	if err := ensureDependencies(c); err != nil {
		return nil, err
	}

	return c, nil
}

// NewClientFromFile initializes a Client by loading configuration from a YAML file.
func NewClientFromFile(ctx context.Context, configPath string, opts ...ClientOption) (*Client, error) {
	// Load configuration from file
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Prepend WithConfig to opts
	newOpts := append([]ClientOption{WithConfig(cfg)}, opts...)
	return NewClient(ctx, newOpts...)
}

// NewClientFromConfig initializes a Client using a pre-loaded Config object.
func NewClientFromConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error) {
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

// Tracer returns the OpenTelemetry Tracer used by the client.
func (c *Client) Tracer() trace.Tracer {
	return c.otelTracer
}

// Logger returns the configured Logger instance.
func (c *Client) Logger() core.Logger {
	return c.logger
}

// NewDefault initializes a Client with sensible default settings.
func NewDefault() (*Client, error) {
	return NewClient(context.Background())
}

// SetLLM manually configures the TextGenerator (LLM) for the client.
// This is useful for code-first wiring or when using provider factories.
func (c *Client) SetLLM(gen core.TextGenerator) {
	c.llm = gen
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

// Shutdown cleans up resources used by the client.
func (c *Client) Shutdown(ctx context.Context) error {
	if c.shutdownFunc != nil {
		return c.shutdownFunc(ctx)
	}
	return nil
}

// Memory returns the active memory provider (if any).
func (c *Client) Memory() core.AgentMemory {
	return c.agentMemory
}
```
---
## [sdk/execute_typed.go]
```go
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
)

// Execute runs a typed action against the client.
// It guarantees that the input matches TIn and attempts to convert the output to TOut.
func Execute[TIn any, TOut any](
	ctx context.Context,
	c *Client,
	handle TypedAction[TIn, TOut],
	input TIn,
) (TOut, error) {
	var zero TOut

	// 1. Delegate to the untyped engine
	// The engine works with core.Envelope and dynamic types.
	env, err := c.ExecuteByName(ctx, handle.Name, input)
	if err != nil {
		return zero, err
	}

	// 2. Type Assertion (Fast Path)
	// If the payload is already the correct pointer or struct, return it.
	if out, ok := env.Payload.(TOut); ok {
		return out, nil
	}

	// 3. Conversion (Slow Path)
	// If the payload is map[string]any (e.g. from JSON/HTTP), we need to convert it to TOut.
	// We use JSON round-trip as a robust fallback.
	return convertPayload[TOut](env.Payload)
}

// convertPayload attempts to convert any payload into T.
func convertPayload[T any](input any) (T, error) {
	var result T

	// Quick check for nil
	if input == nil {
		return result, nil
	}

	// Marshal to JSON
	bytes, err := json.Marshal(input)
	if err != nil {
		return result, fmt.Errorf("failed to marshal payload for type conversion: %w", err)
	}

	// Unmarshal to Target Type
	if err := json.Unmarshal(bytes, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal payload to target type %T: %w", result, err)
	}

	return result, nil
}
```
---
## [sdk/client_execute.go]
```go
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/supervisor"
)

// ExecuteByName executes a named action within the governance loop.
func (c *Client) ExecuteByName(ctx context.Context, actionName string, payload any, opts ...ExecuteOption) (core.Envelope, error) {
	// Find the action in the registry
	action, ok := c.registry[actionName]
	if !ok {
		return core.Envelope{}, fmt.Errorf("action not found: %s", actionName)
	}

	// Apply execute options
	env := core.NewEnvelope(payload)
	for _, opt := range opts {
		if err := opt(&env); err != nil {
			return core.Envelope{}, err
		}
	}

	// Wrap the action with supervision
	supervised := c.Supervise(action)

	// Execute the supervised action
	return supervised.Execute(ctx, env)
}

// ExecuteOption is a functional option for configuring execution.
type ExecuteOption func(*core.Envelope) error

// WithSessionID sets the session ID in the envelope metadata.
func WithSessionID(id string) ExecuteOption {
	return func(env *core.Envelope) error {
		env.SetMeta(core.KeySessionID, id)
		return nil
	}
}

// WithTransientMemory sets a flag to skip memory operations.
func WithTransientMemory() ExecuteOption {
	return func(env *core.Envelope) error {
		env.SetMeta("transient_memory", true)
		return nil
	}
}

// WithMetadata sets a metadata key-value pair.
func WithMetadata(key, value string) ExecuteOption {
	return func(env *core.Envelope) error {
		env.SetMeta(key, value)
		return nil
	}
}
```
---
## [sdk/generics.go]
```go
package sdk

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
)

// Runnable represents a typed action that can be executed by the Client.
type Runnable[In any, Out any] struct {
	client *Client
	name   string
	fn     func(context.Context, In) (Out, error)
}

// Define creates a new typed action from a handler function.
func Define[In any, Out any](
	c *Client,
	name string,
	handler func(context.Context, In) (Out, error),
) *Runnable[In, Out] {
	return &Runnable[In, Out]{
		client: c,
		name:   name,
		fn:     handler,
	}
}

// Metadata returns the action metadata for the runnable.
func (r *Runnable[In, Out]) Metadata() core.ActionMetadata {
	inType := reflect.TypeOf((*Runnable[In, Out])(nil)).In()
	outType := reflect.TypeOf((*Runnable[In, Out])(nil)).Out()

	return core.ActionMetadata{
		Name:        r.name,
		Type:        "typed",
		InputType:   inType.String(),
		OutputType:  outType.String(),
		IsDynamic:   false,
	}
}

// Execute implements the core.Action interface for the runnable.
func (r *Runnable[In, Out]) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Type assert the input
	typedInput, ok := input.Payload.(In)
	if !ok {
		return core.Envelope{}, fmt.Errorf("type assertion failed: expected %T, got %T", reflect.TypeOf((*Runnable[In, Out]).nil, reflect.TypeOf(input.Payload))
	}

	// Call the handler
	result, err := r.fn(ctx, typedInput)
	if err != nil {
		return core.Envelope{}, err
	}

	// Wrap the result in an envelope
	return core.NewEnvelope(result), nil
}
```
---
## [sdk/config.go]
```go
package sdk

import (
	"fmt"

	"github.com/duynguyendang/manglekit/config"
)

// WithConfig sets the configuration for the client.
func WithConfig(cfg *config.Config) ClientOption {
	return func(c *Client) error {
		// Load the blueprint
		if cfg.BlueprintPath != "" {
			if c.engine != nil {
				return fmt.Errorf("engine not initialized for blueprint loading")
			}
			if err := c.engine.LoadPolicy(ctx, cfg.BlueprintPath); err != nil {
				return fmt.Errorf("failed to load blueprint: %w", err)
			}
		}

		// Set the blueprint path
		c.blueprintPath = cfg.BlueprintPath

		// Configure logger
		if cfg.Logger != nil {
			c.logger = cfg.Logger
		}

		// Configure memory
		if cfg.Memory != nil {
			c.agentMemory = cfg.Memory
		}

		// Configure LLM
		if cfg.LLM != nil {
			c.llm = cfg.LLM
		}

		// Configure tracer
		if cfg.Tracer != nil {
			c.tracer = cfg.Tracer
		}

		// Configure failure mode
		if cfg.FailureMode != "" {
			c.failureMode = cfg.FailureMode
		}

		// Configure state manager
		if cfg.StateManager != nil {
			c.stateManager = cfg.StateManager
		}

		return nil
	}
```
---
## [sdk/memory.go]
```go
package sdk

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
)

// HybridMemory combines a history store and vector store for RAG.
type HybridMemory struct {
	historyStore core.State
	vectorStore  core.VectorStore
	embedder    core.Embedder
}

// NewHybridMemory creates a new hybrid memory implementation.
func NewHybridMemory(historyStore core.State, vectorStore core.VectorStore, embedder core.Embedder) *HybridMemory {
	return &HybridMemory{
		historyStore: historyStore,
		vectorStore:  vectorStore,
		embedder:    embedder,
	}
}

// Read retrieves chat history for a session.
func (h *HybridMemory) Read(ctx context.Context, sessionID string) ([]core.Message, error) {
	return h.historyStore.Get(ctx, sessionID)
}

// Append adds messages to the history.
func (h *HybridMemory) Append(ctx context.Context, sessionID string, msgs []core.Message) error {
	return h.historyStore.Set(ctx, sessionID, &core.SessionState{
		SessionID: sessionID,
		ActiveEnvelope: core.Envelope{
			Payload: &core.ConversationHistory{
				Messages: msgs,
			},
		},
	})
}

// Recall retrieves relevant context using RAG.
func (h *HybridMemory) Recall(ctx context.Context, query string) (string, error) {
	// Check if this is an AgentMemoryWithFacts implementation
	if withFacts, ok := h.vectorStore.(core.AgentMemoryWithFacts); ok {
		context, metadata, err := withFacts.RecallWithFacts(ctx, query)
		if err != nil {
			return "", err
		}
		// Format context with metadata
		return formatContext(context, metadata)
	}

	// Embed generates embeddings for the query.
func (h *HybridMemory) Embed(ctx context.Context, text string) ([]float32, error) {
	return h.embedder.Embed(ctx, text)
}

// Memorize stores a query-answer pair for future retrieval.
func (h *HybridMemory) Memorize(ctx context.Context, query string, answer string) error {
	// Generate embeddings
	embeddings, err := h.Embed(ctx, query)
	if err != nil {
		return err
	}

	// Store in vector store (implementation specific)
	// This is a simplified version - real implementation would store in a vector DB
	return h.vectorStore.Set(ctx, core.Document{
		Content:  answer,
		Vector:  embeddings,
	})
}

// Init performs any necessary setup.
func (h *HybridMemory) Init(ctx context.Context) error {
	// Initialize history store if needed
	if init, ok := h.historyStore.(interface{ Init(context.Context) error }); ok {
		return init.Init(ctx)
	}
	// Initialize vector store if needed
	if init, ok := h.vectorStore.(interface{ Init(context.Context) error }); ok {
		return init.Init(ctx)
	}
	return nil
}

// formatContext formats the context with optional metadata.
func formatContext(ctx context.Context, metadata map[string]any) (string, error) {
	// This is a placeholder - real implementation would format the retrieved documents
	return "", nil
}
```
---
## [sdk/helpers.go]
```go
package sdk

import (
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
)

// Must is a helper that panics on error, useful for initialization.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("manglekit initialization failed: %v", err))
	}
	return val
}

// ensureDependencies ensures that required dependencies are initialized.
func ensureDependencies(c *Client) error {
	// Initialize engine if not set
	if c.engine == nil {
		pe, err := engine.New()
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}
		c.engine = pe
	}

	// Initialize tracer if not set
	if c.tracer == nil {
		c.tracer = &core.NopTracer{}
	}

	return nil
}
```
---
## [sdk/loader.go]
```go
package sdk

import (
	"context"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/adapters/extractor"
	"github.com/duynguyendang/manglekit/adapters/func"
	"github.com/duynguyendang/manglekit/adapters/knowledge"
	"github.com/duynguyendang/manglekit/adapters/mcp"
	"github.com/duynguyendang/manglekit/adapters/vector"
	"github.com/duynguyendang/manglekit/core"
)

// LoadActions loads actions from various sources into the client registry.
func LoadActions(ctx context.Context, c *Client, actions ...core.Action) error {
	for _, action := range actions {
		c.RegisterAction(action.Metadata().Name, action)
	}
	return nil
}

// LoadAIActions loads AI actions from a text generator.
func LoadAIActions(ctx context.Context, c *Client, name string, generator core.TextGenerator) error {
	action := ai.NewLLMAction(name, generator)
	return LoadActions(ctx, c, action)
}

// LoadMCPActions loads actions from an MCP server.
func LoadMCPActions(ctx context.Context, c *Client, loader *mcp.Loader) error {
	actions, err := loader.Load(ctx)
	if err != nil {
		return err
	}
	return LoadActions(ctx, c, actions...)
}

// LoadExtractorActions loads an extractor action.
func LoadExtractorActions(ctx context.Context, c *Client, name string, generator core.Action, schema any) error {
	action, err := extractor.New(name, generator, schema)
	if err != nil {
		return err
	}
	return LoadActions(ctx, c, action)
}

// LoadVectorActions loads a vector retrieval action.
func LoadVectorActions(ctx context.Context, c *Client, name string, retriever core.DocumentRetriever) error {
	action := vector.NewRetrieverAction(name, retriever)
	return LoadActions(ctx, c, action)
}

// LoadKnowledgeActions loads knowledge graph facts.
func LoadKnowledgeActions(ctx context.Context, c *Client, loader *knowledge.RDFLoader) error {
	facts, err := loader.Parse("")
	if err != nil {
		return err
	}
	return c.LoadFacts(facts)
}
```
---
## [sdk/executor.go]
```go
package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// Executor handles the execution of actions with retry and routing logic.
type Executor struct {
	client *Client
}

// NewExecutor creates a new executor.
func NewExecutor(c *Client) *Executor {
	return &Executor{client: c}
}

// Execute executes an action with retry and routing support.
func (e *Executor) Execute(ctx context.Context, actionName string, payload any, opts ...ExecuteOption) (core.Envelope, error) {
	// Get the action
	action, ok := e.client.registry[actionName]
	if !ok {
		return core.Envelope{}, fmt.Errorf("action not found: %s", actionName)
	}

	// Apply execute options
	env := core.NewEnvelope(payload)
	for _, opt := range opts {
		if err := opt(&env); err != nil {
			return core.Envelope{}, err
		}
	}

	// Execute the action (with supervision)
	result, err := e.client.Supervise(action).Execute(ctx, env)
	if err != nil {
		return core.Envelope{}, err
	}

	// Check for routing decision
	decision := result.Metadata[core.KeyDecision]
	switch decision {
	case core.DecisionRoute:
		// Route to the next action
		nextAction, ok := result.Metadata[core.KeyNextStep].(string)
		if !ok {
			return core.Envelope{}, fmt.Errorf("route decision without next step")
		}
		return e.Execute(ctx, nextAction, payload, opts...)
	case core.DecisionRetry:
		// Retry with feedback
		// Increment retry count and inject feedback
		retryCount := 0
		if rc, ok := env.Metadata["retry_count"].(int); ok {
			retryCount = rc + 1
		}
		env.SetMeta("retry_count", retryCount)
		env.SetFeedback(result.GetFeedback())
		return e.Execute(ctx, actionName, payload, opts...)
	}

	return result, nil
}
```
---
## [sdk/loop.go]
```go
package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// RunLoop manages the semantic state machine for action execution.
type RunLoop struct {
	client *Client
}

// NewRunLoop creates a new run loop.
func NewRunLoop(c *Client) *RunLoop {
	return &RunLoop{client: c}
}

// Run executes the semantic loop for an action.
func (l *RunLoop) Run(ctx context.Context, actionName string, payload any, opts ...ExecuteOption) (core.Envelope, error) {
	// Get the executor
	executor := NewExecutor(l.client)

	// Execute with retry and routing
	return executor.Execute(ctx, actionName, payload, opts...)
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
// 1. Trace: Starts an OpenTelemetry span for the operation.
// 2. Assess: Checks Pre-Check blueprints (e.g., "infeasible(Req)").
// 3. Execute: Runs the inner action (e.g., calls the LLM).
// 4. Reflect: Checks Post-Check blueprints (e.g., "infeasible(Output)").
// 5. Steering: Evaluates steering blueprints for routing or correction.
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
// 1. Starts a span.
// 2. Injects the logger into the context.
// 3. Runs Assess(). If it fails, execution halts (unless Fail-Open).
// 4. Runs the inner Action.Execute().
// 5. Propagates taint labels from input to output.
// 6. Runs Reflect(). If it fails, the result is blocked.
// 7. Runs EvaluateSteering() to determine next steps (Retry/Route).
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
				attrs := map[string]any{
					core.KeyFeedback: alignErr.Message,
				}
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

// isSensitive checks if the input envelope contains sensitive security labels.
func (g *SupervisedAction) isSensitive(labels []string) bool {
	sensitiveTags := []string{"pii", "secret", "confidential", "auth_token"}
	for _, l := range labels {
		for _, tag := range sensitiveTags {
			if l == tag {
				return true
			}
		}
	}
	}
	return false
}
```
---
## [internal/engine/runtime.go]
```go
package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/google/mangle/analysis"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/engine"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/parse"
)

// MangleRuntime encapsulates the Google Mangle Datalog engine.
type MangleRuntime struct {
	mu sync.RWMutex

	// State fields (protected by mu)
	programInfo   *analysis.ProgramInfo
	strata        []analysis.Nodeset
	predToStratum map[ast.PredicateSym]int
	baseFactStore factstore.SimpleInMemoryStore
	ruleUnits     []parse.SourceUnit
	ready         bool // Flag to indicate if the runtime is initialized
}

// NewMangleRuntime initializes a new, empty MangleRuntime.
func NewMangleRuntime() *MangleRuntime {
	return &MangleRuntime{
		predToStratum: make(map[ast.PredicateSym]int),
		baseFactStore: factstore.NewSimpleInMemoryStore(),
			ready:         false,
	}
}

// Load loads Datalog rules and facts from the specified path.
// CRITICAL CHANGE: This REPLACES the current program state.
func (r *MangleRuntime) Load(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// 1. Resolve files (I/O
