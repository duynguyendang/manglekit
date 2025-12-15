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
	KeyDecision     = "manglekit.decision"  // Values: "PROCEED", "INFEASIBLE", "RETRY", "ROUTE"
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

	// Configuration
	PrefixPromptConfig = "prompt."
)

// Standard Decision Values
const (
	DecisionProceed    = "PROCEED"    // Formerly "ALLOW"
	DecisionInfeasible = "INFEASIBLE" // Formerly "DENY"
	DecisionRetry      = "RETRY"
	DecisionRoute      = "ROUTE"
)

// Datalog System Constants
const (
	EntityInput    = "Req"          // ID for Input Envelope
	EntityOutput   = "Output"       // ID for Output Envelope
	PredInfeasible = "infeasible"   // Predicate for blocking execution
)

// Observability & Trace Attributes
const (
	// Span Names
	SpanPreCheck  = "Datalog.Assess"  // Formerly "Datalog.PreCheck"
	SpanPostCheck = "Datalog.Reflect" // Formerly "Datalog.PostCheck"

	// Attribute Keys
	AttrPolicyName   = "policy.name"
	AttrPolicyType   = "policy.type"
	AttrDecisionType = "decision.type"
	AttrOutcome      = "outcome"       // "PROCEED", "INFEASIBLE"
	AttrLabels       = "mangle.labels" // Taint Propagation
	AttrActionName   = "action.name"
	AttrActionType   = "action.type"
	AttrRuleID       = "mangle.rule_id" // Replaces AttrPolicyRuleID
	AttrAttempt      = "mangle.attempt" // Replaces AttrPolicyAttempt
)

// Outcome Values (for Tracing)
const (
	OutcomeProceed    = "PROCEED"    // Formerly "ALLOWED"
	OutcomeInfeasible = "INFEASIBLE" // Formerly "DENIED"
	OutcomeSuccess    = "success"
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
	Outcome string            // Matches DecisionProceed, DecisionInfeasible, etc.
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

// Answer represents a structured system response.
type Answer struct {
	Text string         `json:"text"`
	Meta map[string]any `json:"meta,omitempty"`
}
