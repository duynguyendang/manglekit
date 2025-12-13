package core

import "go.opentelemetry.io/otel/attribute"

// Observability & Trace Attributes
// These attributes are used to enrich OpenTelemetry spans with Manglekit policy decisions.
const (
	// High-level Outcome
	AttrPolicyOutcome = attribute.Key("policy.outcome") // "allow", "deny", "route", "retry"

	// Details
	AttrPolicyReason = attribute.Key("policy.reason")  // e.g. "Budget Exceeded"
	AttrPolicyTarget = attribute.Key("policy.target")  // e.g. "tool_calculator"
	AttrPolicyRuleID = attribute.Key("policy.rule_id") // (Optional) If available

	// Retry/Loop info
	AttrPolicyAttempt = attribute.Key("policy.attempt") // e.g. 1, 2
)
