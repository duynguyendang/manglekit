package ai

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
)

// GenkitGenerator wraps a Genkit ai.Model and implements the TextGenerator interface.
// This adapter translates between Manglekit's simple TextGenerator interface and Genkit's
// more feature-rich ai.Model API, enabling any Genkit-registered model to be used
// with Manglekit's actions.
type GenkitGenerator struct {
	model ai.Model
}

// NewGenkitGenerator creates a new GenkitGenerator wrapping the provided Genkit model.
//
// model is the Genkit ai.Model to wrap (e.g., from GoogleAI, OpenAI, Ollama plugins).
func NewGenkitGenerator(model ai.Model) *GenkitGenerator {
	return &GenkitGenerator{
		model: model,
	}
}

// Complete implements the TextGenerator interface.
// It takes a prompt and returns generated text by delegating to the underlying Genkit model.
//
// ctx is the request context.
// prompt is the input prompt/query for the model.
//
// It returns the generated text as a string, or an error if generation fails.
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
