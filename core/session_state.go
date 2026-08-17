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

	// AuditRecords captures the governance audit trail across steps.
	// Each entry records which rules matched, their tier, and the decision made.
	// Appended per step — never overwritten.
	AuditRecords []AuditRecord `json:"audit_records,omitempty"`
}

// AuditRecord is a trimmed, serializable snapshot of AuditTrail for persistence.
// It captures the essential rule attribution data without internal pointers.
type AuditRecord struct {
	Step         int            `json:"step"`
	Rules        []RuleSnapshot `json:"rules"`
	Outcome      string         `json:"outcome"` // "PROCEED", "HALT", "RETRY", "ROUTE"
	Timestamp    string         `json:"timestamp"`
	LatencyMs    int64          `json:"latency_ms"`
	FactCount    int            `json:"fact_count"`
	MatchedCount int            `json:"matched_count"`
}

// RuleSnapshot is a serializable snapshot of a single matched rule.
type RuleSnapshot struct {
	RuleName   string            `json:"rule_name"`
	Tier       string            `json:"tier"`
	Definition string            `json:"definition,omitempty"`
	SourceFile string            `json:"source_file,omitempty"`
	Predicate  string            `json:"predicate,omitempty"`
	Bindings   map[string]string `json:"bindings,omitempty"`
}

// NewAuditRecordFromTrail creates an AuditRecord from an AuditTrail for the given step.
func NewAuditRecordFromTrail(trail *AuditTrail, step int, outcome string) AuditRecord {
	if trail == nil {
		return AuditRecord{Step: step, Outcome: outcome}
	}

	rules := make([]RuleSnapshot, 0, len(trail.MatchedRules))
	for _, r := range trail.MatchedRules {
		rules = append(rules, RuleSnapshot{
			RuleName:   r.RuleName,
			Tier:       string(r.Tier),
			Definition: r.Definition,
			SourceFile: r.SourceFile,
			Predicate:  r.Predicate,
			Bindings:   r.Bindings,
		})
	}

	return AuditRecord{
		Step:         step,
		Rules:        rules,
		Outcome:      outcome,
		Timestamp:    trail.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		LatencyMs:    trail.LatencyMs,
		FactCount:    trail.FactCount,
		MatchedCount: trail.MatchedCount,
	}
}

// MarshalJSON implements custom JSON marshaling for SessionState.
// It handles the serialization of the Envelope's Payload field, which can be any type.
func (s *SessionState) MarshalJSON() ([]byte, error) {
	type Alias SessionState

	clone := *s
	if clone.ActiveEnvelope.Payload != nil {
		clone.PayloadType = fmt.Sprintf("%T", clone.ActiveEnvelope.Payload)
	}

	return json.Marshal((*Alias)(&clone))
}

// UnmarshalJSON implements custom JSON unmarshaling for SessionState.
// It handles the deserialization of the Envelope's Payload field.
func (s *SessionState) UnmarshalJSON(data []byte) error {
	type Alias SessionState
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return fmt.Errorf("failed to unmarshal session state: %w", err)
	}

	return nil
}

// Clone creates a deep copy of the SessionState.
func (s *SessionState) Clone() *SessionState {
	if s == nil {
		return nil
	}

	clone := &SessionState{
		SessionID:   s.SessionID,
		PayloadType: s.PayloadType,
	}

	// Clone envelope
	clone.ActiveEnvelope = Envelope{
		ID:          s.ActiveEnvelope.ID,
		Payload:     s.ActiveEnvelope.Payload, // Shallow copy - caller must handle deep copy if needed
		ContentType: s.ActiveEnvelope.ContentType,
		Error:       s.ActiveEnvelope.Error,
	}

	// Deep copy metadata
	if s.ActiveEnvelope.Metadata != nil {
		clone.ActiveEnvelope.Metadata = make(map[string]any, len(s.ActiveEnvelope.Metadata))
		for k, v := range s.ActiveEnvelope.Metadata {
			clone.ActiveEnvelope.Metadata[k] = v
		}
	}

	// Deep copy security labels
	if len(s.ActiveEnvelope.SecurityLabels) > 0 {
		clone.ActiveEnvelope.SecurityLabels = make([]string, len(s.ActiveEnvelope.SecurityLabels))
		copy(clone.ActiveEnvelope.SecurityLabels, s.ActiveEnvelope.SecurityLabels)
	}

	// Deep copy facts
	if len(s.ActiveEnvelope.Facts) > 0 {
		clone.ActiveEnvelope.Facts = make([]string, len(s.ActiveEnvelope.Facts))
		copy(clone.ActiveEnvelope.Facts, s.ActiveEnvelope.Facts)
	}

	// Clone execution context
	clone.ExecutionCtx = ExecutionContext{
		RetryCount: s.ExecutionCtx.RetryCount,
	}

	if len(s.ExecutionCtx.FeedbackHistory) > 0 {
		clone.ExecutionCtx.FeedbackHistory = make([]string, len(s.ExecutionCtx.FeedbackHistory))
		copy(clone.ExecutionCtx.FeedbackHistory, s.ExecutionCtx.FeedbackHistory)
	}

	if len(s.ExecutionCtx.CurrentHistory) > 0 {
		clone.ExecutionCtx.CurrentHistory = make([]Message, len(s.ExecutionCtx.CurrentHistory))
		copy(clone.ExecutionCtx.CurrentHistory, s.ExecutionCtx.CurrentHistory)
	}

	// Deep copy logical facts
	if len(s.LogicalFacts) > 0 {
		clone.LogicalFacts = make([]string, len(s.LogicalFacts))
		copy(clone.LogicalFacts, s.LogicalFacts)
	}

	// Deep copy audit records
	if len(s.AuditRecords) > 0 {
		clone.AuditRecords = make([]AuditRecord, len(s.AuditRecords))
		for i, rec := range s.AuditRecords {
			cloneRec := AuditRecord{
				Step:         rec.Step,
				Outcome:      rec.Outcome,
				Timestamp:    rec.Timestamp,
				LatencyMs:    rec.LatencyMs,
				FactCount:    rec.FactCount,
				MatchedCount: rec.MatchedCount,
			}
			if len(rec.Rules) > 0 {
				cloneRec.Rules = make([]RuleSnapshot, len(rec.Rules))
				for j, rule := range rec.Rules {
					cloneRec.Rules[j] = RuleSnapshot{
						RuleName:   rule.RuleName,
						Tier:       rule.Tier,
						Definition: rule.Definition,
						SourceFile: rule.SourceFile,
						Predicate:  rule.Predicate,
					}
					if len(rule.Bindings) > 0 {
						cloneRec.Rules[j].Bindings = make(map[string]string, len(rule.Bindings))
						for k, v := range rule.Bindings {
							cloneRec.Rules[j].Bindings[k] = v
						}
					}
				}
			}
			clone.AuditRecords[i] = cloneRec
		}
	}

	return clone
}

// Validate checks if the SessionState is valid and complete.
func (s *SessionState) Validate() error {
	if s.SessionID == "" {
		return fmt.Errorf("session_id cannot be empty")
	}

	if s.ActiveEnvelope.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return fmt.Errorf("active_envelope must have a valid ID")
	}

	return nil
}
