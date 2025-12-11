package ai

import (
	"context"

	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// genkitAdapter adapts the Firebase Genkit ai.Model interface to the Manglekit sdk.TextGenerator interface.
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
//   - A sdk.TextGenerator implementation.
func NewGenkitAdapter(model ai.Model, gk *genkit.Genkit) sdk.TextGenerator {
	return &genkitAdapter{
		model: model,
		gk:    gk,
	}
}

// Complete generates text using the underlying Genkit model.
func (g *genkitAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	req := &ai.ModelRequest{
		Messages: []*ai.Message{{
			Role:    ai.RoleUser,
			Content: []*ai.Part{ai.NewTextPart(prompt)},
		}},
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
