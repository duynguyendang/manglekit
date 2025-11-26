package ai

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v2/core"
)

// TextGenerator is a simple interface for a text generation backend.
type TextGenerator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// Adapter implements the core.Action interface for a TextGenerator.
type Adapter struct {
	name string
	gen  TextGenerator
}

// New creates a new Adapter for the given TextGenerator.
func New(name string, gen TextGenerator) *Adapter {
	return &Adapter{
		name: name,
		gen:  gen,
	}
}

// Execute expects a string payload, calls the generator, and returns the response.
func (a *Adapter) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	prompt, ok := input.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("%w: invalid input type, expected string but got %T", core.ErrSystemError, input.Payload)
	}

	resp, err := a.gen.Generate(ctx, prompt)
	if err != nil {
		return core.Envelope{}, err
	}

	return core.NewEnvelope(resp), nil
}

// Metadata returns the metadata for the action.
func (a *Adapter) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: a.name,
		Type: "llm",
	}
}
