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

	// KeyRiskScore indicates the calculated risk level (0-100).
	// Populated by Risk Engines or Pre-Check rules.
	KeyRiskScore = "manglekit.risk_score"

	// KeyLatencyMs records the execution time of the action in milliseconds.
	KeyLatencyMs = "manglekit.latency_ms"

	// KeyTraceID stores the distributed trace ID for correlation.
	KeyTraceID = "manglekit.trace_id"

	// KeyModel stores the name of the model used (if applicable).
	KeyModel = "manglekit.model"
)

// Standard Decision Values
const (
	DecisionAllow = "ALLOW"
	DecisionDeny  = "DENY"
	DecisionRetry = "RETRY"
)
