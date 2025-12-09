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
	prompt := buildPrompt(userReq, sysPrompt, sdk.ContextFacts(ctx))

	// Call AI
	resp, err := gen.Complete(ctx, prompt)
	if err != nil {
		return result, fmt.Errorf("llm completion failed: %w", err)
	}

	// Sanitize response
	cleaned := sanitizeJSON(resp)

	// Unmarshal
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal json: %w\nResponse (sanitized): %s\nOriginal: %s", err, cleaned, resp)
	}

	return result, nil
}

// buildPrompt creates the final prompt string with feedback if available.
func buildPrompt(userReq, sysPrompt string, facts map[string]string) string {
	var sb strings.Builder
	sb.WriteString(sysPrompt)
	sb.WriteString("\n\nUser Request: ")
	sb.WriteString(userReq)

	if feedback, ok := facts["mangle_feedback"]; ok && feedback != "" {
		sb.WriteString("\n\n--- PREVIOUS ATTEMPT REJECTED ---\n")
		sb.WriteString("Reason: ")
		sb.WriteString(feedback)
		sb.WriteString("\n\nInstruction: Please correct your answer to satisfy the policy requirement mentioned above.")
	}

	return sb.String()
}

// sanitizeJSON attempts to extract valid JSON from a raw LLM string.
// It supports extracting from Markdown code blocks (```json ... ```)
// and fallback to finding the first/last JSON delimiters.
func sanitizeJSON(input string) string {
	input = strings.TrimSpace(input)

	// 1. Attempt to find a Markdown code block
	if start := strings.Index(input, "```"); start != -1 {
		// Look for the closing block specifically after the start
		end := strings.LastIndex(input, "```")
		if end > start {
			// Extract content inside the block
			content := input[start+3 : end]
			content = strings.TrimSpace(content)

			// Remove optional language identifier (e.g., "json")
			if strings.HasPrefix(content, "json") {
				content = content[4:]
			}
			return strings.TrimSpace(content)
		}
	}

	// 2. Fallback: Identify the widest possible JSON boundaries
	// We look for the first '[' or '{' and the last ']' or '}'
	start := strings.IndexAny(input, "{[")
	end := strings.LastIndexAny(input, "}]")

	if start != -1 && end != -1 && end > start {
		return input[start : end+1]
	}

	// Return original if no patterns matched
	return input
}
