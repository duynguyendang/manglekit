package core

import (
	"github.com/google/uuid"
)

// Envelope struct defines a standard communication structure
type Envelope struct {
	ID             uuid.UUID
	Payload        any
	Metadata       map[string]string
	SecurityLabels []string // New: Taint tags (e.g., "secret", "pii")
}

// NewEnvelope creates a new envelope with a payload
func NewEnvelope(payload any) Envelope {
	return Envelope{
		ID:             uuid.New(),
		Payload:        payload,
		Metadata:       make(map[string]string),
		SecurityLabels: []string{},
	}
}

// SetMeta sets a value in the envelope's metadata
func (e *Envelope) SetMeta(k, v string) {
	e.Metadata[k] = v
}

// GetMeta gets a value from the envelope's metadata
func (e *Envelope) GetMeta(k string) string {
	return e.Metadata[k]
}

// AddLabel adds a security label if it doesn't exist (deduplication).
func (e *Envelope) AddLabel(label string) {
	if !e.HasLabel(label) {
		e.SecurityLabels = append(e.SecurityLabels, label)
	}
}

// HasLabel checks for existence of a security label.
func (e *Envelope) HasLabel(label string) bool {
	for _, l := range e.SecurityLabels {
		if l == label {
			return true
		}
	}
	return false
}

// MergeLabels appends distinct labels from another source.
func (e *Envelope) MergeLabels(other []string) {
	for _, l := range other {
		e.AddLabel(l)
	}
}
