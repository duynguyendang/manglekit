package core

// Standard Metadata Keys used for Control Plane signaling.
// These keys allow decoupled components (Validator, Router, LLM) to understand
// the state and intent of the data flow.
const (
	// KeyDecision indicates the governance outcome.
	// Values: "ALLOW", "DENY", "RETRY", "ROUTE".
	KeyDecision = "manglekit.decision"

	// KeyFeedback provides human/machine-readable reasons for the decision.
	// Useful for LLM Self-Correction loops.
	KeyFeedback = "manglekit.feedback"

	// KeyPrevFeedback is used to inject feedback into the next input.
	KeyPrevFeedback = "prev_feedback"

	// KeyNextStep provides the name of the next action to route to.
	KeyNextStep = "manglekit.next_step"

	// KeyRiskScore indicates the calculated risk level (0-100).
	// Populated by Risk Engines or Pre-Check rules.
	KeyRiskScore = "manglekit.risk_score"

	// KeyLatencyMs records the execution time of the action in milliseconds.
	KeyLatencyMs = "manglekit.latency_ms"

	// KeyTraceID stores the distributed trace ID for correlation.
	KeyTraceID = "manglekit.trace_id"

	// KeyModel stores the name of the model used (if applicable).
	KeyModel = "manglekit.model"

	// KeyHistory stores serialized chat history.
	KeyHistory = "manglekit_history"
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
	// Entity IDs used during Reflection/Querying
	EntityInput  = "Req"    // The ID representing the Input Envelope
	EntityOutput = "Output" // The ID representing the Output Envelope
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
	AttrLabels       = "mangle.labels" // For Taint Propagation
	AttrActionName   = "action.name"
	AttrActionType   = "action.type"
)

// Outcome Values (for Tracing)
const (
	OutcomeAllowed = "ALLOWED"
	OutcomeDenied  = "DENIED"
	OutcomeSuccess = "success"
)
