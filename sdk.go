package manglekit

// This file re-exports the core Manglekit types and functions for convenience.
// The main Client API is defined in manglekit.go.
//
// Usage:
//
//	import "github.com/duynguyendang/manglekit"
//
//	client, err := manglekit.NewClient(ctx, "policy.dlog")
//	protectedAction := client.Protect(myAction)

import (
	"github.com/duynguyendang/manglekit/core"
)

// Re-export core types for convenience
type (
	// Action is the interface for a unit of work that can be protected by policies.
	Action = core.Action

	// ActionMetadata provides metadata about an action.
	ActionMetadata = core.ActionMetadata

	// Envelope is the standard communication structure for actions.
	Envelope = core.Envelope
)

// NewEnvelope creates a new envelope with the given payload.
// This is a convenience re-export of core.NewEnvelope.
func NewEnvelope(payload any) Envelope {
	return core.NewEnvelope(payload)
}
