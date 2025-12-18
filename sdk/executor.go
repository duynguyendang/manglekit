package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// ExecutePlan executes a generated plan sequentially.
// It chains the output of one step as the input to the next.
//
// Parameters:
//   - ctx: The execution context.
//   - steps: The ordered list of steps to execute.
//   - initialInput: The input envelope for the first step.
//
// Returns:
//   - The final output envelope.
//   - An error if any step fails.
func (c *Client) ExecutePlan(ctx context.Context, steps []PlanStep, initialInput core.Envelope) (core.Envelope, error) {
	currentEnvelope := initialInput

	for i, step := range steps {
		if c.logger != nil {
			c.logger.Info("Executing plan step", "step", i+1, "total", len(steps), "action", step.ActionName)
		}

		// Execute the action by name
		// We extract the payload from the current envelope to pass as input.
		// ExecuteByName will wrap it in a new Envelope, preserving metadata if we handle it correctly.
		// However, ExecuteByName takes `input any`. It doesn't take an Envelope.
		// But it returns an Envelope.
		// We should propagate metadata.
		// `ExecuteByName` internally creates `NewEnvelope(input)`.
		// It accepts `opts ...ExecuteOption`.
		// We can pass metadata via `WithMetadata`.

		result, err := c.ExecuteByName(ctx, step.ActionName, currentEnvelope.Payload, WithMetadataMap(currentEnvelope.Metadata))
		if err != nil {
			return core.Envelope{}, fmt.Errorf("step %d (%s) failed: %w", i+1, step.ActionName, err)
		}

		// Pass output to next step
		currentEnvelope = result
	}

	return currentEnvelope, nil
}
