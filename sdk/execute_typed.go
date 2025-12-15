package sdk

import (
	"context"
	"encoding/json"
	"fmt"
)

// Execute runs a typed action against the client.
// It guarantees that the input matches TIn and attempts to convert the output to TOut.
func Execute[TIn any, TOut any](
	ctx context.Context,
	c *Client,
	handle TypedAction[TIn, TOut],
	input TIn,
) (TOut, error) {
	var zero TOut

	// 1. Delegate to the untyped engine
	// The engine works with core.Envelope and dynamic types.
	env, err := c.ExecuteByName(ctx, handle.Name, input)
	if err != nil {
		return zero, err
	}

	// 2. Type Assertion (Fast Path)
	// If the payload is already the correct pointer or struct, return it.
	if out, ok := env.Payload.(TOut); ok {
		return out, nil
	}

	// 3. Conversion (Slow Path)
	// If the payload is map[string]any (e.g. from JSON/HTTP), we need to convert it to TOut.
	// We use JSON round-trip as a robust fallback.
	return convertPayload[TOut](env.Payload)
}

// convertPayload attempts to convert any payload into T.
func convertPayload[T any](input any) (T, error) {
	var result T

	// Quick check for nil
	if input == nil {
		return result, nil
	}

	// Marshal to JSON
	bytes, err := json.Marshal(input)
	if err != nil {
		return result, fmt.Errorf("failed to marshal payload for type conversion: %w", err)
	}

	// Unmarshal to Target Type
	if err := json.Unmarshal(bytes, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal payload to target type %T: %w", result, err)
	}

	return result, nil
}
