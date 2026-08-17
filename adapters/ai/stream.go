package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/duynguyendang/manglekit/core"
)

// StreamingGate is the governance surface a StreamingSupervisedAction needs:
// the pre-check (Assess) that runs before the first chunk is produced, and
// the post-check (Reflect) that runs on the assembled full response once the
// stream completes. core.Evaluator (sdk.Client.Engine) satisfies it.
type StreamingGate interface {
	Assess(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error
	Reflect(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error)
}

// Compile-time interface satisfaction check.
var _ core.Action = (*StreamingSupervisedAction)(nil)

// StreamingSupervisedAction wraps a core.TextGenerator so that streaming
// generation is supervised by the governance gate:
//
//   - Pre-check (Assess) runs before the provider stream is opened, so a
//     policy deny halts the action before a single chunk is generated.
//   - Chunks are forwarded to the caller as they arrive.
//   - Post-check (Reflect) runs on the assembled full response after the
//     stream completes. If it denies, the caller receives a terminal
//     core.StreamChunk carrying the policy error and no final envelope is
//     exposed.
type StreamingSupervisedAction struct {
	name      string
	generator core.TextGenerator
	gate      StreamingGate

	mu    sync.Mutex
	final *core.Envelope
}

// NewStreamingSupervisedAction creates a streaming supervised LLM action.
//
// Parameters:
//   - name: A unique action name (used in policies and observability).
//   - generator: The TextGenerator providing the underlying stream.
//   - gate: The governance gate (pre/post-check); typically client.Engine().
func NewStreamingSupervisedAction(name string, generator core.TextGenerator, gate StreamingGate) (*StreamingSupervisedAction, error) {
	if generator == nil {
		return nil, fmt.Errorf("NewStreamingSupervisedAction(%s): generator cannot be nil", name)
	}
	if gate == nil {
		return nil, fmt.Errorf("NewStreamingSupervisedAction(%s): gate cannot be nil", name)
	}
	return &StreamingSupervisedAction{
		name:      name,
		generator: generator,
		gate:      gate,
	}, nil
}

// Metadata returns the static metadata for this action.
func (a *StreamingSupervisedAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: a.name,
		Type: "llm",
	}
}

// Execute performs a non-streaming supervised generation: pre-check,
// Complete, post-check on the full response.
func (a *StreamingSupervisedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	prompt, err := a.promptFrom(input)
	if err != nil {
		return core.Envelope{}, err
	}

	// Pre-check: deny halts before any generation happens.
	if err := a.gate.Assess(ctx, a.Metadata(), input); err != nil {
		return core.Envelope{}, fmt.Errorf("streaming action %q pre-check denied: %w", a.name, err)
	}

	text, err := a.generator.Complete(ctx, prompt)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("llm generation failed: %w", err)
	}

	output := core.NewEnvelope(text)
	output.SetMeta("model_type", "llm")
	output.SetMeta("action_name", a.name)

	// Post-check on the assembled response.
	validated, err := a.gate.Reflect(ctx, a.Metadata(), output)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("streaming action %q post-check denied: %w", a.name, err)
	}
	return validated, nil
}

// Stream performs a supervised streaming generation. The pre-check runs
// synchronously before the provider stream is opened: when the policy denies
// the request, Stream returns the policy error and no channel is created, so
// the caller never receives a chunk. On success, chunks are forwarded as they
// arrive; after the underlying stream closes the assembled full response is
// post-checked (Reflect). A post-check denial surfaces as a terminal
// core.StreamChunk with the policy error (available via FinalEnvelope only
// when the post-check passes).
func (a *StreamingSupervisedAction) Stream(ctx context.Context, input core.Envelope) (<-chan core.StreamChunk, error) {
	prompt, err := a.promptFrom(input)
	if err != nil {
		return nil, err
	}

	// Pre-check: deny halts before the first chunk.
	if err := a.gate.Assess(ctx, a.Metadata(), input); err != nil {
		return nil, fmt.Errorf("streaming action %q pre-check denied: %w", a.name, err)
	}

	source, err := a.generator.Stream(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("llm stream failed to open: %w", err)
	}

	out := make(chan core.StreamChunk)
	go func() {
		defer close(out)

		var assembled strings.Builder
		for chunk := range source {
			if chunk.Err != nil {
				emit(ctx, out, core.StreamChunk{Err: chunk.Err})
				return
			}
			assembled.WriteString(chunk.Text)
			if !emit(ctx, out, chunk) {
				return
			}
		}

		// Post-check on the assembled full response.
		output := core.NewEnvelope(assembled.String())
		output.SetMeta("model_type", "llm")
		output.SetMeta("action_name", a.name)

		validated, err := a.gate.Reflect(ctx, a.Metadata(), output)
		if err != nil {
			emit(ctx, out, core.StreamChunk{Err: fmt.Errorf("streaming action %q post-check denied: %w", a.name, err)})
			return
		}

		a.mu.Lock()
		a.final = &validated
		a.mu.Unlock()
	}()

	return out, nil
}

// FinalEnvelope returns the post-checked assembled response, if the stream
// has completed and the post-check passed.
func (a *StreamingSupervisedAction) FinalEnvelope() (core.Envelope, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.final == nil {
		return core.Envelope{}, false
	}
	return *a.final, true
}

// promptFrom extracts the prompt from the input envelope, mirroring the
// Teacher-Student feedback injection used by LLMAction.
func (a *StreamingSupervisedAction) promptFrom(input core.Envelope) (string, error) {
	prompt, ok := input.Payload.(string)
	if !ok {
		return "", fmt.Errorf("%w: invalid input type, expected string but got %T", core.ErrSystemError, input.Payload)
	}
	if feedback, ok := input.Metadata[core.KeyFeedback]; ok && feedback != "" {
		prompt += fmt.Sprintf("\n\n[SYSTEM WARNING]: Your previous attempt failed. Reason: '%s'. Please correct your output to satisfy this rule.", feedback)
	}
	return prompt, nil
}

// emit sends a chunk unless the context is done. It reports whether the
// consumer is still listening.
func emit(ctx context.Context, ch chan<- core.StreamChunk, chunk core.StreamChunk) bool {
	select {
	case ch <- chunk:
		return true
	case <-ctx.Done():
		return false
	}
}
