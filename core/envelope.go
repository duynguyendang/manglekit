package core

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Envelope struct defines a standard communication structure used across Manglekit.
// It encapsulates data, metadata, and security context (taint labels) to ensure
// safe and traceable propagation through the system.
type Envelope struct {
	// ID is the unique identifier for this specific data envelope.
	ID uuid.UUID
	// Payload is the actual data being transported (e.g., a string, a struct, or a map).
	Payload any
	// Metadata stores key-value pairs for control plane signaling (e.g., decision, latency).
	Metadata map[string]string
	// SecurityLabels holds taint tags (e.g., "secret", "pii") for information flow control.
	SecurityLabels []string
	// ContentType indicates whether the payload is a Struct or JSON.
	ContentType ContentType
}

// NewEnvelope creates a new envelope with the provided payload.
// It initializes a new UUID, an empty metadata map, and an empty list of security labels.
//
// Parameters:
//   - payload: The data to be wrapped in the envelope.
//
// Returns:
//   - A new Envelope instance.
func NewEnvelope(payload any) Envelope {
	return Envelope{
		ID:             uuid.New(),
		Payload:        payload,
		Metadata:       make(map[string]string),
		SecurityLabels: []string{},
		ContentType:    TypeStruct, // Default to Typed Mode
	}
}

// SetMeta sets a value in the envelope's metadata map.
//
// Parameters:
//   - k: The metadata key (e.g., core.KeyDecision).
//   - v: The metadata value.
func (e *Envelope) SetMeta(k, v string) {
	e.Metadata[k] = v
}

// GetMeta retrieves a value from the envelope's metadata map.
//
// Parameters:
//   - k: The metadata key to retrieve.
//
// Returns:
//   - The value associated with the key, or an empty string if not found.
func (e *Envelope) GetMeta(k string) string {
	return e.Metadata[k]
}

// AddLabel adds a security label to the envelope if it does not already exist.
// This is used for taint tracking (e.g., marking data as "secret").
//
// Parameters:
//   - label: The security label string to add.
func (e *Envelope) AddLabel(label string) {
	if !e.HasLabel(label) {
		e.SecurityLabels = append(e.SecurityLabels, label)
	}
}

// HasLabel checks for the existence of a specific security label on the envelope.
//
// Parameters:
//   - label: The security label to check for.
//
// Returns:
//   - true if the label exists, false otherwise.
func (e *Envelope) HasLabel(label string) bool {
	for _, l := range e.SecurityLabels {
		if l == label {
			return true
		}
	}
	return false
}

// MergeLabels appends distinct labels from another source (e.g., another Envelope) to this one.
//
// Parameters:
//   - other: A slice of label strings to merge.
func (e *Envelope) MergeLabels(other []string) {
	for _, l := range other {
		e.AddLabel(l)
	}
}

// SetHistory serializes a list of chat messages into the envelope's metadata.
// This is used to persist conversation context across stateless executions.
//
// Parameters:
//   - msgs: The slice of ChatMessage objects to serialize.
func (e *Envelope) SetHistory(msgs []ChatMessage) {
	b, err := json.Marshal(msgs)
	if err == nil {
		e.SetMeta(KeyHistory, string(b))
	}
}
