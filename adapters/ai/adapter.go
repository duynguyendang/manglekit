package ai

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// TextGenerator defines the interface for a raw LLM/Genkit text generation client.
// This represents the "Muscle" layer that performs actual text generation.
type TextGenerator interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// LLMAction wraps a TextGenerator and implements core.Action.
// It treats LLM generation as a universal action (Text-in, Text-out).
type LLMAction struct {
	name      string
	generator TextGenerator
}

// NewLLMAction creates a new LLMAction wrapping the given TextGenerator.
func NewLLMAction(name string, generator TextGenerator) *LLMAction {
	return &LLMAction{
		name:      name,
		generator: generator,
	}
}

// Execute expects a string payload (the prompt/context), calls the generator,
// and returns the generated text wrapped in an Envelope.
func (a *LLMAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	prompt, ok := input.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("%w: invalid input type, expected string but got %T", core.ErrSystemError, input.Payload)
	}

	resp, err := a.generator.Complete(ctx, prompt)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("llm generation failed: %w", err)
	}

	output := core.NewEnvelope(resp)
	output.SetMeta("model_type", "llm")
	output.SetMeta("action_name", a.name)

	return output, nil
}

// Metadata returns the metadata for this LLM action.
func (a *LLMAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: a.name,
		Type: "llm",
	}
}
