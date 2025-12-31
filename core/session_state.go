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

// MarshalJSON implements custom JSON marshaling for SessionState.
// It handles the serialization of the Envelope's Payload field, which can be any type.
func (s *SessionState) MarshalJSON() ([]byte, error) {
	type Alias SessionState

	// Create a copy with the payload type captured
	if s.ActiveEnvelope.Payload != nil {
		s.PayloadType = fmt.Sprintf("%T", s.ActiveEnvelope.Payload)
	}

	return json.Marshal((*Alias)(s))
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
