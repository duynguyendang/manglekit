package core

import (
	"encoding/json"
	"fmt"
	"time"

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

// IntentStr represents the derived intent of a signal.
type IntentStr string

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

// Outcome Values (for Tracing) - aliases to Decision constants for compatibility
const (
	OutcomeProceed = DecisionProceed
	OutcomeHalt    = DecisionHalt
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

	// ContextFacts contains flattened quad facts for Datalog evaluation.
	ContextFacts []Quad `json:"context_facts,omitempty"`
	// Violations holds GenePool axiom violations after Shadow Audit.
	Violations []ViolationRule `json:"violations,omitempty"`
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
	Outcome    string            // Matches DecisionProceed, DecisionHalt, etc.
	Target     string            // Used if Outcome == DecisionRoute
	Reasons    []string          // Explanations
	Meta       map[string]string // Side-channel data (risk scores, latency budget)
	AuditTrail *AuditTrail       // Detailed execution trace
	Action     *ActionEnvelope   // Action to be executed (decoupled from Go implementation)
}

// ActionEnvelope contains the action to be executed and its parameters.
// This decouples Manglekit's logical decisions from their Go implementations.
type ActionEnvelope struct {
	Name      string                 // Action name matching Registry (e.g., "generate_csd")
	Arguments map[string]interface{} // Parameters derived from Datalog bindings
	SessionID string                 // Session ID for accessing TransientStore
}

// NewActionEnvelope creates a new ActionEnvelope.
func NewActionEnvelope(name string, args map[string]interface{}) *ActionEnvelope {
	if args == nil {
		args = make(map[string]interface{})
	}
	return &ActionEnvelope{
		Name:      name,
		Arguments: args,
	}
}

// WithSessionID sets the session ID for the action envelope.
func (a *ActionEnvelope) WithSessionID(sessionID string) *ActionEnvelope {
	a.SessionID = sessionID
	return a
}

// AuditTrail provides detailed explanation of why a decision was made.
// It captures which Datalog rules were triggered and their source tier.
type AuditTrail struct {
	MatchedRules []RuleInference // The rules that matched and contributed to the decision
	Timestamp    time.Time       // When the decision was made
	EngineID     string          // Identifier for the policy engine instance
	Query        string          // The original query that was evaluated
	FactCount    int             // Number of facts evaluated
	MatchedCount int             // Number of matching results
	LatencyMs    int64           // Time taken to evaluate (for performance monitoring)
}

// RuleInference represents a single rule that contributed to a decision.
type RuleInference struct {
	RuleName   string            // The name/ID of the rule (e.g., "can_execute")
	Tier       Tier              // The governance tier (T0_Axiom, T1_Governance, T2_Playbook, T3_User)
	Definition string            // The original Datalog rule definition
	SourceFile string            // Source file where the rule was defined
	Bindings   map[string]string // Variable bindings from unification (e.g., X="agent-001")
	Predicate  string            // The predicate name that matched
}

// Tier represents the governance tier level.
type Tier string

const (
	TierT0_Axiom      Tier = "T0" // Kernel axioms (hard laws)
	TierT1_Governance Tier = "T1" // Governance policies
	TierT2_Playbook   Tier = "T2" // Playbook rules
	TierT3_User       Tier = "T3" // User input / dynamic
	TierUnknown       Tier = "Unknown"
)

// NewAuditTrail creates a new AuditTrail with default values.
func NewAuditTrail(engineID, query string) *AuditTrail {
	return &AuditTrail{
		MatchedRules: make([]RuleInference, 0),
		Timestamp:    time.Now(),
		EngineID:     engineID,
		Query:        query,
	}
}

// AddRule adds a matched rule to the audit trail.
func (a *AuditTrail) AddRule(ruleName, definition, sourceFile, predicate string, tier Tier, bindings map[string]string) {
	a.MatchedRules = append(a.MatchedRules, RuleInference{
		RuleName:   ruleName,
		Tier:       tier,
		Definition: definition,
		SourceFile: sourceFile,
		Bindings:   bindings,
		Predicate:  predicate,
	})
}

// Summary returns a human-readable summary of the audit trail.
func (a *AuditTrail) Summary() string {
	if len(a.MatchedRules) == 0 {
		return "No rules matched"
	}

	summary := fmt.Sprintf("%d rule(s) matched:\n", len(a.MatchedRules))
	for i, rule := range a.MatchedRules {
		summary += fmt.Sprintf("  %d. %s (%s) - %s\n", i+1, rule.RuleName, rule.Tier, rule.SourceFile)
	}
	return summary
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
	// Metadata stores arbitrary per-request metadata (e.g., middleware config).
	Metadata map[string]any
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

// Quad represents a discrete Subject-Predicate-Object-Graph truth used in persistent storage.
type Quad struct {
	Subject   string // Source entity (e.g., "User:Bob")
	Predicate string // Relationship (e.g., "has_role")
	Object    string // Target value (e.g., "Admin", "42")
	Graph     string // Namespace/Context (e.g., "global")
}

// Atom is the smallest unit of knowledge (Subject-Predicate-Object).
type Atom struct {
	Predicate    string    `json:"predicate"`
	Subject      string    `json:"subject"`
	Object       string    `json:"object"`
	Weight       float64   `json:"weight"` // 1.0 (Fact) to 0.1 (Guess)
	OriginIntent IntentStr `json:"origin_intent,omitempty"`
}

// ViolationRule represents a failed policy invariant.
type ViolationRule struct {
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	Severity    int    `json:"severity"` // 0 = Halt, 1 = Retry, 2 = Warn
}
