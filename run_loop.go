package manglekit

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
)

// RunLoop executes a Semantic State Machine starting from startActionName.
// It handles steering decisions (ALLOW, RETRY, ROUTE) returned by the Policy Engine.
func (c *Client) RunLoop(ctx context.Context, startAction string, payload any) (core.Envelope, error) {
	currentAction := startAction
	currentPayload := payload
	var feedbackHistory []string

	for step := 0; step < 10; step++ { // Max depth 10
		if c.logger != nil {
			c.logger.Info("RunLoop step", "step", step, "action", currentAction)
		}

		// 1. Resolve Action
		action, ok := c.registry[currentAction]
		if !ok {
			return core.Envelope{}, fmt.Errorf("action not found: %s", currentAction)
		}

		// 2. Prepare Envelope
		env := core.NewEnvelope(currentPayload)
		// CRITICAL: Pass feedback to the action (as seen in the example)
		if len(feedbackHistory) > 0 {
			env.Metadata[core.KeyPrevFeedback] = strings.Join(feedbackHistory, "; ")
		}

		// 3. Execute (Guard -> Engine -> Steering)
		// The action (GuardedAction) is responsible for running EvaluateSteering and
		// populating Metadata[KeyDecision], KeyFeedback, KeyNextStep.
		result, err := action.Execute(ctx, env)
		// Note: If Engine blocks (DENY), Guard returns error.
		if err != nil {
			return core.Envelope{}, err
		}

		// 4. Handle Decision
		decision := result.Metadata[core.KeyDecision]

		if c.logger != nil {
			c.logger.Debug("RunLoop decision", "decision", decision, "step", step)
		}

		switch decision {
		case core.DecisionRetry:
			// Self-Correction Loop
			hint := result.Metadata[core.KeyFeedback]
			feedbackHistory = append(feedbackHistory, hint)
			if c.logger != nil {
				c.logger.Info("RunLoop: RETRY triggered", "feedback", hint)
			}
			continue // Loop again with same action, new feedback

		case core.DecisionRoute:
			// Dynamic Dispatch
			next := result.Metadata[core.KeyNextStep]
			currentAction = next
			currentPayload = result.Payload // Pipe output to next input
			feedbackHistory = nil           // Reset feedback
			if c.logger != nil {
				c.logger.Info("RunLoop: ROUTE triggered", "next", next)
			}
			continue

		case core.DecisionAllow, "":
			// Done
			return result, nil

		case core.DecisionDeny:
			// Should have been caught by err != nil check if Guard returned error,
			// but if Guard returned success with DENY metadata (unlikely), we handle it.
			return core.Envelope{}, fmt.Errorf("action denied")
		}
	}
	return core.Envelope{}, fmt.Errorf("max steps exceeded")
}
