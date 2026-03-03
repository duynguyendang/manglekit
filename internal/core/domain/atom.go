package domain

import (
	"iter"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

// Atom is an alias to the core Atom type.
type Atom = core.Atom

// Payload is a zero-allocation stream of Atoms.
type Payload iter.Seq[Atom]

// Signal represents an event triggering the OODA loop.
type Signal struct {
	ID         string    `json:"id"`
	Source     PortType  `json:"source"`
	Timestamp  time.Time `json:"timestamp"`
	RawContent string    `json:"raw_content"`
	Intent     IntentStr `json:"intent,omitempty"`
	IntentHint string    `json:"intent_hint,omitempty"`
	IsProposal bool      `json:"is_proposal,omitempty"`
}
