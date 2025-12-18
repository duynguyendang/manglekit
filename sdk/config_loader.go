package sdk

import (
	"context"
	"fmt"

	aiAdapter "github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
)

// HydrateActions iterates through the configuration and instantiates the defined actions.
// It acts as a high-level factory for converting config maps into executable core.Actions.
func HydrateActions(ctx context.Context, actions map[string]config.ActionConfig) ([]core.Action, error) {
	var hydrated []core.Action

	for name, cfg := range actions {
		action, err := NewActionFromConfig(ctx, name, cfg)
		if err != nil {
			// We log/return error. Returning error stops the boot, which is usually safer for config correctness.
			return nil, fmt.Errorf("failed to hydrate action %q: %w", name, err)
		}
		hydrated = append(hydrated, action)
	}
	return hydrated, nil
}

// NewActionFromConfig creates a new Action instance based on the provided configuration.
func NewActionFromConfig(ctx context.Context, name string, cfg config.ActionConfig) (core.Action, error) {
	switch cfg.Type {
	case "llm":
		return createLLMAction(ctx, name, cfg)
	default:
		return nil, fmt.Errorf("unsupported action type: %s", cfg.Type)
	}
}

func createLLMAction(ctx context.Context, name string, cfg config.ActionConfig) (core.Action, error) {
	// 1. Try to find a registered provider
	// Note: ProviderFactory now returns ClientOption, so we cannot instantiate Action directly here.
	// This function is legacy compatibility for HydrateActions.
	// Users should use WithConfig(cfg) instead of HydrateActions.

	// 2. Fallback for Mock (Built-in)
	if cfg.Provider == "mock" {
		return createMockLLMAction(name, cfg)
	}

	// 3. Error if not found
	return nil, fmt.Errorf("failed to create action '%s': provider '%s' not registered (Did you forget to call sdk.RegisterProvider in main?)", name, cfg.Provider)
}

func createMockLLMAction(name string, cfg config.ActionConfig) (core.Action, error) {
	prompt := ""
	if p, ok := cfg.Options["prompt"].(string); ok {
		prompt = p
	}

	gen := &mockGenerator{
		systemPrompt: prompt,
	}

	return aiAdapter.NewLLMAction(name, gen)
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

// createMemory instantiates the memory provider from configuration.
func createMemory(ctx context.Context, cfg config.Config) (core.AgentMemory, error) {
	if cfg.Memory.Provider == "" {
		return nil, nil // Memory is optional
	}

	factory, err := MemoryProvider(cfg.Memory.Provider)
	if err != nil {
		return nil, err
	}

	return factory(ctx, cfg.Memory)
}
