package domain

import (
	"iter"
	"time"
)

// Atom is the smallest unit of knowledge (Subject-Predicate-Object).
type Atom struct {
	Predicate    string    `json:"predicate"`
	Subject      string    `json:"subject"`
	Object       string    `json:"object"`
	Weight       float64   `json:"weight"` // 1.0 (Fact) to 0.1 (Guess)
	OriginIntent IntentStr `json:"origin_intent,omitempty"`
}

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
