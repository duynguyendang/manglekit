package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit-wip/core"
)

// Compile-time interface satisfaction checks
var _ core.Action = (*LLMAction)(nil)

// LLMAction is a concrete implementation of core.Action that wraps a core.TextGenerator.
// It adapts the specific TextGenerator interface to the universal core.Action envelope interface.
type LLMAction struct {
	name      string
	generator core.TextGenerator
}

// NewLLMAction creates a new LLMAction instance.
//
// Parameters:
//   - name: A unique name for this action (used in observability and policies).
//   - generator: The TextGenerator implementation to wrap.
//
// Returns:
//   - A pointer to the initialized LLMAction.
//   - An error if the generator is nil.
func NewLLMAction(name string, generator core.TextGenerator) (*LLMAction, error) {
	if generator == nil {
		return nil, fmt.Errorf("NewLLMAction(%s): generator cannot be nil", name)
	}
	return &LLMAction{
		name:      name,
		generator: generator,
	}, nil
}

// Execute performs the text generation.
// It expects the input Envelope's Payload to be a string (the prompt).
// It returns a new Envelope containing the generated text as the Payload.
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input envelope containing the prompt string.
//
// Returns:
//   - A result envelope with the generated text, or an error.
func (a *LLMAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	prompt, ok := input.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("%w: invalid input type, expected string but got %T", core.ErrSystemError, input.Payload)
	}

	// Teacher-Student Protocol: Inject feedback if present
	if feedback, ok := input.Metadata["mangle_feedback"]; ok && feedback != "" {
		prompt += fmt.Sprintf("\n\n[SYSTEM WARNING]: Your previous attempt failed. Reason: '%s'. Please correct your output to satisfy this rule.", feedback)
	}

	// Inject dynamic config into context
	for k, v := range input.Metadata {
		// Justification: We specifically look for "prompt." prefix injected by Supervisor
		if strings.HasPrefix(k, core.PrefixPromptConfig) {
			vStr := fmt.Sprintf("%v", v)
			ctx = core.WithFact(ctx, k, vStr)
		}
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

// Metadata returns the static metadata for this action.
func (a *LLMAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: a.name,
		Type: "llm",
	}
}

// Complete implements core.TextGenerator.
func (a *LLMAction) Complete(ctx context.Context, prompt string) (string, error) {
	return a.generator.Complete(ctx, prompt)
}

// Generate implements core.TextGenerator.
func (a *LLMAction) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	return a.generator.Generate(ctx, prompt, opts...)
}

// Stream implements core.TextGenerator.
func (a *LLMAction) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	return a.generator.Stream(ctx, prompt)
}
