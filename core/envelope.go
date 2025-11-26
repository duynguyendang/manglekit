package core

import (
	"github.com/google/uuid"
)

// Envelope struct defines a standard communication structure
type Envelope struct {
	ID       uuid.UUID
	Payload  any
	Metadata map[string]string
}

// NewEnvelope creates a new envelope with a payload
func NewEnvelope(payload any) Envelope {
	return Envelope{
		ID:       uuid.New(),
		Payload:  payload,
		Metadata: make(map[string]string),
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
