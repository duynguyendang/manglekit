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
		feedbackSuffix = fmt.Sprintf("\n\n[ALIGNMENT INTERVENTION]: Your previous answer failed blueprint alignment check. Reason: '%s'. Fix it immediately.", feedback)
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

		// Use native genkit.Generate directly on the model
		resp, err := genkit.Generate(ctx, adapter.gk,
			ai.WithModel(adapter.model),
			ai.WithMessages(messages...),
			ai.WithOutputType(new(T)), // <--- Native Structured Output
		)

		if err != nil {
			return result, fmt.Errorf("genkit native generation failed: %w", err)
		}

		if err := resp.Output(&result); err != nil {
			return result, fmt.Errorf("genkit output parsing failed: %w", err)
		}
		return result, nil
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
