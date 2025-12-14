package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
)

// NewActionFromConfig creates a new Action instance based on the provided configuration.
// It serves as a factory for hydrating actions from YAML/JSON configs.
//
// Parameters:
//   - name: The name of the action.
//   - cfg: The configuration object for the action.
//
// Returns:
//   - A configured core.Action instance.
//   - An error if the action type is unsupported or initialization fails.
func NewActionFromConfig(name string, cfg config.ActionConfig) (core.Action, error) {
	switch cfg.Type {
	case "llm":
		return createLLMAction(name, cfg)
	// case "http": return createHTTPAction(cfg) // Future
	default:
		return nil, fmt.Errorf("unsupported action type: %s", cfg.Type)
	}
}

func createLLMAction(name string, cfg config.ActionConfig) (core.Action, error) {
	// For LLMs, we need a core.TextGenerator.
	// In a real scenario, we would initialize Genkit or other providers here based on cfg.Provider.
	// Since adapters/ai currently requires pre-initialized Genkit instances (which are complex to config-hydrate),
	// we will assume a "noop" or "mock" generator if provider is not handled,
	// or ideally use a helper if available.

	// However, looking at the task, we need to return ai.NewAction(...) equivalent.
	// But ai.NewLLMAction requires a generator.
	// We'll create a simple placeholder generator for now to satisfy the interface,
	// as full Genkit hydration from config requires plugin initialization (GoogleAI, OpenAI, etc.).
	//
	// TODO: Add proper provider hydration logic (e.g., ai.NewGeneratorFromConfig).

	var generator core.TextGenerator

	// Minimal implementation to allow build:
	// We create a dummy generator that returns an error or mock response.
	// If the user provided specific instructions on how to hydrate Genkit from config without code,
	// we would follow it. But standard Genkit requires code-based Init.
	//
	// We will log a warning or return error if provider is unknown.

	if cfg.Provider == "mock" {
		generator = &mockGenerator{response: "This is a mock response"}
	} else {
		// Fallback: Return error for now as we can't magic up a Genkit instance without init.
		// Unless we assume the 'ai' adapter has a factory we missed? No.
		// We'll return a helpful error.
		return nil, fmt.Errorf("provider '%s' for LLM action not supported in config hydration yet (requires manual code init)", cfg.Provider)
	}

	return ai.NewLLMAction(name, generator)
}

type mockGenerator struct {
	response string
}

func (m *mockGenerator) Complete(ctx context.Context, prompt string) (string, error) {
	return m.response, nil
}

func (m *mockGenerator) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
    return &core.LLMResponse{Text: m.response}, nil
}

func (m *mockGenerator) Stream(ctx context.Context, prompt string) (<-chan string, error) {
    ch := make(chan string)
    close(ch)
    return ch, nil
}
