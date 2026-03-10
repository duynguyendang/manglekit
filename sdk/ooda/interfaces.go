package ooda

import "context"

// Observer defines the capability to analyze, classify, and normalize raw inputs.
type Observer interface {
	Observe(ctx context.Context, frame *CognitiveFrame) error
}

// Orienter defines the capability to retrieve domain context, rules, or historical knowledge.
type Orienter interface {
	Orient(ctx context.Context, frame *CognitiveFrame) error
}

// Decider defines the capability to formulate a plan or structure based on observation and orientation.
type Decider interface {
	Decide(ctx context.Context, frame *CognitiveFrame) error
}

// Verifier defines the capability to validate the proposed plan before execution.
type Verifier interface {
	Verify(ctx context.Context, frame *CognitiveFrame) error
}

// Actor defines the capability to generate the final artifacts or side-effects.
type Actor interface {
	Act(ctx context.Context, frame *CognitiveFrame) error
}
