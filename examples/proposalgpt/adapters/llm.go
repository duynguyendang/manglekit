package adapters

import (
	"context"

	"github.com/duynguyendang/manglekit-wip/internal/core/domain"
	"github.com/duynguyendang/manglekit-wip/internal/core/ports"
)

// MockGenerative simulates an AI model (like Gemini) generating a structured plan.
type MockGenerative struct{}

func NewMockGenerative() *MockGenerative {
	return &MockGenerative{}
}

func (m *MockGenerative) Generate(ctx context.Context, intent domain.IntentStr, compiledPrompt string, contextAtoms []domain.Atom, genes []domain.DomainGene) (*ports.Plan, error) {
	// A real LLM adapter would send compiledPrompt to the API here.

	return &ports.Plan{
		Steps: []map[string]any{
			{
				"id":          "step-1",
				"description": "Process the user intent: " + string(intent),
				"action":      "ExecuteMockBehavior",
			},
		},
	}, nil
}

func (m *MockGenerative) Embed(ctx context.Context, text string) (ports.Vector, error) {
	// Dummy 64-dim vector
	return make(ports.Vector, 64), nil
}

func (m *MockGenerative) Induce(ctx context.Context, input string) (string, error) {
	return `
// Induced Mock Rule
halt("unauthorized_mock") :-
    input_param("user", "banned_user").
`, nil
}
