package ai

import (
	"context"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

var (
	globalGenkit *genkit.Genkit
	initOnce     sync.Once
)

// GetGenkit returns the global Genkit instance, initializing it if necessary.
// This allows other packages to register models/tools.
func GetGenkit(ctx context.Context) *genkit.Genkit {
	initOnce.Do(func() {
		// Initialize without plugins initially
		globalGenkit = genkit.Init(ctx)
	})
	return globalGenkit
}

// NewGenkitAction creates a new core.Action backed by a Genkit model.
// It ensures the Genkit runtime is initialized and looks up the model by name.
func NewGenkitAction(ctx context.Context, modelName string) (core.Action, error) {
	g := GetGenkit(ctx)

	// Lookup model using genkit.LookupModel
	// Note: We use the full name (e.g., "openai/gpt-4o") directly.
	model := genkit.LookupModel(g, modelName)
	if model == nil {
		// Try parsing if lookup failed directly (in case some registry logic differs)
		// But usually full name is key.

		// Fallback debug info
		return nil, fmt.Errorf("genkit model not found: %s", modelName)
	}

	// Create adapter
	adapter := NewGenkitAdapter(model, g)

	// Wrap in LLMAction
	// We use modelName as the action name
	// Note: We might want a cleaner name? Using modelName is fine.
	return NewLLMAction(modelName, adapter)
}

// genkitAdapter adapts the Firebase Genkit ai.Model interface to the Manglekit core.TextGenerator interface.
type genkitAdapter struct {
	model ai.Model
	gk    *genkit.Genkit
}

// NewGenkitAdapter creates a new adapter from a pre-initialized Genkit model.
//
// Parameters:
//   - model: The Genkit model instance.
//   - gk: The Genkit runtime instance.
//
// Returns:
//   - A core.TextGenerator implementation.
func NewGenkitAdapter(model ai.Model, gk *genkit.Genkit) core.TextGenerator {
	return &genkitAdapter{
		model: model,
		gk:    gk,
	}
}

// Complete generates text using the underlying Genkit model.
func (g *genkitAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	resp, err := g.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// Generate implements the core.TextGenerator interface using Genkit.
func (g *genkitAdapter) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	// Initialize Config
	cfg := &core.GenerationConfig{
		Temperature: 0.7, // Default
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var messages []*ai.Message

	// Dynamic Prompt Configuration
	facts := core.ContextFacts(ctx)
	systemPrompt := ""
	if facts != nil {
		if val, ok := facts[core.PrefixPromptConfig+"tone"]; ok {
			systemPrompt += "\n[INSTRUCTION]: Maintain a " + val + " tone."
		}
		if val, ok := facts[core.PrefixPromptConfig+"strategy"]; ok && val == "cot" {
			systemPrompt += "\n[STRATEGY]: Think step-by-step."
		}
	}

	if systemPrompt != "" {
		messages = append(messages, &ai.Message{
			Role:    ai.RoleSystem,
			Content: []*ai.Part{ai.NewTextPart(systemPrompt)},
		})
	}

	messages = append(messages, &ai.Message{
		Role:    ai.RoleUser,
		Content: []*ai.Part{ai.NewTextPart(prompt)},
	})

	// Prepare Genkit Config
	genkitConfig := ai.GenerationCommonConfig{}

	if cfg.Temperature != 0 {
		genkitConfig.Temperature = cfg.Temperature
	}
	if cfg.MaxTokens != 0 {
		genkitConfig.MaxOutputTokens = cfg.MaxTokens
	}
	if cfg.TopP != 0 {
		genkitConfig.TopP = cfg.TopP
	}
	if len(cfg.StopSequences) > 0 {
		genkitConfig.StopSequences = cfg.StopSequences
	}

	// Map to Genkit Request
	req := &ai.ModelRequest{
		Messages: messages,
		Config:   genkitConfig,
	}

	// Handle Output
	if cfg.OutputType != nil {
		req.Output = &ai.ModelOutputConfig{}
		if cfg.JSONMode {
			req.Output.Format = ai.OutputFormatJSON
		}
	} else if cfg.JSONMode {
		req.Output = &ai.ModelOutputConfig{
			Format: ai.OutputFormatJSON,
		}
	}

	resp, err := g.model.Generate(ctx, req, nil)
	if err != nil {
		return nil, err
	}

	// Extract token usage if available
	usage := make(map[string]int)
	if resp.Usage != nil {
		usage["prompt"] = int(resp.Usage.InputTokens)
		usage["completion"] = int(resp.Usage.OutputTokens)
		usage["total"] = int(resp.Usage.TotalTokens)
	}

	return &core.LLMResponse{
		Text:  resp.Text(),
		Usage: usage,
	}, nil
}

// Stream implements the core.TextGenerator interface.
// Currently returns error as streaming is not fully adapted here.
func (g *genkitAdapter) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	// Simple non-streaming fallback or error
	return nil, fmt.Errorf("streaming not implemented in genkit adapter yet")
}
