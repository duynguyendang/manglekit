package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
)

// StubbornAI simulates an AI agent that tries to spend too much money,
// but corrects itself when it receives specific feedback.
type StubbornAI struct{}

func (a *StubbornAI) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	// 1. Check for feedback (Teacher-Student Protocol)
	feedback, hasFeedback := env.Metadata["mangle_feedback"]

	// 2. Simulate the AI "seeing" the prompt + feedback
	// In a real LLMAction, the feedback is appended to the prompt string.
	// Here, we print it to the console to show the user what's happening.
	prompt := fmt.Sprintf("%v", env.Payload)
	if hasFeedback {
		// Simulate the injected system warning
		prompt = fmt.Sprintf("%s\n[SYSTEM WARNING]: %s", prompt, feedback)

		// Visual cue for the demo
		fmt.Println("👮 Manglekit: DENIED! (Policy Violation Detected)")
	}

	fmt.Printf("\n📥 [StubbornAI] Received Prompt:\n%s\n", prompt)

	// 3. Logic: If warned, behave correctly. If not, behave greedily.
	if hasFeedback {
		fmt.Println("🤖 AI: Oops, I see the feedback. Retrying with $450...")
		return core.NewEnvelope(map[string]int{"amount": 450}), nil
	}

	fmt.Println("🤖 AI: Trying to spend $1000...")
	return core.NewEnvelope(map[string]int{"amount": 1000}), nil
}

func (a *StubbornAI) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "stubborn_ai",
		Type: "mock_llm",
	}
}

func main() {
	ctx := context.Background()

	// Handle running from root or from subdirectory
	policyPath := "examples/semantic_feedback/policy.dl"
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		if _, err := os.Stat("policy.dl"); err == nil {
			policyPath = "policy.dl"
		}
	}

	// 1. Initialize Manglekit Client with the policy
	client := manglekit.Must(manglekit.NewClient(ctx, manglekit.WithPolicyPath(policyPath)))

	// 2. Register the Mock AI Action
	// We wrap it in Protect() so the policy engine runs on its output.
	aiAction := &StubbornAI{}
	client.RegisterAction("stubborn_ai", client.Protect(aiAction))

	fmt.Println("🎬 Starting Semantic Feedback Demo (Teacher-Student Protocol)...")
	fmt.Println("---------------------------------------------------------------")

	// 3. Execute the loop
	// We use ExecuteByName which handles the retry loop when PolicyViolationError occurs.
	result, err := client.ExecuteByName(ctx, "stubborn_ai", map[string]string{"instruction": "submit budget"}, manglekit.WithSessionID("demo-session"))
	if err != nil {
		log.Fatalf("❌ Execution failed: %v", err)
	}

	// 4. Print final success
	fmt.Println("---------------------------------------------------------------")
	fmt.Printf("✅ Final Result: %v\n", result.Payload)
}
