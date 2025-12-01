package ai

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
)

// GenkitGenerator is an adapter that makes a Genkit ai.Model compatible with the TextGenerator interface.
type GenkitGenerator struct {
	model ai.Model
}

// NewGenkitGenerator creates a new adapter for a Genkit model.
//
// Parameters:
//   - model: The initialized Genkit model.
//
// Returns:
//   - A configured GenkitGenerator.
func NewGenkitGenerator(model ai.Model) *GenkitGenerator {
	return &GenkitGenerator{
		model: model,
	}
}

// Complete executes a text generation request using the underlying Genkit model.
//
// Parameters:
//   - ctx: The context.
//   - prompt: The text prompt.
//
// Returns:
//   - The generated response string, or an error.
func (g *GenkitGenerator) Complete(ctx context.Context, prompt string) (string, error) {
	if g.model == nil {
		return "", fmt.Errorf("genkit generator: model not initialized")
	}

	// Create a Genkit model request with the prompt
	// ModelRequest expects a messages array with the user's query
	req := &ai.ModelRequest{
		Messages: []*ai.Message{
			{
				Role: "user",
				Content: []*ai.Part{
					ai.NewTextPart(prompt),
				},
			},
		},
	}

	// Call the model to generate text
	resp, err := g.model.Generate(ctx, req, nil)
	if err != nil {
		return "", fmt.Errorf("genkit model generate failed: %w", err)
	}

	if resp == nil {
		return "", fmt.Errorf("genkit model returned nil response")
	}

	// Extract the generated text from the response
	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("genkit model returned empty response")
	}

	return text, nil
}
