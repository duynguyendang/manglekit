# Manglekit Agent Guide: The "Liquid Software" Standard

**Target Audience:** Human Developers & AI Coding Agents.
**Goal:** Build autonomous agents using Manglekit correctly without breaking the architecture.

-----

## 🛑 The 4 Iron Rules (READ THIS FIRST)

If you are an AI Coding Agent, you **MUST** follow these rules. Violations will cause architecture rot.

### 1\. YOU ARE A USER, NOT A MAINTAINER

  * **FORBIDDEN:** Do **NOT** modify files in `core/`, `sdk/`, `adapters/`, or `internal/`.
  * **ALLOWED:** You may only create/edit files in `cmd/`, `examples/`, or your own application folders.
  * **Reasoning:** The framework is the "Body" (Static Binary). You are writing the "Behavior". Do not try to perform surgery on the body; just use the tools provided.

### 2\. NO Leaky Abstractions

  * **NEVER** import `github.com/firebase/genkit` or `net/http` in your `main.go`.
  * **NEVER** perform raw JSON parsing (`json.Unmarshal`) or string trimming (`strings.Trim`) in `main.go`.
  * **ALWAYS** wrap AI logic inside `adapters/ai`. The `main` package should only see `sdk.TextGenerator` interfaces.

### 3\. NO Hardcoded Logic

  * **NEVER** hardcode System Prompts inside Go variables. Load them from files or use constants.
  * **NEVER** hardcode API Keys. Use `os.Getenv` passed into adapters.

### 4\. ALWAYS Protect

  * Every Action must be wrapped in `client.Supervise(action)`.
  * Unprotected actions are illegal in Manglekit. They bypass the Datalog Policy safety layer.

-----

## 🏗️ The Blueprint: How to Build an Agent

To build a Manglekit Agent, follow this 5-step process:

### Step 1: Define the Data Contract

Define the **Input** (what the user asks) and the **Output** (what the AI returns). Use Go Structs with JSON tags.

```go
type BudgetProposal struct {
    Category string `json:"category"`
    Amount   int    `json:"amount"`
}
```

### Step 2: Initialize the Body (Core)

Initialize the Manglekit client. This loads the "Genes" (Policy Rules).

```go
// "policy.dl" contains the logic: allow/deny rules.
client := manglekit.Must(manglekit.NewClient(ctx, manglekit.WithBlueprintPath("policy.dl")))
```

### Step 3: Initialize the Brain (Adapter)

Use the pre-built adapters in `adapters/ai`. Do not initialize raw SDKs here.

```go
// Clean, one-line initialization. No Genkit/OpenAI imports visible here.
aiModel := ai.NewGemini(os.Getenv("GEMINI_API_KEY"), "gemini-1.5-flash")
```

### Step 4: Define the Skill (Action)

Create an Action that maps Input to Output. Use `manglekit.Define` to encapsulate the logic.

  * **Tip:** Use `ai.GenerateStruct[T]` to handle prompt engineering, JSON schema generation, and self-correction loops automatically.

<!-- end list -->

```go
budgetAction := manglekit.Define("budget_agent", func(ctx context.Context, req string) (BudgetProposal, error) {
    return ai.GenerateStruct[BudgetProposal](
        ctx,
        aiModel,
        req, // User Input
        "You are a finance assistant. Output JSON.", // System Prompt
    )
})
```

### Step 5: Register & Execute

Register the action with the `Protect` wrapper. Then execute it by name.

```go
client.RegisterAction("budget_agent", client.Supervise(budgetAction))
result, err := client.ExecuteByName(ctx, "budget_agent", "Buy coffee")
```

-----

## 📝 The Golden Template (Copy-Paste This)

Use this skeleton for all new examples.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/adapters/ai" // The ONLY AI import allowed
	"github.com/joho/godotenv"
)

// 1. Data Contract
type MyResponse struct {
	Summary string `json:"summary"`
	Score   int    `json:"score"`
}

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	// 2. Core Engine
	client := manglekit.Must(manglekit.NewClient(ctx, manglekit.WithBlueprintPath("policy.dl")))

	// 3. AI Adapter
	// Supported: ai.NewGemini, ai.NewOpenAI (future)
	model := ai.NewGemini(os.Getenv("GEMINI_API_KEY"), "gemini-1.5-flash")

	// 4. Action Definition
	// We map string (User Request) -> MyResponse (Struct)
	myAgent := manglekit.Define("my_agent", func(ctx context.Context, input string) (MyResponse, error) {
		return ai.GenerateStruct[MyResponse](
			ctx,
			model,
			input,
			"You are a helpful assistant. Output valid JSON.",
		)
	})

	// 5. Registration (With Protection)
	client.RegisterAction("my_agent", client.Supervise(myAgent))

	// 6. Execution
	res, err := client.ExecuteByName(ctx, "my_agent", "Analyze this project...")
	if err != nil {
		log.Fatalf("❌ Policy Blocked: %v", err)
	}

	// 7. Result
	data := res.Payload.(MyResponse)
	fmt.Printf("✅ Result: %+v\n", data)
}
```

-----

## 🧠 Advanced Concepts

### The Feedback Loop (Self-Correction)

If `client.Supervise` blocks an action (via Datalog Policy), Manglekit automatically injects the violation error back into `ai.GenerateStruct`.

  * **You don't need to write loop logic.**
  * The `ai` adapter handles the retry and error injection automatically.

### Policy Files (`.dl`)

Your agent's behavior is controlled by `policy.dl`.

  * **Example:** `deny :- amount > 1000.`
  * Changing this file updates the agent's behavior instantly without recompiling the Go binary.

-----

**End of Guide.**