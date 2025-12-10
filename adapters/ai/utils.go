package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/sdk"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// GenerateStruct generates a type-safe response.
// It prioritizes Native Genkit Structured Output if available.

func GenerateStruct[T any](ctx context.Context, gen sdk.TextGenerator, sysPrompt string, userReq string) (T, error) {
	var result T

	// 1. Prepare Inputs
	facts := sdk.ContextFacts(ctx)
	feedbackSuffix := ""
	// Retrieve feedback injected by Manglekit Core
	if feedback, ok := facts["mangle_feedback"]; ok && feedback != "" {
		feedbackSuffix = fmt.Sprintf("\n\n[SYSTEM CORRECTION]: Your previous answer failed policy check. Reason: '%s'. Fix it immediately.", feedback)
	}

	effectiveUserPrompt := userReq + feedbackSuffix

	// 2. OPTIMIZED PATH: Native Genkit
	// Check if the generator is our specific Genkit adapter
	if adapter, ok := gen.(*genkitAdapter); ok {
		// Use Genkit's native feature with explicit System/User separation.

		// ai.NewSystemMessage treats the input as high-priority directives (The Blueprint).
		messages := []*ai.Message{
			ai.NewSystemMessage(ai.NewTextPart(sysPrompt)),
			ai.NewUserMessage(ai.NewTextPart(effectiveUserPrompt)),
		}

		// ai.GenerateData[T] handles:
		// - Schema generation for T (via WithOutputType)
		// - Markdown stripping (sanitization)
		// - Valid JSON parsing (via resp.Output(&value))
		output, _, err := genkit.GenerateData[T](ctx, adapter.gk,
			ai.WithModel(adapter.model),
			ai.WithMessages(messages...),
		)
		if err != nil {
			return result, fmt.Errorf("genkit native generation failed: %w", err)
		}
		// output is *T
		if output != nil {
			return *output, nil
		}
		return result, fmt.Errorf("genkit returned nil output")
	}

	// 3. FALLBACK PATH: Standard JSON (For mocks or other providers)
	// Combine System and User for simple text completion
	fullPrompt := fmt.Sprintf("%s\n\nUser Request: %s", sysPrompt, effectiveUserPrompt)

	rawResp, err := gen.Complete(ctx, fullPrompt)
	if err != nil {
		return result, err
	}

	// Minimal manual cleanup (just in case) using standard strings
	cleaned := strings.TrimSpace(rawResp)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return result, fmt.Errorf("manual json unmarshal failed: %w. Raw: %s", err, rawResp)
	}

	return result, nil
}
