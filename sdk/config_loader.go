package sdk

import (
	"context"
	"fmt"

	aiAdapter "github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/openai"
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
	factory, err := Provider(cfg.Provider)
	if err == nil {
		return factory(ctx, name, cfg)
	}

	// Special Wiring for OpenAI
	if cfg.Provider == "openai" {
		// 1. Extract Options Safely
		apiKey, _ := cfg.Options["api_key"].(string)
		baseURL, _ := cfg.Options["base_url"].(string)

		// Default model if missing
		modelID, ok := cfg.Options["model"].(string)
		if !ok || modelID == "" {
			modelID = "gpt-4o"
		}

		// 2. Initialize Plugin (Wire it!)
		// Get Registry from Adapter
		g := aiAdapter.GetGenkit(ctx)

		// This registers "openai/{modelID}" into Genkit's internal registry.
		if err := openai.Init(g, modelID, openai.Config{APIKey: apiKey, BaseURL: baseURL}); err != nil {
			return nil, fmt.Errorf("failed to wire openai provider: %w", err)
		}

		// 3. Create Action using Universal Adapter
		// We tell the adapter to look for "openai/" + modelID
		genkitModelName := "openai/" + modelID

		// Ensure NewGenkitAction signature matches your codebase (passing name and options)
		// Assuming: NewGenkitAction(ctx context.Context, modelName string, opts ...GenerateOption)
		action, err := aiAdapter.NewGenkitAction(ctx, genkitModelName)
		if err != nil {
			return nil, fmt.Errorf("failed to create genkit action: %w", err)
		}
		return action, nil
	}

	// 2. Fallback for Mock (Built-in)
	if cfg.Provider == "mock" {
		return createMockLLMAction(name, cfg)
	}

	// 3. Error if not found
	return nil, fmt.Errorf("failed to create action '%s': %w (Did you forget to call sdk.RegisterProvider in main?)", name, err)
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
