package ooda

import (
	"context"
	"fmt"
)

// Loop manages the OODA loop execution sequence.
// It orchestrates Observer, Orienter, Decider, Verifier, and Actor capabilities.
type Loop struct {
	observer Observer
	orienter Orienter
	decider  Decider
	verifier Verifier
	actor    Actor
}

// NewLoop creates a new Loop engine.
func NewLoop(observer Observer, orienter Orienter, decider Decider, verifier Verifier, actor Actor) *Loop {
	return &Loop{
		observer: observer,
		orienter: orienter,
		decider:  decider,
		verifier: verifier,
		actor:    actor,
	}
}

// Run executes the full OODA sequence with the provided CognitiveFrame.
// If the frame is nil, a new one is instantiated from the input.
func (l *Loop) Run(ctx context.Context, input string, initialFrame *CognitiveFrame) (*CognitiveFrame, error) {
	frame := initialFrame
	if frame == nil {
		frame = NewCognitiveFrame(input, "", TaskTypeGeneration)
	} else {
		frame.Input = input
	}

	phases := []Phase{PhaseObserve, PhaseOrient, PhaseDecide, PhaseVerify, PhaseAct}

	for _, phase := range phases {
		frame.Phase = phase

		var err error
		switch phase {
		case PhaseObserve:
			if l.observer != nil {
				err = l.observer.Observe(ctx, frame)
			}
		case PhaseOrient:
			if l.orienter != nil {
				err = l.orienter.Orient(ctx, frame)
			}
		case PhaseDecide:
			if l.decider != nil {
				err = l.decider.Decide(ctx, frame)
			}
		case PhaseVerify:
			if l.verifier != nil {
				err = l.verifier.Verify(ctx, frame)
			}
		case PhaseAct:
			if l.actor != nil {
				err = l.actor.Act(ctx, frame)
			}
		}

		if err != nil {
			return frame, fmt.Errorf("OODA loop failed at phase %s: %w", phase, err)
		}
	}

	return frame, nil
}
