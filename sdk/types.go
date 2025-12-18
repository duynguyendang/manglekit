package sdk

import (
	"github.com/duynguyendang/manglekit/core"
)

// Re-export core types for convenience

// ActionMetadata provides metadata about an action.
type ActionMetadata = core.ActionMetadata

// Envelope is the standard communication structure for actions.
type Envelope = core.Envelope

// NewEnvelope creates a new envelope with the given payload.
// This is a convenience re-export of core.NewEnvelope.
func NewEnvelope(payload any) Envelope {
	return core.NewEnvelope(payload)
}
