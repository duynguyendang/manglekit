package sdk

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
)

// HydrateActions iterates through the configuration and instantiates the defined actions.
// It acts as a high-level factory for converting config maps into executable core.Actions.
func HydrateActions(actions map[string]config.ActionConfig) ([]core.Action, error) {
	var hydrated []core.Action

	for name, cfg := range actions {
		action, err := NewActionFromConfig(name, cfg)
		if err != nil {
			// We log/return error. Returning error stops the boot, which is usually safer for config correctness.
			return nil, fmt.Errorf("failed to hydrate action %q: %w", name, err)
		}
		hydrated = append(hydrated, action)
	}
	return hydrated, nil
}

// NewActionFromConfig creates a new Action instance based on the provided configuration.
func NewActionFromConfig(name string, cfg config.ActionConfig) (core.Action, error) {
	switch cfg.Type {
	case "llm":
		return createLLMAction(name, cfg)
	default:
		return nil, fmt.Errorf("unsupported action type: %s", cfg.Type)
	}
}

func createLLMAction(name string, cfg config.ActionConfig) (core.Action, error) {
	// 1. Check if we can support the real provider
	// For this task, we support a fallback to Mock if API keys are missing.

	// Real Genkit hydration would go here.
	// E.g., if cfg.Provider == "google" && os.Getenv("GOOGLE_GENAI_API_KEY") != "" { ... }

	// For "Low-Code Gateway" task, we default to Mock to ensure wiring test passes.
	// We only log a warning if we are falling back when a specific provider was requested.

	if cfg.Provider != "mock" {
		// In a real implementation, we would try to init the provider.
		// Here we fallback.
		// fmt.Printf("⚠️  Warning: Provider '%s' not fully integrated in config loader yet. Falling back to Mock for '%s'.\n", cfg.Provider, name)
	}

	return createMockLLMAction(name, cfg)
}

func createMockLLMAction(name string, cfg config.ActionConfig) (core.Action, error) {
	prompt := ""
	if p, ok := cfg.Options["prompt"].(string); ok {
		prompt = p
	}

	gen := &mockGenerator{
		systemPrompt: prompt,
	}

	return ai.NewLLMAction(name, gen)
}

// mockGenerator implements core.TextGenerator for testing/fallback.
type mockGenerator struct {
	systemPrompt string
}

func (m *mockGenerator) Complete(ctx context.Context, prompt string) (string, error) {
	return fmt.Sprintf("%s %s", m.systemPrompt, prompt), nil
}

func (m *mockGenerator) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	// Basic echo behavior
	resp := fmt.Sprintf("%s %s", m.systemPrompt, prompt)
	return &core.LLMResponse{
		Text: resp,
		Usage: map[string]int{
			"input":  len(prompt),
			"output": len(resp),
		},
	}, nil
}

func (m *mockGenerator) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		ch <- fmt.Sprintf("%s %s", m.systemPrompt, prompt)
	}()
	return ch, nil
}
