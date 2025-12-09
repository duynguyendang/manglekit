package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/sdk"
)

// GenerateStruct generates a structured response from the LLM.
// It handles feedback injection, JSON sanitization, and unmarshaling.
func GenerateStruct[T any](ctx context.Context, gen sdk.TextGenerator, userReq string, sysPrompt string) (T, error) {
	var result T

	// Construct the prompt
	prompt := sysPrompt + "\n\nUser Request: " + userReq

	// Feedback Logic: Retrieve mangle_feedback from context facts
	// We use sdk.ContextFacts because core.MetadataFromContext does not exist in the current codebase.
	// The sdk.ContextFacts function returns map[string]string.
	facts := sdk.ContextFacts(ctx)
	if feedback, ok := facts["mangle_feedback"]; ok && feedback != "" {
		prompt += fmt.Sprintf("\n\n--- PREVIOUS ATTEMPT REJECTED ---\nReason: %s\n\nInstruction: Please correct your answer to satisfy the policy requirement mentioned above.", feedback)
	}

	// Call AI
	resp, err := gen.Complete(ctx, prompt)
	if err != nil {
		return result, fmt.Errorf("llm completion failed: %w", err)
	}

	// Sanitize: Remove ```json and ``` wrappers
	cleaned := strings.TrimSpace(resp)
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	cleaned = strings.TrimSpace(cleaned)

	// Unmarshal
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal json: %w\nResponse: %s", err, resp)
	}

	return result, nil
}
