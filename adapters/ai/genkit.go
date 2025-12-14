package ai

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

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

	req := &ai.ModelRequest{
		Messages: messages,
		// Output describes the desired response format.
		Output: &ai.ModelOutputConfig{
			Format: ai.OutputFormatJSON,
		},
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
