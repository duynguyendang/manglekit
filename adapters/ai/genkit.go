package ai

import (
	"context"

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
		return "", err
	}

	return resp.Text(), nil
}
