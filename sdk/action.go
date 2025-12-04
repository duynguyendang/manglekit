package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	funcAdapter "github.com/duynguyendang/manglekit/adapters/func"
	"github.com/duynguyendang/manglekit/core"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HandlerFunc defines the signature for a type-safe action handler.
type HandlerFunc[In any, Out any] func(context.Context, In) (Out, error)

// Action represents a type-safe wrapper around a registered core.Action.
// It uses Go Generics to enforce input/output types while delegating
// execution to the underlying governance engine.
type Action[In any, Out any] struct {
	name    string
	client  *Client
	handler HandlerFunc[In, Out]
}

// Define registers a new type-safe action with the client.
// It wraps the provided handler function into a core.Action, applies protection
// (policies, tracing), and registers it for execution.
//
// Parameters:
//   - client: The Manglekit Client instance.
//   - name: The unique name for the action.
//   - handler: The type-safe business logic function.
//
// Returns:
//   - A typed Action object that can be used to run the handler.
func Define[In any, Out any](client *Client, name string, handler HandlerFunc[In, Out]) *Action[In, Out] {
	// 1. Create Adapter (Adapts Typed -> Envelope)
	adapter := funcAdapter.New(name, funcAdapter.ToolFunc[In, Out](handler))
	adapter.SetContentType(core.TypeStruct)

	// 2. Protect (Apply Governance)
	protected := client.Protect(adapter)

	// 3. Register (Enable Routing)
	client.RegisterAction(name, protected)

	// 4. Return Typed Handle
	return &Action[In, Out]{
		name:    name,
		client:  client,
		handler: handler,
	}
}

// DefineDynamic registers a new dynamic action (JSON map input/output).
// It sets the content type to TypeJSON, enabling recursive graph fact generation.
func DefineDynamic(client *Client, name string, handler HandlerFunc[map[string]any, map[string]any]) *Action[map[string]any, map[string]any] {
	// 1. Create Adapter
	adapter := funcAdapter.New(name, funcAdapter.ToolFunc[map[string]any, map[string]any](handler))
	adapter.SetContentType(core.TypeJSON)

	// 2. Protect (Apply Governance)
	protected := client.Protect(adapter)

	// 3. Register (Enable Routing)
	client.RegisterAction(name, protected)

	// 4. Return Typed Handle
	return &Action[map[string]any, map[string]any]{
		name:    name,
		client:  client,
		handler: handler,
	}
}

// Run executes the action with the provided input.
// It handles serialization, context extraction, invisible governance, and deserialization.
//
// Parameters:
//   - ctx: The execution context.
//   - input: The strongly-typed input.
//
// Returns:
//   - The strongly-typed output, or an error.
func (a *Action[In, Out]) Run(ctx context.Context, input In) (Out, error) {
	var zero Out

	// 1. Invisible Governance: Start Trace Span
	ctx, span := a.client.Tracer().Start(ctx, "sdk.Action.Run", trace.WithAttributes(
		attribute.String("action.name", a.name),
	))
	defer span.End()

	// Capture Input as Attribute (Best Effort JSON)
	if inBytes, err := json.Marshal(input); err == nil {
		span.SetAttributes(attribute.String("action.input", string(inBytes)))
	}

	// 2. Extract Facts from Context
	facts := ContextFacts(ctx)

	// 3. Execution (Delegates to Client Loop)
	// This triggers Policy Check -> Telemetry -> Execution -> Retry Loop
	opts := []ExecuteOption{}
	if len(facts) > 0 {
		opts = append(opts, WithMetadataMap(facts))
	}

	resEnvelope, err := a.client.ExecuteByName(ctx, a.name, input, opts...)

	if err != nil {
		// 4. Error Handling: Unwrap PolicyViolationError if possible
		var pve *core.PolicyViolationError
		if errors.As(err, &pve) {
			// If it's a policy violation, we might want to return it wrapped or as is
			// Since PolicyViolationError is an error, we return it.
		}

		span.RecordError(err)
		span.SetAttributes(attribute.String("action.status", "error"))
		return zero, err
	}

	// 5. Deserialization
	out, ok := resEnvelope.Payload.(Out)
	if !ok {
		err := fmt.Errorf("output type mismatch: expected %T, got %T", zero, resEnvelope.Payload)
		span.RecordError(err)
		span.SetAttributes(attribute.String("action.status", "type_mismatch"))
		return zero, err
	}

	// Capture Output as Attribute
	if outBytes, err := json.Marshal(out); err == nil {
		span.SetAttributes(attribute.String("action.output", string(outBytes)))
	}
	span.SetAttributes(attribute.String("action.status", "success"))

	return out, nil
}
