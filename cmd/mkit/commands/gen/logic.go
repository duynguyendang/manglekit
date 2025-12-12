package gen

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/duynguyendang/manglekit/internal/engine/resources"
	"github.com/duynguyendang/manglekit/sdk"
)

// GeneratedPolicy represents the structured response from the LLM.
type GeneratedPolicy struct {
	// DatalogContent contains the raw .dl code.
	DatalogContent string `json:"datalog_content"`
	// Explanation provides a human-readable summary of the logic.
	Explanation string `json:"explanation"`
}

// ValidatePolicySyntax checks if the generated Datalog code is valid using the Mangle engine.
func ValidatePolicySyntax(datalog string) error {
	// Initialize a runtime.
	eng := engine.NewMangleRuntime()

	// Prepend standard declarations to the validator so usages of std lib don't fail.
	// We use the single source of truth from internal/engine/resources.
	// We also strictly declare 'deny' as it is the expected output interface.
	stdDecls := resources.GetStdLib()
	denyDecl := "Decl deny(Source, Reason) ."
	fullProgram := stdDecls + "\n" + denyDecl + "\n" + datalog

	// Attempt to parse and compile the policy
	return eng.LoadFromSource(fullProgram)
}

// GenerateWithFeedback orchestrates the Teacher-Student protocol.
func GenerateWithFeedback(ctx context.Context, gen sdk.TextGenerator, userReq string, knownFacts []string) (*GeneratedPolicy, error) {
	// 1. Construct System Prompt
	// We build a template-based prompt manually here.
	factsList := ""
	for _, f := range knownFacts {
		factsList += "- " + f + "\n"
	}

	systemPrompt := fmt.Sprintf(`You are a Senior Knowledge Engineer specializing in Google Mangle Datalog.
Your task is to translate natural language requirements into strict, compilable Datalog rules.

### Standard Library (Always Available):
- json_num(Source, Key, Value)  // Int/Float fields
- json_str(Source, Key, Value)  // String fields
- json_bool(Source, Key, Value) // Boolean fields
- deny(Source, Reason)          // Main policy output

### Schema / Context (Hints):
%s

### Syntax Rules:
1. You MUST declare every predicate using 'Decl name(Arg1, Arg2, ...).' where Arg1, Arg2, etc. MUST start with an Uppercase letter (e.g., Decl amount(Source, Val).).
2. If the hints are JSON field names, prefer using json_num/str/bool or define a wrapper predicate with a Decl.
3. Strings must be double-quoted.
4. Variables start with uppercase (e.g., P, Amount).
5. Do NOT use aggregation (max, count) unless absolutely necessary.

Output JSON only: {"datalog_content": "...", "explanation": "..."}`, factsList)

	currentReq := userReq
	var lastErr error

	// The Loop (Max 5 Retries)
	for i := 0; i < 5; i++ {
		// Step B (Generation)
		policy, err := ai.GenerateStruct[GeneratedPolicy](ctx, gen, systemPrompt, currentReq)
		if err != nil {
			// If generation itself fails (e.g. network), we probably shouldn't blindly retry unless it's transient.
			// But sticking to the protocol, maybe we just fail or retry.
			// Let's assume we retry if it's a generation error too, or just fail.
			// The instructions say "If validation fails...".
			// If GenerateStruct fails, it might be JSON parsing error, so maybe retryable.
			lastErr = err
			// Feedback for JSON error
			feedback := fmt.Sprintf("Generation failed: %v", err)
			currentReq = fmt.Sprintf("%s\n\n[SYSTEM CORRECTION]: Previous attempt invalid.\nError: %s\nFix syntax immediately.", userReq, feedback)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Step C (Validation)
		if err := ValidatePolicySyntax(policy.DatalogContent); err != nil {
			lastErr = err
			// Step D (Decision) - Update feedback
			feedback := err.Error()
			// Step A (Prompting) - Update prompt for next iteration
			currentReq = fmt.Sprintf("%s\n\n[SYSTEM CORRECTION]: Previous attempt invalid.\nError: %s\nFix syntax immediately.", userReq, feedback)

			// Sleep briefly
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Success
		return &policy, nil
	}

	return nil, fmt.Errorf("failed after 5 retries. Last error: %w", lastErr)
}
