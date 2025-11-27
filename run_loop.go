package manglekit

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	engine_memory "github.com/duynguyendang/manglekit/engine/memory"
)

type RunLoopOptions struct {
	SessionID  string
	MemoryMode core.MemoryMode
}

// RunLoop executes a Semantic State Machine starting from startActionName.
// It handles steering decisions (ALLOW, RETRY, ROUTE) returned by the Policy Engine.
func (c *Client) RunLoop(ctx context.Context, startAction string, payload any, opts RunLoopOptions) (core.Envelope, error) {
	// 1. Determine Store Strategy
	var currentHistory []core.ChatMessage
	var store core.MemoryStore

	switch opts.MemoryMode {
	case core.MemoryModePersist:
		store = c.memory
		// Hydrate immediately
		if opts.SessionID != "" {
			var err error
			currentHistory, err = store.Read(ctx, opts.SessionID)
			if err != nil {
				if c.logger != nil {
					c.logger.Warn("RunLoop failed to hydrate history", "error", err)
				}
			}
		}
	case core.MemoryModeTransient:
		store = &engine_memory.VolatileStore{} // Ephemeral map
	default: // None
		store = &core.NoOpStore{}
	}

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
		// CRITICAL: Pass feedback to the action
		if len(feedbackHistory) > 0 {
			env.Metadata[core.KeyPrevFeedback] = strings.Join(feedbackHistory, "; ")
		}

		// Inject History
		if len(currentHistory) > 0 && opts.MemoryMode != core.MemoryModeNone {
			env.SetHistory(currentHistory)
		}

		// 3. Execute (Guard -> Engine -> Steering)
		result, err := action.Execute(ctx, env)
		if err != nil {
			return core.Envelope{}, err
		}

		// 4. Update History (Append User Input + Assistant Response)
		if opts.MemoryMode != core.MemoryModeNone {
			// Note: This assumes simplified text-in/text-out for the prompt.
			newExchange := []core.ChatMessage{
				{Role: "user", Content: fmt.Sprintf("%v", currentPayload)},
				{Role: "assistant", Content: fmt.Sprintf("%v", result.Payload)},
			}
			currentHistory = append(currentHistory, newExchange...)
		}

		// 5. Persist (Async or Sync based on requirements)
		if opts.SessionID != "" && opts.MemoryMode == core.MemoryModePersist {
			if err := store.Write(ctx, opts.SessionID, currentHistory); err != nil {
				if c.logger != nil {
					c.logger.Warn("RunLoop failed to persist history", "error", err)
				}
			}
		}

		// 6. Handle Decision
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
			return core.Envelope{}, fmt.Errorf("action denied")
		}
	}
	return core.Envelope{}, fmt.Errorf("max steps exceeded")
}
