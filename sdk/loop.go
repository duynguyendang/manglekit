package sdk

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	engine_memory "github.com/duynguyendang/manglekit/engine/memory"
)

// ExecuteByName executes a registered action by its name, handling the Semantic State Machine loop.
// It supports steering decisions (RETRY, ROUTE) allowing for self-correction and multi-step flows.
//
// Parameters:
//   - ctx: The execution context.
//   - actionName: The name of the initial action to execute (must be registered via RegisterAction).
//   - input: The initial input payload.
//   - opts: Execution options (SessionID, Metadata, etc.).
//
// Returns:
//   - The final result Envelope, or an error if the flow fails or exceeds max steps.
func (c *Client) ExecuteByName(ctx context.Context, actionName string, input any, opts ...ExecuteOption) (core.Envelope, error) {
	params := ExecutionParams{
		MemoryMode: core.MemoryModeNone, // Default to stateless
	}
	for _, opt := range opts {
		opt(&params)
	}

	return c.runLoopInternal(ctx, actionName, input, params)
}

// runLoopInternal implements the core loop of the Semantic State Machine.
// It iterates up to a maximum depth (10) to prevent infinite loops.
//
// Lifecycle:
//  1. Resolve Action: Find the action in the registry.
//  2. Prepare Envelope: Inject payload, history, and feedback from previous steps.
//  3. Execute: Run the protected action (Guard -> Engine -> Inner Action).
//  4. Update History: Append the interaction to the conversation history.
//  5. Handle Decision:
//     - RETRY: Re-run the same action with feedback (self-correction).
//     - ROUTE: Execute a different action (chaining).
//     - ALLOW: Return the result (success).
//     - DENY: Return an error.
func (c *Client) runLoopInternal(ctx context.Context, startAction string, payload any, params ExecutionParams) (core.Envelope, error) {
	// 1. Determine Store Strategy
	var currentHistory []core.ChatMessage
	var store core.MemoryStore

	switch params.MemoryMode {
	case core.MemoryModePersist:
		store = c.memory
		// Hydrate immediately
		if params.SessionID != "" {
			var err error
			currentHistory, err = store.Read(ctx, params.SessionID)
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
	var lastFeedback string

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
		// Set ContentType from Action Metadata
		env.ContentType = action.Metadata().InputContentType

		// CRITICAL: Pass feedback to the action
		if len(feedbackHistory) > 0 {
			env.Metadata[core.KeyPrevFeedback] = strings.Join(feedbackHistory, "; ")
		}

		// Inject specific Mangle feedback (Teacher-Student Protocol)
		if lastFeedback != "" {
			env.Metadata["mangle_feedback"] = lastFeedback
		}

		// Inject History
		if len(currentHistory) > 0 && params.MemoryMode != core.MemoryModeNone {
			env.SetHistory(currentHistory)
		}

		// Inject Metadata (e.g. source, user_tier)
		if params.Metadata != nil {
			for k, v := range params.Metadata {
				env.Metadata[k] = v
			}
		}

		// 3. Execute (Guard -> Engine -> Steering)
		result, err := action.Execute(ctx, env)
		if err != nil {
			// Check for PolicyViolationError (Teacher-Student Protocol)
			var pve *core.PolicyViolationError
			if errors.As(err, &pve) {
				lastFeedback = pve.Message
				if c.logger != nil {
					c.logger.Info("RunLoop: Policy Violation detected, triggering retry with feedback", "feedback", lastFeedback)
				}
				continue // Trigger retry (Teacher-Student Protocol)
			}
			return core.Envelope{}, err
		}

		// Clear feedback on success
		lastFeedback = ""

		// 4. Update History (Append User Input + Assistant Response)
		if params.MemoryMode != core.MemoryModeNone {
			// Note: This assumes simplified text-in/text-out for the prompt.
			newExchange := []core.ChatMessage{
				{Role: "user", Content: fmt.Sprintf("%v", currentPayload)},
				{Role: "assistant", Content: fmt.Sprintf("%v", result.Payload)},
			}
			currentHistory = append(currentHistory, newExchange...)
		}

		// 5. Persist (Async or Sync based on requirements)
		if params.SessionID != "" && params.MemoryMode == core.MemoryModePersist {
			if err := store.Write(ctx, params.SessionID, currentHistory); err != nil {
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
